package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// ============================ 任务 / 兑换模板 ============================
//
// 这些请求只在后台用。数据范围**不信任前端传的 owner_uid**：
// 商家调接口时服务端强制把 scope=merchant / owner_uid=自己（见
// `service/mall/template.go` 的 TemplateActor）。前端传什么都不算数。

// TemplateSearch 模板列表
type TemplateSearch struct {
	request.PageInfo
	Kind  string `json:"kind" form:"kind"`
	Scope string `json:"scope" form:"scope"`
	// OwnerUID 只有超管能用；商家传了也会被忽略
	OwnerUID *uint `json:"owner_uid" form:"owner_uid"`
}

// TemplateCreateReq 新建模板头
type TemplateCreateReq struct {
	Name string `json:"name" binding:"required,max=64"`
	Kind string `json:"kind" binding:"required,oneof=task redeem"`
	// Scope 官方 / 商家。**商家账号会被服务端强制改成 merchant**，
	// 这里留着是为了超管能给某个商家预置一份。
	Scope string `json:"scope" binding:"omitempty,oneof=official merchant"`
	// OwnerUID scope=merchant 时必填（且必须 > 0）；scope=official 时必须是 0。
	OwnerUID  uint   `json:"owner_uid"`
	Note      string `json:"note" binding:"omitempty,max=255"`
	Sort      int    `json:"sort"`
	IsDefault bool   `json:"is_default"`
}

// TemplateUpdateReq 改模板头。
//
// 全是指针：nil = 这次不动它。用值类型的话，只想改排序
// 也会把没填的名字 / 备注一起清空。
type TemplateUpdateReq struct {
	ID   uint    `json:"id" binding:"required"`
	Name *string `json:"name"`
	Note *string `json:"note"`
	Sort *int    `json:"sort"`
}

// TemplateIDReq 只带一个 id 的操作（详情 / 删除 / 设为默认）
type TemplateIDReq struct {
	ID uint `json:"id" binding:"required"`
}

// TemplateItemSaveReq 新增 / 改一条模板条目。
//
// ID = 0 是新建；> 0 是改。整条覆盖：**每个可编辑列都要传**，
// 少传一个 = 把它清空。和商家「我的商品」保存是同一套口径。
type TemplateItemSaveReq struct {
	ID         uint   `json:"id"`
	TemplateID uint   `json:"template_id" binding:"required"`
	ItemID     string `json:"item_id"`
	Name       string `json:"name" binding:"required,max=64"`
	Emoji      string `json:"emoji"`
	Category   string `json:"category"`
	Cycle      string `json:"cycle"`
	// 任务用
	PlannedMinutes int `json:"planned_minutes"`
	BasePoints     int `json:"base_points"`
	// 兑换用
	Price       int64 `json:"price"`
	LimitPerDay int   `json:"limit_per_day"`
	Stock       int   `json:"stock"`
	Once        bool  `json:"once"`
	// 通用
	Note string `json:"note"`
	Sort int    `json:"sort"`
	// Extra 玩法参数（JSON 对象，camelCase）。空串 = 不带额外参数。
	// 服务端会校验它必须是对象，并剔掉 typed 列的 snake_case 键。
	Extra string `json:"extra"`
}

// TemplateItemDeleteReq 删一条模板条目
type TemplateItemDeleteReq struct {
	ID         uint `json:"id" binding:"required"`
	TemplateID uint `json:"template_id" binding:"required"`
}
