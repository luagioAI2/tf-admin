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
| `server/config.yaml` | **从仓库移除 + 加进 .gitignore**（上游会提交它，那会把 jwt 密钥和数据库绝对路径带进 git）。改成只提交 `server/config.example.yaml`。见 §3.0 |
| `server/config.example.yaml` | **新增**（DB 路径与密钥留空） |
| `web/.env.development` | `VITE_SERVER_PORT` `8888` → **`8890`**（本项目后端端口） |
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

### 3.0 ⚠️ 配置文件不进 git（与上游不同）

上游把 `server/config.yaml` **提交进仓库**。本项目**不**这么做，理由：

- 它会把 **jwt 签名密钥**带进 git（上游默认值是 `qmPlus`，改过之后就是真密钥）
- 它会把**数据库绝对路径**带进 git —— 别人 clone 下来跑不起来

所以：

```
server/config.example.yaml   ← 提交（DB 路径与密钥都留空）
server/config.yaml           ← 已在 .gitignore，各人本地维护
```

首次使用：

```bash
cd server
cp config.example.yaml config.yaml
# 然后走下面的初始化流程（initdb 会把 DB 配置写回 config.yaml）
```

### 3.1 初始化数据库（顺序有讲究）

`/init/initdb` 有一道守卫：`global.GVA_DB != nil` 时直接拒绝
（报「已存在数据库配置」）。而 `config.example.yaml` 里
`sqlite.path` / `db-name` **本来就是空的**，所以流程是：

```bash
cd server
cp config.example.yaml config.yaml      # path/db-name 为空 → 服务以「无库」状态启动

go run .                                # 启动（此时 GVA_DB == nil）

curl -X POST http://127.0.0.1:8890/init/checkdb -d '{}'
curl -X POST http://127.0.0.1:8890/init/initdb \
  -H 'Content-Type: application/json' \
  -d '{"adminPassword":"<你的密码>","dbType":"sqlite",
       "dbPath":"E:/plan/test25/tf-platform-api/data","dbName":"tf"}'
# initdb 成功后会**把真实配置写回 config.yaml**（含自动生成的 jwt 密钥）

go run .                                # 重启，这次连得上库了
```

> 💡 `dbPath` 填的是 **tf-platform-api 的 data 目录**（两边共用一个库，见 §2）。
> 也可以直接填 `../tf-platform-api/data`（相对 server 目录）。

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

> ⚠️ **三个容易踩的坑**：
> 1. `syncApi` 是 **GET**；用 POST 会 404（而且不报「方法不对」，就是 404）
> 2. `syncApi` **只是预览差异，不会落库**；真正落库要再调 `enterSyncApi`
> 3. 落库之后 casbin **还在用旧策略**（进程启动时加载的）⇒ 必须
>    `GET /api/freshCasbin` 或重启后台。不然「库里有行、请求还是 403」，
>    看着像写错了表。
>
> 不注册的症状：接口返回 `{"code":7,"msg":"权限不足"}`。

> ⚠️ **mall 模块里读接口不全是 POST**：`mall/dashboard/overview`、
> `mall/user/detail`、`mall/user/ledger`、`mall/merchant/pending-count` 是 **GET**，
> 其余是 POST。方法写错时 Gin 走 NoRoute，回的是**纯文本**
> `404 page not found` —— 拿 `res.json()` 解析它的脚本会红在
> `Unexpected non-whitespace character after JSON at position 4`，
> 报错位置离真正原因四层远。`scripts/_admin-lib.mjs` 的 `adminApi`
> 现在先看 `content-type`，不是 JSON 就把「方法 / 路径 / status / 前 120 字节」
> 一起报出来；读接口用 `adminGet(path)`。

#### 离线批量补齐（拿不到管理员密码 / 想放进部署流程）

```bash
python scripts/seed-mall.py            # 22 个 mall 接口 → sys_apis + casbin_rule（幂等）
python scripts/seed-mall.py --dry-run  # 只看要做什么
python scripts/check-mall-perms.py     # 逐个体检：哪个接口还是「权限不足」
```

