package mall

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	mall "github.com/flipped-aurora/gin-vue-admin/server/model/mall"
	mallReq "github.com/flipped-aurora/gin-vue-admin/server/model/mall/request"
	system "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	mallSvc "github.com/flipped-aurora/gin-vue-admin/server/service/mall"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PortalApi 家长 / 商家门户。
//
// 这些接口**只有影子账号能调**（用户名 `app_{uid}`）。真正的准入有三层，
// 各自独立、缺一层都能出数据事故：
//
//  1. casbin    —— 这个角色能不能调这个地址（1001 / 1002 的 policy）
//  2. appUID()  —— 这个 token 到底是谁（解析不出就直接拒，**绝不回落成 0**）
//  3. service   —— 每条 SQL 都带 `WHERE uid = ?`
//
// 少第 1 层：接口能被运营账号调；少第 2 层：uid 变成 0（平台自营 / 游客）；
// 少第 3 层：家长 A 拿自己的 token 翻家长 B 的订单 —— 而界面完全正常。
type PortalApi struct{}

// appUID 取当前 token 对应的 App 用户 id。
//
// ⚠️ 解析失败**必须**直接返回 401 语义，不能兜底成 0：
// `uid = 0` 在平台表里是「游客 / 平台自营」，兜底等于把平台自己的数据
// 当成这个人的数据交出去，而且不报错。
func appUID(c *gin.Context) (uint, bool) {
	uid, err := mallSvc.AppUIDOf(utils.GetUserName(c))
	if err != nil {
		response.FailWithMessage("这个账号不能访问门户接口", c)
		return 0, false
	}
	return uid, true
}

// mallLoginLog 组装一条登录日志。
//
// 复用 GVA 的 `sys_login_log`，不另建表：运营只需要在一个地方看
// 「谁在什么时候从哪个 IP 登录失败」，多一张表就是多一个会被忘掉的地方。
func mallLoginLog(username, ip, agent, errMsg string) system.SysLoginLog {
	return system.SysLoginLog{
		Username:     username,
		Ip:           ip,
		Agent:        agent,
		Status:       errMsg == "",
		ErrorMessage: errMsg,
	}
}

// ============================ 登录桥 ============================

// AppLogin 用 App 账号登录后台（家长 / 商家）。
//
// ★ 公网可达（和 `base/login` 同级），所以：
//   - 错误措辞**不区分**「没这个号」和「密码错」（否则等于送一个探测接口）
//   - 失败要落 `sys_login_log`，运营才看得到「有人在拿手机号撞库」
//
// @Tags     MallPortal
// @Summary  用 App 账号登录后台（家长 / 商家）
// @Produce  application/json
// @Param    data  body      mallReq.AppLoginReq                                     true  "手机号 / 密码 / 门户类型"
// @Success  200   {object}  response.Response{data=systemRes.LoginResponse,msg=string}  "token 与用户信息"
// @Router   /base/appLogin [post]
func (a *PortalApi) AppLogin(c *gin.Context) {
	var req mallReq.AppLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	res, err := portalService.AppLogin(c.Request.Context(), req)
	if err != nil {
		// 业务拒绝（未审核 / 密码错）不是系统故障，走 Warn 不刷 ERROR 堆栈
		global.GVA_LOG.Warn("门户登录被拒",
			zap.String("role", req.Role),
			zap.String("mobile", maskMobile(req.Mobile)),
			zap.String("ip", c.ClientIP()),
			zap.Error(err))
		loginLogService.CreateLoginLog(mallLoginLog(
			"app:"+req.Role, c.ClientIP(), c.Request.UserAgent(), err.Error()))
		response.FailWithMessage(err.Error(), c)
		return
	}

	global.GVA_LOG.Info("门户登录成功",
		zap.String("role", req.Role),
		zap.String("user", res.User.Username),
		zap.String("ip", c.ClientIP()))
	loginLogService.CreateLoginLog(mallLoginLog(res.User.Username, c.ClientIP(), c.Request.UserAgent(), ""))
	response.OkWithDetailed(res, "登录成功", c)
}

// maskMobile 日志里只留前 3 后 4。
//
// 后台日志会被截图发到群里、会被备份到别的地方；完整手机号是个人信息，
// 排查「谁在撞库」只要认出是同一个号就够了。
func maskMobile(m string) string {
	if len(m) <= 7 {
		return "***"
	}
	return m[:3] + "****" + m[len(m)-4:]
}

