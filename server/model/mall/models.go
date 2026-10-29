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
//   - `user`（这里）  = C 端小程序用户（孩子/家长）
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
	// Pool 盲盒池（JSON 数组字符串），后台只展示不编辑
	Pool       string `gorm:"column:pool" json:"pool"`
	PriceCents int64  `gorm:"column:price_cents" json:"price_cents"`
	Note       string `gorm:"column:note" json:"note"`
	Notice     string `gorm:"column:notice" json:"notice"`
	// BuyURL 外链（**只在家长端出现**）
	BuyURL    string `gorm:"column:buy_url" json:"buy_url"`
	BuyLabel  string `gorm:"column:buy_label" json:"buy_label"`
	Archived  bool   `gorm:"column:archived" json:"archived"`
	SortOrder int    `gorm:"column:sort_order" json:"sort_order"`
}

func (WishItem) TableName() string { return "wish_item" }

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