**2026-11-03 实测：14 个接口只有 3 个通**（`user/list`、`merchant/list`、
`merchant/pending-count`），其余全是「权限不足」—— 而页面看着完全正常，
点下去才发现。**新加接口之后一定要跑一遍 `check-mall-perms.py`。**

> ⚠️ `code=7` **不等于**「权限不足」：参数校验失败也返回 `code=7`
> （`{"code":7,"msg":"Key: 'UpdateGameReq.GameID' Error:Field validation…"}`）。
> 判据要看 **`msg`**，只看 code 会把「参数没填」当成「没授权」。

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
| `POST /mall/user/list` | C 端用户列表（`is_guest=1/0` 只看游客 / 只看正式账号） |
| `GET /mall/user/detail?id=N` | 用户详情 + 钱包 |
| `GET /mall/user/ledger?id=N` | 账本流水 |
| `POST /mall/user/adjust` | **调整余额（走账本）** |
| `POST /mall/user/reset-password` | **重置登录密码（生成一次性密码 + 撤旧令牌）** |

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

**「余额 = 账本 delta 之和」怎么验**（`scripts/_probe-user.mjs` ⑦）：

界面按 `id DESC` 排流水，所以**每种币第一次出现的那一行就是最新那行**，
它的 `balance_after` 必须等于当前钱包余额。对不上就说明余额和账本
已经不是一回事了 —— 而界面上**不会报任何错**，只有人肉看才发现。

> ★ 详情弹窗里的账本是**异步补上**的：弹窗一出现「账本流水」标题就在了，
> 但那一刻表格是空的、分页写着「共 0 条」。探针要等**行出现**，
> 等标题会红成「账本没有流水」，而接口明明有 315 条。

**为什么要能按「账号类型」筛**：游客是 App 一启动就自动建的临时账号，`id` 最大
⇒ **永远排在最前面**。不筛的话运营点进来第一页全是一屏「小客人」（0 分 0 流水），
真正要看的用户被挤到后面几页 —— 而「现在有多少人还在用临时账号」本身也是个
有意义的运营数字。字段是 `*int`：`nil` = 不筛，`0` = 只看正式账号（**0 是有效值**，
不能用 0 当「没传」，否则筛了等于没筛，而界面上照样显示全部）。

**家长锁（`parentPin`）在界面上只显示状态，不显示内容**：

- 列表列只有「已设置 / 未设 / —」三种文案；`pin_known=false`（老库还没有
  `user_settings` 表）画「—」，不能画「未设」—— 把「不知道」画成「没有」是在撒谎。
- ⚠️ **详情接口不返回 `has_pin`**（那是列表里批量算的）。`openDetail` 整份
  覆盖 `detail` 会把它变成 `undefined` ⇒ 明明设了密码，详情里说没设。
  页面专门保留了列表那一份，探针里有一条断言守它。
- 探针**不去设 / 清 PIN** —— 那是凭据，不该被测试动。

操作人从 JWT 里取（`utils.GetUserName(c)`），写进账本 memo ——
后台直接改的是生产数据，没有操作记录等于出了事查不到人。

**`user/reset-password` 为什么密码由服务端生成、不让后台手填**：

生产环境还没有短信通道（平台 `DevSmsSender` 在 `IsProduction()` 下直接报错），
所以 C 端「忘记密码」走不通 —— 用户忘了密码就是永久进不去，而积分、券、存档
全在那个号里。这是唯一的兜底，但它有三个容易做错的地方：

| 做错 | 症状 |
|---|---|
| 写明文密码 | 「重置成功」但谁都登不上 —— 平台端是 `bcrypt.CompareHashAndPassword` 比的，**零报错** |
| 不撤 `user_token` | 旧设备手里的刷新令牌还能续 7 天。而「要重置密码」的场景里，有一半是「号被别人拿走了」 |
| 不写 `updated_at` | `BaseModel` 是 `autoUpdateTime:false`，落 0 ⇒ 列表上显示「1970-01-01」 |

- 密码由 `newTempPassword()` 生成（8 位，字母表去掉 `0/O`、`1/l/I`）：这串要念给
  家长 / 抄在纸上，长得像的字符只会让人登不进去，然后以为「后台给的是错的」。
