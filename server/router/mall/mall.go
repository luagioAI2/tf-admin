package mall

import (
	api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

// RouterGroup 组合本模块所有 Router。
//
// 新增一个 Router 时：在这里加一行，并在 `router/enter.go` 里
// 把 MallRouterGroup 挂到 RouterGroupApp，最后在
// `initialize/router.go` 的 Routers() 里调用它的 Init 方法。
type RouterGroup struct {
	GameRouter
	GameReleaseRouter
	PlatformUserRouter
	RedeemRouter
	WishRouter
	TaskRouter
	DashboardRouter
	MerchantRouter
}

var (
	gameApi         = api.ApiGroupApp.MallApiGroup.GameApi
	gameReleaseApi  = api.ApiGroupApp.MallApiGroup.GameReleaseApi
	platformUserApi = api.ApiGroupApp.MallApiGroup.PlatformUserApi
	redeemApi       = api.ApiGroupApp.MallApiGroup.RedeemApi
	wishApi         = api.ApiGroupApp.MallApiGroup.WishApi
	taskApi         = api.ApiGroupApp.MallApiGroup.TaskApi
	dashboardApi    = api.ApiGroupApp.MallApiGroup.DashboardApi
	merchantApi     = api.ApiGroupApp.MallApiGroup.MerchantApi
)

// ============================ 游戏与版本 ============================

type GameRouter struct{}

// InitGameRouter 游戏管理路由
//
// ⚠️ 写操作挂 `middleware.OperationRecord()` ——
// 它会把「谁、什么时候、改了什么」记进 `sys_operation_records`。
// 后台直接改的是生产数据（价格、上下架），没有操作日志等于出了事查不到人。
func (r *GameRouter) InitGameRouter(router *gin.RouterGroup) {
	group := router.Group("mall/game").Use(middleware.OperationRecord())
	groupWithoutRecord := router.Group("mall/game")
	{
		group.POST("update", gameApi.Update)
	}
	{
		groupWithoutRecord.POST("list", gameApi.List)
		groupWithoutRecord.POST("options", gameApi.Options)
	}
}

type GameReleaseRouter struct{}

// InitGameReleaseRouter 游戏版本路由
func (r *GameReleaseRouter) InitGameReleaseRouter(router *gin.RouterGroup) {
	group := router.Group("mall/release").Use(middleware.OperationRecord())
	groupWithoutRecord := router.Group("mall/release")
	{
		group.POST("publish", gameReleaseApi.Publish)
		group.POST("deactivate", gameReleaseApi.Deactivate)
		group.DELETE("delete", gameReleaseApi.Delete)
	}
	{
		groupWithoutRecord.POST("list", gameReleaseApi.List)
	}
}

// ============================ 平台用户 ============================

type PlatformUserRouter struct{}

// InitPlatformUserRouter 平台用户路由
func (r *PlatformUserRouter) InitPlatformUserRouter(router *gin.RouterGroup) {
	group := router.Group("mall/user").Use(middleware.OperationRecord())
	groupWithoutRecord := router.Group("mall/user")
	{
		// 改余额是敏感操作，必须留痕
		group.POST("adjust", platformUserApi.AdjustBalance)
	}
	{
		groupWithoutRecord.POST("list", platformUserApi.List)
		groupWithoutRecord.GET("detail", platformUserApi.Detail)
		groupWithoutRecord.GET("ledger", platformUserApi.Ledger)
	}
}

// ============================ 兑换商城 ============================

type RedeemRouter struct{}

// InitRedeemRouter 兑换商城路由
func (r *RedeemRouter) InitRedeemRouter(router *gin.RouterGroup) {
	group := router.Group("mall/redeem").Use(middleware.OperationRecord())
	groupWithoutRecord := router.Group("mall/redeem")
	{
		group.POST("item/update", redeemApi.ItemUpdate)
	}
	{
		groupWithoutRecord.POST("item/list", redeemApi.ItemList)
		groupWithoutRecord.POST("record/list", redeemApi.RecordList)
	}
}

// ============================ 心愿商城 ============================

type WishRouter struct{}

// InitWishRouter 心愿商城路由
func (r *WishRouter) InitWishRouter(router *gin.RouterGroup) {
	group := router.Group("mall/wish").Use(middleware.OperationRecord())
	groupWithoutRecord := router.Group("mall/wish")
	{
		group.POST("item/update", wishApi.ItemUpdate)
	}
	{
		groupWithoutRecord.POST("item/list", wishApi.ItemList)
		groupWithoutRecord.POST("order/list", wishApi.OrderList)
	}
}

// ============================ 任务 ============================

type TaskRouter struct{}

// InitTaskRouter 任务路由
func (r *TaskRouter) InitTaskRouter(router *gin.RouterGroup) {
	groupWithoutRecord := router.Group("mall/task")
	{
		groupWithoutRecord.POST("list", taskApi.List)
		groupWithoutRecord.POST("instance/list", taskApi.InstanceList)
	}
}

// ============================ 看板 ============================

type DashboardRouter struct{}

// InitDashboardRouter 经营看板路由
func (r *DashboardRouter) InitDashboardRouter(router *gin.RouterGroup) {
	groupWithoutRecord := router.Group("mall/dashboard")
	{
		groupWithoutRecord.GET("overview", dashboardApi.Overview)
	}
}

// ============================ 招商 ============================

type MerchantRouter struct{}

// InitMerchantRouter 招商审核路由
//
// ⚠️ `review` 必须挂 `middleware.OperationRecord()`：
// 它改的是「谁能拿到后台入口」—— 出了事要查得到是谁在什么时候点的通过。
// 列表 / 待办数只是读，不挂。
func (r *MerchantRouter) InitMerchantRouter(router *gin.RouterGroup) {
	group := router.Group("mall/merchant").Use(middleware.OperationRecord())
	groupWithoutRecord := router.Group("mall/merchant")
	{
		group.POST("review", merchantApi.Review)
	}
	{
		groupWithoutRecord.POST("list", merchantApi.List)
		groupWithoutRecord.GET("pending-count", merchantApi.PendingCount)
	}
}
