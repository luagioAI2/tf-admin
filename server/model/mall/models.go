// Package mall 是后台的**业务模块**，管理 tf-platform-api 的业务数据。
//
// # 为什么后台和 API 共用同一个库
//
// 后台直接读写平台自己的表（`game` / `user` / `redeem_item` / `wish_order` …），
// 不另建一套「后台表 + 同步机制」。
//
// 理由：一套数据只有一份才是对的。如果后台自己存一份「游戏列表」，
// 那么运营在后台改了一个价格、而 C 端读到的是另一份 —— 这种错**不报错**，
// 只会表现为「我明明改了怎么没生效」。
//
// # 两条必须遵守的规矩
//
//  1. **每张表都要显式写 `TableName()`**。
//     tf-platform-api 用的是 `SingularTable: true`（表名 `game` 单数），
//     而 gin-vue-admin 用的是 GORM 默认复数命名（`games`）。
//     不显式指定就会读到一张不存在的表，报「no such table」——
//     或者更糟：读到一张恰好同名的空表，然后「查不到数据」。
//
//  2. **这里的模型不参与 AutoMigrate**。
//     表是平台的资产，结构由平台自己的 `Models()` 决定。
//     后台如果也去 migrate，两个进程会互相改表结构，
//     而且后台的字段定义只要少一个，就可能被 migrate 掉一列。
//     见 `initialize/gorm_biz.go` 的注释。
package mall

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 与 tf-platform-api 的 entities.BaseModel **逐字段对齐**。
//
// ⚠️ 时间戳是 **int64**（Unix 秒），不是 time.Time ——
// 平台那边就是这么存的。用 time.Time 会读出零值，
// 而且**不报错**（GORM 会尝试解析失败后给零值）。
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt int64          `gorm:"autoCreateTime:false" json:"created_at"`
	UpdatedAt int64          `gorm:"autoUpdateTime:false" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Timestamp 便于前端格式化
func (b *BaseModel) Timestamp() string {
	if b.CreatedAt == 0 {
		return ""
	}
	return time.Unix(b.CreatedAt, 0).Format("2006-01-02 15:04:05")
}

// ============================ 游戏 ============================

// Game 游戏（对应 `game` 表）
type Game struct {
	BaseModel

	GameID       string `gorm:"column:game_id" json:"game_id"`
	Name         string `gorm:"column:name" json:"name"`
	Emoji        string `gorm:"column:emoji" json:"emoji"`
	Tagline      string `gorm:"column:tagline" json:"tagline"`
	Kind         string `gorm:"column:kind" json:"kind"`
	Tint         string `gorm:"column:tint" json:"tint"`
	Status       string `gorm:"column:status" json:"status"`
	PricePoints  int64  `gorm:"column:price_points" json:"price_points"`
	PriceDiamond int64  `gorm:"column:price_diamond" json:"price_diamond"`
	DefaultOwned bool   `gorm:"column:default_owned" json:"default_owned"`
	EntryURL     string `gorm:"column:entry_url" json:"entry_url"`
	CoverURL     string `gorm:"column:cover_url" json:"cover_url"`
	AgeHint      string `gorm:"column:age_hint" json:"age_hint"`
	SessionHint  string `gorm:"column:session_hint" json:"session_hint"`
	Sort         int    `gorm:"column:sort" json:"sort"`
}

func (Game) TableName() string { return "game" }

// ============================ 游戏版本 ============================

// GameRelease 游戏版本发布（对应 `game_release` 表）
type GameRelease struct {
	BaseModel

	GameID        string `gorm:"column:game_id" json:"game_id"`
	Version       string `gorm:"column:version" json:"version"`
	Channel       string `gorm:"column:channel" json:"channel"`
	URL           string `gorm:"column:url" json:"url"`
	SHA256        string `gorm:"column:sha256" json:"sha256"`
	Size          int64  `gorm:"column:size" json:"size"`
	AssetBase     string `gorm:"column:asset_base" json:"asset_base"`
	MinHostApi    int    `gorm:"column:min_host_api" json:"min_host_api"`
	Active        bool   `gorm:"column:active" json:"active"`
	Forced        bool   `gorm:"column:forced" json:"forced"`
	ReleaseNotes  string `gorm:"column:release_notes" json:"release_notes"`
	DownloadCount int64  `gorm:"column:download_count" json:"download_count"`
	PublishedAt   int64  `gorm:"column:published_at" json:"published_at"`
	PublishedBy   uint   `gorm:"column:published_by" json:"published_by"`
}

func (GameRelease) TableName() string { return "game_release" }

// ============================ 平台用户 ============================

// PlatformUser C 端用户（对应 `user` 表）。
//
// ⚠️ 字段名全是 C 端的口径（`nickname` / `avatar` / `mobile`），
// 不要和后台自己的 `sys_users` 混。两者是完全不同的两套账号体系：
//   - `sys_users`     = 后台管理员（运营人员）
//   - `user`（这里）  = C 端用户，一人一行；家长和孩子共用同一行
//     （家长只是多设了一个 4 位密码，没有第二种账号）
type PlatformUser struct {
	BaseModel

	Username       string `gorm:"column:username" json:"username"`
	Mobile         string `gorm:"column:mobile" json:"mobile"`
	MobileVerified bool   `gorm:"column:mobile_verified" json:"mobile_verified"`
	Nickname       string `gorm:"column:nickname" json:"nickname"`
	Avatar         string `gorm:"column:avatar" json:"avatar"`
	Gender         uint8  `gorm:"column:gender" json:"gender"`
	Status         int    `gorm:"column:status" json:"status"`
	Language       string `gorm:"column:language" json:"language"`
	IsGuest        bool   `gorm:"column:is_guest" json:"is_guest"`
	RegisterIP     string `gorm:"column:register_ip" json:"register_ip"`
	LastLoginIP    string `gorm:"column:last_login_ip" json:"last_login_ip"`
	LastLoginAt    int64  `gorm:"column:last_login_at" json:"last_login_at"`
}

func (PlatformUser) TableName() string { return "user" }

// Wallet 钱包（对应 `wallet` 表）
type Wallet struct {
	BaseModel

	UID     uint  `gorm:"column:uid" json:"uid"`
	Points  int64 `gorm:"column:points" json:"points"`
	Coupon  int64 `gorm:"column:coupon" json:"coupon"`
	Diamond int64 `gorm:"column:diamond" json:"diamond"`
	Version int64 `gorm:"column:version" json:"-"`
}

func (Wallet) TableName() string { return "wallet" }

// PlatformUserRow 用户列表的一行：用户 + 钱包余额。
//
// 余额**不在 `user` 表里**（在 `wallet`），而列表页最想先看的就是余额。
// 逐行查钱包就是 N+1；这里由 `PlatformUserService.List` 用一次
// `uid IN (...)` 补齐。
//
// ⚠️ 没有钱包行的用户（从没入过账）补 0 —— 那是「还没花过」，
//
//	不是「查不到」。判据别写成「余额为 0 就当成异常」。
type PlatformUserRow struct {
	PlatformUser
	Points  int64 `json:"points"`
	Coupon  int64 `json:"coupon"`
	Diamond int64 `json:"diamond"`

	/* 有没有设家长锁。
	 *
	 * ⚠️ 这不是「家长账号」—— 一个账号就是一个家庭，家长和孩子**共用同一个
	 *    账号**，家长区只是被 `settings.parentPin` 这道锁挡着（见
	 *    `docs/online-version.md` §7.20）。所以运营能看到的只有「这把锁设没设」，
	 *    没有「家长列表」这种列表。
	 *
	 * PIN 本身住在 `user_settings.payload` 那段 JSON 里，这里只读出来判断
	 * 有没有值，**不返回 PIN**（后台不需要它，泄露面越小越好）。
	 */
	HasPin bool `json:"has_pin"`
	/* PinKnown=false 表示没查到（`user_settings` 表还不存在 —— 老库）。
	   界面上显示「—」而不是「未设置」：把「不知道」画成「没有」是在撒谎。 */
	PinKnown bool `json:"pin_known"`
}

// UserSettings 家长设置（对应 `user_settings` 表，tf-platform-api 建的）。
//
// 后台只读：整份 JSON 由客户端定义，后台**不解释**字段（除了判断 PIN 有没有）。
type UserSettings struct {
	UID     uint   `gorm:"column:uid"`
	Payload string `gorm:"column:payload"`
}

func (UserSettings) TableName() string { return "user_settings" }

// LedgerEntry 账本（对应 `ledger_entry` 表）
type LedgerEntry struct {
	BaseModel

	UID          uint   `gorm:"column:uid" json:"uid"`
	Currency     string `gorm:"column:currency" json:"currency"`
	Delta        int64  `gorm:"column:delta" json:"delta"`
	BalanceAfter int64  `gorm:"column:balance_after" json:"balance_after"`
	Source       string `gorm:"column:source" json:"source"`
	RefID        string `gorm:"column:ref_id" json:"ref_id"`
	Memo         string `gorm:"column:memo" json:"memo"`
}

func (LedgerEntry) TableName() string { return "ledger_entry" }

// ============================ 兑换商城 ============================

// RedeemItem 兑换商品（对应 `redeem_item` 表）
type RedeemItem struct {
	BaseModel

	UID         uint   `gorm:"column:uid" json:"uid"`
	ItemID      string `gorm:"column:item_id" json:"item_id"`
	Name        string `gorm:"column:name" json:"name"`
	Emoji       string `gorm:"column:emoji" json:"emoji"`
	Category    string `gorm:"column:category" json:"category"`
	Price       int64  `gorm:"column:price" json:"price"`
	Note        string `gorm:"column:note" json:"note"`
	LimitPerDay int    `gorm:"column:limit_per_day" json:"limit_per_day"`
	Once        bool   `gorm:"column:once" json:"once"`
	Stock       int    `gorm:"column:stock" json:"stock"`
	Archived    bool   `gorm:"column:archived" json:"archived"`
	Sort        int    `gorm:"column:sort" json:"sort"`
}

func (RedeemItem) TableName() string { return "redeem_item" }

// RedeemRecord 兑换记录（对应 `redeem_record` 表）
type RedeemRecord struct {
	BaseModel

	UID                uint   `gorm:"column:uid" json:"uid"`
	RecordID           string `gorm:"column:record_id" json:"record_id"`
	ItemID             string `gorm:"column:item_id" json:"item_id"`
	Name               string `gorm:"column:name" json:"name"`
	Emoji              string `gorm:"column:emoji" json:"emoji"`
	Quantity           int    `gorm:"column:quantity" json:"quantity"`
	Cost               int64  `gorm:"column:cost" json:"cost"`
	CouponPaid         int64  `gorm:"column:coupon_paid" json:"coupon_paid"`
	PointsPaid         int64  `gorm:"column:points_paid" json:"points_paid"`
	PointsBalanceAfter int64  `gorm:"column:points_balance_after" json:"points_balance_after"`
	CouponBalanceAfter int64  `gorm:"column:coupon_balance_after" json:"coupon_balance_after"`
	Fulfilled          bool   `gorm:"column:fulfilled" json:"fulfilled"`
	FulfilledNote      string `gorm:"column:fulfilled_note" json:"fulfilled_note"`
	Cancelled          bool   `gorm:"column:cancelled" json:"cancelled"`
	CancelReason       string `gorm:"column:cancel_reason" json:"cancel_reason"`
}

func (RedeemRecord) TableName() string { return "redeem_record" }

// ============================ 心愿商城 ============================

// WishItem 心愿商品（对应 `wish_item` 表）
type WishItem struct {
	BaseModel

	ItemID      string `gorm:"column:item_id" json:"item_id"`
	Kind        string `gorm:"column:kind" json:"kind"`
	Name        string `gorm:"column:name" json:"name"`
	Emoji       string `gorm:"column:emoji" json:"emoji"`
	Tint        string `gorm:"column:tint" json:"tint"`
	PartnerName string `gorm:"column:partner_name" json:"partner_name"`
	GameID      string `gorm:"column:game_id" json:"game_id"`
	// Pool 盲盒池（JSON 数组字符串）
	Pool       string `gorm:"column:pool" json:"pool"`
	PriceCents int64  `gorm:"column:price_cents" json:"price_cents"`
	// AgeRange 适龄范围（JSON 数组字符串，如 `[6,12]`）。
	// 表里是 `text` 存的 JSON，和 `pool` 一个路子。
	AgeRange string `gorm:"column:age_range" json:"age_range"`
	Note     string `gorm:"column:note" json:"note"`
	Notice   string `gorm:"column:notice" json:"notice"`
	// BuyURL 外链（**只在家长端出现**）
	BuyURL    string `gorm:"column:buy_url" json:"buy_url"`
	BuyLabel  string `gorm:"column:buy_label" json:"buy_label"`
	Archived  bool   `gorm:"column:archived" json:"archived"`
	SortOrder int    `gorm:"column:sort_order" json:"sort_order"`
	// UnlockCount 一次开出几款游戏（只对 blindbox 有意义，1 = 单开）
	//
	// ★ 「已经解锁的不计算」不是靠这个字段，而是靠开奖时先减掉已拥有：
	//   池子里写全部游戏，孩子已有的不占名额。
	// ⚠️ 剩余不足这个数时 C 端会**整单拒绝**，不会「有几款给几款」。
	UnlockCount int `gorm:"column:unlock_count" json:"unlock_count"`
}

func (WishItem) TableName() string { return "wish_item" }

// WishCode 激活码（对应 `wish_code` 表）。
//
// ⚠️ 表里**只有 sha256**，没有明文。明文只在「生成那一瞬间」返回一次 ——
// 打印卡片要用它。丢了只能作废重发，找不回来。
//
// ⚠️ 这张表和 C 端 `tf-platform-api` 共用。哈希口径必须和那边
// `internal/app/service/wish_srv.go` 的 `hashWishCode` **逐字一致**，
// 否则后台生成的码在 App 里全是「没有这个码」，而且**两边都不报错**。
type WishCode struct {
	BaseModel

	// CodeHash 明文码的 sha256（唯一）。`json:"-"`：列表里没有它的用处，
	// 而它是「能反查出码」的唯一线索，不往外送。
	CodeHash string `gorm:"column:code_hash" json:"-"`
	// CodeTail 后 4 位，方便家长核对是哪张卡
	CodeTail string `gorm:"column:code_tail" json:"code_tail"`
	// ItemID 这张码解锁哪件商品（空 = 通用码，不解锁东西）
	ItemID string `gorm:"column:item_id" json:"item_id"`
	// Batch 批次号（运营自己起的名字，如「2026双11-游戏盲盒」）
	Batch string `gorm:"column:batch" json:"batch"`

	Used      bool   `gorm:"column:used" json:"used"`
	UsedByUID uint   `gorm:"column:used_by_uid" json:"used_by_uid"`
	UsedAt    *int64 `gorm:"column:used_at" json:"used_at"`
	// ExpireAt 过期时间（0 = 永不过期）
	ExpireAt int64 `gorm:"column:expire_at" json:"expire_at"`
}

func (WishCode) TableName() string { return "wish_code" }

// WishOffer 家长定价（对应 `wish_offer` 表）
type WishOffer struct {
	BaseModel

	UID          uint   `gorm:"column:uid" json:"uid"`
	ItemID       string `gorm:"column:item_id" json:"item_id"`
	VoucherPrice int64  `gorm:"column:voucher_price" json:"voucher_price"`
	DiamondPrice int64  `gorm:"column:diamond_price" json:"diamond_price"`
	OpenedAt     int64  `gorm:"column:opened_at" json:"opened_at"`
	ClosedAt     int64  `gorm:"column:closed_at" json:"closed_at"`
}

func (WishOffer) TableName() string { return "wish_offer" }

// WishOrder 心愿单（对应 `wish_order` 表）
type WishOrder struct {
	BaseModel

	UID          uint   `gorm:"column:uid" json:"uid"`
	OrderID      string `gorm:"column:order_id" json:"order_id"`
	ItemID       string `gorm:"column:item_id" json:"item_id"`
	Kind         string `gorm:"column:kind" json:"kind"`
	ItemName     string `gorm:"column:item_name" json:"item_name"`
	ItemEmoji    string `gorm:"column:item_emoji" json:"item_emoji"`
	PartnerName  string `gorm:"column:partner_name" json:"partner_name"`
	VoucherPrice int64  `gorm:"column:voucher_price" json:"voucher_price"`
	Status       string `gorm:"column:status" json:"status"`
	VouchersHeld int64  `gorm:"column:vouchers_held" json:"vouchers_held"`
	// PointsHeld 这一单实际扣掉的**积分**。
	//
	// 2026-10-29 起心愿商城可券 + 积分混付（和「兑换」同一套规则），
	// 所以订单上多了这个字段。它和 `VouchersHeld` 是**退款凭据的两半**，
	// 缺一不可 —— 一笔成交扣两种币，退款只退一种的话另一半就凭空蒸发了。
	PointsHeld   int64  `gorm:"column:points_held" json:"points_held"`
	OpenedGameID string `gorm:"column:opened_game_id" json:"opened_game_id"`
	TrackingNo   string `gorm:"column:tracking_no" json:"tracking_no"`
	// Ship 收货信息（**敏感**：后台只在有权限时展示，见 service 层）
	Ship         string `gorm:"column:ship" json:"ship"`
	CommittedAt  *int64 `gorm:"column:committed_at" json:"committed_at"`
	ConfirmedAt  *int64 `gorm:"column:confirmed_at" json:"confirmed_at"`
	DoneAt       *int64 `gorm:"column:done_at" json:"done_at"`
	CancelledAt  *int64 `gorm:"column:cancelled_at" json:"cancelled_at"`
	CancelReason string `gorm:"column:cancel_reason" json:"cancel_reason"`
}

func (WishOrder) TableName() string { return "wish_order" }

// ============================ 任务 ============================

// Task 任务定义（对应 `task` 表）
type Task struct {
	BaseModel

	UID            uint   `gorm:"column:uid" json:"uid"`
	Title          string `gorm:"column:title" json:"title"`
	Note           string `gorm:"column:note" json:"note"`
	Category       string `gorm:"column:category" json:"category"`
	Cycle          string `gorm:"column:cycle" json:"cycle"`
	PlannedMinutes int    `gorm:"column:planned_minutes" json:"planned_minutes"`
	BasePoints     int    `gorm:"column:base_points" json:"base_points"`
	AllowOvertime  bool   `gorm:"column:allow_overtime" json:"allow_overtime"`
	CheckInEnabled bool   `gorm:"column:check_in_enabled" json:"check_in_enabled"`
	Archived       bool   `gorm:"column:archived" json:"archived"`
	Sort           int    `gorm:"column:sort" json:"sort"`
}

func (Task) TableName() string { return "task" }

// TaskInstance 任务实例（对应 `task_instance` 表）
type TaskInstance struct {
	BaseModel

	UID            uint   `gorm:"column:uid" json:"uid"`
	TaskID         uint   `gorm:"column:task_id" json:"task_id"`
	PeriodKey      string `gorm:"column:period_key" json:"period_key"`
	Date           string `gorm:"column:date" json:"date"`
	Title          string `gorm:"column:title" json:"title"`
	Status         string `gorm:"column:status" json:"status"`
	PlannedMinutes int    `gorm:"column:planned_minutes" json:"planned_minutes"`
	ActualMinutes  *int   `gorm:"column:actual_minutes" json:"actual_minutes"`
	EarnedPoints   *int   `gorm:"column:earned_points" json:"earned_points"`
	Quality        string `gorm:"column:quality" json:"quality"`
}

func (TaskInstance) TableName() string { return "task_instance" }

// ============================ 招商 ============================

// MerchantApplication 招商申请（对应 `merchant_application` 表）
//
// C 端（`kid-quest-farm-2d` 心愿页右上角「🤝 招商」）只负责**提交**，
// 审核和 `AdminURL` 回填只能在这里做。用户原话：
//
//	「申请后端会返回后台链接，已经申请后台审核过的 就是直接显示链接」
//
// ⚠️ 唯一索引在 **`(uid, role)`**（`uk_merchant_uid_role`）：
// 同一角色重复提交是**改申请**，不是提第二份。所以后台这一侧
// **不要**做成「新增一条」，只做「改这一条的状态」。
type MerchantApplication struct {
	BaseModel

	UID    uint   `gorm:"column:uid" json:"uid"`
	Role   string `gorm:"column:role" json:"role"`
	Status string `gorm:"column:status" json:"status"`
	// Name 家长昵称 / 店铺名（**服务端从账号补的**，不是用户填的）
	Name string `gorm:"column:name" json:"name"`
	// Contact 手机号或微信（同上，手机号是明文，C 端拿到的是脱敏的）
	Contact string `gorm:"column:contact" json:"contact"`
	Note    string `gorm:"column:note" json:"note"`
	// AdminURL 通过时回填的后台地址。**只有 approved 时 C 端才看得到**
	AdminURL   string `gorm:"column:admin_url" json:"admin_url"`
	ReviewedAt int64  `gorm:"column:reviewed_at" json:"reviewed_at"`
	ReviewedBy uint   `gorm:"column:reviewed_by" json:"reviewed_by"`
	ReviewNote string `gorm:"column:review_note" json:"review_note"`
}

func (MerchantApplication) TableName() string { return "merchant_application" }