- **明文只在响应里出现一次**，后台不落库、不打日志（日志里只记 uid / 操作人 / 备忘）。
- 游客（`is_guest`）和无手机号的账号直接拒绝，并给出人话原因：设了密码也没法登录
  （登录要手机号），该做的是让用户先在 App 里点「保存进度」绑定。
- ⚠️ **旧 access token 最多还能用 2 小时**：平台端 JWT 无状态，没给单条 access
  做黑名单。跟客服说清楚，别承诺「立刻就进不去了」。

> ★ 弹窗里那个「重置登录密码」按钮放在**详情**弹窗的 footer 左侧，不放列表操作列 ——
> 那一列已经正好 175px 够一行，再塞一个就换行，整张表高度全变。

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

> ⚠️ **兑换商品是按用户建的**（`redeem_item.uid`）：每个孩子一张自己的清单，
> 价格和库存由自己家长定。所以后台列表里会同时出现几个人的同名商品 ——
> **不是重复数据**。页面第一列「所属」就是为此存在的，别删。
>
> ⚠️ 后台**没有**「新建兑换商品」，也**没有**「标记已兑现」：
> 前者是家长在 App 里做的，后者是孩子点「换到了」。别加这两个按钮，
> 接口不存在。
>
> ⚠️ 库存口径（`tf-platform-api/internal/app/service/redeem_srv.go`）：
> **`-1` = 无限；`0` = 卖光了**（`soldOut := it.Stock == 0`）。
> 后台里把 0 当「无限」写，孩子那边就换不了了 —— 这个坑我第一版页面
> 的提示文案就写反了。

#### 心愿商城：和兑换商城**口径相反**的三处

两页长得像，但照着 redeem 的直觉改就会静默错：

| | 兑换商城 | 心愿商城 |
|---|---|---|
| 归属 | **每个孩子一份**（`redeem_item.uid`） | **全站一份**（`wish_item` 没有 uid） |
| 价格字段 | `price`，单位 = **券** | `price_cents`，单位 = **人民币分**（6900 = 69 元），且只是**参考价** |
| 兑换价在哪 | 就是 `price` | 在订单上：`voucher_price`（单位 = 券） |

- 把 `price_cents` 当券数显示，会写出「6900 券」—— 不报错、不越界，纯错。
- 心愿商品的列表排序和 C 端一致（**盲盒永远最前**，其次 `sort_order`）。
  后台按 id 排的话，运营在这儿调顺序、孩子那边看到的是另一个顺序。
- 后台只能改 **`buy_url` / `buy_label` / `archived` / `sort_order`**。
  订单状态是孩子和家长推的（`wished → committed → confirmed → done`，
  `cancelled` 从任何非终态都能进）。后台长出「改状态」= 能凭空扣券、退券。
- `buy_url` **孩子看不到**，只在家长确认页出现。
- 订单上的 `vouchers_held` 决定券在谁手里：`committed` / `confirmed` 时
  **押在平台**，其余已经回了孩子钱包。对账只看 `status` 会把押着的券
  算成孩子的余额，所以页面单独列了一列「券在谁手里」。

#### 激活码（2026-11-04，心愿商城页第三个页签）

一张卡 = 一个 6 位码。家长在淘宝买到卡 → 孩子把码敲进 App → 那款游戏进游戏库。
后台只做三件事：**生成 / 查看 / 作废**，核销在 C 端（`/api/wish/redeem-code`，
同一个 `tf.db`）。

| 接口 | 说明 |
|---|---|
| `POST /mall/wish/code/list` | 码列表（按批次 / 是否已用筛） |
| `POST /mall/wish/code/generate` | 批量生成（返回**明文**，只此一次） |
| `POST /mall/wish/code/void` | 作废单张 / 整批未使用的 |
| `POST /mall/wish/code/batches` | 批次号下拉 |

- ★ **表里只存 sha256，明文只在生成那一次返回**。界面必须当场让运营复制走
  （关掉窗口就找不回来，只能作废重发）—— 这一条写在了生成弹窗的提示里。
