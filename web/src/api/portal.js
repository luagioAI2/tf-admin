import service from '@/utils/request'

// 家长 / 商家门户的接口。
//
// ⚠️ 这些接口**只能**用 `/base/appLogin` 换来的 token 调：
// 后端拿用户名反查 App 的 uid（`app_{uid}`），解析不出来就拒。
// 所以这里的每一个请求都天然只能看到「自己」的数据 ——
// 传 uid 参数也没用，后端根本不读。

// ============================ 家长 ============================

// @Summary 家长概览（余额 + 心愿各状态计数 + 最近 5 单）
// @Router /mall/portal/parent/overview [get]
export const getParentOverview = () => {
  return service({
    url: '/mall/portal/parent/overview',
    method: 'get'
  })
}

// @Summary 家长心愿单
// @Router /mall/portal/parent/wishes [post]
export const getParentWishes = (data) => {
  return service({
    url: '/mall/portal/parent/wishes',
    method: 'post',
    data
  })
}

// @Summary 家长兑换记录
// @Router /mall/portal/parent/records [post]
export const getParentRecords = (data) => {
  return service({
    url: '/mall/portal/parent/records',
    method: 'post',
    data
  })
}

// ============================ 商家 ============================

// @Summary 商家入驻状态（没申请过返回 null）
// @Router /mall/portal/merchant/status [get]
export const getMerchantStatus = () => {
  return service({
    url: '/mall/portal/merchant/status',
    method: 'get'
  })
}

// @Summary 我的商品
// @Router /mall/portal/merchant/items [post]
export const getMerchantItems = (data) => {
  return service({
    url: '/mall/portal/merchant/items',
    method: 'post',
    data
  })
}

// @Summary 上架 / 改价 / 下架自己的一件商品（itemId 留空 = 新建）
// @Router /mall/portal/merchant/item/save [post]
export const saveMerchantItem = (data) => {
  return service({
    url: '/mall/portal/merchant/item/save',
    method: 'post',
    data
  })
}
