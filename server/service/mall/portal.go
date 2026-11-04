package mall

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/mall"
	mallReq "github.com/flipped-aurora/gin-vue-admin/server/model/mall/request"
	system "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Package mall 里的 portal.go 是「家长 / 商家门户」这一半。
//
// # 一句话
//
// 后台只有一个，权限系统也只有一个（GVA 的 `sys_user` + `sys_authority` +
// casbin + 菜单）。家长和商家**不是第二套账号体系**，他们用 **App 的账号**
// 登录，登录时惰性换出一个**影子 `sys_user`**，角色固定为 1001 / 1002。
//
// # 为什么是影子账号，而不是「另做一套 token」
//
// 因为 GVA 的菜单、按钮、接口权限全部是**数据**：
// `sys_authority_menus` 决定左边有哪些菜单，`casbin_rule` 决定哪个接口能调。
// 自己另做一套 token，就要把这两件事各再实现一遍 —— 而它们一旦分叉，
// 症状是「菜单看得见、点下去 403」或者反过来，**两边都不报错**。
// 换成影子账号，这两张表照用，一行判断都不用重写。
//
// # 数据范围不能只靠 casbin
//
// casbin 管的是「这个角色能不能调这个接口」，**管不到「能看谁的数据」**。
// 所以每个家长 / 商家接口都必须**在 service 层再按 uid 过滤一次** ——
// 只靠 casbin 的话，家长 A 拿自己的 token 就能翻家长 B 的订单。
// 这是本文件里最重要的一条，改任何方法前先看它有没有 `uid = ?`。

const (
	// PortalRoleParent 家长门户
	PortalRoleParent = "parent"
	// PortalRoleMerchant 商家门户
	PortalRoleMerchant = "merchant"

	// AuthorityParent 家长角色（`sys_authorities.authority_id`）
	//
	// ⚠️ 1001 / 1002 是**本项目自己占的号**，GVA 默认只有 888 / 9528 / 8881。
	// 改动这里必须同步改 `scripts/seed-portal.py`，否则影子账号会挂到一个
	// 不存在的角色上 —— 表现是登录成功、进去一片空白，**没有任何报错**。
	AuthorityParent uint = 1001
	// AuthorityMerchant 商家角色
	AuthorityMerchant uint = 1002

	// shadowPrefix 影子账号的用户名前缀。`app_{App 用户 id}`
	//
	// ⚠️ 这个前缀是**从 token 反查 App uid 的唯一依据**（见 `AppUIDOf`）。
	// 改它等于让所有已登录的家长 / 商家在下一个请求里失去数据范围 ——
	// 所以它必须和 `scripts/seed-portal.py` 里写的名字一致。
	shadowPrefix = "app_"
)

// PortalService 家长 / 商家门户。
//
// 它读写的是 **tf-platform-api 的表**（和本模块其它 service 一样），
// 所以这里所有 `TableName` / 时间戳（int64 秒）/ 软删口径的坑都适用。
type PortalService struct{}

// ============================ 登录桥 ============================

// appCredential 只取校验需要的几列。
//
// ⚠️ 不能直接查 `mall.PlatformUser`：那个模型**故意没有 Password 字段**
// （后台用户列表不该把哈希带出去）。这里单独 select，别为了省事给它加字段。
type appCredential struct {
	ID       uint   `gorm:"column:id"`
	Password string `gorm:"column:password"`
	Nickname string `gorm:"column:nickname"`
	Status   int    `gorm:"column:status"`
}

var mobileRe = regexp.MustCompile(`^\+?[0-9]{8,20}$`)

// normalizeMobile 与 tf-platform-api 的 `normalizeMobile` **同一口径**：
// 去空格、去横线，保留 `+`。两边不一致的话，App 能登、后台说「查无此人」。
func normalizeMobile(mobile string) string {
	m := strings.TrimSpace(mobile)
	m = strings.ReplaceAll(m, " ", "")
	m = strings.ReplaceAll(m, "-", "")
	if !mobileRe.MatchString(m) {
		return ""
	}
	return m
}