// ============================ 家长端 ============================

// ParentOverview 家长首页概览（余额 + 心愿各状态计数 + 最近 5 单）
// @Tags     MallPortal
// @Summary  家长首页概览
// @Produce  application/json
// @Success  200  {object}  response.Response{data=map[string]interface{}}  "概览"
// @Router   /mall/portal/parent/overview [get]
func (a *PortalApi) ParentOverview(c *gin.Context) {
	uid, ok := appUID(c)
	if !ok {
		return
	}
	data, err := portalService.ParentOverview(c.Request.Context(), uid)
	if err != nil {
		global.GVA_LOG.Error("获取家长概览失败", zap.Uint("uid", uid), zap.Error(err))
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithData(data, c)
}

// ParentWishes 家长的心愿单
// @Tags     MallPortal
// @Summary  家长心愿单列表
// @Produce  application/json
// @Param    data  body      mallReq.ParentWishSearch                       true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}    "心愿单"
// @Router   /mall/portal/parent/wishes [post]
func (a *PortalApi) ParentWishes(c *gin.Context) {
	uid, ok := appUID(c)
	if !ok {
		return
	}
	var info mallReq.ParentWishSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := portalService.ParentWishes(c.Request.Context(), uid, info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// ParentRecords 家长的兑换记录
// @Tags     MallPortal
// @Summary  家长兑换记录
// @Produce  application/json
// @Param    data  body      mallReq.ParentRecordSearch                     true  "分页"
// @Success  200   {object}  response.Response{data=response.PageResult}    "兑换记录"
// @Router   /mall/portal/parent/records [post]
func (a *PortalApi) ParentRecords(c *gin.Context) {
	uid, ok := appUID(c)
	if !ok {
		return
	}
	var info mallReq.ParentRecordSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := portalService.ParentRecords(c.Request.Context(), uid, info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// ============================ 商家端 ============================

// MerchantStatus 商家自己的入驻状态
// @Tags     MallPortal
// @Summary  商家入驻状态（含驳回理由）
// @Produce  application/json
// @Success  200  {object}  response.Response{data=mall.MerchantApplication}  "状态"
// @Router   /mall/portal/merchant/status [get]
func (a *PortalApi) MerchantStatus(c *gin.Context) {
	uid, ok := appUID(c)
	if !ok {
		return
	}
	// 显式声明类型只为让 swag 能解析上面那行 `data=mall.MerchantApplication`
	var (
		app *mall.MerchantApplication
		err error
	)
	app, err = portalService.MerchantStatus(c.Request.Context(), uid)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	// 没申请过时 app 为 nil，返回 null 而不是错误 —— 前端据此显示「去 App 里申请」
	response.OkWithData(app, c)
}

// MerchantItems 商家自己的商品
// @Tags     MallPortal
// @Summary  我的商品
// @Produce  application/json
// @Param    data  body      mallReq.MerchantItemSearch                     true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}    "商品列表"
// @Router   /mall/portal/merchant/items [post]
func (a *PortalApi) MerchantItems(c *gin.Context) {
	uid, ok := appUID(c)
	if !ok {
		return
	}
	var info mallReq.MerchantItemSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := portalService.MerchantItems(c.Request.Context(), uid, info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// MerchantSaveItem 上架 / 改价 / 下架自己的一件商品
// @Tags     MallPortal
// @Summary  保存我的商品（itemId 留空 = 新建）
// @Produce  application/json
// @Param    data  body      mallReq.MerchantSaveItemReq  true  "商品字段"
// @Success  200   {object}  response.Response{}          "itemId"
// @Router   /mall/portal/merchant/item/save [post]
func (a *PortalApi) MerchantSaveItem(c *gin.Context) {
	uid, ok := appUID(c)
	if !ok {
		return
	}
	var req mallReq.MerchantSaveItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	itemID, err := portalService.MerchantSaveItem(c.Request.Context(), uid, req)
	if err != nil {
		// 「不在你名下」是越权尝试，值得 Warn 留痕
		global.GVA_LOG.Warn("商家保存商品失败",
			zap.Uint("uid", uid), zap.String("item_id", req.ItemID), zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"itemId": itemID}, "已保存", c)
}
