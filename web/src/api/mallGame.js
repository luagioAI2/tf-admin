import service from '@/utils/request'

/* ============================================================
   游戏（后台只读的下拉选项）
   ------------------------------------------------------------
   现在只用到 `/mall/game/options`：心愿商城配「单款游戏」和「盲盒池」
   时要按 game_id 选。以前这两个字段只能去改客户端那份目录，
   所以后台从来没拉过游戏列表。
   ============================================================ */

// @Summary 游戏下拉选项（game_id / name / emoji / status）
// @Router /mall/game/options [post]
export const getGameOptions = () => {
  return service({
    url: '/mall/game/options',
    method: 'post'
  })
}