// mobileCandidates 库里可能存的几种写法。
//
// ⚠️ 平台按**精确匹配**查手机号（`user_repo.GetByMobile` 就是 `mobile = ?`），
// 而注册时存的是带国家码的 `+8613800138000`，家长在后台多半只输 `13800138000`。
// 不补这一步，表现是「App 里能登，后台说账号不存在」—— 而账号是同一个。
func mobileCandidates(mobile string) []string {
	if strings.HasPrefix(mobile, "+") {
		return []string{mobile, strings.TrimPrefix(mobile, "+86"), strings.TrimPrefix(mobile, "+")}
	}
	return []string{mobile, "+86" + mobile, "+" + mobile}
}

// AppLogin 用 App 账号登录后台，返回与 GVA 原生登录**完全同形**的响应。
//
// 三步，缺一不可：
//  1. App 的密码（bcrypt，和 `/api/auth/login` 同一份哈希）
//  2. `merchant_application` 里**这个角色已经 approved** —— 没申请过、
//     还在待审、被驳回的，一律不许进（这是「后台入口不能凭空发」的唯一闸门）
//  3. 惰性建 / 复用影子 `sys_user`，角色 1001 / 1002
func (s *PortalService) AppLogin(ctx context.Context, req mallReq.AppLoginReq) (*systemRes.LoginResponse, error) {
	role := strings.TrimSpace(req.Role)
	if role != PortalRoleParent && role != PortalRoleMerchant {
		return nil, errors.New("门户类型不对")
	}
	mobile := normalizeMobile(req.Mobile)
	if mobile == "" {
		return nil, errors.New("手机号格式不对")
	}

	var cred appCredential
	err := global.GVA_DB.WithContext(ctx).Table("user").
		Select("id, password, nickname, status").
		Where("mobile IN ?", mobileCandidates(mobile)).
		Take(&cred).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 措辞**不区分**「没这个号」和「密码错」：后台登录页是公网可达的，
		// 分开说等于送一个「这个手机号注册过没有」的探测接口。
		return nil, errors.New("手机号或密码不对")
	}
	if err != nil {
		return nil, err
	}
	if cred.Password == "" {
		return nil, errors.New("这个账号还没有设置密码，先在 App 里设一个")
	}
	if !utils.BcryptCheck(req.Password, cred.Password) {
		return nil, errors.New("手机号或密码不对")
	}
	if cred.Status != 0 && cred.Status != 1 {
		// 平台的 status：0/1 正常，其它视为停用（与 C 端 `IsActive` 同口径）
		return nil, errors.New("这个账号已被停用")
	}

	app, err := s.approvedApplication(ctx, cred.ID, role)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errors.New("这个账号还没有通过审核 —— 先在 App 里提交申请")
	}

	authorityID := AuthorityParent
	if role == PortalRoleMerchant {
		authorityID = AuthorityMerchant
	}
	shadow, err := s.ensureShadowUser(ctx, cred.ID, cred.Nickname, mobile, authorityID)
	if err != nil {
		return nil, err
	}

	var full system.SysUser
	if err := global.GVA_DB.WithContext(ctx).
		Preload("Authority").Preload("Authorities").
		Where("id = ?", shadow.ID).First(&full).Error; err != nil {
		return nil, err
	}

	token, claims, err := utils.LoginToken(&full)
	if err != nil {
		return nil, errors.New("签发登录凭证失败")
	}
	return &systemRes.LoginResponse{
		User:      full,
		Token:     token,
		ExpiresAt: claims.RegisteredClaims.ExpiresAt.Unix() * 1000,
	}, nil
}

