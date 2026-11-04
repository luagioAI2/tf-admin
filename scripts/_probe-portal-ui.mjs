/**
 * 家长 / 商家门户的**界面**探针：登录页一个框、两个门户的页面到底能不能打开。
 *
 * 用法（三步，顺序不能反）：
 *
 *   # ① 造探针数据（顺手把接口判据也跑一遍，现在 82 条）
 *   python scripts/check-portal-perms.py --keep
 *
 *   # ② 看界面（需要 web 的 dev server 在 8891）
 *   node scripts/_probe-portal-ui.mjs
 *
 *   # ③ 清掉探针数据（不带 --keep 会自己清）
 *   python scripts/check-portal-perms.py
 *
 * ⚠️ ① 和 ② **不能并行**：① 的收尾会把 `+861990000*` 整个前缀删掉，
 *    而本探针用的家长就是 `+8619900000101` —— 并行跑的表现是
 *    「手机号或密码不对」，看着像登录坏了，其实是被另一个脚本清掉了。
 *
 * # 为什么必须有一个「看界面」的探针
 *
 * 接口全绿 + `vite build` 全绿，仍然可能是一屏空白：
 *   - 菜单的 `component` 路径写错 ⇒ 路由进去了，chunk 404，白屏
 *   - 模板里用了一个没 import 的函数 ⇒ 渲染时抛错，白屏
 *   - `defaultRouter` 和菜单 name 对不上 ⇒ 停在空白页，只弹一句提示
 * 这三种都**不产生后端错误**，只有浏览器能看见。
 *
 * # 判据
 *
 * 1. 登录页**只有一个输入框**、**没有身份切换**（角色由审核记录决定，不由用户声明）
 * 2. 家长登录后停在「概览」，**卡片上的数字和接口返回的一致**
 *    （只看「有卡片」不够：卡片先画 0 再填数，先看到 0 也算通过）
 * 3. 左侧点得开「心愿单」，能看到那条探针心愿
 * 4. 商家登录后「我的商品」里能看到自己那件
 * 5. 商家的「我的模板」：两个页签都在、有自己的推广码、**看不到官方模板**，
 *    新建出来的模板落库是 scope=merchant + 本人 uid，删得掉
 * 6. 全程没有 pageerror、没有本机 4xx/5xx
 */
import { adminApi, openAdmin } from './_admin-lib.mjs'

const WEB = process.env.TF_ADMIN_WEB ?? 'http://localhost:8891/'
const API = process.env.TF_ADMIN_API ?? 'http://127.0.0.1:8890'
const PW = 'Probe@123456'
const PARENT = '+8619900000101'
const MERCH = '+8619900000201'
const ORDER_MARK = '探针心愿-只有家长A看得到'
/* 登录页唯一那个输入框。按 placeholder 找，别按 index ——
   页面上还有密码框，`input` 顺序一变就填错框。 */
const ACCOUNT_SEL = 'input[placeholder="用户名 / App 手机号"]'
const ITEM_NAME = '探针商品A'
const BUY_URL = 'https://example.com/probe-item'

const results = []
const check = (ok, label, detail = '') => {
  results.push({ ok: !!ok, label, detail })
  console.log((ok ? '  ✓ ' : '  ✗ ') + label + (!ok && detail ? '  — ' + detail : ''))
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

const post = async (path, body) => {
  const r = await fetch(API + path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  })
  return r.json()
}

const a = await openAdmin({ url: WEB })

/* 截图前等 400ms：`v-loading` 的遮罩和 el-tag 的入场都在下一帧之后才稳定，
   抢着截会得到一张「半透明、标签看不见」的图 —— 而那张图会被当成「页面坏了」。 */
const shot = async (name) => {
  await sleep(400)
  return a.shot(name)
}

