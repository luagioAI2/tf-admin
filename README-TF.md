# tf-admin —— 后台管理系统

> 本项目**基于 [gin-vue-admin](https://github.com/flipped-aurora/gin-vue-admin) 二次开发**，
> 用于管理 `tf-platform-api`（儿童任务/游戏平台）的业务数据。
>
> 上游版本：**v2.9.2-stable**（commit `cee04b08d1262036ca33e3026e1e13d29471c7d5`）
> 许可证：**Apache License 2.0**（`LICENSE` 文件保留原样）

---

## ⚠️ 与上游的差异（Apache-2.0 §4(b) 要求声明）

本项目对上游做了以下**实质性修改**，按 Apache-2.0 要求在此声明：

| 项 | 修改内容 |
|---|---|
| `go.mod` | `go 1.26.10` → **`go 1.24.11`**（见下文「为什么降 Go 版本」） |
| `server/config.yaml` | jwt 签名密钥、监听端口、数据库类型、验证码策略、sqlite 连接 |
| `server/service/enter.go` | 注册 `MallServiceGroup` |
| `server/api/v1/enter.go` | 注册 `MallApiGroup` |
| `server/router/enter.go` | 注册 `Mall` 路由组 |
| `server/initialize/router.go` | 在 `Routers()` 里注册 7 个 Mall 路由 |
| **新增** `server/model/mall/` | 平台业务模型（只读/受限写） |
| **新增** `server/service/mall/` | 平台业务服务 |
| **新增** `server/api/v1/mall/` | 平台业务 API |
| **新增** `server/router/mall/` | 平台业务路由 |

**未改动**：上游的 `LICENSE`、版权声明、`aiDoc/`、插件系统、代码生成器。
本项目**不声称**是官方 gin-vue-admin，也不代表上游。

---

## 1. 为什么降 Go 版本

上游 `server/go.mod` 声明 `go 1.26.10`，而本机工具链是 **Go 1.24.11**
（`GOTOOLCHAIN=local`，不允许自动下载新工具链）。

两个选择：

1. 装 Go 1.26.10 —— 改环境，影响面大
2. 把 `go` 指令降到 `1.24.11` —— **实测可行**

选了 2，实证依据：

```
go mod tidy   → 成功（依赖闭包在 1.24 下完整解析）
go build ./... → 成功（无 1.25/1.26 专属语言特性依赖）
```

⚠️ **升级上游时要注意**：上游若用了 1.25+ 的新语法（如泛型类型别名的新用法），
降指令会直接编译失败。那时应该装对应版本的 Go，而不是改代码。

---

## 2. 与 `tf-platform-api` 共用同一个数据库

后台**直接读写**平台的业务表（`game` / `user` / `redeem_item` / `wish_order` …），
不另建一套「后台表 + 同步机制」。

**理由**：一套数据只有一份才是对的。如果后台自己存一份「游戏列表」，
运营在后台改了价格、而 C 端读到的是另一份 —— 这种错**不报错**，
只表现为「我明明改了怎么没生效」。

配置（`server/config.yaml`）：

```yaml
system:
  db-type: sqlite
sqlite:
  path: "E:/plan/test25/tf-platform-api/data"
  db-name: "tf"
```

即 `E:/plan/test25/tf-platform-api/data/tf.db`。

> ⚠️ **SQLite 多进程并发写会锁库**。开发环境够用；
> 生产环境两边都该指向同一个 **MySQL**，那时没有这个问题（见 §7 生产清单）。

### 2.1 两条必须遵守的规矩

**① 每张平台表都要显式写 `TableName()`**

`tf-platform-api` 用 `SingularTable: true`（表名 `game` 单数），
而 gin-vue-admin 用 GORM 默认复数命名（`games`）。不显式指定就会读到
一张不存在的表，报 `no such table` —— 或者更糟：读到一张恰好同名的空表，
然后「查不到数据」。

**② 平台模型不参与 AutoMigrate**

表是平台的资产，结构由平台自己的 `Models()` 决定。
后台如果也去 migrate，两个进程会互相改表结构；
而且后台的字段定义只要少一个，就可能被 migrate 掉一列。

所以 `initialize/gorm_biz.go` 的 `bizModel()` **保持为空**。

---

## 3. 首次运行

### 3.1 初始化数据库（顺序有讲究）

`/init/initdb` 有一道守卫：`global.GVA_DB != nil` 时直接拒绝
（报「已存在数据库配置」）。所以流程是：

```bash
cd server

# ① 先把 config.yaml 的 sqlite.path / db-name 清空
#    让服务以「无库」状态启动

# ② 启动
go run .

# ③ 调初始化接口（succeeded 后会把真实配置写回 config.yaml）
curl -X POST http://127.0.0.1:8890/init/checkdb -d '{}'
curl -X POST http://127.0.0.1:8890/init/initdb \
  -H 'Content-Type: application/json' \
  -d '{"adminPassword":"<你的密码>","dbType":"sqlite",
       "dbPath":"E:/plan/test25/tf-platform-api/data","dbName":"tf"}'

# ④ 重启（这时配置已写回）
go run .
```

> ⚠️ **初始化前必须让 `tf-platform-api` 先跑过一次**，
> 否则库里只有 `sys_*` 表，后台会报 `no such table: game`。
> 平台的 `AutoMigrate` 负责建它的业务表。
>
> 正确顺序：**平台先跑（建业务表）→ 后台再初始化**。

### 3.2 注册新 API 到权限系统

gin-vue-admin 用 **casbin** 做接口级鉴权。新加的接口**不会自动可用** ——
必须先注册进 `sys_apis`，再授权给角色。

```bash
TOKEN="<登录拿到的 token>"

# ① 取差异（GET，不是 POST）
curl -X GET http://127.0.0.1:8890/api/syncApi -H "x-token: $TOKEN"
#    → 返回 {newApis, deleteApis, ignoreApis}

# ② 把差异原样 POST 回去（这才是「执行同步」）
curl -X POST http://127.0.0.1:8890/api/enterSyncApi \
  -H "x-token: $TOKEN" -H 'Content-Type: application/json' \
  -d '<上一步返回的 data 字段>'

# ③ 逐条授权给角色 888（超级管理员）
#    接口：POST /api/setApiRoles  {"path":"...","method":"...","authorityIds":[888]}
```

> ⚠️ **两个容易踩的坑**：
> 1. `syncApi` 是 **GET**；用 POST 会 404（而且不报「方法不对」，就是 404）
> 2. `syncApi` **只是预览差异，不会落库**；真正落库要再调 `enterSyncApi`
>
> 不注册的症状：接口返回 `{"code":7,"msg":"权限不足"}`。

### 3.3 默认账号

| 用户名 | 密码 |
|---|---|
| `admin` | 初始化时输入的 `adminPassword` |

> 上游还会种一个演示账号 `a303176530`（同密码）。
> **上生产前请删掉它。**

---

## 4. 业务模块（Mall）

新增的 `mall` 模块提供 20 个接口，全部挂在 `PrivateGroup`
（JWT + Casbin）下，写操作额外挂 `OperationRecord`（操作留痕）。

### 4.1 经营看板

| 接口 | 说明 |
|---|---|
| `GET /mall/dashboard/overview?days=7` | 实时统计（用户/游戏/兑换/心愿/任务/积分总量） |

数据全部来自平台表，**不做缓存** —— 后台是低频访问，
实时数据比「快 100ms」有价值。

### 4.2 游戏与版本

| 接口 | 说明 |
|---|---|
| `POST /mall/game/list` | 游戏列表（分页/筛选） |
| `POST /mall/game/options` | 游戏下拉选项（给版本发布用） |
| `POST /mall/game/update` | 改价格 / 上下架 / 排序 |
| `POST /mall/release/list` | 版本列表 |
| `POST /mall/release/publish` | 发布版本 |
| `POST /mall/release/deactivate` | 下架版本 |
| `DELETE /mall/release/delete` | 删除版本 |

**`game/update` 只允许改运营参数**（价格、状态、排序、是否默认拥有），
**不允许改名字/图标/文案** —— 那些是游戏包自己的 manifest 决定的，
后台改了会和包里的声明对不上，而 C 端显示的是包里那份。

**`release/publish` 的校验**（与平台端逐条对齐）：
- `sha256` 必须 64 位小写 hex（不做补零/截断这类宽容匹配）
- `asset_base` 必须以 `/` 结尾
- 版本不能重复
- **事务边界**：「旧的置 non-active」+「插入新的」必须原子 ——
  拆开会出现「两个 active」或「一个 active 都没有」的瞬间

### 4.3 平台用户与钱包

| 接口 | 说明 |
|---|---|
| `POST /mall/user/list` | C 端用户列表 |
| `GET /mall/user/detail?id=N` | 用户详情 + 钱包 |
| `GET /mall/user/ledger?id=N` | 账本流水 |
| `POST /mall/user/adjust` | **调整余额（走账本）** |

> ⚠️ **注意区分两套账号体系**：
> `sys_users` = 后台管理员（运营人员）；
> `user` = C 端小程序用户（孩子/家长）。**完全不同的两套。**

**`user/adjust` 为什么必须走账本**：

平台的不变量是「**余额 = 账本 delta 之和**」。直接 `UPDATE wallet.points`
会破坏它，而且**不报错** —— 症状要等到某次对账才暴露：
「钱多了，但没有对应的流水」。

所以后台模拟平台的 `ApplyChange`：

```
1. 事务内先加锁读钱包（FOR UPDATE）
2. 算新余额，负数直接拒绝（ErrInsufficientBalance）
3. 带版本号乐观锁更新（并发写同一钱包时只会有一个人成功）
4. 写账本（source = manual_adjust，memo 里带操作人）
```

操作人从 JWT 里取（`utils.GetUserName(c)`），写进账本 memo ——
后台直接改的是生产数据，没有操作记录等于出了事查不到人。

### 4.4 兑换商城 / 心愿商城 / 任务

| 接口 | 说明 |
|---|---|
| `POST /mall/redeem/item/list` | 兑换商品列表 |
| `POST /mall/redeem/item/update` | 改价 / 改库存 / 上下架 |
| `POST /mall/redeem/record/list` | 兑换记录 |
| `POST /mall/wish/item/list` | 心愿商品列表 |
| `POST /mall/wish/item/update` | 改外链 / 上下架 |
| `POST /mall/wish/order/list` | 心愿单 |
| `POST /mall/task/list` | 任务定义 |
| `POST /mall/task/instance/list` | 任务实例 |

> ⚠️ `wish/item/update` 能改 `buy_url`，但**不要**把它当成
> 「孩子能看到的推广位」—— C 端接口对孩子的响应里根本不返回这个字段，
> 它只在家长确认页出现。

---

## 5. 实测验证

```
登录 admin                              ✓
经营看板（21 款游戏）                    ✓
游戏列表 total=21                       ✓
心愿商品 total=27                       ✓
平台用户列表 total=1（后台测试娃）        ✓
用户详情 points=600                      ✓
后台加 500 积分 → 钱包 600               ✓
账本：manual_adjust(500) + register_gift(100)  ✓  ← 账实相符
扣成负数 → 被拒（insufficient balance）  ✓
```

---

## 6. 前端页面（**待补**）

后端接口已完整可用，但**管理界面的 Vue 页面还没写**。

上游 gin-vue-admin 的界面是「菜单 → 动态路由 → `web/src/view/<path>`」，
所以还需要两步：

1. **加菜单**：往 `sys_base_menus` 插入记录，
   `component` 指向 `view/mall/xxx/index.vue`
   （可用后台「超级管理员 → 菜单管理」界面手动加，
   或调 `/menu/addBaseMenu`）
2. **写页面**：在 `web/src/view/mall/` 下建页面 + `web/src/api/mall/*.js` 封装

### 已预留的菜单建议

| 菜单名 | path | component |
|---|---|---|
| 经营看板 | `mallDashboard` | `view/mall/dashboard/index.vue` |
| 游戏管理 | `mallGame` | `view/mall/game/index.vue` |
| 游戏版本 | `mallRelease` | `view/mall/release/index.vue` |
| 平台用户 | `mallUser` | `view/mall/user/index.vue` |
| 兑换商城 | `mallRedeem` | `view/mall/redeem/index.vue` |
| 心愿商城 | `mallWish` | `view/mall/wish/index.vue` |
| 任务记录 | `mallTask` | `view/mall/task/index.vue` |

> 不改前端也能用：这些接口可以用 Postman / curl 直接调，
> 或者用后端自带的 `/swagger/index.html`（swagger 注释已经写全）。

---

## 7. 生产清单

| 项 | 开发值 | 生产要求 |
|---|---|---|
| `jwt.signing-key` | `tf-admin-dev-key-change-me` | **必须改**（环境变量注入） |
| 数据库 | SQLite（与平台共用文件） | **MySQL**（两边指向同一个库） |
| `system.addr` | 8890 | 按部署环境 |
| `captcha.open-captcha` | 5 | 保持 3~5（0 = 每次都要求） |
| 演示账号 `a303176530` | 已存在 | **删掉** |
| `system.env` | `local` | `public`（关掉路由日志） |
| CORS | 中间件默认注释掉 | 按需打开 `middleware.CorsByRules()` |

---

## 8. 构建与运行

```bash
cd server

# 开发
go run .                      # 或 go run main.go
# 构建（本机 Go 1.24.11）
GOTOOLCHAIN=local go build -o tf-admin.exe .

# Swagger
swag init                     # → http://127.0.0.1:8890/swagger/index.html
```

前端：

```bash
cd web
pnpm install
pnpm dev                      # → http://localhost:8080，/api 代理到 8890
pnpm build
```

> ⚠️ `web/.env.development` 里 `VITE_SERVER_PORT` 默认是 **8888**
> （上游默认端口）。本项目后端跑在 **8890**，需要改这个值，
> 否则前端请求会打到「什么都没监听」的端口。

---

## 9. 已知问题

### 9.1 初始化时「菜单-权限关联」可能失败

**现象**：`initdb` 报「创建 [菜单-权限] 关联失败, 未找到菜单表初始化数据」，
登录后侧边栏**空的**。

**原因**：插件（announcement / auto）在 DB ready 后会注册自己的菜单，
使得后续 `sys_base_menus` 的 `DataInserted()` 检查认为「已经有数据」而跳过标准菜单播种。

**修复**：调 `/menu/addMenuAuthority` 手动建立角色 888 与全部菜单的关联。
（本项目已修复，登录后能看到 9 个顶级菜单。）

### 9.2 SQLite 多进程并发写

后台和平台同时写同一个 SQLite 文件会锁库。
开发环境低频访问够用；生产用 MySQL。

### 9.3 Go 版本指令

见 §1。降指令是权宜之计，长期应跟上上游的 Go 版本。