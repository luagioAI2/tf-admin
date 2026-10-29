import service from '@/utils/request'

// @Summary 招商申请列表（待审核的排最前）
// @Router /mall/merchant/list [post]
export const getMerchantList = (data) => {
  return service({
    url: '/mall/merchant/list',
    method: 'post',
    data
  })
}

// @Summary 审核招商申请（通过时 admin_url 必填）
// @Router /mall/merchant/review [post]
export const reviewMerchant = (data) => {
  return service({
    url: '/mall/merchant/review',
    method: 'post',
    data
  })
}

// @Summary 待审核数（菜单红点）
// @Router /mall/merchant/pending-count [get]
export const getMerchantPendingCount = () => {
  return service({
    url: '/mall/merchant/pending-count',
    method: 'get'
  })
}
