import service from '@/utils/request'

/* ============================================================
   游戏（后台）
   ------------------------------------------------------------
   · `/mall/game/options`：心愿商城配「单款游戏 / 盲盒池」时的下拉。
   · `/mall/game/list`：游戏列表页（本轮新增，跟「商城运营」同级）。
   · `/mall/game/update`：只放行价格 / 状态 / 默认拥有 / 排序，
     名字、图片、入口 URL 由客户端包带，后台不改（改了也对不上包）。
   ============================================================ */

// @Summary 游戏下拉选项（game_id / name / emoji / status）
// @Router /mall/game/options [post]
export const getGameOptions = () => {
  return service({
    url: '/mall/game/options',
    method: 'post'
  })
}

// @Summary 游戏列表（分页 + game_id / 名称 / 分类 / 状态）
// @Router /mall/game/list [post]
export const getGameList = (data) => {
  return service({
    url: '/mall/game/list',
    method: 'post',
    data
  })
}

// @Summary 调整游戏（只传要改的那几个字段）
// @Router /mall/game/update [post]
export const updateGame = (data) => {
  return service({
    url: '/mall/game/update',
    method: 'post',
    data
  })
}