- ★ **作废整批之前必须先按批次筛**：不筛就等于「作废全部未使用的码」。
  已经用掉的不动（那个孩子的游戏已经到账）。
- 码只能挂在 **`kind='game'` 且绑了 `game_id`** 的商品上（一款卡对应一款游戏）。
- 生成时的码字符集去掉了易混的 `I/L/O`；服务端 `normalizeWishCode`
  把 `I`/`L` 纠成 `1`、`O` 纠成 `0`，分隔符（空格 / `-` / `_` / `·` / `・`）
  一律去掉 —— 归一化规则必须和客户端 `domain/gameCode.ts` **逐条一致**，
  否则症状是「码明明是对的，界面说无效」。
- 权限：四条接口已进 `source/system/api.go` 与 `casbin.go`（角色 888）。
  新库直接生效；**老库要跑一次 3.2 的同步**，否则页面上点哪都是 403。

> 心愿商品还多了一个「一次开出几款」（`unlock_count`，1~20，只对盲盒有意义）。
> 非盲盒保存时一律写回 1 —— 否则「盲盒改成游戏盒子」之后这个字段还留着 5。
> 口径见 `kid-quest-farm-online/docs/wish-mall-spec.md` §13。

### 4.5 招商审核（2026-10-29 新增，**第一个有页面的模块**）

C 端（`kid-quest-farm-2d` 心愿页右上角「🤝 招商」）只负责**提交**，
审核和回填后台地址只能在这里做。用户原话：

> 「申请后端会返回后台链接，已经申请后台审核过的 就是直接显示链接。」

| 接口 | 说明 |
|---|---|
| `POST /mall/merchant/list` | 申请列表（**待审核的排最前**） |
| `POST /mall/merchant/review` | 审核：通过 / 拒绝 |
| `GET /mall/merchant/pending-count` | 待审核数 |

页面：`web/src/view/mall/merchant/index.vue`，
菜单「商城运营 → 招商审核」（`source/system/menu.go` 已种子）。

**四条不能改的约束**：

1. **通过时 `admin_url` 必填**。C 端那一屏写的是「审核通过后，这里会出现
   一个后台链接」—— 通过了却不给地址，用户看到「已通过」却永远拿不到入口，
   只能来问客服。服务层**直接拒绝**这种调用，而不是先落库再指望有人补。
2. **只允许从 `pending` 改（CAS）**：`WHERE id = ? AND status = 'pending'`，
   `RowsAffected == 0` 就报「已经审核过了」。先查再写的话，两个人同时点
   「通过」，后一个会覆盖前一个的 `reviewed_at`，审计上就看不出谁什么时候审的。
3. **拒绝时把 `admin_url` 清空**。留着它，C 端一旦漏判状态就会把一个
   「已拒绝」的单子渲染出「打开后台」。
4. **`updated_at` 必须手动写**。平台模型的 `autoUpdateTime:false`，
   GORM 不会自动填 —— 漏了会写进 0（1970 年）。

> ⚠️ **`review` 挂了 `OperationRecord`**：它改的是「谁能拿到后台入口」，
> 出了事要查得到是谁在什么时候点的通过。

> ⚠️ **新接口要注册进 casbin 才会生效**。这次把 3 条同时加进了
> `source/system/api.go` 与 `source/system/casbin.go` 的**种子**（只对首次
> `initdb` 生效）。已经初始化过的库仍要走 §3.2 的 `syncApi` + `setApiRoles`，
> 否则点「通过」返回 `{"code":7,"msg":"权限不足"}` —— 而页面看着完全正常。

### 4.6 上游 v3 变更（2026-10-29）对后台的影响

`kid-quest-farm-2d` 当天把心愿商城的付款规则从「只收券」改成
**券 + 积分混付**（与「兑换」一字不差），订单上多了 `points_held`。

后台的改动：

| 项 | 说明 |
|---|---|
| `model/mall/models.go` 的 `WishOrder` | 新增 `PointsHeld` 字段（`points_held` 列） |

