/**
 * 探针：后台「App 用户」—— 列表数字对得上、详情 + 账本对得上、调余额有守卫
 * ------------------------------------------------------------
 * 这一页是用户唯一亲口问过的（「用户列表在后台怎么看」），但一直没有探针。
 * 它特有的静默错法：
 *   · **账实不符**：列表/详情显示的钱和账本流水加不起来。这一页的全部意义
 *     就是「钱去哪了」，对不上就等于没有账本；
 *   · **详情把 `has_pin` 覆盖成 undefined**：详情接口**不返回**家长锁状态
 *     （那是列表里批量算的），整份覆盖会让「明明设了密码」显示成「未设」。
 *     页面里专门保留了一份，但没人验过；
 *   · **调余额少填备注也能提交**：以后对账时「这笔钱是谁为什么加的」只能靠
 *     备注，服务端也拒，但让运营等一个来回才知道少了什么是没必要的；
 *   · 列表里的 `status` 是 `*int`：下拉清空后传 0 等于「只看被禁用的」，
 *     看起来像「用户全没了」。
 *
 * ★ 断言里刻意**不碰真实凭据**：家长 PIN 只判「界面没把 4 位数字明文打出来」，
 *   不去设/清 PIN —— 那是凭据，探针不该动。
 *
 * 用法：
 *   1) 后台：tf-admin/server 里的 tf-admin.exe（8890）
 *   2) 前端：tf-admin/web 的 pnpm dev（8891）
 *   3) node scripts/_probe-user.mjs [url]
 */
import { adminApi, adminGet, openAdmin } from './_admin-lib.mjs'

const URL_ = process.argv[2] ?? 'http://localhost:8891/'

let failed = 0
const ok = (label, pass, detail = '') => {
  if (!pass) failed++
  console.log(`  ${pass ? '✓' : '✗'} ${label}${detail ? ` — ${detail}` : ''}`)
}

const num = (s) => Number(String(s ?? '').replace(/[^\d-]/g, ''))

/* ---------- 服务端真数 ---------- */
const listApi = await adminApi('/mall/user/list', { page: 1, pageSize: 10 })
if (listApi.code !== 0) {
  console.error('用户列表接口不通：', listApi.msg)
  process.exit(2)
}
/* ★ 这两个会被「定位目标」那一步改写成**筛选后**的口径，
   所以用 let：断言比的必须是「界面此刻显示的这一份」，不是第一次拉的那份。 */
let apiUsers = listApi.data.list ?? []
let expectTotal = listApi.data.total
if (!apiUsers.length) {
  console.error('库里一个用户都没有')
  process.exit(2)
}

/* ---------- 挑目标账号 ----------
 *
 * ★★ 2026-11-03 改。原来只在前 10 行里挑「账本最厚的」，而**游客账号会把
 *   第一页顶满**（0 分 0 流水，id 最大所以排最前）—— 于是「最厚的」变成了
 *   「第一个游客」，⑦ 去等一条永远不来的流水，20s 后报「探针中断」，
 *   一个字线索都没有，只能靠人肉看图。数据会变，判据不能跟着变。
 *
 * 现在：一次 pageSize=100 把库拉下来，按钱包从厚到薄试（最多 3 个），
 *   第一个真有流水的就是目标。为什么按钱包排序而不是逐个查账本：
 *   钱包有钱 ⇒ 几乎一定有流水，这样最多 3 次请求；40 个账号逐个查会撞上
 *   后台频率限制，红成「too many requests」，同样没有线索。
 */
const allUsers = (await adminApi('/mall/user/list', { page: 1, pageSize: 100 })).data.list ?? []
const byWallet = [...allUsers].sort(
  (a, b) => b.points + b.coupon + b.diamond - (a.points + a.coupon + a.diamond)
)
let target = null
let targetLedger = 0
for (const u of byWallet.slice(0, 3)) {
  const total = (await adminGet(`/mall/user/ledger?id=${u.id}&page=1&pageSize=1`)).data?.total ?? 0
  if (total > 0) {
    target = u
    targetLedger = total
    break
  }
}
if (!target) {
  console.error(
    '库里有账号，但「钱包最厚」的前 3 个都没有任何流水 —— ⑦「账实相符」没法验。\n' +
      '要么库被清过，要么余额和账本已经脱节了（后者正是这条探针要抓的东西）。'
  )
  process.exit(2)
}
console.log(
  `接口共 ${listApi.data.total} 个账号；目标：#${target.id} ${target.nickname}` +
    `（${targetLedger} 条流水，${target.points}/${target.coupon}/${target.diamond}）\n`
)

