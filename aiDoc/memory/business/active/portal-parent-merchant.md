# 家长 / 商家门户（复用 GVA 权限系统）

## 基本信息

- 提出日期：2026-11-04
- 当前状态：`active`
- 需求类型：权限体系 / 后台模块 / 数据归属
- 优先级：高
- 需求文件：`aiDoc/memory/business/active/portal-parent-merchant.md`

## 用户原始意图摘要

> 还有家长后台的权限，以及商家的权限。admin 本身就有权限系统。你可以创建号，对应的处理。
> 而家长和商家的后台登录账号，就用 App 的登录账号。

即：**不另起一套账号体系**，让家长 / 商家用 App 的手机号 + 密码登录**同一个** gin-vue-admin，
各自只看到自己的菜单和数据。

## 影响范围

- 后端：
  - `service/mall/portal.go`（新）—— 登录桥 + 两个门户的全部读写
  - `api/v1/mall/portal.go`（新）—— 处理器 + `appUID()` 身份闸门
  - `model/mall/request/portal.go`（新）—— 请求结构（商家侧字段刻意收窄）
  - `router/mall/mall.go` —— `PortalRouter`
  - `router/system/sys_base.go` —— 公开路由 `POST /base/appLogin`
  - `model/mall/models.go` + `tf-platform-api/internal/app/entities/wish.go` —— `wish_item.merchant_uid`
- 前端：
  - `view/login/index.vue` —— 管理员 / 家长 / 商家三个 tab
  - `pinia/modules/user.js` —— 抽出 `afterLogin()`，新增 `PortalLoginIn()`
  - `api/portal.js`（新）、`view/portal/parent/*`、`view/portal/merchant/*`（新）
  - `view/mall/wish/index.vue` —— 商品表加「归属」列
- 脚本：
  - `scripts/seed-portal.py`（新）—— 角色 1001/1002 + 菜单 + `sys_apis` + `casbin_rule`
  - `scripts/check-portal-perms.py`（新）—— 43 条判据
- 文档：本文件 + `demand-index.md`

## 涉及对象

- 模块：`mall`（admin）、`wish`（platform）
- 接口：
  - `POST /base/appLogin`
  - `GET  /mall/portal/parent/overview`
  - `POST /mall/portal/parent/wishes`
  - `POST /mall/portal/parent/records`
  - `GET  /mall/portal/merchant/status`
  - `POST /mall/portal/merchant/items`
  - `POST /mall/portal/merchant/item/save`
- 页面：`view/portal/parent/{overview,wishes,records}.vue`、`view/portal/merchant/{items,status}.vue`
- 配置：`sys_authorities` 1001 / 1002；`sys_base_menus` id 57–63

## 已确认约束

- **一个后台、一套权限**：不新建 token 体系，家长 / 商家走 `sys_user` 的影子行
  （`username = app_{uid}`，角色 1001 / 1002），菜单与 casbin 全部复用。
- **准入闸门 = `merchant_application` 已 approved**：没申请 / 待审 / 被驳回都在
  **登录这一步**被拒，不会进到一个空后台里。
- **casbin 不管数据范围**：每个门户接口在 service 层再按 `uid` 过滤一次。
- **`merchant_uid = 0` 是平台自营**，不是「没填」，不能被当成「所有人可见」的兜底。
- 商家**不能**改 `kind` / `game_id` / `pool` / `unlock_count` / `merchant_uid`
  ⇒ `MerchantSaveItemReq` 故意只有 10 个字段。
- 家长端**只读**：确认 / 回绝 / 取消留在 App（那里有券托管与退款规则）。
- 本轮只做「商品归属」，不做商家结算 / 分账。

## 当前进展

- [x] `wish_item.merchant_uid` 双端加字段（平台侧 AutoMigrate 已落库）
- [x] 登录桥 `AppLogin`（bcrypt + 审核 + 影子账号 + `LoginToken`）
- [x] 家长端 3 个只读接口；商家端 3 个接口（含 uid 作用域的写）
- [x] 种子脚本（幂等，含 `--dry-run`），已执行
- [x] 探针 `check-portal-perms.py`：**43/43 通过**（闸门 / 角色 / 身份 / 数据范围 / 外壳）
- [x] 登录页三 tab + 家长 / 商家五个页面 + 商品归属列
- [x] `go build ./...` = 0；`vite build` = 0

## 后续待办

- App 端「家长绑定 / 商家入驻」入口的文案要与后台的拒绝理由对齐
  （目前后台只说「先在 App 里提交申请」）。
- 运营后台目前只能**看**归属，还不能把一件平台自营商品**划给**某个商家；
  需要时在 `mall/wish` 的编辑弹窗加一个商家选择器。
- 商家删除商品仍走运营的 `item/delete`；商家侧只有上下架。
- 商家结算 / 分账、家长端「确认收到」尚未设计。
- 影子账号会随登录累积（一个 App uid 一行），后台「用户管理」里会看到 `app_*`；
  如需隐藏，应在用户列表里按前缀过滤而不是删行（删行会让已登录的人失去角色）。

## 更新规则

- 同一需求始终维护在同一个文件中
- 新信息优先补充到对应段落，不要另起一份重复记录
- 只有需求状态变化时，才在 `active/` 与 `done/` 之间移动文件
