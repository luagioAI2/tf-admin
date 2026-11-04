package mall

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	mallReq "github.com/flipped-aurora/gin-vue-admin/server/model/mall/request"
	mallSvc "github.com/flipped-aurora/gin-vue-admin/server/service/mall"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
)

// TemplateApi 任务 / 兑换模板配置。
//
// 准入有三层，缺一层都会出数据事故：
//  1. casbin —— 这个角色能不能调这个地址（1001 家长不给）
//  2. templateActor —— 这个 token 到底是谁；解析不出直接拒，绝不回落 uid=0
//  3. service —— 每条 SQL 都带 scope / owner_uid 范围
//
// ★ 商家账号能配自己的模板（用户口径「商家自己配自己的」），
//   但**不能**碰官方模板、更不能碰别的商家的。范围由 token 推导，
//   请求体里的 owner_uid 一律不信任。
type TemplateApi struct{}

// templateActor 从 token 推导数据范围。
//
// 影子账号用户名是 `app_{uid}`；解析失败 = 不是 App 账号 = 拒绝。
// 家长 1001 在这里直接挡掉：casbin 已经不给它这些地址，
// 服务层再挡一次是「两端都测了、中间那根线也钉住」。
func templateActor(c *gin.Context) (mallSvc.TemplateActor, bool) {
	aid := utils.GetUserAuthorityId(c)
	switch aid {
	case 888:
		return mallSvc.TemplateActor{IsAdmin: true}, true
	case mallSvc.AuthorityMerchant, mallSvc.AuthorityParentMerchant:
		uid, err := mallSvc.AppUIDOf(utils.GetUserName(c))
		if err != nil || uid == 0 {
			response.FailWithMessage("这个账号不能配置模板", c)
			return mallSvc.TemplateActor{}, false
		}
		return mallSvc.TemplateActor{OwnerUID: uid}, true
	default:
		response.FailWithMessage("这个账号不能配置模板", c)
		return mallSvc.TemplateActor{}, false
	}
}

// List 模板列表
// @Tags     MallTemplate
// @Summary  模板列表
// @Param    data  body      mallReq.TemplateSearch                        true  "筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}   "模板列表"
// @Router   /mall/template/list [post]
func (a *TemplateApi) List(c *gin.Context) {
	actor, ok := templateActor(c)
	if !ok {
		return
	}
	var info mallReq.TemplateSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := templateService.List(c.Request.Context(), actor, info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// Detail 模板详情（头 + 条目）
// @Tags     MallTemplate
// @Summary  模板详情
// @Param    data  body      mallReq.TemplateIDReq  true  "id"
// @Success  200   {object}  response.Response{}    "模板详情"
// @Router   /mall/template/detail [post]
func (a *TemplateApi) Detail(c *gin.Context) {
	actor, ok := templateActor(c)
	if !ok {
		return
	}
	var req mallReq.TemplateIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	detail, err := templateService.Detail(c.Request.Context(), actor, req.ID)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithData(detail, c)
}

// Create 新建模板
// @Tags     MallTemplate
// @Summary  新建模板
// @Param    data  body      mallReq.TemplateCreateReq  true  "模板头"
// @Success  200   {object}  response.Response{}        "id"
// @Router   /mall/template/create [post]
func (a *TemplateApi) Create(c *gin.Context) {
	actor, ok := templateActor(c)
	if !ok {
		return
	}
	var req mallReq.TemplateCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	id, err := templateService.Create(c.Request.Context(), actor, req)
	if err != nil {
		response.FailWithMessage("新建失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"id": id}, "已新建", c)
}

// Update 改模板头
// @Tags     MallTemplate
// @Summary  改模板头
// @Param    data  body      mallReq.TemplateUpdateReq  true  "字段"
// @Success  200   {object}  response.Response{}        "结果"
// @Router   /mall/template/update [post]
func (a *TemplateApi) Update(c *gin.Context) {
	actor, ok := templateActor(c)
	if !ok {
		return
	}
	var req mallReq.TemplateUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := templateService.Update(c.Request.Context(), actor, req); err != nil {
		response.FailWithMessage("保存失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("已保存", c)
}

// SetDefault 设为该分组的默认模板
// @Tags     MallTemplate
// @Summary  设为默认
// @Param    data  body      mallReq.TemplateIDReq  true  "id"
// @Success  200   {object}  response.Response{}    "结果"
// @Router   /mall/template/set-default [post]
func (a *TemplateApi) SetDefault(c *gin.Context) {
	actor, ok := templateActor(c)
	if !ok {
		return
	}
	var req mallReq.TemplateIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := templateService.SetDefault(c.Request.Context(), actor, req.ID); err != nil {
		response.FailWithMessage("设置失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("已设为默认", c)
}

// Delete 删除模板（连带条目）
// @Tags     MallTemplate
// @Summary  删除模板
// @Param    data  body      mallReq.TemplateIDReq  true  "id"
// @Success  200   {object}  response.Response{}    "结果"
// @Router   /mall/template/delete [post]
func (a *TemplateApi) Delete(c *gin.Context) {
	actor, ok := templateActor(c)
	if !ok {
		return
	}
	var req mallReq.TemplateIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := templateService.Delete(c.Request.Context(), actor, req.ID); err != nil {
		response.FailWithMessage("删除失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("已删除", c)
}

// SaveItem 新增 / 改一条模板条目
// @Tags     MallTemplate
// @Summary  保存模板条目
// @Param    data  body      mallReq.TemplateItemSaveReq  true  "条目"
// @Success  200   {object}  response.Response{}          "id"
// @Router   /mall/template/item/save [post]
func (a *TemplateApi) SaveItem(c *gin.Context) {
	actor, ok := templateActor(c)
	if !ok {
		return
	}
	var req mallReq.TemplateItemSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	id, err := templateService.SaveItem(c.Request.Context(), actor, req)
	if err != nil {
		response.FailWithMessage("保存失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"id": id}, "已保存", c)
}

// DeleteItem 删一条模板条目
// @Tags     MallTemplate
// @Summary  删除模板条目
// @Param    data  body      mallReq.TemplateItemDeleteReq  true  "id + template_id"
// @Success  200   {object}  response.Response{}            "结果"
// @Router   /mall/template/item/delete [post]
func (a *TemplateApi) DeleteItem(c *gin.Context) {
	actor, ok := templateActor(c)
	if !ok {
		return
	}
	var req mallReq.TemplateItemDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := templateService.DeleteItem(c.Request.Context(), actor, req); err != nil {
		response.FailWithMessage("删除失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("已删除", c)
}

// InviteCode 商家自己的推广邀请码（印在物料上，注册时带给 App）。
// @Tags     MallTemplate
// @Summary  我的邀请码
// @Success  200   {object}  response.Response{}  "邀请码"
// @Router   /mall/template/invite-code [post]
func (a *TemplateApi) InviteCode(c *gin.Context) {
	actor, ok := templateActor(c)
	if !ok {
		return
	}
	uid := actor.OwnerUID
	if actor.IsAdmin {
		// 超管没有「自己的」商家码；给 0 就是空串，界面上要说明。
		uid = 0
	}
	response.OkWithData(gin.H{
		"uid":         uid,
		"invite_code": mallSvc.MakeInviteCode(uid),
	}, c)
}
