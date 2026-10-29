package mall

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	mallReq "github.com/flipped-aurora/gin-vue-admin/server/model/mall/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// parseUint 解析无符号整数（解析失败返回 0）
func parseUint(s string) uint {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(v)
}

// parseInt 解析整数（解析失败返回默认值）
func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// ============================ 游戏 ============================

type GameApi struct{}

// List 游戏列表
// @Tags     MallGame
// @Summary  游戏列表
// @Produce  application/json
// @Param    data  body      mallReq.GameSearch                          true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}  "游戏列表"
// @Router   /mall/game/list [post]
func (a *GameApi) List(c *gin.Context) {
	var info mallReq.GameSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := gameService.List(c.Request.Context(), info)
	if err != nil {
		global.GVA_LOG.Error("获取游戏列表失败", zap.Error(err))
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total,
		Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// Options 游戏下拉选项
// @Tags     MallGame
// @Summary  游戏下拉选项
// @Produce  application/json
// @Success  200  {object}  response.Response{data=[]mall.Game}  "选项"
// @Router   /mall/game/options [post]
func (a *GameApi) Options(c *gin.Context) {
	list, err := gameService.Options(c.Request.Context())
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithData(list, c)
}

// Update 更新游戏（改价 / 上下架）
// @Tags     MallGame
// @Summary  更新游戏价格与状态
// @Produce  application/json
// @Param    data  body      mallReq.UpdateGameReq  true  "更新参数"
// @Success  200   {object}  response.Response{}    "更新结果"
// @Router   /mall/game/update [post]
func (a *GameApi) Update(c *gin.Context) {
	var req mallReq.UpdateGameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := gameService.Update(c.Request.Context(), req); err != nil {
		global.GVA_LOG.Error("更新游戏失败", zap.String("game_id", req.GameID), zap.Error(err))
		response.FailWithMessage("更新失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// ============================ 游戏版本 ============================

type GameReleaseApi struct{}

// List 版本列表
// @Tags     MallGameRelease
// @Summary  游戏版本列表
// @Produce  application/json
// @Param    data  body      mallReq.GameReleaseSearch                    true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}  "版本列表"
// @Router   /mall/release/list [post]
func (a *GameReleaseApi) List(c *gin.Context) {
	var info mallReq.GameReleaseSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := gameReleaseService.List(c.Request.Context(), info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total,
		Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// Publish 发布版本
// @Tags     MallGameRelease
// @Summary  发布游戏版本
// @Produce  application/json
// @Param    data  body      mallReq.PublishReleaseReq  true  "发布参数"
// @Success  200   {object}  response.Response{}         "发布结果"
// @Router   /mall/release/publish [post]
func (a *GameReleaseApi) Publish(c *gin.Context) {
	var req mallReq.PublishReleaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	rel, err := gameReleaseService.Publish(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("发布游戏版本失败", zap.String("game_id", req.GameID), zap.Error(err))
		response.FailWithMessage("发布失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(rel, "发布成功", c)
}

// Delete 删除版本
// @Tags     MallGameRelease
// @Summary  删除游戏版本
// @Produce  application/json
// @Param    id  query     uint  true  "版本 ID"
// @Success  200  {object}  response.Response{}  "删除结果"
// @Router   /mall/release/delete [delete]
func (a *GameReleaseApi) Delete(c *gin.Context) {
	id := parseUint(c.Query("id"))
	if id == 0 {
		response.FailWithMessage("id 不能为空", c)
		return
	}
	if err := gameReleaseService.Delete(c.Request.Context(), id); err != nil {
		response.FailWithMessage("删除失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// Deactivate 下架版本
// @Tags     MallGameRelease
// @Summary  下架游戏版本（active = false）
// @Produce  application/json
// @Param    id  query     uint  true  "版本 ID"
// @Success  200  {object}  response.Response{}  "下架结果"
// @Router   /mall/release/deactivate [post]
func (a *GameReleaseApi) Deactivate(c *gin.Context) {
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := gameReleaseService.Deactivate(c.Request.Context(), req.ID); err != nil {
		response.FailWithMessage("下架失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("已下架", c)
}

// ============================ 平台用户 ============================

type PlatformUserApi struct{}

// List C 端用户列表
// @Tags     MallUser
// @Summary  平台用户列表
// @Produce  application/json
// @Param    data  body      mallReq.PlatformUserSearch                    true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}   "用户列表"
// @Router   /mall/user/list [post]
func (a *PlatformUserApi) List(c *gin.Context) {
	var info mallReq.PlatformUserSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := platformUserService.List(c.Request.Context(), info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total,
		Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// Detail 用户详情（含钱包）
// @Tags     MallUser
// @Summary  用户详情与钱包
// @Produce  application/json
// @Param    id  query     uint  true  "用户 ID"
// @Success  200  {object}  response.Response{}  "详情"
// @Router   /mall/user/detail [get]
func (a *PlatformUserApi) Detail(c *gin.Context) {
	uid := parseUint(c.Query("id"))
	if uid == 0 {
		response.FailWithMessage("id 不能为空", c)
		return
	}
	user, wallet, err := platformUserService.Detail(c.Request.Context(), uid)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"user": user, "wallet": wallet}, "获取成功", c)
}

// Ledger 用户账本
// @Tags     MallUser
// @Summary  用户账本流水
// @Produce  application/json
// @Param    id         query     uint  true  "用户 ID"
// @Param    page       query     int   false "页码"
// @Param    pageSize   query     int   false "每页数量"
// // @Success  200  {object}  response.Response{data=response.PageResult}  "账本"
// @Router   /mall/user/ledger [get]
func (a *PlatformUserApi) Ledger(c *gin.Context) {
	uid := parseUint(c.Query("id"))
	if uid == 0 {
		response.FailWithMessage("id 不能为空", c)
		return
	}
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("pageSize"), 30)

	list, total, err := platformUserService.Ledger(c.Request.Context(), uid, page, pageSize)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: page, PageSize: pageSize,
	}, "获取成功", c)
}

// AdjustBalance 调整余额
// @Tags     MallUser
// @Summary  调整用户余额（走账本，保留流水）
// @Produce  application/json
// @Param    data  body      mallReq.AdjustBalanceReq  true  "调整参数"
// @Success  200   {object}  response.Response{}       "调整结果"
// @Router   /mall/user/adjust [post]
func (a *PlatformUserApi) AdjustBalance(c *gin.Context) {
	var req mallReq.AdjustBalanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	// 操作人：后台登录用户名（审计用，会写进账本 memo）
	operator := utils.GetUserName(c)
	if operator == "" {
		operator = "admin"
	}
	if err := platformUserService.AdjustBalance(c.Request.Context(), req, operator); err != nil {
		global.GVA_LOG.Error("调整余额失败",
			zap.Uint("uid", req.UID), zap.String("currency", req.Currency), zap.Error(err))
		response.FailWithMessage("调整失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("调整成功", c)
}

// ============================ 兑换商城 ============================

type RedeemApi struct{}

// ItemList 兑换商品列表
// @Tags     MallRedeem
// @Summary  兑换商品列表
// @Produce  application/json
// @Param    data  body      mallReq.RedeemItemSearch                      true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}   "商品列表"
// @Router   /mall/redeem/item/list [post]
func (a *RedeemApi) ItemList(c *gin.Context) {
	var info mallReq.RedeemItemSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := redeemService.ItemList(c.Request.Context(), info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// ItemUpdate 调整兑换商品
// @Tags     MallRedeem
// @Summary  调整兑换商品（改价 / 库存 / 上下架）
// @Produce  application/json
// @Param    data  body      mallReq.UpdateRedeemItemReq  true  "调整参数"
// @Success  200   {object}  response.Response{}          "调整结果"
// @Router   /mall/redeem/item/update [post]
func (a *RedeemApi) ItemUpdate(c *gin.Context) {
	var req mallReq.UpdateRedeemItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := redeemService.UpdateItem(c.Request.Context(), req); err != nil {
		response.FailWithMessage("调整失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("调整成功", c)
}

// RecordList 兑换记录
// @Tags     MallRedeem
// @Summary  兑换记录列表
// @Produce  application/json
// @Param    data  body      mallReq.RedeemRecordSearch                    true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}   "记录列表"
// @Router   /mall/redeem/record/list [post]
func (a *RedeemApi) RecordList(c *gin.Context) {
	var info mallReq.RedeemRecordSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := redeemService.RecordList(c.Request.Context(), info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// ============================ 心愿商城 ============================

type WishApi struct{}

// ItemList 心愿商品列表
// @Tags     MallWish
// @Summary  心愿商品列表
// @Produce  application/json
// @Param    data  body      mallReq.WishItemSearch                        true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}   "商品列表"
// @Router   /mall/wish/item/list [post]
func (a *WishApi) ItemList(c *gin.Context) {
	var info mallReq.WishItemSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := wishService.ItemList(c.Request.Context(), info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// ItemUpdate 调整心愿商品
// @Tags     MallWish
// @Summary  调整心愿商品（外链 / 上下架）
// @Produce  application/json
// @Param    data  body      mallReq.UpdateWishItemReq  true  "调整参数"
// @Success  200   {object}  response.Response{}        "调整结果"
// @Router   /mall/wish/item/update [post]
func (a *WishApi) ItemUpdate(c *gin.Context) {
	var req mallReq.UpdateWishItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := wishService.UpdateItem(c.Request.Context(), req); err != nil {
		response.FailWithMessage("调整失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("调整成功", c)
}

// OrderList 心愿单列表
// @Tags     MallWish
// @Summary  心愿单列表
// @Produce  application/json
// @Param    data  body      mallReq.WishOrderSearch                       true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}   "心愿单列表"
// @Router   /mall/wish/order/list [post]
func (a *WishApi) OrderList(c *gin.Context) {
	var info mallReq.WishOrderSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := wishService.OrderList(c.Request.Context(), info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// ============================ 任务 ============================

type TaskApi struct{}

// List 任务定义列表
// @Tags     MallTask
// @Summary  任务定义列表
// @Produce  application/json
// @Param    data  body      mallReq.TaskSearch                            true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}   "任务列表"
// @Router   /mall/task/list [post]
func (a *TaskApi) List(c *gin.Context) {
	var info mallReq.TaskSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := taskService.List(c.Request.Context(), info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// InstanceList 任务实例列表
// @Tags     MallTask
// @Summary  任务实例列表
// @Produce  application/json
// @Param    data  body      mallReq.TaskInstanceSearch                    true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}   "实例列表"
// @Router   /mall/task/instance/list [post]
func (a *TaskApi) InstanceList(c *gin.Context) {
	var info mallReq.TaskInstanceSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := taskService.InstanceList(c.Request.Context(), info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// ============================ 看板 ============================

type DashboardApi struct{}

// Overview 经营看板
// @Tags     MallDashboard
// @Summary  经营看板（实时统计）
// @Produce  application/json
// @Param    days  query     int  false  "统计最近几天，默认 7"
// @Success  200   {object}  response.Response{}  "统计数据"
// @Router   /mall/dashboard/overview [get]
func (a *DashboardApi) Overview(c *gin.Context) {
	days := parseInt(c.Query("days"), 7)
	data, err := dashboardService.Overview(c.Request.Context(), days)
	if err != nil {
		global.GVA_LOG.Error("获取经营看板失败", zap.Error(err))
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithData(data, c)
}

// ============================ 招商 ============================

type MerchantApi struct{}

// List 招商申请列表
// @Tags     MallMerchant
// @Summary  招商申请列表（待审核的排最前）
// @Produce  application/json
// @Param    data  body      mallReq.MerchantSearch                        true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}   "申请列表"
// @Router   /mall/merchant/list [post]
func (a *MerchantApi) List(c *gin.Context) {
	var info mallReq.MerchantSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := merchantService.List(c.Request.Context(), info)
	if err != nil {
		global.GVA_LOG.Error("获取招商申请失败", zap.Error(err))
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// Review 审核一条招商申请
// @Tags     MallMerchant
// @Summary  审核招商申请（通过时 admin_url 必填）
// @Produce  application/json
// @Param    data  body      mallReq.ReviewMerchantReq  true  "审核参数"
// @Success  200   {object}  response.Response{}        "审核结果"
// @Router   /mall/merchant/review [post]
func (a *MerchantApi) Review(c *gin.Context) {
	var req mallReq.ReviewMerchantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	operator := utils.GetUserName(c)
	if operator == "" {
		operator = "admin"
	}
	if err := merchantService.Review(c.Request.Context(), req, utils.GetUserID(c), operator); err != nil {
		// 业务错误（没填地址 / 已经审过）不是系统故障，别刷 ERROR 日志
		global.GVA_LOG.Warn("审核招商申请失败",
			zap.Uint("id", req.ID), zap.Bool("approve", req.Approve), zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	global.GVA_LOG.Info("招商申请已审核",
		zap.Uint("id", req.ID),
		zap.Bool("approve", req.Approve),
		zap.String("operator", operator))
	response.OkWithMessage("审核完成", c)
}

// PendingCount 待审核数
// @Tags     MallMerchant
// @Summary  待审核数（菜单红点）
// @Produce  application/json
// @Success  200   {object}  response.Response{data=int64}  "待审核数"
// @Router   /mall/merchant/pending-count [get]
func (a *MerchantApi) PendingCount(c *gin.Context) {
	n, err := merchantService.PendingCount(c.Request.Context())
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithData(n, c)
}
