import service from '@/utils/request'

/* ============================================================
   心愿商城（后台）
   ------------------------------------------------------------
   ⚠️ 和兑换商城**不是一回事**，别照着 redeem 的直觉改：
     · 兑换商品是**每个孩子一份**（`redeem_item.uid`）；
       心愿商品是**全站一份**（`wish_item` 没有 uid）。
       所以这一页没有「所属」筛选，列表里也不该出现人名。
     · 兑换价是 `price`（单位 = 券）；心愿商品上的 `price_cents` 是
       **人民币分**（6900 = 69 元），而且只是给家长看的**参考价**。
       真正的兑换价在订单上（`voucher_price`，单位 = 券）。
       把 `price_cents` 当券数显示，会把「69 元」写成「6900 券」。

   ⚠️ 后台能改商品本身（`item/create` / `item/update` / `item/delete`），
      **但仍然不能改订单状态** —— 状态机是孩子和家长在 App 里推的
      （wished → committed → confirmed → done / cancelled）。
      后台长出「改状态」按钮就等于能凭空扣券、凭空退券。

   ⚠️ `buy_url` 孩子那边**看不到**：C 端给孩子的响应里不返回这个字段，
      它只出现在家长的确认页。所以这里是「家长去哪买」的链接，
      不是推广位。
   ============================================================ */

// @Summary 心愿商品列表（全局一份）
// @Router /mall/wish/item/list [post]
export const getWishItemList = (data) => {
  return service({
    url: '/mall/wish/item/list',
    method: 'post',
    data
  })
}

// @Summary 调整心愿商品（全字段；只传要改的那几个）
// @Router /mall/wish/item/update [post]
export const updateWishItem = (data) => {
  return service({
    url: '/mall/wish/item/update',
    method: 'post',
    data
  })
}

// @Summary 新建心愿商品
// @Router /mall/wish/item/create [post]
export const createWishItem = (data) => {
  return service({
    url: '/mall/wish/item/create',
    method: 'post',
    data
  })
}

// @Summary 删除心愿商品（只允许删没被兑换过的）
// @Router /mall/wish/item/delete [post]
export const deleteWishItem = (data) => {
  return service({
    url: '/mall/wish/item/delete',
    method: 'post',
    data
  })
}

// @Summary 心愿单列表（只读）
// @Router /mall/wish/order/list [post]
export const getWishOrderList = (data) => {
  return service({
    url: '/mall/wish/order/list',
    method: 'post',
    data
  })
}
