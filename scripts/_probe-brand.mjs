/**
 * 「后台长得像不像这个产品」的界面探针。
 *
 * 用法（dev server 在 8891、接口在 8890）：
 *   node scripts/_probe-brand.mjs
 *
 * # 为什么要单独一条
 *
 * 这次改的全是**文字和入口**：站点名、标题、页脚、顶栏那个「授权版演示」按钮、
 * 侧边栏里框架自己的演示菜单。这类改动有个共同点 ——
 * **接口全绿、`vite build` 全绿、页面也不报错**，改漏了只是「那一行还在」。
 * 只有真的把页面打开、把 innerText 读出来才看得见。
 *
 * # 判据
 *
 * 1. 登录页：标题是产品名、图标加载成功、没有「前往初始化」、没有框架字样
 * 2. 登录后：顶栏是产品名、没有「授权版演示」/「视频教程」
 * 3. 侧边栏没有框架的演示菜单（示例文件 / 官方网站 / 插件系统）
 * 4. 首页四个数**和接口一致**（不是「有卡片」——卡片先画「—」再填数）
 * 5. 页脚是产品自己的话，没有「Powered by」
 *
 * ⚠️ 第 4 条特意不做「非零」断言：待审核数本来就是 0 才是常态，
 *    这里比的是「界面 == 服务端」，两边都是 0 也是一个有效结论。
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { openAdmin, adminGet } from './_admin-lib.mjs'

const WEB = process.env.TF_ADMIN_WEB ?? 'http://localhost:8891/'

const results = []
const check = (ok, label, detail = '') => {
  results.push({ ok: !!ok, label, detail })
  console.log((ok ? '  ✓ ' : '  ✗ ') + label + (!ok && detail ? '  — ' + detail : ''))
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

/* 框架字样：页面正文里出现任何一个都算没清干净。
   `gva-container` 这种 class 名不算 —— 它们不进 innerText。 */
const FRAMEWORK_WORDS = ['Gin-Vue', 'gin-vue', 'GVA', 'flipped-aurora']

const a = await openAdmin({ url: WEB })

const shot = async (name) => {
  await sleep(400)
  return a.shot(name)
}