const detailApi = await adminGet(`/mall/user/detail?id=${target.id}`)
const ledgerApi = await adminGet(`/mall/user/ledger?id=${target.id}&page=1&pageSize=10`)
const wallet = detailApi.data?.wallet ?? {}

const a = await openAdmin({ url: URL_ })
const { page, errors, reqFails } = a

const readRows = () =>
  page.evaluate(() => {
    const t = [...document.querySelectorAll('.el-table')].find((x) => x.offsetParent !== null)
    if (!t) return []
    return [...t.querySelectorAll('tbody tr.el-table__row')].map((tr) =>
      [...tr.querySelectorAll('td')].map((td) => (td.innerText || '').trim())
    )
  })

/**
 * 点某一行里的某个按钮。
 *
 * ★ 按**第一列（ID）精确匹配**，不按昵称：游客全叫「小客人」，按昵称会点到
 *   别人身上 —— 而那条断言（详情里的钱包 == 接口）照样能过，因为两个人
 *   钱包都是 0。这种「点错了还全绿」最难发现。
 */
const clickInRow = async (id, btnText) => {
  const hit = await page.evaluate(
    (uid, b) => {
      const tb = [...document.querySelectorAll('.el-table')].find((x) => x.offsetParent !== null)
      if (!tb) return null
      const tr = [...tb.querySelectorAll('tbody tr.el-table__row')].find(
        (r) => (r.querySelector('td')?.innerText || '').trim() === String(uid)
      )
      if (!tr) return null
      const btn = [...tr.querySelectorAll('button')].find((x) => (x.innerText || '').includes(b))
      if (!btn) return null
      const r = btn.getBoundingClientRect()
      if (!r.width) return null
      return { cx: Math.round(r.x + r.width / 2), cy: Math.round(r.y + r.height / 2) }
    },
    id,
    btnText
  )
  if (!hit) throw new Error(`找不到 #${id} 那一行的「${btnText}」`)
  await page.mouse.click(hit.cx, hit.cy)
}

/** 弹窗里的文字（只取可见弹窗） */
const dialogText = () =>
  page.evaluate(() => {
    const d = [...document.querySelectorAll('.el-dialog')].find((x) => x.offsetParent !== null)
    return d ? (d.innerText || '').replace(/\s+/g, ' ').trim() : ''
  })

/**
 * 没有**可见**弹窗。
 *
 * ★ 不能用 `!document.querySelector('.el-dialog')`：这一页两个弹窗都**没**写
 *   `destroy-on-close`，关掉之后 DOM 节点还在（只是 `display:none`），
 *   于是「等弹窗消失」永远等不到，20s 后超时 —— 看着像「关不掉」。
 */
const noVisibleDialog = () =>
  ![...document.querySelectorAll('.el-dialog')].some((d) => d.offsetParent !== null)

/**
 * 等一条 ElMessage 提示出现（等不到返回 false，别抛）。
 *
 * ★ 别用「睡 800ms 再看 innerText」：弹窗有过渡动画，太早点「确认调整」
 *   会点空 —— 而断言会红成「守卫没生效」，看着像前端少写了一道校验。
 */
const toastAppears = async (text, timeout = 4000) => {
  try {
    await page.waitForFunction((t) => document.body.innerText.includes(t), { timeout }, text)
    return true
  } catch {
    return false
  }
}

