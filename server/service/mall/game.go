package mall

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/mall"
	"github.com/flipped-aurora/gin-vue-admin/server/model/mall/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// sha256Re 64 位小写十六进制
var sha256Re = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ============================ 游戏 ============================

type GameService struct{}

// List 游戏列表
func (s *GameService) List(ctx context.Context, info request.GameSearch) (list []mall.Game, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.Game{})

	if info.GameID != "" {
		db = db.Where("game_id LIKE ?", "%"+info.GameID+"%")
	}
	if info.Name != "" {
		db = db.Where("name LIKE ?", "%"+info.Name+"%")
	}
	if info.Kind != "" {
		db = db.Where("kind = ?", info.Kind)
	}
	if info.Status != "" {
		db = db.Where("status = ?", info.Status)
	}

	if err = db.Count(&total).Error; err != nil {
		return
	}
	if info.PageSize == 0 {
		info.PageSize = 20
	}
	err = db.Order("sort DESC, id ASC").
		Scopes(paginate(info.Page, info.PageSize)).
		Find(&list).Error
	return
}

// Update 更新游戏（运营改价 / 上下架）
//
// 只允许改**运营参数**，不允许改名字/图标/文案 ——
// 那些是游戏包自己的 manifest 决定的，后台改了会和包里的声明对不上，
// 而 C 端显示的是包里的那份（界面会「看起来没生效」）。
func (s *GameService) Update(ctx context.Context, req request.UpdateGameReq) error {
	fields := map[string]interface{}{}
	if req.PricePoints != nil {
		if *req.PricePoints < 0 {
			return errors.New("价格不能为负")
		}
		fields["price_points"] = *req.PricePoints
	}
	if req.PriceDiamond != nil {
		if *req.PriceDiamond < 0 {
			return errors.New("价格不能为负")
		}
		fields["price_diamond"] = *req.PriceDiamond
	}
	if req.Status != nil {
		switch *req.Status {
		case "available", "coming_soon", "offline":
			fields["status"] = *req.Status
		default:
			return errors.New("状态只能是 available / coming_soon / offline")
		}
	}
	if req.DefaultOwned != nil {
		fields["default_owned"] = *req.DefaultOwned
	}
	if req.Sort != nil {
		fields["sort"] = *req.Sort
	}
	if len(fields) == 0 {
		return nil
	}
	return global.GVA_DB.WithContext(ctx).Model(&mall.Game{}).
		Where("game_id = ?", req.GameID).Updates(fields).Error
}

// Options 游戏下拉选项（给版本发布页用）
func (s *GameService) Options(ctx context.Context) (list []mall.Game, err error) {
	err = global.GVA_DB.WithContext(ctx).Model(&mall.Game{}).
		Select("game_id, name, emoji, status").Order("game_id ASC").Find(&list).Error
	return
}

// ============================ 游戏版本 ============================

type GameReleaseService struct{}

// List 版本列表
func (s *GameReleaseService) List(ctx context.Context, info request.GameReleaseSearch) (list []mall.GameRelease, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.GameRelease{})

	if info.GameID != "" {
		db = db.Where("game_id = ?", info.GameID)
	}
	if info.Channel != "" {
		db = db.Where("channel = ?", info.Channel)
	}
	if info.OnlyActive {
		db = db.Where("active = ?", true)
	}

	if err = db.Count(&total).Error; err != nil {
		return
	}
	if info.PageSize == 0 {
		info.PageSize = 20
	}
	err = db.Order("published_at DESC, id DESC").
		Scopes(paginate(info.Page, info.PageSize)).
		Find(&list).Error
	return
}

