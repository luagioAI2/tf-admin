package mall

import (
	"context"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/mall"
	"github.com/flipped-aurora/gin-vue-admin/server/model/mall/request"
	"gorm.io/gorm"
)

// ============================ 兑换商城 ============================

type RedeemService struct{}

// ItemList 兑换商品列表
func (s *RedeemService) ItemList(ctx context.Context, info request.RedeemItemSearch) (list []mall.RedeemItem, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.RedeemItem{})
	if info.UID != nil {
		db = db.Where("uid = ?", *info.UID)
	}
	if info.Category != "" {
		db = db.Where("category = ?", info.Category)
	}
	if info.Archived != nil {
		db = db.Where("archived = ?", *info.Archived)
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	if info.PageSize == 0 {
		info.PageSize = 20
	}
	err = db.Order("id DESC").Scopes(paginate(info.Page, info.PageSize)).Find(&list).Error
	return
}

// UpdateItem 调整兑换商品（改价 / 改库存 / 上下架）
func (s *RedeemService) UpdateItem(ctx context.Context, req request.UpdateRedeemItemReq) error {
	fields := map[string]interface{}{}
	if req.Price != nil {
		if *req.Price <= 0 {
			return errInvalid("价格必须大于 0")
		}
		fields["price"] = *req.Price
	}
	if req.Stock != nil {
		if *req.Stock < -1 {
			return errInvalid("库存不能小于 -1（-1 = 无限）")
		}
		fields["stock"] = *req.Stock
	}
	if req.Archived != nil {
		fields["archived"] = *req.Archived
	}
	if req.Sort != nil {
		fields["sort"] = *req.Sort
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = nowUnix()
	return global.GVA_DB.WithContext(ctx).Model(&mall.RedeemItem{}).
		Where("item_id = ?", req.ItemID).Updates(fields).Error
}

// RecordList 兑换记录列表
func (s *RedeemService) RecordList(ctx context.Context, info request.RedeemRecordSearch) (list []mall.RedeemRecord, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.RedeemRecord{})
	if info.UID != nil {
		db = db.Where("uid = ?", *info.UID)
	}
	if info.ItemID != "" {
		db = db.Where("item_id = ?", info.ItemID)
	}
	if info.Cancelled != nil {
		db = db.Where("cancelled = ?", *info.Cancelled)
	}
	if info.Fulfilled != nil {
		db = db.Where("fulfilled = ?", *info.Fulfilled)
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	if info.PageSize == 0 {
		info.PageSize = 20
	}
	err = db.Order("id DESC").Scopes(paginate(info.Page, info.PageSize)).Find(&list).Error
	return
}

// ============================ 心愿商城 ============================

type WishService struct{}

// ItemList 心愿商品列表
func (s *WishService) ItemList(ctx context.Context, info request.WishItemSearch) (list []mall.WishItem, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.WishItem{})
	if info.Kind != "" {
		db = db.Where("kind = ?", info.Kind)
	}
	if info.GameID != "" {
		db = db.Where("game_id = ?", info.GameID)
	}
	if info.Archived != nil {
		db = db.Where("archived = ?", *info.Archived)
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	if info.PageSize == 0 {
		info.PageSize = 20
	}
	// 与 C 端一致的排序：盲盒永远最前，其次 sort_order
	err = db.Order("CASE WHEN kind = 'blindbox' THEN 0 ELSE 1 END, sort_order ASC, name ASC").
		Scopes(paginate(info.Page, info.PageSize)).Find(&list).Error
	return
}

// UpdateItem 调整心愿商品（改外链 / 上下架）
//
// ⚠️ `buy_url` 只在**家长端**出现。后台可以改它，
// 但**不要**把它当成「孩子能看到的推广位」——
// C 端接口对孩子的响应里根本不返回这个字段。
func (s *WishService) UpdateItem(ctx context.Context, req request.UpdateWishItemReq) error {
	fields := map[string]interface{}{}
	if req.BuyURL != nil {
		fields["buy_url"] = *req.BuyURL
	}
	if req.BuyLabel != nil {
		fields["buy_label"] = *req.BuyLabel
	}
	if req.Archived != nil {
		fields["archived"] = *req.Archived
	}
	if req.SortOrder != nil {
		fields["sort_order"] = *req.SortOrder
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = nowUnix()
	return global.GVA_DB.WithContext(ctx).Model(&mall.WishItem{}).
		Where("item_id = ?", req.ItemID).Updates(fields).Error
}

// OrderList 心愿单列表
func (s *WishService) OrderList(ctx context.Context, info request.WishOrderSearch) (list []mall.WishOrder, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.WishOrder{})
	if info.UID != nil {
		db = db.Where("uid = ?", *info.UID)
	}
	if info.Status != "" {
		db = db.Where("status = ?", info.Status)
	}
	if info.Kind != "" {
		db = db.Where("kind = ?", info.Kind)
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	if info.PageSize == 0 {
		info.PageSize = 20
	}
	err = db.Order("id DESC").Scopes(paginate(info.Page, info.PageSize)).Find(&list).Error
	return
}

// ============================ 任务 ============================

type TaskService struct{}

// List 任务定义列表
func (s *TaskService) List(ctx context.Context, info request.TaskSearch) (list []mall.Task, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.Task{})
	if info.UID != nil {
		db = db.Where("uid = ?", *info.UID)
	}
	if info.Category != "" {
		db = db.Where("category = ?", info.Category)
	}
	if info.Cycle != "" {
		db = db.Where("cycle = ?", info.Cycle)
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	if info.PageSize == 0 {
		info.PageSize = 20
	}
	err = db.Order("id DESC").Scopes(paginate(info.Page, info.PageSize)).Find(&list).Error
	return
}

// InstanceList 任务实例列表
func (s *TaskService) InstanceList(ctx context.Context, info request.TaskInstanceSearch) (list []mall.TaskInstance, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.TaskInstance{})
	if info.UID != nil {
		db = db.Where("uid = ?", *info.UID)
	}
	if info.Status != "" {
		db = db.Where("status = ?", info.Status)
	}
	if info.Date != "" {
		db = db.Where("date = ?", info.Date)
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	if info.PageSize == 0 {
		info.PageSize = 20
	}
	err = db.Order("id DESC").Scopes(paginate(info.Page, info.PageSize)).Find(&list).Error
	return
}

// ============================ 经营看板 ============================

type DashboardService struct{}

// Overview 经营看板。
//
// 数据全部来自平台自己的表，**不做任何缓存** ——
// 后台是低频访问，实时数据比「快 100ms」有价值。
func (s *DashboardService) Overview(ctx context.Context, days int) (map[string]interface{}, error) {
	if days <= 0 {
		days = 7
	}
	since := time.Now().AddDate(0, 0, -days).Unix()
	db := global.GVA_DB.WithContext(ctx)

	var (
		userTotal, userNew     int64
		gameTotal              int64
		redeemTotal, wishTotal int64
		taskInstTotal          int64
		releaseActive          int64
		coinsSum               int64
	)

	if err := db.Model(&mall.PlatformUser{}).Count(&userTotal).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&mall.PlatformUser{}).Where("created_at >= ?", since).Count(&userNew).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&mall.Game{}).Where("status <> ?", "offline").Count(&gameTotal).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&mall.RedeemRecord{}).Where("created_at >= ? AND cancelled = ?", since, false).Count(&redeemTotal).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&mall.WishOrder{}).Where("created_at >= ?", since).Count(&wishTotal).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&mall.TaskInstance{}).Where("created_at >= ?", since).Count(&taskInstTotal).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&mall.GameRelease{}).Where("active = ?", true).Count(&releaseActive).Error; err != nil {
		return nil, err
	}
	// 全站积分总量（用于观察「发了多少分」是否需要干预）
	if err := db.Model(&mall.Wallet{}).Select("COALESCE(SUM(points),0)").Scan(&coinsSum).Error; err != nil {
		return nil, err
	}

	// 心愿单待办（运营视角：家长没处理的有多少）
	//
	// ⚠️ 这两句原来写的是 `_ = ...Count().Error` —— 把错误吞掉了。
	// 表名 / 列名一旦写错，两个数就**永远显示 0**，而 0 看起来像一个正常的
	// 业务结论（「今天没有待办」），没人会去查。宁可让看板整屏报错。
	var pendingWished, pendingAction int64
	if err := db.Model(&mall.WishOrder{}).Where("status = ?", "wished").Count(&pendingWished).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&mall.WishOrder{}).
		Where("status IN ?", []string{"committed", "confirmed"}).
		Count(&pendingAction).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"days":                days,
		"user_total":          userTotal,
		"user_new":            userNew,
		"game_total":          gameTotal,
		"game_release_active": releaseActive,
		"redeem_records":      redeemTotal,
		"wish_orders":         wishTotal,
		"task_instances":      taskInstTotal,
		"points_in_wallets":   coinsSum,
		"wish_pending_wished": pendingWished,
		"wish_pending_action": pendingAction,
	}, nil
}

