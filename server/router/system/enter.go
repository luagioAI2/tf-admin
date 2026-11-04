package system

import api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	ApiRouter
	JwtRouter
	SysRouter
	BaseRouter
	InitRouter
	MenuRouter
	UserRouter
	CasbinRouter
	AutoCodeRouter
	AuthorityRouter
	DictionaryRouter
	OperationRecordRouter
	DictionaryDetailRouter
	AuthorityBtnRouter
	SysExportTemplateRouter
	SysParamsRouter
	SysVersionRouter
	SysErrorRouter
	LoginLogRouter
	ApiTokenRouter
	SkillsRouter
}

var (
	dbApi                = api.ApiGroupApp.SystemApiGroup.DBApi
	jwtApi               = api.ApiGroupApp.SystemApiGroup.JwtApi
	baseApi              = api.ApiGroupApp.SystemApiGroup.BaseApi
	casbinApi            = api.ApiGroupApp.SystemApiGroup.CasbinApi
	systemApi            = api.ApiGroupApp.SystemApiGroup.SystemApi
	sysParamsApi         = api.ApiGroupApp.SystemApiGroup.SysParamsApi
	autoCodeApi          = api.ApiGroupApp.SystemApiGroup.AutoCodeApi
	authorityApi         = api.ApiGroupApp.SystemApiGroup.AuthorityApi
	apiRouterApi         = api.ApiGroupApp.SystemApiGroup.SystemApiApi
	dictionaryApi        = api.ApiGroupApp.SystemApiGroup.DictionaryApi
	authorityBtnApi      = api.ApiGroupApp.SystemApiGroup.AuthorityBtnApi
	authorityMenuApi     = api.ApiGroupApp.SystemApiGroup.AuthorityMenuApi
	autoCodePluginApi    = api.ApiGroupApp.SystemApiGroup.AutoCodePluginApi
	autocodeHistoryApi   = api.ApiGroupApp.SystemApiGroup.AutoCodeHistoryApi
	operationRecordApi   = api.ApiGroupApp.SystemApiGroup.OperationRecordApi
	autoCodePackageApi   = api.ApiGroupApp.SystemApiGroup.AutoCodePackageApi
	dictionaryDetailApi  = api.ApiGroupApp.SystemApiGroup.DictionaryDetailApi
	autoCodeTemplateApi  = api.ApiGroupApp.SystemApiGroup.AutoCodeTemplateApi
	exportTemplateApi    = api.ApiGroupApp.SystemApiGroup.SysExportTemplateApi
	sysVersionApi        = api.ApiGroupApp.SystemApiGroup.SysVersionApi
	sysErrorApi          = api.ApiGroupApp.SystemApiGroup.SysErrorApi
	skillsApi            = api.ApiGroupApp.SystemApiGroup.SkillsApi
	aiWorkflowSessionApi = api.ApiGroupApp.SystemApiGroup.AIWorkflowSessionApi
	// mallPortalApi 家长 / 商家门户的处理器。
	//
	// 放在这里只为一件事：`base/appLogin` 必须挂在**公开**路由组上
	// （登录时还没有 token），而那个组由 BaseRouter 建。
	// 处理器本身仍然属于 mall 模块，别把它的逻辑搬进 system。
	mallPortalApi = api.ApiGroupApp.MallApiGroup.PortalApi
)
