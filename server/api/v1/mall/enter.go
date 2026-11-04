package mall

import (
	"github.com/flipped-aurora/gin-vue-admin/server/service"
)

// ApiGroup 组合本模块所有 API。
//
// 新增一个 API 时：在这里加一行，并在 `api/v1/enter.go` 里
// 把 MallApiGroup 挂到 ApiGroupApp。
type ApiGroup struct {
	GameApi
	GameReleaseApi
	PlatformUserApi
	RedeemApi
	WishApi
	WishCodeApi
	TaskApi
	DashboardApi
	MerchantApi
	// PortalApi 家长 / 商家门户
	PortalApi
	// TemplateApi 任务 / 兑换模板配置
	TemplateApi
}

// 从 ServiceGroupApp 取 Service 实例（gin-vue-admin 的约定：
// API 层不自己 new，统一从这里拿，保证单例）
var (
	gameService         = service.ServiceGroupApp.MallServiceGroup.GameService
	gameReleaseService  = service.ServiceGroupApp.MallServiceGroup.GameReleaseService
	platformUserService = service.ServiceGroupApp.MallServiceGroup.PlatformUserService
	redeemService       = service.ServiceGroupApp.MallServiceGroup.RedeemService
	wishService         = service.ServiceGroupApp.MallServiceGroup.WishService
	wishCodeService     = service.ServiceGroupApp.MallServiceGroup.WishCodeService
	taskService         = service.ServiceGroupApp.MallServiceGroup.TaskService
	dashboardService    = service.ServiceGroupApp.MallServiceGroup.DashboardService
	merchantService     = service.ServiceGroupApp.MallServiceGroup.MerchantService
	portalService       = service.ServiceGroupApp.MallServiceGroup.PortalService
	templateService     = service.ServiceGroupApp.MallServiceGroup.TemplateService
	// loginLogService 门户登录也落 `sys_login_log`：
	// 公网可达的登录入口如果不记失败，撞库只能从 nginx 日志里翻
	loginLogService = service.ServiceGroupApp.SystemServiceGroup.LoginLogService
)
