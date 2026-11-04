package system

import (
	"github.com/gin-gonic/gin"
)

type BaseRouter struct{}

func (s *BaseRouter) InitBaseRouter(Router *gin.RouterGroup) (R gin.IRoutes) {
	baseRouter := Router.Group("base")
	{
		baseRouter.POST("login", baseApi.Login)
		baseRouter.POST("captcha", baseApi.Captcha)
		// appLogin 家长 / 商家用 **App 账号** 登录（手机号 + App 密码）。
		//
		// ⚠️ 它必须在这组（公开），因为这时候用户还没有 token。
		// 闸门不在路由上，在 service：`merchant_application` 里
		// 这个角色必须已经是 `approved`，否则一律拒（见 PortalService.AppLogin）。
		//
		// 处理器挂在 **Mall** 的 ApiGroup 上而不是 BaseApi：
		// 它读写的是平台表，放进 system 包会让「平台数据」这件事从 system 里渗出来。
		baseRouter.POST("appLogin", mallPortalApi.AppLogin)
	}
	return baseRouter
}