**为什么必须同步**：这是**退款凭据的另一半**。后台如果读不到它，
运营看到的「这一单扣了多少」就不完整，而对账时
「券少了 30、积分少了 20」在后台只显示得出前一半。

> ⚠️ 后台的模型**不参与 AutoMigrate**（见 §2.1 第 ② 条），
> 所以加这个字段**不需要**改表结构 —— 它只是让后台能读到平台已经建好的列。
> 反过来说：新字段的建列由**平台自己的** `Models()` 负责。

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

node scripts/_probe-dashboard.mjs        ✓ 13/13（数字和接口逐项对上，两个待处理数对账）
node scripts/_probe-redeem.mjs           ✓ 18/18（改价真落库、测完改回原价）
node scripts/_probe-wish.mjs             ✓ 21/21（参考价按元不按券、改外链真落库、筛选真生效）
node scripts/_probe-user.mjs             ✓ 29/29（账实相符、家长锁不泄露、调余额两道守卫、账号类型筛选）
node scripts/_probe-reset-password.mjs   ✓ 18/18（旧密码登不上、新密码能登上、旧令牌作废）
node scripts/_run-all-probes.mjs         ✓ 六条全绿（顺序跑，中间留 6s 避开限流）
python scripts/check-mall-perms.py       ✓ 14/14（2026-11-03 补齐前只有 3/14）
```

> ★ **别连着跑探针**：后台登录和几个列表接口有频率限制，12+ 次请求连着发
> 会返回 `too many requests`，探针会红在「有页面级报错 — AxiosError$1」
> 这种一点线索都没有的地方，而单独跑一遍又全绿。用
> `node scripts/_run-all-probes.mjs`，它自带间隔。

---

## 6. 前端页面

上游 gin-vue-admin 的界面是「菜单 → 动态路由 → `web/src/view/<path>`」，
所以加一页要两步：**加菜单**（`sys_base_menus`，`component` 指向
`view/mall/xxx/index.vue`）+ **写页面**。

### 6.1 已有页面

| 菜单 | 页面 | 接口封装 | 探针 |
|---|---|---|---|
| 商城运营 → 经营看板 | `web/src/view/mall/dashboard/index.vue` | `web/src/api/mallDashboard.js` | `scripts/_probe-dashboard.mjs`（13） |
| 商城运营 → App 用户 | `web/src/view/mall/user/index.vue` | `web/src/api/tfUser.js` | `scripts/_probe-user.mjs`（29）+ `scripts/_probe-reset-password.mjs`（18） |
| 商城运营 → 招商审核 | `web/src/view/mall/merchant/index.vue` | `web/src/api/mallMerchant.js` | — |
| 商城运营 → 兑换商城 | `web/src/view/mall/redeem/index.vue` | `web/src/api/mallRedeem.js` | `scripts/_probe-redeem.mjs`（18） |
| 商城运营 → 心愿商城 | `web/src/view/mall/wish/index.vue` | `web/src/api/mallWish.js` | `scripts/_probe-wish.mjs`（20） |

这些菜单写进了 `source/system/menu.go` 的**种子**，所以**首次 `initdb`
之后就有了**，不用手动加。已经初始化过的库要自己加一次（用
`python scripts/seed-menu.py --only <menuName>`）。

### 6.2 还没有页面的模块

下面这些后端接口**已经完整可用**，但页面还没写：

| 菜单名 | path | component |
|---|---|---|
| 游戏管理 | `mallGame` | `view/mall/game/index.vue` |
| 游戏版本 | `mallRelease` | `view/mall/release/index.vue` |
| 任务记录 | `mallTask` | `view/mall/task/index.vue` |

> 游戏管理 / 游戏版本这两页**先别急着做**：游戏本体是打进 APK 的
> （`kid-quest-farm-online/src/games/localGames.ts`），`game_release` 表现在是空的。
> 做出来就是两张空表，而「空」会被读成「游戏被下掉了」。

> ⚠️ **别照这张表把菜单一次性补齐**。菜单加了但 `component` 指向的
> `.vue` 不存在时，点进去是**空白页**（前端只报一个 404 chunk），
> 而人的第一反应是「权限配错了」。**先写页面，再加菜单。**

> 不改前端也能用：这些接口可以用 Postman / curl 直接调，
> 或者用后端自带的 `/swagger/index.html`（swagger 注释已经写全）。

#### 加一页的完整顺序（四步，别跳）

1. 写 `web/src/view/mall/<x>/index.vue` + `web/src/api/mallXxx.js`
2. `python scripts/seed-mall.py` —— 接口登记 + casbin（否则第一屏就 403）
3. `python scripts/seed-menu.py --only <menuName>` —— 菜单 + 菜单授权
4. 照 `scripts/_probe-wish.mjs` 写一条探针，`node scripts/_run-all-probes.mjs` 跑全绿

`seed-menu.py` 里那张 `PAGES` 表**会先看 `.vue` 在不在**，不在就跳过 ——
所以可以提前把还没写的页面列进去，它不会建出「点进去空白」的菜单。

> ★★ **写探针时最容易红错地方的六件事**（1、2、6 三条 `_admin-lib.mjs` 已处理）：
> 1. **点按钮点到了隐藏页签里的同名按钮**。`el-tab-pane` 是 v-show，
>    隐藏页签的按钮照样在 DOM 里、`innerText` 照样有字。`clickByText`
>    现在先按「占面积」过滤（不能用 `offsetParent`：`position:fixed`
>    的元素它的 `offsetParent` 恒为 null）。
> 2. **选下拉项时点到了页面正文里的同名文字**。筛「已取消」时表格里
>    本来就有一堆「已取消」标签，而它们在 DOM 里排在 body 末尾的 popper
>    **前面** ⇒ 点到表格里，什么都不会发生，断言红成「筛选没生效」。
>    用 `pickOption()`，它只在 `.el-select-dropdown` 里找。
> 3. **等了一个「上一页已经满足」的条件**。切到 App 用户时等
>    「表格里有行」—— 仪表盘本来就有一张 5 行的表，条件立刻满足，
>    读到的是上一页的表格，然后 7 条断言一起红在「列对不上」。
>    **等这一页特有的文字**（如「个账号」），别等通用结构。
> 4. **关掉的弹窗节点还在**。这一页两个弹窗**没写** `destroy-on-close`，
>    关掉后 DOM 还在（只是 `display:none`）⇒「等弹窗消失」永远等不到。
>    判据要用**可见性**（`offsetParent !== null`），不是存在性。
>
> 5. **★ 断言跟着数据一起烂掉**。`_probe-user.mjs` 原来只在前 10 行里挑
>    「账本最厚的」当目标，可游客账号会不断把第一页顶满（0 分 0 流水，
>    id 最大排最前）⇒ 挑中的是「第一个游客」，⑦ 去等一条永远不来的流水，
>    20s 后报「探针中断」，**一个字线索都没有**。现在改成「按钱包从厚到薄
>    试前 3 个，第一个真有流水的当目标」，目标不在第一页就用手机号筛出来
>    （**不能按昵称**：游客全叫「小客人」，筛出来一堆会点到别人身上，
>    而两个人钱包都是 0，点错了照样全绿）。
>
> 6. **★★ 第三方站点把整个套件拖红**。GVA 首页那张「提交记录」卡会去
>    `api.github.com` 拉 commit，而 Chrome 走系统代理（本机 7897）——
>    实测**直连 200、走代理 403**。403 ⇒ axios reject ⇒ 没人 catch ⇒
>    `pageerror` ⇒ **五条探针一起红在「没有页面级报错」上**，报的还只有
>    `AxiosError$1` 一个字的线索（`String(e)` 对 axios 的错就是这样）。
>    表现最阴的地方：**代码一行没改、开关一下代理，套件从全绿变全红**，
>    而接口日志里 0 个 4xx / 5xx —— 很容易误判成「我改坏了」。
>    `openAdmin()` 现在用请求拦截把**所有非本机请求**就地回个空壳
>    （JSON → `[]`、图片 → 1×1 透明 PNG），一条都不真发出去。
>    ⚠️ 空壳**必须自己带 `Access-Control-Allow-Origin`**：跨域回包没这个头，
>    Chrome 按 CORS 拦成 `net::ERR_FAILED`，axios 照样 reject，
>    看起来像「stub 没生效」。
>
> ★ 还有一条：这一页的筛选**不是即时的**，选完要再点「查询」才发请求。
> 不点的话请求体一个都不会发 —— 页面上那个下拉已经显示「已取消」了，
> 看着非常像后端坏了。
>
> ★ 详情弹窗里的账本是**异步补上**的：弹窗一出现「账本流水」这个标题就在了，
> 但那一刻表格还是空的、分页写着「共 0 条」。要等**行出现**，不是等标题。

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

> ✅ `web/.env.development` 里的 `VITE_SERVER_PORT` **已改成 8890**
> （上游默认 8888，而本项目后端跑在 8890）。
>
> 不改这个值的话，前端请求会打到「什么都没监听」的 8888，
> 症状是**点登录没反应**、控制台一堆 `ECONNREFUSED` ——
> 而不是任何一条能指向「端口配错了」的提示。

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
---

## 10. 移动端适配（实测）

**结论：布局外壳适配了，内容页没适配。**

hooks/responsive.js 会在 ody 宽度 < 992px 时把 device 切成 mobile，
side/index.vue 随之把所有 side_mode 回落到 
ormalMode，
而 
ormalMode.vue 的 watchEffect 在 mobile 下强制 isCollapse = true。

### 实测数据（Chromium，390×844，账号 admin）

| 项 | 结果 |
|---|---|
| 页面级横向溢出 | **无**（8 个页面全都没溢出） |
| 侧栏 | **80px 折叠图标栏**（不是抽屉），主内容区只剩 **310px** |
| 点顶栏左上角 | 无反应 |
| 点侧栏底部箭头 | 展开到 **256px** → 主内容区只剩 **134px** |
| 表格 | **全部横向滚动** |
| 触摸目标 | 大量 32px 高按钮（业界下限 44px） |

表格实测（内容宽 / 容器宽）：

`
用户管理      1215px / 266px   →  需要滚动 4.6 屏
菜单管理      1660px / 266px   →  6.2 屏
API 管理       795px / 266px   →  3.0 屏
角色管理       920px / 266px   →  3.5 屏
操作历史      1215px / 266px   →  4.6 屏
代码生成器    2570px / 266px   →  9.7 屏
`

> 数字不是「功能坏了」—— 表格**能**横向滚动，功能可用。
> 但「看一行用户要左右拖 4 屏」在实际操作里等于不可用。

**触摸目标偏小**（Element Plus 默认尺寸）：

`
查询 / 重置 / 新增    80×32
表格行内「编辑」       54×20
「分配角色」           82×20
复选框                14×14
`

### 截图与测量脚本

本次验证用的脚本在仓库外（E:\plan\test25\），没有提交：

- shoot.js — 手机/桌面双视口截图 → E:\plan\test25\shots\
- measure.js — 量化测量（溢出 / 表格 / 字号 / 触摸目标）
- probe*.js — 抽屉菜单、表格边界、侧栏宽度的针对性确认

> 一个**测量陷阱**值得记下：一开始我把「el-table 的
> getBoundingClientRect().right > viewport」当成布局断裂，
> 但实际它在一个 overflow-auto 的祖先里（310px 宽，scrollWidth 1293）——
> **是受控滚动，不是断裂**。判断越界必须先找可滚祖先。

### 如果要做移动端

按成本从低到高：

1. **只给运营用的少数页面做适配**（最省）——
   后台大概率不是给手机用的，是给电脑用的
2. **表格改卡片**：< 992px 时把 el-table 换成卡片列表
   （gin-vue-admin 没有这个能力，要自己写）
3. **换一套移动端 UI**：引入 Vant / NutUI，给移动端单独做一套页面
4. **做成响应式 PWA**

> 本项目还没做任何一项 —— **自建的业务模块（mall）连前端页面都还没有**
> （/#/layout/mall/* 全是 404），所以「适配手机」这件事
> 目前只关乎上游那套管理界面。
