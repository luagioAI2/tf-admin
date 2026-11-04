import service from '@/utils/request'

/* ============================================================
   任务 / 兑换模板（后台）
   ------------------------------------------------------------
   ⚠️ 模板是**下发用的蓝图**，不是用户的任务表：
     · `task` / `redeem_item` 是每用户一份的实例（有 `uid`）；
     · `template` / `template_item` 没有 uid，是公共定义。
     改模板对**已经装过**的老用户不生效 —— App 只在首次启动时铺一次。
     界面上必须写清楚，否则运营会以为「改完用户立刻变」。

   ⚠️ 数据范围由 token 推导，**前端传 owner_uid 不算数**：
     超管能看全部；商家只能看 `scope=merchant AND owner_uid=自己`。
     所以这一页没有「归属」筛选（超管那份由后端自己收敛）。

   ⚠️ `item/save` 是**整条覆盖**：每个可编辑列都要传，少传一个 = 清空它。
     和「我的商品」保存是同一套口径。
   ============================================================ */

// @Summary 模板列表（带条目数）
// @Router /mall/template/list [post]
export const getTemplateList = (data) => {
  return service({
    url: '/mall/template/list',
    method: 'post',
    data
  })
}

// @Summary 模板详情（头 + 全部条目）
// @Router /mall/template/detail [post]
export const getTemplateDetail = (data) => {
  return service({
    url: '/mall/template/detail',
    method: 'post',
    data
  })
}

// @Summary 新建模板头（条目随后在 item/save 里加）
// @Router /mall/template/create [post]
export const createTemplate = (data) => {
  return service({
    url: '/mall/template/create',
    method: 'post',
    data
  })
}

// @Summary 改模板头（名字 / 备注 / 排序，全是指针：不传 = 不动）
// @Router /mall/template/update [post]
export const updateTemplate = (data) => {
  return service({
    url: '/mall/template/update',
    method: 'post',
    data
  })
}

// @Summary 设为该分组的默认模板（同组其它默认会被清掉）
// @Router /mall/template/set-default [post]
export const setDefaultTemplate = (data) => {
  return service({
    url: '/mall/template/set-default',
    method: 'post',
    data
  })
}

// @Summary 删除模板（连带条目硬删）
// @Router /mall/template/delete [post]
export const deleteTemplate = (data) => {
  return service({
    url: '/mall/template/delete',
    method: 'post',
    data
  })
}

// @Summary 新增 / 改一条模板条目（整条覆盖）
// @Router /mall/template/item/save [post]
export const saveTemplateItem = (data) => {
  return service({
    url: '/mall/template/item/save',
    method: 'post',
    data
  })
}

// @Summary 删除模板条目（硬删）
// @Router /mall/template/item/delete [post]
export const deleteTemplateItem = (data) => {
  return service({
    url: '/mall/template/item/delete',
    method: 'post',
    data
  })
}

// @Summary 我的推广邀请码（商家印在物料上，注册时带给 App）
// @Router /mall/template/invite-code [post]
export const getInviteCode = () => {
  return service({
    url: '/mall/template/invite-code',
    method: 'post',
    data: {}
  })
}