// Publish 发布版本。
//
// ## 事务边界（与平台端同一套语义）
//
// 「把旧的置 non-active」和「插入新版本」**必须原子** ——
// 拆开会出现「两个 active」或者「一个 active 都没有」的瞬间：
// 前者让客户端随机拿到两个版本之一，后者让所有下载 404。
//
// ## 为什么后台要重做一遍而不是调平台的 HTTP 接口
//
// 因为后台和平台**共用同一个库**（见 model/mall 包注释），
// 直接操作更简单，也避免后台依赖平台是否在线。
// 代价是这套规则有**两份实现** —— 所以校验逻辑必须完全一致，
// 任何一边改了都要同步改另一边。
func (s *GameReleaseService) Publish(ctx context.Context, req request.PublishReleaseReq) (rel *mall.GameRelease, err error) {
	// SHA-256 格式（不做补零/截断这类宽容匹配）
	if !sha256Re.MatchString(req.SHA256) {
		return nil, errors.New("sha256 必须是 64 位小写十六进制")
	}
	// 资源根必须以 / 结尾
	if req.AssetBase != "" && req.AssetBase[len(req.AssetBase)-1] != '/' {
		return nil, errors.New("asset_base 必须以 / 结尾")
	}
	// 游戏必须存在
	var game mall.Game
	if err = global.GVA_DB.WithContext(ctx).Where("game_id = ?", req.GameID).First(&game).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("游戏不存在：" + req.GameID)
		}
		return nil, err
	}

	channel := req.Channel
	if channel == "" {
		channel = "stable"
	}
	minHostApi := req.MinHostApi
	if minHostApi <= 0 {
		minHostApi = 1
	}

	// 版本重复检查
	var count int64
	if err = global.GVA_DB.WithContext(ctx).Model(&mall.GameRelease{}).
		Where("game_id = ? AND channel = ? AND version = ?", req.GameID, channel, req.Version).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("该版本已发布：" + req.Version)
	}

	now := nowUnix()
	rel = &mall.GameRelease{
		GameID:       req.GameID,
		Version:      req.Version,
		Channel:      channel,
		URL:          req.URL,
		SHA256:       req.SHA256,
		Size:         req.Size,
		AssetBase:    req.AssetBase,
		MinHostApi:   minHostApi,
		Active:       true,
		Forced:       req.Forced,
		ReleaseNotes: req.ReleaseNotes,
		PublishedAt:  now,
	}
	rel.CreatedAt = now
	rel.UpdatedAt = now

	// 事务边界：「旧的置 non-active」和「插入新版本」必须原子 ——
	// 拆开会出现「两个 active」或「一个 active 都没有」的瞬间。
	err = global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&mall.GameRelease{}).
			Where("game_id = ? AND channel = ? AND active = ?", req.GameID, channel, true).
			Update("active", false).Error; e != nil {
			return e
		}
		return tx.Create(rel).Error
	})
	if err != nil {
		global.GVA_LOG.Error("发布游戏版本失败", zap.Error(err))
		return nil, err
	}
	return rel, nil
}

// Delete 删除版本
//
// ★ 转发给平台，**不要直接 `GVA_DB.Delete`**：两边共用同一个库，后台直接写
// 一秒钟就生效，但平台内存里的目录缓存（默认 300 秒）不会知道 ——
// 于是「删了」之后客户端最长还能再看到它 5 分钟。理由详见 game_release_control.go。
func (s *GameReleaseService) Delete(ctx context.Context, id uint) error {
	code, msg, err := s.forwardToPlatform(
		ctx, "/api/admin/game/release/delete", map[string]any{"release_id": id},
	)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("平台拒绝（%d）：%s", code, msg)
	}
	return nil
}

// SetActive 上架 / 下架一个版本（active=false 即下架）。同样转发给平台。
func (s *GameReleaseService) SetActive(ctx context.Context, id uint, active bool) error {
	code, msg, err := s.forwardToPlatform(
		ctx, "/api/admin/game/release/set-active",
		map[string]any{"release_id": id, "active": active},
	)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("平台拒绝（%d）：%s", code, msg)
	}
	return nil
}

// Deactivate 下架版本（active = false）
func (s *GameReleaseService) Deactivate(ctx context.Context, id uint) error {
	return s.SetActive(ctx, id, false)
}