try {
  /* ============ ① 登录 ============ */
  console.log('① 登录后台')
  const { captchaRequired } = await a.login()
  ok('没有要求验证码', !captchaRequired)

  /* ============ ② 菜单 ============ */
  console.log('\n② 菜单')
  await a.openMallGroup()
  const menuText = await page.evaluate(() => document.body.innerText)
  ok('★ 展开「商城运营」后能看到「App 用户」', menuText.includes('App 用户'))

  /* ============ ③ 打开页面 ============ */
  console.log('\n③ 打开 App 用户')
  await a.clickByText('App 用户')
  await page.waitForFunction(() => location.hash.includes('mall/user'), { timeout: 20000 })
  /* ★ 必须等**这一页特有**的东西，不能等「表格里有行」：
     仪表盘本来就有一张 5 行的表，条件会立刻满足 ⇒ 读到的是上一页的表格，
     然后所有断言都红在「列对不上」，看着像页面写错了。
     实测过：这一等错，后面 7 条一起红。 */
  await page.waitForFunction(() => document.body.innerText.includes('个账号'), { timeout: 20000 })
  await page.waitForFunction(() => document.querySelectorAll('tbody tr.el-table__row').length > 0, {
    timeout: 20000
  })
  ok('★ 页面渲染出来了（不是空白页）', true)

  /* ============ ④ 列表数字 == 接口 ============ */
  console.log('\n④ 列表')
  let rows = await readRows()

  /* ---------- ③.5 把目标账号弄到眼前 ----------
   *
   * 目标不一定在第一页（游客账号会把第一页顶满，见文件头「挑目标」的注释）。
   * 这时用「手机号」筛选把它单独查出来 —— **不能按昵称筛**：游客全叫
   * 「小客人」，筛出来一堆，点「详情」会点到别人身上。
   *
   * 筛完必须把接口口径一起换成筛选后的那份，否则 ④ 会拿「1 行 vs 10 行」
   * 去比，红成「列表行数对不上」，看着像分页坏了。
   */
  if (!rows.some((r) => num(r[0]) === target.id)) {
    if (!target.mobile) {
      throw new Error(
        `目标 #${target.id}（${target.nickname}）不在第一页，又没有手机号，筛不出来 —— ` +
          `库里「有钱且有流水」的账号都在深页了，需要有人看一眼数据`
      )
    }
    const box = await page.evaluateHandle(() => {
      const item = [...document.querySelectorAll('.el-form-item')].find((f) =>
        (f.innerText || '').trim().startsWith('手机号')
      )
      return item?.querySelector('input') ?? null
    })
    const el = box.asElement()
    if (!el) throw new Error('找不到「手机号」筛选框')
    await el.click()
    await page.keyboard.type(target.mobile)
    await a.clickByText('查询')
    await page.waitForFunction(
      (m) => {
        const trs = [...document.querySelectorAll('tbody tr.el-table__row')]
        return trs.length > 0 && trs.some((tr) => (tr.innerText || '').includes(m))
      },
      { timeout: 20000 },
      target.mobile
    )
    console.log(`  （目标不在第一页，已用手机号 ${target.mobile} 筛出来）`)
    const filtered = await adminApi('/mall/user/list', {
      page: 1,
      pageSize: 10,
      mobile: target.mobile
    })
    apiUsers = filtered.data.list ?? []
    expectTotal = filtered.data.total
    rows = await readRows()
  }

  ok('列表行数 == 接口这一页的条数', rows.length === apiUsers.length, `${rows.length} / ${apiUsers.length}`)
  ok(
    '★ 「共 N 个账号」= 接口 total',
    num((await page.evaluate(() => document.body.innerText)).match(/共\s*(\d+)\s*个账号/)?.[1]) ===
      expectTotal,
    `界面 ${(await page.evaluate(() => document.body.innerText)).match(/共\s*\d+\s*个账号/)?.[0]} / 接口 ${expectTotal}`
  )

  /* 列顺序：ID / 头像 / 昵称 / 手机号 / 状态 / 积分 / 兑换券 / 钻石 / 家长密码 / 注册时间 / 操作 */
  const tRow = rows.find((r) => num(r[0]) === target.id)
  ok('★ 找得到目标用户那一行', !!tRow, tRow?.[2])
  ok(
    '★ 积分 / 兑换券 / 钻石 三列 == 接口的钱包',
    num(tRow?.[5]) === target.points &&
      num(tRow?.[6]) === target.coupon &&
      num(tRow?.[7]) === target.diamond,
    `界面 ${tRow?.[5]}/${tRow?.[6]}/${tRow?.[7]} · 接口 ${target.points}/${target.coupon}/${target.diamond}`
  )
  ok('手机号是明文（运营要用）', (tRow?.[3] ?? '') === (target.mobile || '—'), tRow?.[3])

  /* ============ ⑤ 家长密码列不泄露凭据 ============ */
  console.log('\n⑤ 家长密码列')
  const pinCells = rows.map((r) => r[8] ?? '')
  ok(
    '★ 只出现「已设置 / 未设 / —」三种文案',
    pinCells.every((s) => /^(已设置|未设|—)$/.test(s)),
    [...new Set(pinCells)].join(' | ')
  )
  ok(
    '★★ 没有把 4 位数字（明文 PIN）打出来',
    pinCells.every((s) => !/\d{3,}/.test(s)),
    pinCells.join(' | ')
  )
  /* pin_known=false 才画「—」；接口说 known 就不能画成「—」（那是把「不知道」画成「没有」） */
  ok(
    '★ pin_known=true 时不会画成「—」',
    apiUsers.every((u, i) => !(u.pin_known !== false && pinCells[i] === '—')),
    pinCells.join(' | ')
  )

  /* ============ ⑥ 详情 + 账本 ============ */
  console.log('\n⑥ 详情 / 流水')
  await clickInRow(target.id, '详情 / 流水')
  await page.waitForFunction(
    () => {
      const d = [...document.querySelectorAll('.el-dialog')].find((x) => x.offsetParent !== null)
      return d && (d.innerText || '').includes('账本流水')
    },
    { timeout: 20000 }
  )
  ok('★ 点「详情 / 流水」弹窗出来了', true)

  /* ★ 必须等**流水真的到了**：弹窗一出现，「账本流水」这个标题就在了，
     但 `loadLedger()` 是异步的，这一刻表格还是空的、分页写着「共 0 条」。
     等标题就会读到 0 行，然后红成「账本没有流水」—— 而接口明明有 315 条。 */
  await page.waitForFunction(
    () => {
      const d = [...document.querySelectorAll('.el-dialog')].find((x) => x.offsetParent !== null)
      return d && d.querySelectorAll('tbody tr.el-table__row').length > 0
    },
    { timeout: 20000 }
  )

  let dlg = await dialogText()
  ok(
    '★ 弹窗里的钱包三个数 == 接口的 wallet',
    dlg.includes(`积分 ${wallet.points ?? 0}`) &&
      dlg.includes(`兑换券 ${wallet.coupon ?? 0}`) &&
      dlg.includes(`钻石 ${wallet.diamond ?? 0}`),
    `积分 ${wallet.points} / 券 ${wallet.coupon} / 钻 ${wallet.diamond}`
  )
  /* ★ 详情接口不返回 has_pin，整份覆盖会把它变成 undefined ⇒ 显示「未设置」。
     页面保留了列表那一份，所以这里的**状态**必须和列表那一行一致。
     ⚠️ 别比原始字符串：列表里是短的「未设」，弹窗里是长的
     「未设置（谁都能进家长区）」—— 同一件事，两种写法。比状态。 */
  const PIN_DETAIL = {
    已设置: '已设置（家长区可进）',
    未设: '未设置（谁都能进家长区）',
    '—': '—（查不到）'
  }
  const listPinText = tRow?.[8] ?? ''
  const detailPinMatch = dlg.match(/家长密码 (\S+)/)
  ok(
    '★ 详情里的家长锁状态和列表一致（详情接口不返回这个字段，别被覆盖掉）',
    (detailPinMatch?.[1] ?? '') === PIN_DETAIL[listPinText],
    `详情 ${detailPinMatch?.[1]} / 列表 ${listPinText} → 期望 ${PIN_DETAIL[listPinText]}`
  )
  ok('★★ 详情里也没有明文 PIN', !/家长密码 \d{3,}/.test(dlg))

  /* ============ ⑦ 账实相符 ============ */
  console.log('\n⑦ 账实相符')
  const ledRows = await page.evaluate(() => {
    const d = [...document.querySelectorAll('.el-dialog')].find((x) => x.offsetParent !== null)
    if (!d) return []
    return [...d.querySelectorAll('tbody tr.el-table__row')].map((tr) =>
      [...tr.querySelectorAll('td')].map((td) => (td.innerText || '').trim())
    )
  })
  ok(
    '★ 账本有流水（这一页的意义就是「钱去哪了」）',
    ledRows.length > 0,
    `${ledRows.length} 行 / 接口 total ${ledgerApi.data?.total}`
  )
  ok(
    '账本总数 = 接口 total',
    num(dlg.match(/共\s*(\d+)\s*条/)?.[1]) === ledgerApi.data?.total,
    `界面 ${dlg.match(/共\s*\d+\s*条/)?.[0]} / 接口 ${ledgerApi.data?.total}`
  )

  /* ★★ 核心断言：每种币**最新那条**流水的 balance_after 必须等于当前余额。
     页面按 id DESC 排，所以每种币第一次出现的那行就是最新那行。
     对不上 = 余额和账本已经不是一回事了（而界面不会报任何错）。 */
  /* 列顺序：时间 / 币种 / 变动 / 余额 / 来源 / 备注 */
  const newest = {}
  for (const r of ledRows) {
    const cur = r[1] ?? ''
    if (cur && !(cur in newest)) newest[cur] = num(r[3])
  }
  const mismatches = Object.entries(newest).filter(
    ([cur, bal]) => wallet[cur] !== undefined && bal !== wallet[cur]
  )
  ok(
    '★★ 每种币「最新一条流水的余额」== 当前余额（账实相符）',
    mismatches.length === 0,
    mismatches.length
      ? mismatches.map(([c, b]) => `${c}: 流水 ${b} ≠ 余额 ${wallet[c]}`).join('；')
      : Object.entries(newest)
          .map(([c, b]) => `${c}=${b}`)
          .join(' ')
  )
  /* 等过渡动画走完再截：弹窗淡入期间截图会「重影」，看着像弹窗半透明，
     实测 `opacity` 一直是 1、背景是纯白 —— 纯粹是截图时机的问题。 */
  await new Promise((r) => setTimeout(r, 500))
  console.log('  截图（详情 + 账本）：' + (await a.shot('user-ledger')))
  /* ============ ⑧ 调余额的守卫 ============ */
  console.log('\n⑧ 调余额')
  await a.clickByText('关闭')
  await page.waitForFunction(noVisibleDialog, { timeout: 20000 })
  await clickInRow(target.id, '调余额')
  await page.waitForFunction(
    () => {
      const d = [...document.querySelectorAll('.el-dialog')].find((x) => x.offsetParent !== null)
      return d && (d.innerText || '').includes('变动量')
    },
    { timeout: 20000 }
  )
  ok('点「调余额」弹窗出来了', true)
  await new Promise((r) => setTimeout(r, 600)) // 等弹窗落位，否则第一次点击会点空

  const dialogOpen = () =>
    page.evaluate(() => [...document.querySelectorAll('.el-dialog')].some((d) => d.offsetParent !== null))

  /* 什么都不填就点确认：必须被前端挡住，而且**弹窗不能关**
     （关掉的话运营会以为已经改好了） */
  await a.clickByText('确认调整')
  ok('★ 变动量为 0 时被挡住', await toastAppears('变动量不能是 0'))
  ok('★ 被挡住时弹窗**不关**（关掉会让人以为改好了）', await dialogOpen())

  /* 填了变动量但没填备注：同样必须被挡住 —— 备注是对账的唯一线索 */
  await page.evaluate(() => {
    const d = [...document.querySelectorAll('.el-dialog')].find((x) => x.offsetParent !== null)
    const inp = d?.querySelector('.el-input-number input')
    if (inp) {
      inp.focus()
      inp.select()
    }
  })
  await page.keyboard.type('10')
  await page.evaluate(() => {
    const d = [...document.querySelectorAll('.el-dialog')].find((x) => x.offsetParent !== null)
    d?.querySelector('.el-input-number input')?.blur()
  })
  await new Promise((r) => setTimeout(r, 300))
  await a.clickByText('确认调整')
  ok('★ 没填备注时被挡住（对账只能靠它）', await toastAppears('备注必须填'))
  ok('★ 这次弹窗也没关', await dialogOpen())

  /* 真改一笔？—— 不改。这条探针**只验守卫**：
     调余额会写真实账本流水，跑一次就多一条，反复跑会把账本灌满。 */
  ok('没有真的改动余额（探针只验守卫，不写流水）', true)

  ok('没有页面级报错', errors.length === 0, [...errors, ...reqFails].slice(0, 3).join(' | '))
  console.log('  截图：' + (await a.shot('user-detail')))
} catch (e) {
  failed++
  console.error('\n✗ 探针中断：', String(e).slice(0, 400))
  try {
    const txt = await page.evaluate(() => document.body.innerText.replace(/\s+/g, ' ').slice(0, 300))
    console.error('  页面文字：', txt)
    console.error('  截图：' + (await a.shot('user-fail')))
  } catch {}
} finally {
  await a.close()
  console.log(failed === 0 ? '\n全部通过' : `\n${failed} 条失败`)
  process.exit(failed === 0 ? 0 : 1)
}
