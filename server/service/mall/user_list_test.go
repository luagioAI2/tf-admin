package mall

import (
	"context"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	mallReq "github.com/flipped-aurora/gin-vue-admin/server/model/mall/request"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

/* ============================================================
   平台用户列表：余额必须**一次批量补齐**，缺钱包的补 0
   ------------------------------------------------------------
   这一层错了都不报错，只会「看着像」：
     · 逐行查钱包 ⇒ 一页 20 行 20 次往返（页面越翻越慢，像后台卡）；
     · 缺钱包行就跳过 ⇒ 那个用户的余额是 0 还是「没查到」分不清；
     · `uid IN (...)` 的结果 map 键写错 ⇒ **所有人的余额都是 0**，
       而列表本身完全正常 —— 运营会以为「这些用户一分没有」。

   这里用内存库真跑一遍 SQL，不去连 tf.db。
   ============================================================ */

// newTestDB 建一张最小的 `user` / `wallet`，塞三个用户（只有前两个有钱包行）。
//
// ⚠️ 这里**故意不 AutoMigrate**：项目规矩是这两张表属于 tf-platform-api，
//    后台只读不建（`models.go` 头注释）。测试里手写 CREATE TABLE，
//    是为了「表结构写错」也能被测出来，而不是让 GORM 按模型猜一份。
func newTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:userlist?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存库失败：%v", err)
	}
	/* ⚠️ 同一个进程里 `cache=shared` 的内存库是**共用**的，
	   第二个测试会撞上第一个建好的表（`table user already exists`）。
	   先 DROP 再建，别改成「每个测试一个 DSN」—— 那样表结构各测一份，
	   反而看不出「测试之间数据串了」。 */
	stmts := []string{
		`DROP TABLE IF EXISTS user`,
		`DROP TABLE IF EXISTS wallet`,
		`DROP TABLE IF EXISTS user_settings`,
		`CREATE TABLE user (
			id integer PRIMARY KEY AUTOINCREMENT,
			created_at integer, updated_at integer, deleted_at integer,
			username text, mobile text, mobile_verified integer,
			nickname text, avatar text, gender integer, status integer,
			register_ip text, last_login_ip text, last_login_at integer,
			language text, is_guest integer
		)`,
		`CREATE TABLE wallet (
			id integer PRIMARY KEY AUTOINCREMENT,
			created_at integer, updated_at integer, deleted_at integer,
			uid integer, points integer, coupon integer, diamond integer, version integer
		)`,
		/* 家长锁住在这一张里（整份设置 JSON，平台建的）。
		   uid=1 设了 PIN，uid=2 有行但没设，uid=3 连行都没有。 */
		`CREATE TABLE user_settings (
			id integer PRIMARY KEY AUTOINCREMENT,
			created_at integer, updated_at integer, deleted_at integer,
			uid integer, payload text, size integer, version integer
		)`,
		`INSERT INTO user_settings (uid, payload, size, version) VALUES
			(1, '{"childName":"测试家长","parentPin":"1234"}', 40, 3),
			(2, '{"childName":"第二位"}', 20, 1)`,
		/* uid=3 是**游客**：App 一启动就自动建的临时账号，没有手机号。
		   夹具要造得出「两种账号混在一页」，否则筛游客的测试是空对空。 */
		`INSERT INTO user (id, mobile, nickname, status, created_at, is_guest) VALUES
			(1, '13800138000', '测试家长', 1, 1700000000, 0),
			(2, '13800138001', '第二位', 1, 1700000100, 0),
			(3, '', '小客人', 1, 1700000200, 1)`,
		`INSERT INTO wallet (uid, points, coupon, diamond, version) VALUES
			(1, 219, 43, 0, 7),
			(2, 100, 0, 5, 1)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("准备数据失败：%v\nSQL: %s", err, s)
		}
	}

	old := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = old })
}

func search(page, pageSize int) mallReq.PlatformUserSearch {
	return mallReq.PlatformUserSearch{PageInfo: commonReq.PageInfo{Page: page, PageSize: pageSize}}
}

func TestPlatformUserListAttachesWallet(t *testing.T) {
	newTestDB(t)
	svc := &PlatformUserService{}

	rows, total, err := svc.List(context.Background(), search(1, 10))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d，应该是 3", total)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d，应该是 3", len(rows))
	}
	// 按 id DESC
	if rows[0].ID != 3 || rows[2].ID != 1 {
		t.Fatalf("排序不对：%d, %d, %d", rows[0].ID, rows[1].ID, rows[2].ID)
	}

	byID := map[uint]struct{ points, coupon, diamond int64 }{}
	for _, r := range rows {
		byID[r.ID] = struct{ points, coupon, diamond int64 }{r.Points, r.Coupon, r.Diamond}
	}
	if w := byID[1]; w.points != 219 || w.coupon != 43 || w.diamond != 0 {
		t.Fatalf("uid=1 的余额不对：%+v", w)
	}
	if w := byID[2]; w.points != 100 || w.coupon != 0 || w.diamond != 5 {
		t.Fatalf("uid=2 的余额不对：%+v", w)
	}
	/* ★ 没有钱包行的用户补 0 —— 那是「还没花过」，不是「查不到」。
	   这一条最容易写成「跳过」，然后界面显示 `—` 和 `0` 混在一起。 */
	if w := byID[3]; w.points != 0 || w.coupon != 0 || w.diamond != 0 {
		t.Fatalf("uid=3 没有钱包行，余额应该是 0：%+v", w)
	}

	/* 家长锁：同一个账号「有没有设密码」—— 不是另一类账号。 */
	pin := map[uint]struct{ has, known bool }{}
	for _, r := range rows {
		pin[r.ID] = struct{ has, known bool }{r.HasPin, r.PinKnown}
	}
	if p := pin[1]; !p.has || !p.known {
		t.Fatalf("uid=1 设了 PIN，has_pin 应该 true：%+v", p)
	}
	if p := pin[2]; p.has || !p.known {
		t.Fatalf("uid=2 有设置行但没设 PIN：%+v", p)
	}
	if p := pin[3]; p.has || !p.known {
		t.Fatalf("uid=3 没有设置行：%+v", p)
	}
}

/* 老库没有 `user_settings` 表时：列表照常返回，`pin_known=false`。
   把「不知道」画成「没设置」就是在撒谎 —— 这条守的就是它。 */
func TestPlatformUserListWithoutSettingsTable(t *testing.T) {
	newTestDB(t)
	if err := global.GVA_DB.Exec(`DROP TABLE user_settings`).Error; err != nil {
		t.Fatalf("drop: %v", err)
	}
	svc := &PlatformUserService{}

	rows, total, err := svc.List(context.Background(), search(1, 10))
	if err != nil {
		t.Fatalf("没有设置表时不该失败：%v", err)
	}
	if total != 3 || len(rows) != 3 {
		t.Fatalf("total=%d rows=%d，应该是 3/3", total, len(rows))
	}
	for _, r := range rows {
		if r.PinKnown {
			t.Fatalf("表都不存在，pin_known 不能是 true：%+v", r)
		}
		if r.HasPin {
			t.Fatalf("查不到就不能说「设了」：%+v", r)
		}
	}
}

func TestPlatformUserListFilters(t *testing.T) {
	newTestDB(t)
	svc := &PlatformUserService{}

	// 手机号片段
	info := search(1, 10)
	info.Mobile = "138"
	rows, total, err := svc.List(context.Background(), info)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 || len(rows) != 2 {
		t.Fatalf("按手机号过滤：total=%d rows=%d，应该是 2/2", total, len(rows))
	}

	// 昵称片段
	info = search(1, 10)
	info.Nickname = "小客人"
	_, total, err = svc.List(context.Background(), info)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("按昵称过滤：total=%d，应该是 1", total)
	}

	/* 状态：0 = 禁用。库里三个都是 1，所以应该是 0 条。
	   ⚠️ 这条同时守住「传 0 别被当成『没传』」—— 指针为 nil 才是没传。 */
	zero := 0
	info = search(1, 10)
	info.Status = &zero
	_, total, err = svc.List(context.Background(), info)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 0 {
		t.Fatalf("status=0 过滤：total=%d，应该是 0", total)
	}

	/* 账号类型：游客 vs 正式。
	   ★ 这条守的是「传 0 别被当成没传」—— 0 = 只看正式账号，是个有效值；
	     写成 `if info.IsGuest != nil && *info.IsGuest != 0` 就会静默退化成全量，
	     而界面上「只看正式账号」照样显示全部（游客混在里面，看不出错）。 */
	guest, formal := 1, 0
	info = search(1, 10)
	info.IsGuest = &guest
	rows, total, err = svc.List(context.Background(), info)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ID != 3 {
		t.Fatalf("只看游客：total=%d rows=%+v，应该只有 id=3", total, rows)
	}

	info = search(1, 10)
	info.IsGuest = &formal
	rows, total, err = svc.List(context.Background(), info)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Fatalf("只看正式账号：total=%d，应该是 2", total)
	}
	for _, r := range rows {
		if r.IsGuest {
			t.Fatalf("「只看正式账号」里混进了游客：%+v", r)
		}
	}

	// 分页
	rows, total, err = svc.List(context.Background(), search(2, 2))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("分页时 total 应该还是全量 3，实际 %d", total)
	}
	if len(rows) != 1 || rows[0].ID != 1 {
		t.Fatalf("第 2 页（每页 2 条）应该只剩 id=1，实际 %+v", rows)
	}
}