// ============================ 平台用户与钱包 ============================

type PlatformUserService struct{}

// List C 端用户列表
func (s *PlatformUserService) List(ctx context.Context, info request.PlatformUserSearch) (rows []mall.PlatformUserRow, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.PlatformUser{})

	if info.Mobile != "" {
		db = db.Where("mobile LIKE ?", "%"+info.Mobile+"%")
	}
	if info.Nickname != "" {
		db = db.Where("nickname LIKE ?", "%"+info.Nickname+"%")
	}
	if info.Status != nil {
		db = db.Where("status = ?", *info.Status)
	}
	/* 账号类型：游客（App 一启动就自动建的临时账号）还是正式账号（绑了手机号）。
	   ⚠️ 传 false 而不是写 `is_guest = 0`：SQLite 里这列是 0/1，
	     换到 MySQL 的 tinyint(1) 也一样，但写成布尔参数两边都不用改。 */
	if info.IsGuest != nil {
		db = db.Where("is_guest = ?", *info.IsGuest == 1)
	}

	if err = db.Count(&total).Error; err != nil {
		return
	}
	if info.PageSize == 0 {
		info.PageSize = 20
	}

	var users []mall.PlatformUser
	if err = db.Order("id DESC").Scopes(paginate(info.Page, info.PageSize)).Find(&users).Error; err != nil {
		return
	}

	rows = make([]mall.PlatformUserRow, 0, len(users))
	if len(users) == 0 {
		return rows, total, nil
	}

	/* 余额一次补齐（`uid IN (...)`），不要逐行查 ——
	   一页 20 行就是 20 次往返，而且页面越翻越慢，看起来像「后台卡」。 */
	uids := make([]uint, 0, len(users))
	for _, u := range users {
		uids = append(uids, u.ID)
	}
	var wallets []mall.Wallet
	if err = global.GVA_DB.WithContext(ctx).Where("uid IN ?", uids).Find(&wallets).Error; err != nil {
		return
	}
	byUID := make(map[uint]mall.Wallet, len(wallets))
	for _, w := range wallets {
		byUID[w.UID] = w
	}
	/* 家长锁：同样一次批量查 `user_settings`。
	   ⚠️ 表可能**还不存在**（老库没跑过平台的 AutoMigrate）——
	      这时不能整个列表跟着失败，而是把 `pin_known` 置 false，
	      界面上显示「—」。把「不知道」画成「没设置」是在撒谎。 */
	pinOf, pinKnown := pinStates(ctx, uids)

	for _, u := range users {
		row := mall.PlatformUserRow{PlatformUser: u, PinKnown: pinKnown}
		if w, ok := byUID[u.ID]; ok {
			row.Points, row.Coupon, row.Diamond = w.Points, w.Coupon, w.Diamond
		}
		if pinKnown {
			row.HasPin = pinOf[u.ID]
		}
		rows = append(rows, row)
	}
	return rows, total, nil
}

/* pinStates 批量判断「这一页里谁设了家长锁」。
 *
 * 返回 (uid → 有没有 PIN, 这次到底查到没有)。
 * `user_settings.payload` 是客户端定义的整份 JSON，后台只取 `parentPin` 一个键，
 * **不返回它的值**。 */
func pinStates(ctx context.Context, uids []uint) (map[uint]bool, bool) {
	out := make(map[uint]bool, len(uids))
	var settings []mall.UserSettings
	err := global.GVA_DB.WithContext(ctx).
		Select("uid", "payload").
		Where("uid IN ?", uids).
		Find(&settings).Error
	if err != nil {
		// 老库没有这张表：报一次就够，别把整个列表拖垮
		return out, false
	}
	for _, s := range settings {
		var m map[string]any
		if json.Unmarshal([]byte(s.Payload), &m) != nil {
			continue
		}
		if pin, ok := m["parentPin"].(string); ok && pin != "" {
			out[s.UID] = true
		}
	}
	return out, true
}

