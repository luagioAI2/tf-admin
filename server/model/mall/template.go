package mall

// ============================ 任务 / 兑换模板 ============================
//
// 这两张表和 tf-platform-api 的 `entities.Template` / `TemplateItem`
// **逐字段对齐**（同一个库、同一份数据）。字段定义以那边为准，
// 这里只加后台要用的 json 标签，别在这一侧改列名或加列。
//
// ⚠️ 时间戳是 int64 秒且 `autoCreateTime:false`：每个写入点都要手写，
// 否则落库是 0（1970 年），列表按时间倒序时全沉底、且不报错。
//
// ⚠️ 模板是**下发用的蓝图**，不是用户的任务实例：
//   `task` / `redeem_item` 是每用户一份（`uid` 列），
//   `template` / `template_item` 没有 uid。
// 改模板对**已经装过**的老用户不生效（App 首启才铺一次），
// 界面上必须写清楚，否则运营会以为「改了后台用户立刻变」。

// Template 模板头
type Template struct {
	BaseModel

	Name string `gorm:"column:name;size:64;not null" json:"name"`
	Kind string `gorm:"column:kind;size:16;index;not null" json:"kind"`

	// Scope / OwnerUID 一起决定「这份模板归谁」。
	// ★ owner_uid = 0 且 scope=official 才是官方；0 不是「没填」的占位符。
	Scope    string `gorm:"column:scope;size:16;index;not null" json:"scope"`
	OwnerUID uint   `gorm:"column:owner_uid;index;default:0;not null" json:"owner_uid"`

	IsDefault bool   `gorm:"column:is_default;index;default:false" json:"is_default"`
	Note      string `gorm:"column:note;size:255" json:"note"`
	Sort      int    `gorm:"column:sort;default:0" json:"sort"`
	Source    string `gorm:"column:source;size:16;default:'manual'" json:"source"`
}

func (Template) TableName() string { return "template" }

// TemplateItem 模板条目（两种 kind 共用一张表）
//
// Extra 存「服务端不解释、客户端要保留」的玩法参数（JSON，camelCase）：
// 签到开关、阶梯倍率、提醒时间、createdByParent …
// 加一个玩法参数不用改表 —— 所以它必须是**合法 JSON 对象**，
// 而且不许带 typed 列的 snake_case 键（否则下发时会把运营改的值盖回去）。
type TemplateItem struct {
	BaseModel

	TemplateID uint   `gorm:"column:template_id;uniqueIndex:uk_tpl_item;index;not null" json:"template_id"`
	ItemID     string `gorm:"column:item_id;size:64;uniqueIndex:uk_tpl_item;not null" json:"item_id"`

	Name           string `gorm:"column:name;size:64;not null" json:"name"`
	Emoji          string `gorm:"column:emoji;size:16" json:"emoji"`
	Category       string `gorm:"column:category;size:16" json:"category"`
	Cycle          string `gorm:"column:cycle;size:16" json:"cycle"`
	PlannedMinutes int    `gorm:"column:planned_minutes;default:30" json:"planned_minutes"`
	BasePoints     int    `gorm:"column:base_points;default:10" json:"base_points"`
	Price          int64  `gorm:"column:price;default:0" json:"price"`
	Note           string `gorm:"column:note;size:255" json:"note"`
	LimitPerDay    int    `gorm:"column:limit_per_day;default:0" json:"limit_per_day"`
	Stock          int    `gorm:"column:stock;default:-1" json:"stock"`
	Once           bool   `gorm:"column:once;default:false" json:"once"`
	Sort           int    `gorm:"column:sort;default:0" json:"sort"`
	Extra          string `gorm:"column:extra;type:text" json:"extra"`
}

func (TemplateItem) TableName() string { return "template_item" }

// TemplateRow 列表里的一行 = 模板头 + 条目数。
//
// 条目数用一次 GROUP BY 查完，别在循环里 count（N+1，一页 20 行就是 20 次）。
type TemplateRow struct {
	Template
	ItemCount int64 `gorm:"column:item_count" json:"item_count"`
}

// TemplateDetail 详情 = 模板头 + 全部条目
type TemplateDetail struct {
	Template
	Items []TemplateItem `json:"items"`
}