// ============================ 招商 ============================

// MerchantService 招商申请审核。
//
// 这一屏是**唯一**能把申请推到 `approved` 的地方 —— C 端不许自己改状态
// （改了就等于给所有用户凭空发一个后台入口）。
type MerchantService struct{}

// List 申请列表。
//
// 排序：**待审核的永远在最前面**，同档按 id 倒序（新的在上）。
// 运营打开这一页就是为了处理待办，让他自己翻页去找 pending 等于白做一屏。
func (s *MerchantService) List(ctx context.Context, info request.MerchantSearch) (list []mall.MerchantApplication, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&mall.MerchantApplication{})
	if info.Role != "" {
		db = db.Where("role = ?", info.Role)
	}
	if info.Status != "" {
		db = db.Where("status = ?", info.Status)
	}
	if info.Keyword != "" {
		// 名字 / 联系方式 / 备注 一起搜 —— 运营手上通常只有其中一样
		like := "%" + info.Keyword + "%"
		db = db.Where("name LIKE ? OR contact LIKE ? OR note LIKE ?", like, like, like)
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	if info.PageSize == 0 {
		info.PageSize = 20
	}
	err = db.Order("CASE WHEN status = 'pending' THEN 0 ELSE 1 END, id DESC").
		Scopes(paginate(info.Page, info.PageSize)).Find(&list).Error
	return
}