// Detail 用户详情（含钱包）
func (s *PlatformUserService) Detail(ctx context.Context, uid uint) (user *mall.PlatformUser, wallet *mall.Wallet, err error) {
	user = &mall.PlatformUser{}
	if err = global.GVA_DB.WithContext(ctx).Where("id = ?", uid).First(user).Error; err != nil {
		return nil, nil, err
	}
	wallet = &mall.Wallet{}
	err = global.GVA_DB.WithContext(ctx).Where("uid = ?", uid).First(wallet).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		wallet = &mall.Wallet{UID: uid}
		err = nil
	}
	return user, wallet, err
}

// Ledger 用户账本
func (s *PlatformUserService) Ledger(ctx context.Context, uid uint, page, pageSize int) (list []mall.LedgerEntry, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.LedgerEntry{}).Where("uid = ?", uid)
	if err = db.Count(&total).Error; err != nil {
		return
	}
	if pageSize == 0 {
		pageSize = 30
	}
	err = db.Order("id DESC").Scopes(paginate(page, pageSize)).Find(&list).Error
	return
}

// AdjustBalance 后台调整余额。
//
// ## 为什么必须走账本（而不是直接 UPDATE wallet）
//
// 平台的不变量是「**余额 = 账本 delta 之和**」。直接改 wallet 会破坏它，
// 而且**不报错** —— 症状要等到某次对账才暴露：
// 「钱多了，但没有对应的流水」。
//
// 所以这里模拟平台的 `ApplyChange`：行锁 + 版本号 + 同时写账本。
//
// ## 实现要点（与平台端逐条对齐）
//
//  1. 事务内**先加锁读钱包**（`FOR UPDATE`）
//  2. 幂等键：后台调整也用 `manual_adjust:{uid}:{时间戳}` 之类，
//     但**不强制** —— 后台是人工操作，重复点两次就是真的加两次
//  3. 版本号乐观锁：`WHERE version = ?`，不匹配就重试/报错
//  4. 写账本时带 `balance_after` 快照
func (s *PlatformUserService) AdjustBalance(ctx context.Context, req request.AdjustBalanceReq, operator string) error {
	if req.Delta == 0 {
		return errors.New("变动量不能为 0")
	}

	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 加锁读钱包
		var w mall.Wallet
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", req.UID).First(&w).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			now := nowUnix()
			// ⚠️ BaseModel 是**嵌入**的，Go 不允许在结构体字面量里
			// 直接写提升字段（CreatedAt / UpdatedAt），必须分行赋值。
			w = mall.Wallet{UID: req.UID}
			w.CreatedAt = now
			w.UpdatedAt = now
			if e := tx.Create(&w).Error; e != nil {
				return e
			}
			if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("uid = ?", req.UID).First(&w).Error; e != nil {
				return e
			}
		} else if err != nil {
			return err
		}

		// 2. 算新余额
		cur := w.Points
		col := "points"
		switch req.Currency {
		case "coupon":
			cur, col = w.Coupon, "coupon"
		case "diamond":
			cur, col = w.Diamond, "diamond"
		}
		newBalance := cur + req.Delta
		if newBalance < 0 {
			return ErrInsufficientBalance
		}

		// 3. 带版本号更新（并发写同一钱包时只会有一个人成功）
		res := tx.Model(&mall.Wallet{}).
			Where("uid = ? AND version = ?", req.UID, w.Version).
			Updates(map[string]interface{}{
				col:          newBalance,
				"version":    w.Version + 1,
				"updated_at": nowUnix(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrVersionConflict
		}

		// 4. 写账本（**这一步不能省**）
		now := nowUnix()
		entry := &mall.LedgerEntry{
			UID:          req.UID,
			Currency:     req.Currency,
			Delta:        req.Delta,
			BalanceAfter: newBalance,
			Source:       "manual_adjust",
			Memo:         fmt.Sprintf("[后台] %s：%s", operator, req.Memo),
		}
		entry.CreatedAt = now
		entry.UpdatedAt = now
		return tx.Create(entry).Error
	})
}