// approvedApplication 查「这个 uid 以这个角色通过了审核」。
//
// 只认 `approved`：`pending` / `rejected` 都不放行。
// ⚠️ 别把 `rejected` 当成「没申请过」去放行 —— 被驳回的人正是最想再试一次的那个。
func (s *PortalService) approvedApplication(ctx context.Context, uid uint, role string) (*mall.MerchantApplication, error) {
	var app mall.MerchantApplication
	err := global.GVA_DB.WithContext(ctx).
		Where("uid = ? AND role = ?", uid, role).
		Take(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if app.Status != "approved" {
		return nil, nil
	}
	return &app, nil
}

// ensureShadowUser 拿（没有就建）影子 `sys_user`。
//
// ⚠️ 三个必须注意的点：
//  1. **用户名带前缀**，`app_125`。这是 `AppUIDOf` 反查 uid 的唯一依据。
//  2. **密码列写随机串**：登录永远走 App 的密码（`AppLogin` 里已经校验过），
//     这一列不会被 `BcryptCheck` 用到。写随机值而不是空串，是为了万一有人
//     在「用户管理」里点了重置密码，也不会得到一个能猜的默认口令。
//  3. **角色变了要跟着改**（同一个人先申请家长、后来又申请商家）：
//     复用同一行、只改 `authority_id`，不新建第二个影子账号 ——
//     否则同一个人会有两个 id，操作日志里对不上。
func (s *PortalService) ensureShadowUser(
	ctx context.Context, appUID uint, nickname, mobile string, authorityID uint,
) (*system.SysUser, error) {
	username := shadowPrefix + strconv.FormatUint(uint64(appUID), 10)
	if nickname == "" {
		nickname = username
	}

	var existing system.SysUser
	err := global.GVA_DB.WithContext(ctx).Where("username = ?", username).Take(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		row := system.SysUser{
			UUID:        uuid.New(),
			Username:    username,
			Password:    utils.BcryptHash(uuid.NewString()),
			NickName:    nickname,
			AuthorityId: authorityID,
			Phone:       mobile,
			Enable:      1,
		}
		if err := global.GVA_DB.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	case err != nil:
		return nil, err
	}

	updates := map[string]interface{}{}
	if existing.AuthorityId != authorityID {
		updates["authority_id"] = authorityID
	}
	if existing.Enable != 1 {
		// 被后台停用过的人，重新登录不该被永久挡住 —— 但也不会自动解封：
		// 这里只在「本来就是 1」时不动它。所以只有 0（脏数据）才修回来。
		if existing.Enable == 0 {
			updates["enable"] = 1
		}
	}
	if len(updates) > 0 {
		if err := global.GVA_DB.WithContext(ctx).Model(&system.SysUser{}).
			Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return &existing, nil
}

// AppUIDOf 从影子账号的用户名反查 App 的 `user.id`。
//
// 每个家长 / 商家接口的**第一件事**都是调它，拿到的 uid 决定这一屏能看谁的数据。
// 解析不出来就返回错误 —— 绝不返回 0 兜底：`uid = 0` 在平台表里是
// 「游客 / 平台自营」，那会把别人的数据当自己的。
func AppUIDOf(username string) (uint, error) {
	if !strings.HasPrefix(username, shadowPrefix) {
		return 0, errors.New("这不是 App 账号")
	}
	n, err := strconv.ParseUint(strings.TrimPrefix(username, shadowPrefix), 10, 64)
	if err != nil || n == 0 {
		return 0, errors.New("App 账号标识不合法")
	}
	return uint(n), nil
}

// ============================ 家长端（只读） ============================

// ParentOverview 家长首页概览：余额 + 心愿单各状态计数 + 最近几单。
//
// ⚠️ 这一屏**故意只读**。确认 / 回绝 / 定价这些会动钱的动作用户在 App 里做 ——
// 那里已经有完整规则（托管两半、退款凭据、日界），在后台再实现一遍，
// 两份规则迟早分叉，而分叉的表现是「券扣了但订单没变」，不报错。
func (s *PortalService) ParentOverview(ctx context.Context, uid uint) (map[string]interface{}, error) {
	var wallet mall.Wallet
	err := global.GVA_DB.WithContext(ctx).Where("uid = ?", uid).Take(&wallet).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	type statusCount struct {
		Status string `gorm:"column:status"`
		N      int64  `gorm:"column:n"`
	}
	var counts []statusCount
	if err := global.GVA_DB.WithContext(ctx).Model(&mall.WishOrder{}).
		Select("status, count(*) as n").
		Where("uid = ?", uid).Group("status").Scan(&counts).Error; err != nil {
		return nil, err
	}
	byStatus := map[string]int64{}
	for _, c := range counts {
		byStatus[c.Status] = c.N
	}

	var recent []mall.WishOrder
	if err := global.GVA_DB.WithContext(ctx).
		Where("uid = ?", uid).Order("created_at DESC").Limit(5).Find(&recent).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"uid":     uid,
		"points":  wallet.Points,
		"coupon":  wallet.Coupon,
		"diamond": wallet.Diamond,
		"wish":    byStatus,
		"recent":  recent,
	}, nil
}

// ParentWishRow 家长心愿单的一行 = 订单 + 这件东西的购买链接。
//
// ⚠️ 订单表**不存** `buy_url`（那是商品属性，订单只存 `item_id` 快照）。
// 但家长看这一屏的唯一目的就是「去下单」—— 不给链接，这个页面等于白开。
// 所以这里按 `item_id` 回查一次商品表补上，而不是往订单表里冗余一份
// （冗余的话，商家改链接后老订单会指向旧地址）。
type ParentWishRow struct {
	mall.WishOrder
	BuyURL   string `json:"buy_url"`
	BuyLabel string `json:"buy_label"`
}

// ParentWishes 家长的心愿单列表（按 uid 过滤）。
func (s *PortalService) ParentWishes(
	ctx context.Context, uid uint, info mallReq.ParentWishSearch,
) (list []ParentWishRow, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.WishOrder{}).Where("uid = ?", uid)
	if info.Status != "" {
		db = db.Where("status = ?", info.Status)
	}
	if info.Keyword != "" {
		db = db.Where("item_name LIKE ?", "%"+info.Keyword+"%")
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	var orders []mall.WishOrder
	if err = db.Order("created_at DESC").
		Scopes(info.Paginate()).Find(&orders).Error; err != nil {
		return
	}
	list = make([]ParentWishRow, 0, len(orders))
	if len(orders) == 0 {
		return
	}

	// 一次查完这一页用到的商品（别在循环里查 —— N+1，一页 10 行就是 10 次往返）
	ids := make([]string, 0, len(orders))
	seen := map[string]bool{}
	for _, o := range orders {
		if o.ItemID != "" && !seen[o.ItemID] {
			seen[o.ItemID] = true
			ids = append(ids, o.ItemID)
		}
	}
	type link struct {
		ItemID   string `gorm:"column:item_id"`
		BuyURL   string `gorm:"column:buy_url"`
		BuyLabel string `gorm:"column:buy_label"`
	}
	var links []link
	if len(ids) > 0 {
		if err = global.GVA_DB.WithContext(ctx).Table("wish_item").
			Select("item_id, buy_url, buy_label").
			Where("item_id IN ?", ids).Scan(&links).Error; err != nil {
			return
		}
	}
	byItem := make(map[string]link, len(links))
	for _, l := range links {
		byItem[l.ItemID] = l
	}
	for _, o := range orders {
		l := byItem[o.ItemID]
		list = append(list, ParentWishRow{WishOrder: o, BuyURL: l.BuyURL, BuyLabel: l.BuyLabel})
	}
	return
}

// ParentRecords 家长的兑换记录（按 uid 过滤）。
func (s *PortalService) ParentRecords(
	ctx context.Context, uid uint, info mallReq.ParentRecordSearch,
) (list []mall.RedeemRecord, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.RedeemRecord{}).Where("uid = ?", uid)
	if info.Keyword != "" {
		db = db.Where("name LIKE ?", "%"+info.Keyword+"%")
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	err = db.Order("created_at DESC").Scopes(info.Paginate()).Find(&list).Error
	return
}

// ============================ 商家端 ============================

// MerchantStatus 商家自己的入驻状态（含驳回理由 / 后台地址）。
func (s *PortalService) MerchantStatus(ctx context.Context, uid uint) (*mall.MerchantApplication, error) {
	var app mall.MerchantApplication
	err := global.GVA_DB.WithContext(ctx).
		Where("uid = ? AND role = ?", uid, PortalRoleMerchant).Take(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// MerchantItems 商家自己的商品（**只按 merchant_uid 过滤**）。
//
// ⚠️ `merchant_uid = 0` 是平台自营，不属于任何商家。
// 如果这里写成 `merchant_uid != 0` 或者漏掉这个条件，
// 商家就能看到 / 改到平台自己的商品 —— 而列表看上去完全正常。
func (s *PortalService) MerchantItems(
	ctx context.Context, uid uint, info mallReq.MerchantItemSearch,
) (list []mall.WishItem, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.WishItem{}).Where("merchant_uid = ?", uid)
	if info.Archived != nil {
		db = db.Where("archived = ?", *info.Archived)
	}
	if info.Keyword != "" {
		db = db.Where("name LIKE ?", "%"+info.Keyword+"%")
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	err = db.Order("sort_order ASC, id DESC").Scopes(info.Paginate()).Find(&list).Error
	return
}

// MerchantSaveItem 商家上架 / 改价 / 下架**自己的一件**商品。
//
// 两条硬约束：
//   - 改：`WHERE item_id = ? AND merchant_uid = ?` —— 少写后半句就能改别人的
//   - 新建：`merchant_uid` 强制写成自己，`kind` 强制 `goods`
//     （商家卖的是实物；能改 `kind` 就等于能把扣券的东西改成发游戏）
//
// ⚠️ 改是**整条覆盖**（见 `MerchantSaveItemReq` 的注释）：`fields` 里带上了
// 每一个可编辑列，所以少传一个字段 = 把它清空。别在这里改成「只更新传了的字段」——
// JSON 区分不出「没传」和「传了空串」，那样改会让商家**清不掉**链接。
func (s *PortalService) MerchantSaveItem(ctx context.Context, uid uint, req mallReq.MerchantSaveItemReq) (string, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return "", errors.New("商品名不能为空")
	}
	if req.PriceCents < 0 {
		return "", errors.New("参考价不能是负数")
	}
	if u := strings.TrimSpace(req.BuyURL); u != "" &&
		!strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		// C 端会把它渲染成 `<a href>`；`javascript:` 之类必须挡在入口
		return "", errors.New("购买链接必须是 http(s) 开头")
	}

	fields := map[string]interface{}{
		"name":         name,
		"emoji":        req.Emoji,
		"tint":         req.Tint,
		"partner_name": req.PartnerName,
		"price_cents":  req.PriceCents,
		"note":         req.Note,
		"notice":       req.Notice,
		"buy_url":      req.BuyURL,
		"buy_label":    req.BuyLabel,
		"sort_order":   req.SortOrder,
		"updated_at":   nowUnix(),
	}
	if req.Archived != nil {
		fields["archived"] = *req.Archived
	}

	if req.ItemID == "" {
		itemID := fmt.Sprintf("goods_%d_%d", uid, nowUnix())
		row := mall.WishItem{
			ItemID:      itemID,
			Kind:        "goods",
			MerchantUID: uid,
			Archived:    req.Archived != nil && *req.Archived,
		}
		row.Name = name
		row.Emoji = req.Emoji
		row.Tint = req.Tint
		row.PartnerName = req.PartnerName
		row.PriceCents = req.PriceCents
		row.Note = req.Note
		row.Notice = req.Notice
		row.BuyURL = req.BuyURL
		row.BuyLabel = req.BuyLabel
		row.SortOrder = req.SortOrder
		row.CreatedAt = nowUnix()
		row.UpdatedAt = row.CreatedAt
		if err := global.GVA_DB.WithContext(ctx).Create(&row).Error; err != nil {
			return "", err
		}
		return itemID, nil
	}

	res := global.GVA_DB.WithContext(ctx).Model(&mall.WishItem{}).
		Where("item_id = ? AND merchant_uid = ?", req.ItemID, uid).
		Updates(fields)
	if res.Error != nil {
		return "", res.Error
	}
	if res.RowsAffected == 0 {
		// 不存在 / 不是自己的，**同一个错误**：区分开等于告诉对方「这件存在但不是你的」
		return "", errors.New("这件商品不在你名下")
	}
	return req.ItemID, nil
}
