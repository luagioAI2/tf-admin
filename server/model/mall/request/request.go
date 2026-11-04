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
	// IsGuest 账号类型：1 = 只看游客（临时账号），0 = 只看正式账号（绑了手机号）。
	//
	// 为什么要有它：游客是 App 一启动就自动建的，id 最大 ⇒ **永远排在最前面**。
	// 运营点进来看到一屏「小客人」（0 分 0 流水），真正要看的用户全被挤到
	// 后面几页去 —— 而「游客有 40 个」这件事本身也是有意义的运营数字。
	// 指针：nil = 不筛；0 是有效值（只看正式账号），不能用 0 当「没传」。
	IsGuest *int `json:"is_guest" form:"is_guest"`
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

// ResetPasswordReq 后台重置某个 C 端用户的登录密码。
//
// ## 为什么是「重置」而不是「改」
//
// 后台**看不到**原密码（库里只有 bcrypt 哈希，而且不该能反推）。
// 用户忘了密码时，能做的只有「设一个新的，让他用新的登进来」。
//
// ## 为什么不传新密码
//
// 由服务端生成，不让人填。理由很实际：让人填，早晚会有人图省事填 `123456`，
// 而这是**线上**账号的登录凭据。生成器也排掉了 `0/O`、`1/l/I` 这些
// 电话里念会念错、手抄会抄错的字符。
//
// ## Memo 必填
//
// 重置密码是**动别人账号**的操作。工单号 / 用户诉求写在这里，
// 将来有人问「这个号怎么被改过密码」时，只能靠它。
type ResetPasswordReq struct {
	UID  uint   `json:"uid" binding:"required"`
	Memo string `json:"memo" binding:"required,max=128"`
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

// CreateWishItemReq 后台新建心愿商品。
//
// 以前新建一件商品只能去改客户端那份 `wishCatalog.ts` 再发一次版 ——
// 运营加一个实物要等 App 更新。这张表本来就是**全站一份**的商品目录
// （`wish_item`），后台补上「新增」才算真的可配置。
type CreateWishItemReq struct {
	// ItemID 业务 id（C 端 `WishItem.id` 用的就是它）。留空自动生成。
	// 一旦生成就别再改 —— 心愿单、家长定价都按它关联。
	ItemID string `json:"item_id"`
	// Kind 商品类型：blindbox / gamebox / toy / book / other
	Kind string `json:"kind" binding:"required"`
	Name string `json:"name" binding:"required"`

	Emoji       string `json:"emoji"`
	Tint        string `json:"tint"`
	PartnerName string `json:"partner_name"`
	// GameID 游戏盒子指定的那一款（kind=gamebox 必填）
	GameID string `json:"game_id"`
	// Pool 盲盒池：候选游戏 id。**写全部游戏**是对的 ——
	// 已拥有的那几款由 C 端在开奖时减掉（每个家庭不一样，不能写进常量）。
	Pool       []string `json:"pool"`
	PriceCents int64    `json:"price_cents"`
	AgeRange   []int    `json:"age_range"`
	Note       string   `json:"note"`
	Notice     string   `json:"notice"`
	// BuyURL 外链。只在家长确认页出现，孩子端读不到
	BuyURL    string `json:"buy_url"`
	BuyLabel  string `json:"buy_label"`
	Archived  bool   `json:"archived"`
	SortOrder int    `json:"sort_order"`
	// UnlockCount 一次开出几款（盲盒用）。0 = 按 1 处理（单开）
	UnlockCount int `json:"unlock_count"`
}

// UpdateWishItemReq 后台调整心愿商品。
//
// 全部字段都是**指针**：nil = 这次不动它。用值类型的话，
// 界面上只想改排序，也会把没填的名字/须知一起清空。
type UpdateWishItemReq struct {
	ItemID string `json:"item_id" binding:"required"`

	Kind        *string   `json:"kind"`
	Name        *string   `json:"name"`
	Emoji       *string   `json:"emoji"`
	Tint        *string   `json:"tint"`
	PartnerName *string   `json:"partner_name"`
	GameID      *string   `json:"game_id"`
	Pool        *[]string `json:"pool"`
	PriceCents  *int64    `json:"price_cents"`
	AgeRange    *[]int    `json:"age_range"`
	Note        *string   `json:"note"`
	Notice      *string   `json:"notice"`
	BuyURL      *string   `json:"buy_url"`
	BuyLabel    *string   `json:"buy_label"`
	Archived    *bool     `json:"archived"`
	SortOrder   *int      `json:"sort_order"`
	// UnlockCount 一次开出几款（盲盒用）。nil = 不动
	UnlockCount *int `json:"unlock_count"`
}

// DeleteWishItemReq 删除心愿商品（只允许删「没人换过」的）
type DeleteWishItemReq struct {
	ItemID string `json:"item_id" binding:"required"`
}

// ============================ 激活码 ============================

// WishCodeSearch 激活码查询
type WishCodeSearch struct {
	request.PageInfo
	ItemID string `json:"item_id" form:"item_id"`
	Batch  string `json:"batch" form:"batch"`
	// Used nil = 不筛；true/false 都要能用（`0` 是有效值，见 IsGuest 那条注释）
	Used *bool `json:"used" form:"used"`
}

// GenerateWishCodeReq 批量生成激活码
type GenerateWishCodeReq struct {
	// ItemID 这批码解锁哪件商品。**必须是 gamebox 类的商品** ——
	// 其它类型没有 `game_id`，核销时解锁不出任何东西。
	ItemID string `json:"item_id" binding:"required"`
	// Count 生成多少个。上限 500：一次印几千张不现实，而且明文只返回一次，
	// 前端复制不了那么多。
	Count int `json:"count" binding:"required"`
	// Batch 批次号（运营自己起名，便于「按批作废」）
	Batch string `json:"batch"`
	// ExpireDays 多少天后过期。0 = 永不过期
	ExpireDays int `json:"expire_days"`
}

// VoidWishCodeReq 作废激活码。
//
// 两种粒度：给 `id` 作废单张；给 `batch` 作废整批里**还没用过**的。
// 已经用掉的不动 —— 那个孩子的游戏已经到账了，作废码不能回收游戏。
type VoidWishCodeReq struct {
	ID    uint   `json:"id"`
	Batch string `json:"batch"`
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
