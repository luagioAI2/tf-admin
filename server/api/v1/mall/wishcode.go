package mall

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	mallReq "github.com/flipped-aurora/gin-vue-admin/server/model/mall/request"
	"github.com/gin-gonic/gin"
)

// WishCodeApi 激活码（后台）
//
// 生成 / 查看 / 作废。核销不在这里 —— 那是孩子在 App 里输码时发生的事，
// 由 `tf-platform-api` 的 `/api/wish/redeem-code` 处理（同一个 tf.db）。
type WishCodeApi struct{}

// CodeList 激活码列表
// @Tags     MallWishCode
// @Summary  激活码列表
// @Produce  application/json
// @Param    data  body      mallReq.WishCodeSearch                        true  "分页与筛选"
// @Success  200   {object}  response.Response{data=response.PageResult}   "码列表"
// @Router   /mall/wish/code/list [post]
func (a *WishCodeApi) CodeList(c *gin.Context) {
	var info mallReq.WishCodeSearch
	if err := c.ShouldBindJSON(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := wishCodeService.CodeList(c.Request.Context(), info)
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List: list, Total: total, Page: info.Page, PageSize: info.PageSize,
	}, "获取成功", c)
}

// GenerateCodes 批量生成激活码
//
// ★ 返回的 `codes` 是**明文**，只此一次。表里只有 sha256，
// 丢了只能作废重发 —— 界面上必须当场让运营复制走。
// @Tags     MallWishCode
// @Summary  批量生成激活码
// @Produce  application/json
// @Param    data  body      mallReq.GenerateWishCodeReq  true  "生成参数"
// @Success  200   {object}  response.Response{}          "明文码列表"
// @Router   /mall/wish/code/generate [post]
func (a *WishCodeApi) GenerateCodes(c *gin.Context) {
	var req mallReq.GenerateWishCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	codes, err := wishCodeService.GenerateCodes(c.Request.Context(), req)
	if err != nil {
		response.FailWithMessage("生成失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{
		"codes": codes,
		"count": len(codes),
		"batch": req.Batch,
	}, "生成成功", c)
}

// VoidCode 作废激活码（单张 / 整批未使用的）
// @Tags     MallWishCode
// @Summary  作废激活码
// @Produce  application/json
// @Param    data  body      mallReq.VoidWishCodeReq  true  "id 或 batch"
// @Success  200   {object}  response.Response{}      "作废结果"
// @Router   /mall/wish/code/void [post]
func (a *WishCodeApi) VoidCode(c *gin.Context) {
	var req mallReq.VoidWishCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	n, err := wishCodeService.VoidCode(c.Request.Context(), req)
	if err != nil {
		response.FailWithMessage("作废失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"voided": n}, "已作废", c)
}

// CodeBatches 批次号下拉
// @Tags     MallWishCode
// @Summary  已有批次号
// @Produce  application/json
// @Success  200  {object}  response.Response{}  "批次号"
// @Router   /mall/wish/code/batches [post]
func (a *WishCodeApi) CodeBatches(c *gin.Context) {
	list, err := wishCodeService.CodeBatches(c.Request.Context())
	if err != nil {
		response.FailWithMessage("获取失败："+err.Error(), c)
		return
	}
	response.OkWithData(list, c)
}
