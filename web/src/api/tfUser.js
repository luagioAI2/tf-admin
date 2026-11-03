import service from '@/utils/request'

/* ============================================================
   平台用户（App 的用户，不是后台管理员）
   ------------------------------------------------------------
   ⚠️ **家长和孩子不是两种账号。** `user` 表里一人一行，家长只是在这行上
      多设了一个 4 位密码（`user_settings.payload` 的 `parentPin`）用来进
      家长区。所以这一组接口只有「用户」，没有「家长列表」。

   ⚠️ 这一组管的是 `user` 表 —— 和 GVA 自带的「用户管理」
      （`sys_users`，后台登录账号）**不是一回事**。
      在后台看到两个「用户」入口时别搞混：改错那边等于把自己的管理员踢掉。

   ⚠️ 余额调整走 `adjust`，服务端会写一条 `manual_adjust` 账本流水。
      不要另做「直接改 wallet」的接口：余额和账本一旦对不上，
      对账时只看到「钱多了但没流水」，查不出是谁、什么时候、为什么。
   ============================================================ */

// @Summary App 用户列表（含钱包余额）
// @Router /mall/user/list [post]
export const getPlatformUserList = (data) => {
  return service({
    url: '/mall/user/list',
    method: 'post',
    data
  })
}

// @Summary App 用户详情（含钱包）
// @Router /mall/user/detail [get]
export const getPlatformUserDetail = (id) => {
  return service({
    url: '/mall/user/detail',
    method: 'get',
    params: { id }
  })
}

// @Summary App 用户账本流水
// @Router /mall/user/ledger [get]
export const getPlatformUserLedger = (params) => {
  return service({
    url: '/mall/user/ledger',
    method: 'get',
    params
  })
}

// @Summary 调整余额（走账本，必填备注）
// @Router /mall/user/adjust [post]
export const adjustPlatformUserBalance = (data) => {
  return service({
    url: '/mall/user/adjust',
    method: 'post',
    data
  })
}