/* ResetPassword 后台重置 C 端用户的登录密码，返回**一次性的**新密码。
 *
 * ## 它解决的是什么
 *
 * 生产环境还没有短信通道（平台的 `DevSmsSender` 在 `IsProduction()` 下直接
 * 返回错误），所以 C 端「忘记密码」那条路走不通。用户忘了密码 = 永久进不去，
 * 而他的积分、兑换券、农场、存档全在那个账号里。这是唯一的兜底。
 *
 * ## 三条必须做的事
 *
 *  1. **写 bcrypt 哈希**，不能写明文 —— 平台端 `Login` 是
 *     `bcrypt.CompareHashAndPassword` 比的，写明文等于密码永远不对。
 *  2. **撤掉所有 refresh token**（`user_token.revoked = 1`）。
 *     不撤的话，旧设备手里的刷新令牌还能续 7 天 ——
 *     而「要重置密码」的场景里，有一半是「账号被别人拿走了」。
 *     ⚠️ 旧 **access token** 仍然最多还能用 2 小时（JWT 无状态，
 *     平台端没有给单条 access 做黑名单）。这一点必须在工单里说清楚。
 *  3. **写 `updated_at`**：`BaseModel` 是 `autoUpdateTime:false`，
 *     不手写就是 0（1970），后台列表上会显示成「1970-01-01」。
 */
func (s *PlatformUserService) ResetPassword(ctx context.Context, req request.ResetPasswordReq, operator string) (string, error) {
	var u mall.PlatformUser
	if err := global.GVA_DB.WithContext(ctx).Where("id = ?", req.UID).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("用户不存在")
		}
		return "", err
	}

	/* 游客没有密码可重置：他连手机号都没有，设了密码也没法登录
	   （登录要手机号）。真正的做法是让他先在 App 里绑定手机号。 */
	if u.IsGuest {
		return "", errors.New("这是临时账号（游客），还没有手机号；请让用户在 App 里点「保存进度」绑定手机号")
	}
	if u.Mobile == "" {
		return "", errors.New("这个账号没有手机号，重置了也登不进来；请先让用户绑定手机号")
	}

	pwd, err := newTempPassword()
	if err != nil {
		return "", err
	}
	hash := utils.BcryptHash(pwd)

	err = global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&mall.PlatformUser{}).Where("id = ?", req.UID).
			Updates(map[string]interface{}{
				"password":   hash,
				"updated_at": nowUnix(),
			}).Error; e != nil {
			return e
		}
		/* 旧设备的刷新令牌全部作废。`revoked` 是 bool 列（SQLite 存 0/1），
		   用 `= ?` 传 false 而不是写 `revoked = 0`，免得换库时又踩一次。 */
		return tx.Table("user_token").
			Where("uid = ? AND revoked = ?", req.UID, false).
			Update("revoked", true).Error
	})
	if err != nil {
		return "", err
	}

	global.GVA_LOG.Info("admin reset user password",
		zap.Uint("uid", req.UID),
		zap.String("operator", operator),
		zap.String("memo", req.Memo))
	return pwd, nil
}

/* newTempPassword 生成一次性密码。
 *
 * 长度 8，字母表**故意**去掉 `0/O`、`1/l/I`：这串要被人念出来 / 抄在纸上
 * 交给家长，长得像的字符只会让人登不进去，然后以为「后台给的是错的」。
 * 8 位 × 31 个字符 ≈ 40 bit，对一个「用一次就改」的临时密码够用。
 */
func newTempPassword() (string, error) {
	const alphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	const n = 8
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out), nil
}

// paginate 分页 scope
func paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}
		return db.Offset((page - 1) * pageSize).Limit(pageSize)
	}
}
