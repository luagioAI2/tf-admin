package mall

import (
	"context"
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
	var pendingWished, pendingAction int64
	_ = db.Model(&mall.WishOrder{}).Where("status = ?", "wished").Count(&pendingWished).Error
	_ = db.Model(&mall.WishOrder{}).Where("status IN ?", []string{"committed", "confirmed"}).Count(&pendingAction).Error

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

// errInvalid 简单业务错误
func errInvalid(msg string) error {
	return &invalidError{msg}
}

type invalidError struct{ msg string }

func (e *invalidError) Error() string { return e.msg }

// 确保 gorm 被使用（txOrDB 的签名需要）
var _ = func(db *gorm.DB) *gorm.DB { return db }