try {
  // ---------- 登录页：一个框 ----------
  console.log('\n== 1. 登录页 ==')
  await a.page.goto(WEB, { waitUntil: 'domcontentloaded', timeout: 60000 })
  await a.page.waitForSelector(ACCOUNT_SEL, { timeout: 60000 })

  const shape = await a.page.evaluate((sel) => ({
    account: !!document.querySelector(sel),
    radios: [...document.querySelectorAll('.el-radio-button')].map((el) =>
      (el.innerText || '').trim()
    ),
    hint: ['通过审核', '不用另记账号', '账号的权限'].some((w) =>
      document.body.innerText.includes(w)
    )
  }), ACCOUNT_SEL)
  check(shape.account, '登录页只有一个「用户名 / App 手机号」输入框')
  // ★ 这里断言的是**没有**身份切换：那个开关等于让用户自己声明权限，
  //   选错的表现是「菜单少了一半」，用户会以为申请没过。
  check(shape.radios.length === 0, '没有「管理员 / 家长 / 商家」切换',
    JSON.stringify(shape.radios))
  check(!shape.hint, '登录页没有解释性提示语', JSON.stringify(shape))
  await shot('01-login')

  // ---------- 家长登录 ----------
  console.log('\n== 2. 家长登录与概览 ==')
  await a.fill(ACCOUNT_SEL, PARENT)
  await a.fill('input[placeholder="请输入密码"]', PW)
  await a.clickByText('登 录')

  // 等的不只是「离开登录页」—— 路由进去了也可能是空白页，
  // 所以等的是「概览卡片上的数字到了」
  const tok = (await post('/base/appLogin', { mobile: PARENT, password: PW }))
  const pTok = tok?.data?.token
  check(!!pTok, '家长 token 拿到了（否则下面的数字对比没有意义）', JSON.stringify(tok))
  const ov = await fetch(API + '/mall/portal/parent/overview', {
    headers: { 'x-token': pTok ?? '' }
  }).then((r) => r.json())
  const want = [
    String(ov?.data?.points ?? '?'),
    String(ov?.data?.coupon ?? '?'),
    String(ov?.data?.diamond ?? '?')
  ]
  check(want.every((v) => v !== '?'), '概览接口返回了三个余额', JSON.stringify(ov))
  // ★ 基线不能是 0：钱包为空时接口和卡片都是 0，`0 == 0` 恒真 ——
  //   卡片根本没渲染也能「通过」。探针脚本已经把它设成 777 / 13 / 5。
  check(want[0] !== '0', '探针基线非零（否则下面那条断言恒真）', want.join('/'))

  await a.waitFor(
    (w) => {
      const vs = [...document.querySelectorAll('.card .v')].map((el) => (el.innerText || '').trim())
      return vs[0] === w[0] && vs[1] === w[1] && vs[2] === w[2]
    },
    `概览卡片显示 积分=${want[0]} 券=${want[1]} 钻石=${want[2]}`,
    { timeout: 30000, arg: want }
  )
  check(true, `概览的数字和接口一致（积分 ${want[0]} / 券 ${want[1]} / 钻石 ${want[2]}）`)
  check(!(await a.page.evaluate(() => location.hash.includes('login'))), '已经离开登录页')

  // 各状态计数：静态数组写成 `s.n` 时这里会是空的，而且不报错
  const stats = await a.page.evaluate(() =>
    [...document.querySelectorAll('.stat')].map((e) => (e.innerText || '').replace(/\s+/g, ' ').trim())
  )
  check(
    stats.length === 4 && stats.every((t) => /\d/.test(t)),
    '四个状态计数都渲染出了数字',
    JSON.stringify(stats)
  )
  await shot('02-parent-overview')

  // ---------- 家长：心愿单 ----------
  console.log('\n== 3. 家长心愿单 ==')
  await a.clickByText('家长中心')
  await sleep(600)
  await a.clickByText('心愿单')
  // ★ 锚点必须选**只有这一页才有**的东西。
  //   「探针那条心愿」在概览页的「最近 5 单」里也有 —— 拿它当锚点，
  //   页面根本没切过去也会通过（实测过一次：标签页开了、正文还是概览）。
  //   这里用筛选栏的 placeholder，它在概览页不存在。
  await a.page.waitForSelector('input[placeholder="搜心愿名字"]', { timeout: 20000 })
  await a.waitFor(
    (m) => {
      const rows = [...document.querySelectorAll('tbody tr.el-table__row')]
      return rows.some((r) => (r.innerText || '').includes(m))
    },
    '心愿单表格里出现探针那条心愿',
    { timeout: 20000, arg: ORDER_MARK }
  )
  check(true, '心愿单页能打开，表格里有自己的心愿')

  /* #16：游戏 / 心愿商城升成后台一级菜单后，家长不该看到它们。
     只看「菜单没授权」不够 —— 界面层再钉一次，防止哪天种子脚本顺手
     给 1001 加了授权而 DB 探针没覆盖到。 */
  const parentAside = await a.page.evaluate(() => {
    const el = document.querySelector('.el-menu')
    return el ? (el.innerText || '') : ''
  })
  check(
    !parentAside.includes('游戏') && !parentAside.includes('心愿商城'),
    '★ 家长侧栏里没有「游戏」/「心愿商城」后台菜单',
    parentAside.slice(0, 160)
  )

  // 「去下单」= 订单 → item_id 回查 wish_item.buy_url。订单表本身不存 URL，
  // 这条链断掉时页面上什么都不显示，接口也不会报错。
  const link = await a.page.evaluate(() => {
    const el = [...document.querySelectorAll('tbody a')].find((x) =>
      (x.innerText || '').includes('去下单')
    )
    return el ? { href: el.getAttribute('href'), text: (el.innerText || '').trim() } : null
  })
  check(
    link && link.href === BUY_URL,
    '「去下单」链接指向商品的 buy_url',
    JSON.stringify(link)
  )
  await shot('03-parent-wishes')

  // ---------- 商家 ----------
  console.log('\n== 4. 商家登录与商品 ==')
  // 换身份：token 存在 localStorage + cookie 两处，只清一处会被路由守卫拉回去
  await a.page.evaluate(() => {
    localStorage.clear()
    sessionStorage.clear()
    document.cookie.split(';').forEach((c) => {
      const k = c.split('=')[0].trim()
      if (k) document.cookie = k + '=; Max-Age=0; path=/'
    })
  })
  await a.page.goto(WEB, { waitUntil: 'domcontentloaded' })
  await a.page.waitForSelector(ACCOUNT_SEL, { timeout: 60000 })
  await a.fill(ACCOUNT_SEL, MERCH)
  await a.fill('input[placeholder="请输入密码"]', PW)
  await a.clickByText('登 录')

  // 同样要一个「只有这一页才有」的锚点：defaultRouter 就是我的商品，
  // 但先确认按钮在，才能说明这一页真的渲染了
  await a.page.waitForFunction(
    () => [...document.querySelectorAll('button')].some((b) => (b.innerText || '').includes('上架新商品')),
    { timeout: 30000 }
  )
  await a.waitFor(
    (n) => document.body.innerText.includes(n),
    '我的商品里出现探针商品',
    { timeout: 20000, arg: ITEM_NAME }
  )
  check(true, '商家「我的商品」能看到自己的商品')

  const onlyMine = await a.page.evaluate(() => {
    const t = document.body.innerText
    return !t.includes('心愿单') && !t.includes('兑换记录')
  })
  check(onlyMine, '商家侧栏里没有家长页面')

  const merchantAside = await a.page.evaluate(() => {
    const el = document.querySelector('.el-menu')
    return el ? (el.innerText || '') : ''
  })
  check(
    !merchantAside.includes('游戏') && !merchantAside.includes('心愿商城'),
    '★ 商家侧栏里没有「游戏」/「心愿商城」后台菜单',
    merchantAside.slice(0, 160)
  )
  check(
    !merchantAside.includes('模板配置') && merchantAside.includes('我的模板'),
    '★ 商家侧栏没有超管那个「模板配置」，只有「我的模板」',
    merchantAside.slice(0, 160)
  )
  await shot('04-merchant-items')

  // ---------- 我的模板（商家配「我推广的用户看到什么」） ----------
  console.log('\n== 4.5 商家「我的模板」 ==')
  /* 上一次跑挂了可能留下同名模板：先按名字清掉，保证可以反复跑。
     ★ 走的是超管 token（adminApi），但只删这一条探针自己建的。 */
  const PROBE_TPL = '探针模板-可删'
  const stale =
    (await adminApi('/mall/template/list', { page: 1, pageSize: 200, kind: 'task' })).data
      ?.list ?? []
  for (const t of stale.filter((x) => x.name === PROBE_TPL)) {
    await adminApi('/mall/template/delete', { id: t.id })
  }

  await a.clickByText('商家中心')
  await sleep(600)
  await a.clickByText('我的模板')
  await a.page.waitForFunction(() => location.hash.includes('templates'), { timeout: 20000 })
  await a.page.waitForFunction(
    () =>
      [...document.querySelectorAll('.el-tabs__item')].some(
        (x) => (x.innerText || '').trim() === '任务模板'
      ),
    { timeout: 20000 }
  )
  /* ★ 先等推广码那块**渲染出来**再动别的：
     它是异步拉回来的（v-if 挂上去会把下面的工具栏整体下移 ~70px），
     而 clickByText 是「量坐标 → 移过去 → 点」，中间那 300ms 里布局一变，
     点击就落在表格空白处 —— 表现是「点了新建模板，弹窗没出来」，
     和「按钮坏了」一模一样。实测踩过一次。 */
  await a.page.waitForFunction(
    () => /^TF[0-9A-Z]+-/.test(document.querySelector('[data-probe="invite-code"]')?.innerText.trim() ?? ''),
    { timeout: 20000 }
  )

  const tplShell = await a.page.evaluate(() => ({
    tabs: [...document.querySelectorAll('.el-tabs__item')].map((x) => (x.innerText || '').trim()),
    invite: document.querySelector('[data-probe="invite-code"]')?.innerText.trim() ?? '',
    rows: [...document.querySelectorAll('[data-probe="tpl-name"]')].map((e) =>
      e.innerText.trim()
    )
  }))
  check(
    tplShell.tabs.includes('任务模板') && tplShell.tabs.includes('兑换模板'),
    '★ 商家这一页两个页签都在（同一个 .vue，按路由 name 定默认页签）',
    tplShell.tabs.join(',')
  )
  check(
    /^TF[0-9A-Z]+-[0-9A-F]{4}$/.test(tplShell.invite),
    '★ 显示了自己的推广码（印在物料上，注册时带给 App）',
    tplShell.invite
  )
  check(
    !tplShell.rows.includes('官方任务模板'),
    '★★ 商家列表里没有官方模板（数据范围被后端收成 owner_uid=自己）',
    tplShell.rows.join(',')
  )

  /* 新建 → 真的挂在商家自己名下 → 删掉。
     ★ 关键判据不是「界面多了一行」，而是**落库的 scope / owner_uid**：
       前端没传 scope，后端必须自己收敛成 merchant + 本人。 */
  await a.clickByText('新建模板')
  await a.page.waitForSelector('.el-dialog', { timeout: 20000 })
  await a.fill('input[placeholder="例如：寒假作业模板"]', PROBE_TPL)
  await a.clickByText('新建')
  await a.page.waitForFunction(
    (n) => [...document.querySelectorAll('[data-probe="tpl-name"]')].some((e) => e.innerText.trim() === n),
    { timeout: 20000 },
    PROBE_TPL
  )
  check(true, '新建的模板出现在列表里')

  const listed =
    (await adminApi('/mall/template/list', { page: 1, pageSize: 200, kind: 'task' })).data
      ?.list ?? []
  const made = listed.find((x) => x.name === PROBE_TPL)
  check(
    !!made && made.scope === 'merchant' && made.owner_uid > 0,
    '★★ 落库是「商家模板 + 本人 uid」（前端不传 scope 也算数）',
    made ? `scope=${made.scope} owner_uid=${made.owner_uid}` : '没找到'
  )

  /* 删掉：先点行内「删除」，再点确认框的确认按钮。
     ★ 确认框里的「删除」和行内那个同名 ⇒ 不能按文字点，直接点主按钮。 */
  await a.page.evaluate((n) => {
    const tr = [...document.querySelectorAll('tbody tr.el-table__row')].find((r) =>
      (r.innerText || '').includes(n)
    )
    const btn = tr && [...tr.querySelectorAll('button')].find((b) => (b.innerText || '').trim() === '删除')
    btn?.click()
  }, PROBE_TPL)
  await a.page.waitForSelector('.el-message-box', { timeout: 20000 })
  await a.page.evaluate(() => {
    const btn = document.querySelector('.el-message-box__btns .el-button--primary')
    btn?.click()
  })
  await a.page.waitForFunction(
    (n) => ![...document.querySelectorAll('[data-probe="tpl-name"]')].some((e) => e.innerText.trim() === n),
    { timeout: 20000 },
    PROBE_TPL
  )
  check(true, '删掉之后列表里也没有了')
  await shot('04b-merchant-templates')

  // 入驻状态页
  await a.clickByText('商家中心')
  await sleep(600)
  await a.clickByText('入驻状态')
  await a.waitFor(
    () => document.body.innerText.includes('已通过'),
    '入驻状态页显示「已通过」',
    { timeout: 20000 }
  )
  check(true, '入驻状态页显示「已通过」')
  await shot('05-merchant-status')

  // ---------- 收尾 ----------
  console.log('\n== 5. 干净程度 ==')
  // favicon / HMR 之类的噪音不算；只关心真正的页面异常和接口失败
  const errs = a.errors.filter((e) => e && !/favicon/i.test(e))
  const fails = a.reqFails.filter((f) => !/favicon/i.test(f))
  check(errs.length === 0, '没有页面级报错', errs.join(' | '))
  check(fails.length === 0, '没有本机 4xx/5xx', fails.join(' | '))
} catch (e) {
  check(false, '探针中断：' + (e?.message ?? e))
} finally {
  await a.close()
}

const bad = results.filter((r) => !r.ok)
console.log(`\n${results.length - bad.length}/${results.length} 通过`)
if (bad.length) {
  console.log('没通过：')
  for (const b of bad) console.log('  -', b.label, '|', b.detail)
  process.exit(1)
}
