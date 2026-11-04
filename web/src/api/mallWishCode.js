import service from '@/utils/request'

/* ============================================================
   心愿商城 · 激活码（后台）
   ------------------------------------------------------------
   一张卡 = 一个 6 位码。家长线下买到卡，孩子把码敲进 App，
   那款游戏就进他的游戏库（核销在 C 端 `/api/wish/redeem-code`）。

   ⚠️ **明文只出现在生成接口的返回值里，只此一次。**
      表里存的是 sha256（`code_hash`），没有明文列。
      所以「生成」弹窗必须当场把码复制走 —— 关掉就再也拿不回来，
      只能作废重发。

   ⚠️ 码只能挂在**有 `game_id` 的商品**上（分类 = 「游戏」）。
      挂在实物或盲盒上，核销会成功却什么都不解锁 —— 那张卡就白卖了。
      后端 `GenerateCodes` 会拦住，这里也先说一遍。
   ============================================================ */

// @Summary 激活码列表
// @Router /mall/wish/code/list [post]
export const getWishCodeList = (data) => {
  return service({
    url: '/mall/wish/code/list',
    method: 'post',
    data
  })
}

// @Summary 批量生成激活码（返回明文，只此一次）
// @Router /mall/wish/code/generate [post]
export const generateWishCodes = (data) => {
  return service({
    url: '/mall/wish/code/generate',
    method: 'post',
    data
  })
}

// @Summary 作废激活码（单张 id / 整批 batch，都只作废没用的）
// @Router /mall/wish/code/void [post]
export const voidWishCodes = (data) => {
  return service({
    url: '/mall/wish/code/void',
    method: 'post',
    data
  })
}

// @Summary 已有批次号（筛选下拉）
// @Router /mall/wish/code/batches [post]
export const getWishCodeBatches = () => {
  return service({
    url: '/mall/wish/code/batches',
    method: 'post',
    data: {}
  })
}