// Review 审核（通过 / 拒绝）。
//
// ## 通过时 AdminURL 必填
//
// C 端那一屏写的是「审核通过后，这里会出现一个后台链接」。
// 通过了却不给地址 ⇒ 用户看到「已通过」但永远拿不到入口。
// 所以这里**先拒绝这种调用**，而不是先落库再指望有人补。
//
// ## 只允许从 pending 改（CAS）
//
// 用 `WHERE status = 'pending'` 做条件更新，`RowsAffected == 0` 就报
// 「已经审核过了」。不用先查再写：两个人同时点「通过」，后一个会覆盖前一个的
// `reviewed_at`，审计上就看不出谁什么时候审的。
func (s *MerchantService) Review(
	ctx context.Context,
	req request.ReviewMerchantReq,
	operatorID uint,
	operatorName string,
) error {
	target := "approved"
	if !req.Approve {
		target = "rejected"
	}
	adminURL := strings.TrimSpace(req.AdminURL)
	if req.Approve {
		if adminURL == "" {
			return errInvalid("通过审核必须填后台地址，否则对方拿不到入口")
		}
		// 格式检查放这里而不是 `binding:"url"`：校验器的报错是英文的
		// `Key: 'ReviewMerchantReq.AdminURL' ...`，运营看不懂。
		if !strings.HasPrefix(adminURL, "http://") && !strings.HasPrefix(adminURL, "https://") {
			return errInvalid("后台地址要以 http:// 或 https:// 开头")
		}
	} else {
		// 拒绝不该留下上一次通过时写的地址 —— 留着它，C 端一旦漏判状态
		// 就会把一个「已拒绝」的单子渲染出「打开后台」。
		adminURL = ""
	}

	note := strings.TrimSpace(req.Note)
	if operatorName != "" {
		// 备注里带上操作人，省得以后翻 sys_operation_records 才知道是谁
		note = strings.TrimSpace(note + " @" + operatorName)
	}

	// 一次取时间：`reviewed_at` 与 `updated_at` 差一秒的话，
	// 审计上会出现「更新早于审核」这种看不懂的记录
	now := nowUnix()
	fields := map[string]interface{}{
		"status":      target,
		"admin_url":   adminURL,
		"review_note": note,
		"reviewed_at": now,
		"reviewed_by": operatorID,
		// ⚠️ 模型的 `autoUpdateTime:false` ⇒ 必须手动写，否则 updated_at 停在 0
		"updated_at": now,
	}

	res := global.GVA_DB.WithContext(ctx).Model(&mall.MerchantApplication{}).
		Where("id = ? AND status = ?", req.ID, "pending").
		Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// 区分「不存在」和「已审过」—— 两种原因对运营是两件事
		var n int64
		if err := global.GVA_DB.WithContext(ctx).Model(&mall.MerchantApplication{}).
			Where("id = ?", req.ID).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return errInvalid("这条申请不存在（可能已经被删了）")
		}
		return errInvalid("这条申请已经审核过了，刷新看看最新状态")
	}
	return nil
}

// PendingCount 待审核数（菜单红点）
func (s *MerchantService) PendingCount(ctx context.Context) (int64, error) {
	var n int64
	err := global.GVA_DB.WithContext(ctx).Model(&mall.MerchantApplication{}).
		Where("status = ?", "pending").Count(&n).Error
	return n, err
}

// errInvalid 简单业务错误
func errInvalid(msg string) error {
	return &invalidError{msg}
}

type invalidError struct{ msg string }

func (e *invalidError) Error() string { return e.msg }

// 确保 gorm 被使用（txOrDB 的签名需要）
var _ = func(db *gorm.DB) *gorm.DB { return db }
