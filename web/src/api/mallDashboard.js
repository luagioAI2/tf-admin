import service from '@/utils/request'

/* ============================================================
   经营看板
   ------------------------------------------------------------
   一个只读接口，返回**已经算好**的十来个数字（服务端一次查完）。
   为什么不在前端各页分别拉：那是 N 个请求 + 前端自己求和，
   而「一共有多少用户 / 一共发了多少分」这种口径必须只有一处。

   ⚠️ 这些数是**平台表**（`user` / `wallet` / `redeem_records` /
      `wish_orders` / `task_instances` / `games`）的实时统计，
      不是后台自己的 `sys_*` 表。别拿「用户管理」里的人数对比。
   ============================================================ */

// @Summary 经营看板总览
// @Router /mall/dashboard/overview [get]
export const getMallOverview = () => {
  return service({
    url: '/mall/dashboard/overview',
    method: 'get'
  })
}

// @Summary 待审核招商数（看板上那条提醒用）
// @Router /mall/merchant/pending-count [get]
export const getMerchantPendingCount = () => {
  return service({
    url: '/mall/merchant/pending-count',
    method: 'get'
  })
}