try {
  // ---------- ① 登录页 ----------
  console.log('\n== 1. 登录页 ==')
  await a.page.goto(WEB, { waitUntil: 'domcontentloaded', timeout: 60000 })
  await a.page.waitForSelector('input[placeholder*="用户名"]', { timeout: 60000 })
  await sleep(600)

  const head = await a.page.evaluate(() => ({
    title: document.title,
    text: document.body.innerText,
    logo: (() => {
      const img = document.querySelector('img[src="/logo.png"]')
      return img ? { w: img.naturalWidth, h: img.naturalHeight } : null
    })()
  }))

  check(head.title.includes('小任务农场'), '浏览器标题是产品名', JSON.stringify(head.title))
  check(
    !FRAMEWORK_WORDS.some((w) => head.text.includes(w)),
    '登录页正文没有框架字样',
    FRAMEWORK_WORDS.filter((w) => head.text.includes(w)).join(',')
  )
  check(!head.text.includes('前往初始化'), '没有「前往初始化」入口')
  // 图标没加载出来时是 0×0，页面照样「不报错」——只有 naturalWidth 能看出来
  check(head.logo && head.logo.w > 0, '图标加载出来了', JSON.stringify(head.logo))
  check(head.text.includes('完成小任务'), '右栏是产品的标语，不是框架的演示图')

  /* ★ 主色要**读渲染结果**，不能只看源码里写了什么。
     这条链有三段：`app.js` 的 primaryColor → `setBodyPrimaryColor()` 写内联
     CSS 变量 → 按钮读 `var(--el-color-primary)`。
     实测过一次「改了样式表但按钮还是蓝的」：内联样式优先级更高，
     样式表那份被无声覆盖 —— 而 build 和其余断言全绿。 */
  const brand = await a.page.evaluate(() => {
    const b = document.querySelector('.el-button--primary')
    return {
      bg: b ? getComputedStyle(b).backgroundColor : null,
      primary: getComputedStyle(document.documentElement)
        .getPropertyValue('--el-color-primary')
        .trim()
    }
  })
  check(
    brand.bg === 'rgb(21, 128, 61)',
    '主按钮是农场绿 #15803d（不是框架默认的蓝）',
    JSON.stringify(brand)
  )
  await shot('brand-01-login')

  // ---------- ② 登录后：顶栏 ----------
  console.log('\n== 2. 顶栏 ==')
  await a.login()
  await a.page.waitForFunction(
    () => document.querySelector('.gva-container') || document.body.innerText.includes('小任务农场'),
    { timeout: 30000 }
  )
  await sleep(800)

  /* ★ 顶栏 / 侧边栏都**必须按 DOM 找**，不能退化成 `document.body`：
     退化了之后「顶栏显示产品名」会被首页标题顶过去（恒真），
     而「没有授权版演示」又变成全页断言（更严但看不出是哪一层）。
     顶栏 = `.fixed.top-0`（GVA 用 fixed 顶栏），侧边栏 = `.el-menu`。 */
  const shell = await a.page.evaluate(() => {
    const pick = (sel) => {
      const el = document.querySelector(sel)
      return el ? (el.innerText || '').replace(/\s+/g, ' ').trim() : null
    }
    return { header: pick('.fixed.top-0'), aside: pick('.el-menu') }
  })

  check(shell.header !== null, '找得到顶栏（否则下面两条等于没测）', JSON.stringify(shell.header))
  check(
    (shell.header ?? '').includes('小任务农场'),
    '顶栏显示产品名',
    (shell.header ?? '').slice(0, 120)
  )
  check(!(shell.header ?? '').includes('授权版演示'), '顶栏没有「授权版演示」按钮')
  check(!(shell.header ?? '').includes('视频教程'), '顶栏没有「视频教程」下拉')

  // ---------- ③ 侧边栏 ----------
  console.log('\n== 3. 侧边栏 ==')
  check(shell.aside !== null, '找得到侧边栏', JSON.stringify(shell.aside))
  const gone = ['示例文件', '官方网站', '插件系统']
  for (const g of gone) {
    check(!(shell.aside ?? '').includes(g), `侧边栏没有「${g}」`)
  }
  check((shell.aside ?? '').includes('商城运营'), '侧边栏还有「商城运营」（没把业务菜单一起藏掉）')

  // ---------- ④ 首页四个数 ----------
  console.log('\n== 4. 首页 ==')
  await a.page.goto(WEB + '#/layout/dashboard', { waitUntil: 'domcontentloaded' })
  await sleep(1200)

  const [ov, pc] = await Promise.all([
    adminGet('/mall/dashboard/overview'),
    adminGet('/mall/merchant/pending-count')
  ])
  const want = {
    merchant: String(pc?.data ?? 0),
    wish: String(ov?.data?.wish_pending_action ?? 0),
    user: String(ov?.data?.user_new ?? 0),
    coin: String(ov?.data?.points_in_wallets ?? 0)
  }
  check(ov?.code === 0 && pc?.code === 0, '两个看板接口都通了', JSON.stringify({ ov, pc }).slice(0, 200))

  await a.waitFor(
    (w) =>
      Object.entries(w).every(([k, v]) => {
        const el = document.querySelector(`[data-kpi="${k}"] .kpi-v`)
        return el && (el.innerText || '').trim() === v
      }),
    `首页四个数 == 接口值 ${JSON.stringify(want)}`,
    { timeout: 30000, arg: want }
  )
  check(true, `首页四个数 == 接口值（待审核 ${want.merchant} / 待兑现 ${want.wish} / 新增 ${want.user} / 积分 ${want.coin}）`)

  const home = await a.page.evaluate(() => document.body.innerText.replace(/\s+/g, ' '))
  check(home.includes('常用入口'), '首页有「常用入口」')
  check(!home.includes('购买商业授权'), '首页没有「购买商业授权」')
  check(!FRAMEWORK_WORDS.some((w) => home.includes(w)), '首页正文没有框架字样')
  await shot('brand-02-dashboard')

  // ---------- ⑤ 页脚 ----------
  console.log('\n== 5. 页脚 ==')
  const foot = await a.page.evaluate(() => document.body.innerText.replace(/\s+/g, ' '))
  check(!foot.includes('Powered by'), '页脚没有「Powered by」')
  check(foot.includes('家长 / 商家请用 App 的手机号和密码登录'), '页脚写清了账号从哪儿来')

  // ---------- ⑤·5 用户管理 ----------
  /* 后端列表把 `app_*` 影子账号挡掉了（家长 / 商家第一次登录自动建的那些）。
     挡掉的副作用是「运营在这儿找不到某个家长」，所以页面上必须写清
     「App 用户去哪看」。这一段验的就是那句话 + 列表里真的没有 app_ 行。 */
  console.log('\n== 5·5 用户管理 ==')
  await a.clickByText('超级管理员')
  await sleep(600)
  await a.clickByText('用户管理')
  await a.waitFor(
    () => [...document.querySelectorAll('button')].some((b) => (b.innerText || '').includes('新增用户')),
    '用户管理页打开',
    { timeout: 20000 }
  )
  const um = await a.page.evaluate(() => ({
    rows: [...document.querySelectorAll('tbody tr.el-table__row')].map((r) =>
      (r.innerText || '').replace(/\s+/g, ' ')
    ),
    text: document.body.innerText.replace(/\s+/g, ' ')
  }))
  check(um.rows.length > 0, '用户管理有行（否则下面那条恒真）', String(um.rows.length))
  check(
    !um.rows.some((r) => r.includes('app_')),
    '列表里没有 app_* 影子账号',
    um.rows.slice(0, 3).join(' | ')
  )
  check(um.text.includes('App 用户请看'), '页面写清了 App 用户去「商城运营」看')
  await shot('brand-03-users')

  // ---------- ⑥ 干净程度 ----------
  console.log('\n== 6. 干净程度 ==')
  const errs = a.errors.filter((e) => e && !/favicon/i.test(e))
  const fails = a.reqFails.filter((f) => !/favicon/i.test(f))
  check(errs.length === 0, '没有页面级报错', errs.join(' | '))
  check(fails.length === 0, '没有本机 4xx/5xx', fails.join(' | '))

  // ---------- ⑦ 生产产物 ----------
  /* ★★ dev server **不会**注入框架的「未授权」遮罩，构建才会。
     实测过一次：本地全绿，`vite build` 出来的后台一进去就被
     一个 z-index:9999 的全屏弹层挡住（文案「此模板未授权且未获得授权使用权」）。
     所以这一段必须直接读 `dist/`，不能靠打开页面。
     没有 dist 时跳过（不假装通过）。 */
  console.log('\n== 7. 生产产物 ==')
  const distIndex = resolve('web/dist/index.html')
  if (!existsSync(distIndex)) {
    console.log('  · 没有 web/dist/index.html，跳过（先跑一次 vite build）')
  } else {
    const html = readFileSync(distIndex, 'utf8')
    const kw = (html.match(/<meta[^>]*name=["']keywords["'][^>]*>/i) ?? [''])[0]
    check(
      kw.includes('小任务农场') && !/gin-vue|Gin-Vue|GVA/i.test(kw),
      '产物里的 keywords 是产品自己的',
      kw.slice(0, 160)
    )
    check(html.includes('<symbol'), '产物里有 svg sprite（图标不是空的）')
    const jsFiles = readdirSync(resolve('web/dist/assets')).filter((f) => f.endsWith('.js'))
    /* 注入的遮罩脚本里有一段 base64（`position:fixed;bottom:10px;...`），
       原文里搜「未授权」搜不到 —— 它是 `\uXXXX` 转义 + base64 双层藏的。 */
    const injected = jsFiles.filter((f) =>
      readFileSync(resolve('web/dist/assets', f), 'utf8').includes(
        'cG9zaXRpb246Zml4ZWQ7Ym90dG9tOjEwcHg'
      )
    )
    check(injected.length === 0, '产物里没有框架的「未授权」遮罩脚本', injected.join(','))
  }
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
