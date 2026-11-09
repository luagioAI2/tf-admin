import service from '@/utils/request'

/* ============================================================
   游戏版本发布（后台）
   ------------------------------------------------------------
   · `list`      版本列表（分页 + game_id / channel / 只看在架）
   · `upload`    上传包并发布（multipart，**服务端转发给平台落盘**）
   · `publish`   只发元数据（URL + SHA-256 自己填）—— 给「包已经在 CDN 上」用
   · `deactivate` 下架（active = false，记录保留）
   · `delete`    删掉这条版本记录

   ★ 为什么 `upload` 不是 `publish` 的替代品：`publish` 要求调用方自己填
   SHA-256，而人算不出 64 位十六进制；能填的只有「从别处复制」。所以日常发版
   走 `upload`（哈希由服务端对**落盘后的文件**算），`publish` 留给 CDN 流程。
   ============================================================ */

// @Summary 版本列表
// @Router /mall/release/list [post]
export const getReleaseList = (data) => {
  return service({
    url: '/mall/release/list',
    method: 'post',
    data
  })
}

// @Summary 上传包并发布（FormData）
// @Router /mall/release/upload [post]
export const uploadRelease = (data) => {
  return service({
    url: '/mall/release/upload',
    method: 'post',
    data,
    // axios 在浏览器里遇到 FormData 会把这个头去掉、让浏览器自己带 boundary
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

// @Summary 只发元数据（包已在托管上）
// @Router /mall/release/publish [post]
export const publishRelease = (data) => {
  return service({
    url: '/mall/release/publish',
    method: 'post',
    data
  })
}

// @Summary 下架版本
// @Router /mall/release/deactivate [post]
export const deactivateRelease = (id) => {
  return service({
    url: '/mall/release/deactivate',
    method: 'post',
    data: { id }
  })
}

// @Summary 上架 / 下架（active=false 下架、true 重新上架）
// @Router /mall/release/set-active [post]
export const setReleaseActive = (id, active) => {
  return service({
    url: '/mall/release/set-active',
    method: 'post',
    data: { id, active }
  })
}

// @Summary 删除版本记录
// @Router /mall/release/delete [delete]
export const deleteRelease = (id) => {
  return service({
    url: '/mall/release/delete',
    method: 'delete',
    params: { id }
  })
}
