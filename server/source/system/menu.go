package system

import (
	"context"

	. "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const initOrderMenu = initOrderAuthority + 1

type initMenu struct{}

// auto run
func init() {
	system.RegisterInit(initOrderMenu, &initMenu{})
}

func (i *initMenu) InitializerName() string {
	return SysBaseMenu{}.TableName()
}

func (i *initMenu) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(
		&SysBaseMenu{},
		&SysBaseMenuParameter{},
		&SysBaseMenuBtn{},
	)
}

func (i *initMenu) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	m := db.Migrator()
	return m.HasTable(&SysBaseMenu{}) &&
		m.HasTable(&SysBaseMenuParameter{}) &&
		m.HasTable(&SysBaseMenuBtn{})
}

func (i *initMenu) InitializeData(ctx context.Context) (next context.Context, err error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	// 定义所有菜单
	allMenus := []SysBaseMenu{
		{MenuLevel: 0, Hidden: false, ParentId: 0, Path: "dashboard", Name: "dashboard", Component: "view/dashboard/index.vue", Sort: 1, Meta: Meta{Title: "首页", Icon: "odometer"}},
		{MenuLevel: 0, Hidden: false, ParentId: 0, Path: "about", Name: "about", Component: "view/about/index.vue", Sort: 9, Meta: Meta{Title: "关于我们", Icon: "info-filled"}},
		{MenuLevel: 0, Hidden: false, ParentId: 0, Path: "admin", Name: "superAdmin", Component: "view/superAdmin/index.vue", Sort: 3, Meta: Meta{Title: "超级管理员", Icon: "user"}},
		{MenuLevel: 0, Hidden: true, ParentId: 0, Path: "person", Name: "person", Component: "view/person/person.vue", Sort: 4, Meta: Meta{Title: "个人信息", Icon: "message"}},
		{MenuLevel: 0, Hidden: true, ParentId: 0, Path: "example", Name: "example", Component: "view/example/index.vue", Sort: 7, Meta: Meta{Title: "示例文件", Icon: "management"}},
		{MenuLevel: 0, Hidden: true, ParentId: 0, Path: "systemTools", Name: "systemTools", Component: "view/systemTools/index.vue", Sort: 5, Meta: Meta{Title: "编程辅助", Icon: "tools"}},
		{MenuLevel: 0, Hidden: true, ParentId: 0, Path: "https://www.gin-vue-admin.com", Name: "https://www.gin-vue-admin.com", Component: "/", Sort: 0, Meta: Meta{Title: "官方网站", Icon: "customer-gva"}},
		{MenuLevel: 0, Hidden: false, ParentId: 0, Path: "state", Name: "state", Component: "view/system/state.vue", Sort: 8, Meta: Meta{Title: "运行状态", Icon: "cloudy"}},
		{MenuLevel: 0, Hidden: true, ParentId: 0, Path: "plugin", Name: "plugin", Component: "view/routerHolder.vue", Sort: 6, Meta: Meta{Title: "插件系统", Icon: "cherry"}},
		// 商城运营 / 心愿商城 / 游戏 是三个平级的一级菜单，各自挂子页面。
		// ★ component 必须是 web/src/view 下真实存在的 .vue：不存在时 gin-vue-admin
		//   只报一个 404 chunk，看着像「权限不足」，其实点进去是空白页。
		// ★ 心愿商品 / 心愿单 / 游戏激活码 指向同一个 view/mall/wish/index.vue，
		//   由路由 name 决定默认页签（见该文件的 MODE_BY_NAME）。
		{MenuLevel: 0, Hidden: false, ParentId: 0, Path: "mall", Name: "mall", Component: "view/routerHolder.vue", Sort: 2, Meta: Meta{Title: "商城运营", Icon: "shop"}},
		// 模板配置：新用户首次启动铺的任务 / 兑换内容。
		// ★ 商家也有这一页（`商家中心 → 我的模板`），但走的是 seed-portal.py 那棵树，
		//   不在这里 —— initdb 只建超管这套菜单。
		{MenuLevel: 0, Hidden: false, ParentId: 0, Path: "templateRoot", Name: "templateRoot", Component: "view/routerHolder.vue", Sort: 6, Meta: Meta{Title: "模板配置", Icon: "files"}},
		{MenuLevel: 0, Hidden: false, ParentId: 0, Path: "wishMall", Name: "wishMall", Component: "view/routerHolder.vue", Sort: 4, Meta: Meta{Title: "心愿商城", Icon: "present"}},
		{MenuLevel: 0, Hidden: false, ParentId: 0, Path: "gameRoot", Name: "gameRoot", Component: "view/routerHolder.vue", Sort: 5, Meta: Meta{Title: "游戏", Icon: "platform"}},
	}

	// 先创建父级菜单（ParentId = 0 的菜单）
	if err = db.Create(&allMenus).Error; err != nil {
		return ctx, errors.Wrap(err, SysBaseMenu{}.TableName()+"父级菜单初始化失败!")
	}

	// 建立菜单映射 - 通过Name查找已创建的菜单及其ID
	menuNameMap := make(map[string]uint)
	for _, menu := range allMenus {
		menuNameMap[menu.Name] = menu.ID
	}

	// 定义子菜单，并设置正确的ParentId
	childMenus := []SysBaseMenu{
		// superAdmin子菜单
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["superAdmin"], Path: "authority", Name: "authority", Component: "view/superAdmin/authority/authority.vue", Sort: 1, Meta: Meta{Title: "角色管理", Icon: "avatar"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["superAdmin"], Path: "menu", Name: "menu", Component: "view/superAdmin/menu/menu.vue", Sort: 2, Meta: Meta{Title: "菜单管理", Icon: "tickets", KeepAlive: true}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["superAdmin"], Path: "api", Name: "api", Component: "view/superAdmin/api/api.vue", Sort: 3, Meta: Meta{Title: "接口管理", Icon: "platform", KeepAlive: true}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["superAdmin"], Path: "user", Name: "user", Component: "view/superAdmin/user/user.vue", Sort: 4, Meta: Meta{Title: "用户管理", Icon: "coordinate"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["superAdmin"], Path: "dictionary", Name: "dictionary", Component: "view/superAdmin/dictionary/sysDictionary.vue", Sort: 5, Meta: Meta{Title: "字典管理", Icon: "notebook"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["superAdmin"], Path: "operation", Name: "operation", Component: "view/superAdmin/operation/sysOperationRecord.vue", Sort: 6, Meta: Meta{Title: "操作历史", Icon: "pie-chart"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["superAdmin"], Path: "sysParams", Name: "sysParams", Component: "view/superAdmin/params/sysParams.vue", Sort: 7, Meta: Meta{Title: "参数管理", Icon: "compass"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["superAdmin"], Path: "system", Name: "system", Component: "view/systemTools/system/system.vue", Sort: 8, Meta: Meta{Title: "系统配置", Icon: "operation"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["superAdmin"], Path: "apiToken", Name: "apiToken", Component: "view/systemTools/apiToken/index.vue", Sort: 9, Meta: Meta{Title: "API Token", Icon: "key"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["superAdmin"], Path: "loginLog", Name: "loginLog", Component: "view/systemTools/loginLog/index.vue", Sort: 10, Meta: Meta{Title: "登录日志", Icon: "monitor"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["superAdmin"], Path: "sysVersion", Name: "sysVersion", Component: "view/systemTools/version/version.vue", Sort: 11, Meta: Meta{Title: "版本管理", Icon: "server"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["superAdmin"], Path: "sysError", Name: "sysError", Component: "view/systemTools/sysError/sysError.vue", Sort: 12, Meta: Meta{Title: "错误日志", Icon: "warn"}},

		// example子菜单
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["example"], Path: "upload", Name: "upload", Component: "view/example/upload/upload.vue", Sort: 5, Meta: Meta{Title: "媒体库（上传下载）", Icon: "upload"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["example"], Path: "breakpoint", Name: "breakpoint", Component: "view/example/breakpoint/breakpoint.vue", Sort: 6, Meta: Meta{Title: "断点续传", Icon: "upload-filled"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["example"], Path: "customer", Name: "customer", Component: "view/example/customer/customer.vue", Sort: 7, Meta: Meta{Title: "客户列表（资源示例）", Icon: "avatar"}},

		// systemTools子菜单
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["systemTools"], Path: "autoPkg", Name: "autoPkg", Component: "view/systemTools/autoPkg/autoPkg.vue", Sort: 0, Meta: Meta{Title: "模板配置", Icon: "folder"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["systemTools"], Path: "autoCode", Name: "autoCode", Component: "view/systemTools/autoCode/index.vue", Sort: 1, Meta: Meta{Title: "代码生成器", Icon: "cpu", KeepAlive: true}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["systemTools"], Path: "autoCodeAdmin", Name: "autoCodeAdmin", Component: "view/systemTools/autoCodeAdmin/index.vue", Sort: 2, Meta: Meta{Title: "自动化代码管理", Icon: "magic-stick"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["systemTools"], Path: "formCreate", Name: "formCreate", Component: "view/systemTools/formCreate/index.vue", Sort: 3, Meta: Meta{Title: "表单生成器", Icon: "magic-stick", KeepAlive: true}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["systemTools"], Path: "aiWorkflow", Name: "aiWorkflow", Component: "view/systemTools/aiWrokflow/index.vue", Sort: 4, Meta: Meta{Title: "AI需求工作流", Icon: "magic-stick", KeepAlive: true}},
		{MenuLevel: 1, Hidden: true, ParentId: menuNameMap["systemTools"], Path: "autoCodeEdit/:id", Name: "autoCodeEdit", Component: "view/systemTools/autoCode/index.vue", Sort: 0, Meta: Meta{Title: "自动化代码-${id}", Icon: "magic-stick"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["systemTools"], Path: "exportTemplate", Name: "exportTemplate", Component: "view/systemTools/exportTemplate/exportTemplate.vue", Sort: 5, Meta: Meta{Title: "导出模板", Icon: "reading"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["systemTools"], Path: "mcpTest", Name: "mcpTest", Component: "view/systemTools/autoCode/mcpTest.vue", Sort: 6, Meta: Meta{Title: "Mcp Tools管理", Icon: "partly-cloudy"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["systemTools"], Path: "mcpTool", Name: "mcpTool", Component: "view/systemTools/autoCode/mcp.vue", Sort: 7, Meta: Meta{Title: "Mcp Tools模板", Icon: "magnet"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["systemTools"], Path: "skills", Name: "skills", Component: "view/systemTools/skills/index.vue", Sort: 8, Meta: Meta{Title: "Skills管理", Icon: "document"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["systemTools"], Path: "picture", Name: "picture", Component: "view/systemTools/autoCode/picture.vue", Sort: 9, Meta: Meta{Title: "AI页面绘制", Icon: "picture-filled"}},

		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["plugin"], Path: "https://plugin.gin-vue-admin.com/", Name: "https://plugin.gin-vue-admin.com/", Component: "https://plugin.gin-vue-admin.com/", Sort: 0, Meta: Meta{Title: "插件市场", Icon: "shop"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["plugin"], Path: "installPlugin", Name: "installPlugin", Component: "view/systemTools/installPlugin/index.vue", Sort: 1, Meta: Meta{Title: "插件安装", Icon: "box"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["plugin"], Path: "pubPlug", Name: "pubPlug", Component: "view/systemTools/pubPlug/pubPlug.vue", Sort: 3, Meta: Meta{Title: "打包插件", Icon: "files"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["plugin"], Path: "plugin-email", Name: "plugin-email", Component: "plugin/email/view/index.vue", Sort: 4, Meta: Meta{Title: "邮件插件", Icon: "message"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["plugin"], Path: "anInfo", Name: "anInfo", Component: "plugin/announcement/view/info.vue", Sort: 5, Meta: Meta{Title: "公告管理[示例]", Icon: "scaleToOriginal"}},

		// mall子菜单
		// mall子菜单（和 scripts/seed-menu.py / seed-menu-level2.py 对齐）
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["mall"], Path: "user", Name: "appUser", Component: "view/mall/user/index.vue", Sort: 0, Meta: Meta{Title: "App 用户", Icon: "user"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["mall"], Path: "dashboard", Name: "mallDashboard", Component: "view/mall/dashboard/index.vue", Sort: 1, Meta: Meta{Title: "经营看板", Icon: "data-analysis"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["mall"], Path: "redeem", Name: "mallRedeem", Component: "view/mall/redeem/index.vue", Sort: 2, Meta: Meta{Title: "兑换商城", Icon: "shopping-cart"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["mall"], Path: "merchant", Name: "merchant", Component: "view/mall/merchant/index.vue", Sort: 3, Meta: Meta{Title: "招商审核", Icon: "message"}},

		// 心愿商城（一级菜单 wishMall 的子页；三个菜单共用 wish/index.vue）
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["wishMall"], Path: "item", Name: "mallWish", Component: "view/mall/wish/index.vue", Sort: 0, Meta: Meta{Title: "心愿商品", Icon: "goods"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["wishMall"], Path: "order", Name: "wishOrder", Component: "view/mall/wish/index.vue", Sort: 1, Meta: Meta{Title: "心愿单", Icon: "tickets"}},

		// 模板配置（一级菜单 templateRoot 的子页；两个菜单共用 template/index.vue）
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["templateRoot"], Path: "task", Name: "templateTask", Component: "view/mall/template/index.vue", Sort: 0, Meta: Meta{Title: "任务模板", Icon: "document"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["templateRoot"], Path: "redeem", Name: "templateRedeem", Component: "view/mall/template/index.vue", Sort: 1, Meta: Meta{Title: "兑换模板", Icon: "shopping-cart"}},

		// 游戏（一级菜单 gameRoot 的子页）
		// ★ 只授权给 888：家长 1001 / 商家 1002 / 家长+商家 1003 都看不到这组，
		//   /mall/game/* 与 /mall/wish/* 的 casbin 也只有 888，别顺手给门户开。
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["gameRoot"], Path: "list", Name: "gameList", Component: "view/mall/game/index.vue", Sort: 0, Meta: Meta{Title: "游戏列表", Icon: "grid"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["gameRoot"], Path: "code", Name: "gameCode", Component: "view/mall/wish/index.vue", Sort: 1, Meta: Meta{Title: "游戏激活码", Icon: "key"}},
		{MenuLevel: 1, Hidden: false, ParentId: menuNameMap["gameRoot"], Path: "release", Name: "gameRelease", Component: "view/mall/release/index.vue", Sort: 2, Meta: Meta{Title: "版本发布", Icon: "upload"}},
	}

	// 创建子菜单
	if err = db.Create(&childMenus).Error; err != nil {
		return ctx, errors.Wrap(err, SysBaseMenu{}.TableName()+"子菜单初始化失败!")
	}

	// 组合所有菜单作为返回结果
	allEntities := append(allMenus, childMenus...)
	next = context.WithValue(ctx, i.InitializerName(), allEntities)
	return next, nil
}

func (i *initMenu) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("path = ?", "autoPkg").First(&SysBaseMenu{}).Error, gorm.ErrRecordNotFound) { // 判断是否存在数据
		return false
	}
	return true
}
