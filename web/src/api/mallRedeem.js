import service from '@/utils/request'

/* ============================================================
   兑换商城（后台）
   ------------------------------------------------------------
   ⚠️ 兑换商品是**按用户建的**（`redeem_item.uid`）：每个孩子一张自己的
      兑换清单，价格和库存都由自己家长定。所以列表默认是**全部用户**混在一起，
      第一列必须显示「是谁的」—— 不显示的话，运营看到三条同名商品
      会以为是重复数据，其实是三个孩子各有一条。

   ⚠️ 后台只有「改价 / 改库存 / 上下架」（`item/update`），**没有**「新建商品」，
      也**没有**「标记已兑现」。前者是家长在 App 里做的事，后者是孩子在 App 里
      点「换到了」。后台不要长出这两个按钮 —— 接口不存在，点了只会 404。

   ⚠️ 金额单位是「分」（1 兑换券 = 1 分），和 App 里显示的数字一致，
      别再乘 100。库存 `-1` = 无限。
   ============================================================ */

// @Summary 兑换商品列表
// @Router /mall/redeem/item/list [post]
export const getRedeemItemList = (data) => {
  return service({
    url: '/mall/redeem/item/list',
    method: 'post',
    data
  })
}

// @Summary 调整兑换商品（改价 / 库存 / 上下架 / 排序）
// @Router /mall/redeem/item/update [post]
export const updateRedeemItem = (data) => {
  return service({
    url: '/mall/redeem/item/update',
    method: 'post',
    data
  })
}

// @Summary 兑换记录（只读）
// @Router /mall/redeem/record/list [post]
export const getRedeemRecordList = (data) => {
  return service({
    url: '/mall/redeem/record/list',
    method: 'post',
    data
  })
}
