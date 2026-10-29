package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// ============================ 通用 ============================

// PlatformPageInfo 平台数据分页（复用后台的分页结构）
type PlatformPageInfo struct {
	request.PageInfo
}

// ============================ 游戏 ============================

// GameSearch 游戏查询
type GameSearch struct {
	request.PageInfo
	GameID string `json:"game_id" form:"game_id"`
	Name   string `json:"name" form:"name"`
	Kind   string `json:"kind" form:"kind"`
	Status string `json:"status" form:"status"`
}

// UpdateGameReq 更新游戏（运营改价格/上下架）。
//
// 只允许改这些字段 —— 名字/图标/文案**不在**这里，那些是游戏包自己的
// manifest 决定的，后台改了会和包里的声明对不上。
type UpdateGameReq struct {
	GameID       string  `json:"game_id" binding:"required"`
	PricePoints  *int64  `json:"price_points"`
	PriceDiamond *int64  `json:"price_diamond"`
	Status       *string `json:"status"`
	DefaultOwned *bool   `json:"default_owned"`
	Sort         *int    `json:"sort"`
}

// ============================ 游戏版本 ============================

// GameReleaseSearch 版本查询
type GameReleaseSearch struct {
	request.PageInfo
	GameID     string `json:"game_id" form:"game_id"`
	Channel    string `json:"channel" form:"channel"`
	OnlyActive bool   `json:"only_active" form:"only_active"`
}

// PublishReleaseReq 发布版本
type PublishReleaseReq struct {
	GameID       string `json:"game_id" binding:"required"`
	Version      string `json:"version" binding:"required"`
	Channel      string `json:"channel"`
	URL          string `json:"url" binding:"required"`
	SHA256       string `json:"sha256" binding:"required"`
	Size         int64  `json:"size"`
	AssetBase    string `json:"asset_base"`
	MinHostApi   int    `json:"min_host_api"`
	Forced       bool   `json:"forced"`
	ReleaseNotes string `json:"release_notes"`
}

// ============================ 平台用户 ============================

// PlatformUserSearch 用户查询
type PlatformUserSearch struct {
	request.PageInfo
	Mobile   string `json:"mobile" form:"mobile"`
	Nickname string `json:"nickname" form:"nickname"`
	Status   *int   `json:"status" form:"status"`
}

// AdjustBalanceReq 后台调整余额。
//
// ⚠️ 调整**必须走账本**（写一条 `manual_adjust` 的记录），
// 不能直接 UPDATE wallet.points ——
// 那样钱包余额和账本就会对不上，而且**不报错**，
// 直到某天对账才发现「钱多了但没流水」。
type AdjustBalanceReq struct {
	UID      uint   `json:"uid" binding:"required"`
	Currency string `json:"currency" binding:"required,oneof=points coupon diamond"`
	// Delta 变动量：正数增加，负数扣减
	Delta int64  `json:"delta" binding:"required"`
	Memo  string `json:"memo" binding:"required,max=128"`
}

// ============================ 兑换商城 ============================

// RedeemItemSearch 兑换商品查询
type RedeemItemSearch struct {
	request.PageInfo
	UID      *uint  `json:"uid" form:"uid"`
	Category string `json:"category" form:"category"`
	Archived *bool  `json:"archived" form:"archived"`
}

// UpdateRedeemItemReq 后台调整兑换商品（改价/上下架）
type UpdateRedeemItemReq struct {
	ItemID   string `json:"item_id" binding:"required"`
	Price    *int64 `json:"price"`
	Stock    *int   `json:"stock"`
	Archived *bool  `json:"archived"`
	Sort     *int   `json:"sort"`
}

// RedeemRecordSearch 兑换记录查询
type RedeemRecordSearch struct {
	request.PageInfo
	UID       *uint  `json:"uid" form:"uid"`
	ItemID    string `json:"item_id" form:"item_id"`
	Cancelled *bool  `json:"cancelled" form:"cancelled"`
	Fulfilled *bool  `json:"fulfilled" form:"fulfilled"`
}

// ============================ 心愿商城 ============================

// WishItemSearch 心愿商品查询
type WishItemSearch struct {
	request.PageInfo
	Kind     string `json:"kind" form:"kind"`
	GameID   string `json:"game_id" form:"game_id"`
	Archived *bool  `json:"archived" form:"archived"`
}

// UpdateWishItemReq 后台调整心愿商品（外链/上下架）
type UpdateWishItemReq struct {
	ItemID    string  `json:"item_id" binding:"required"`
	BuyURL    *string `json:"buy_url"`
	BuyLabel  *string `json:"buy_label"`
	Archived  *bool   `json:"archived"`
	SortOrder *int    `json:"sort_order"`
}

// WishOrderSearch 心愿单查询
type WishOrderSearch struct {
	request.PageInfo
	UID    *uint  `json:"uid" form:"uid"`
	Status string `json:"status" form:"status"`
	Kind   string `json:"kind" form:"kind"`
}

// ============================ 任务 ============================

// TaskSearch 任务查询
type TaskSearch struct {
	request.PageInfo
	UID      *uint  `json:"uid" form:"uid"`
	Category string `json:"category" form:"category"`
	Cycle    string `json:"cycle" form:"cycle"`
}

// TaskInstanceSearch 任务实例查询
type TaskInstanceSearch struct {
	request.PageInfo
	UID    *uint  `json:"uid" form:"uid"`
	Status string `json:"status" form:"status"`
	Date   string `json:"date" form:"date"`
}

// ============================ 统计 ============================

// DashboardReq 经营看板
type DashboardReq struct {
	// Days 统计最近几天（默认 7）
	Days int `json:"days" form:"days"`
}

// ============================ 招商 ============================

// MerchantSearch 招商申请查询
type MerchantSearch struct {
	request.PageInfo
	Role    string `json:"role" form:"role"`
	Status  string `json:"status" form:"status"`
	Keyword string `json:"keyword" form:"keyword"`
}

// ReviewMerchantReq 审核一条申请。
//
// ⚠️ **通过时必须给 `admin_url`**：C 端那一屏写的是
// 「审核通过后，这里会出现一个后台链接」。通过了却不给地址，
// 用户看到「已通过」却永远拿不到入口，只能来问客服。
// 服务层会直接拒绝这种调用，而不是先落库再说。
type ReviewMerchantReq struct {
	ID uint `json:"id" binding:"required"`
	// Approve true = 通过，false = 拒绝
	Approve bool `json:"approve"`
	// AdminURL 通过时必填（给对方的后台入口）
	//
	// ⚠️ 这里**不用** `binding:"url"` —— 它的报错是
	// `Key: 'ReviewMerchantReq.AdminURL' Error:Field validation for ... failed`，
	// 运营看不懂。格式检查放在 service 里，报「要以 http:// 或 https:// 开头」。
	AdminURL string `json:"admin_url" binding:"omitempty,max=512"`
	// Note 审核备注 / 拒绝原因（拒绝时会被 C 端显示出来）
	Note string `json:"note" binding:"omitempty,max=255"`
}
