/**
 * 探针：后台「经营看板」—— 菜单能看见、页面能打开、数字是真数
 * ------------------------------------------------------------
 * 这条探针要抓的是后台**特有**的那类静默坏法：
 *   · 菜单行建了但没授权 ⇒ 登录后左侧根本没有这一项（不报错）；
 *   · `component` 指向的 .vue 不存在 ⇒ 点进去是**空白页**（只报一个 404 chunk），
 *     而人的第一反应是「权限配错了」，会跑去查 casbin；
 *   · 接口没登记 `sys_apis` / `casbin_rule` ⇒ 页面画出来了，数字全是 0 或
 *     控制台里 `{"code":7,"权限不足"}` —— 而 0 看起来像一个真实的数。
 *
 * 所以断言是三条：菜单在、页面渲染出卡片、卡片上的数字**和接口返回一致**。
 *
 * 用法：
 *   1) 后台：tf-admin/server 里的 tf-admin.exe（8890）
 *   2) 前端：tf-admin/web 的 pnpm dev（8891）
 *   3) node scripts/_probe-dashboard.mjs [url]
 */

import { createRequire } from 'node:module'
import { existsSync, mkdirSync } from 'node:fs'
import { pathToFileURL } from 'node:url'
import { resolve } from 'node:path'

/* `puppeteer-core` 不在本工程的依赖里（后台前端不需要它），而 ESM **不认
   NODE_PATH** —— `import 'puppeteer-core'` 直接 ERR_MODULE_NOT_FOUND。
   所以按候选目录用 CJS 解析去找；找到哪个都行，找不到就报清楚要装在哪。 */
const CANDIDATES = [
  process.env.PUPPETEER_FROM,
  '../kid-quest-farm-online/node_modules', // 本机现状：探针都装在那边的工程里
  './web/node_modules',
  './node_modules'
].filter(Boolean)
/* ⚠️ `createRequire()` 不会因为目录不存在就抛 —— 必须自己先看包在不在，
   否则会「成功」拿到一个解析不到包的 require，错误被推到下一行。 */
const found = CANDIDATES.find((d) => existsSync(resolve(d, 'puppeteer-core', 'package.json')))
const req = found
  ? createRequire(pathToFileURL(resolve(found, 'index.js')).href)
  : createRequire(import.meta.url)
let puppeteer
try {
  puppeteer = req('puppeteer-core')
} catch {
  console.error(
    '找不到 puppeteer-core。装一下，或用 PUPPETEER_FROM 指到某个装了它的 node_modules：\n' +
      CANDIDATES.map((d) => '  ' + resolve(d)).join('\n')
  )
  process.exit(2)
}

const URL_ = process.argv[2] ?? 'http://localhost:8891/'
const API = process.env.TF_ADMIN_API ?? 'http://127.0.0.1:8890'
const ADMIN = { username: 'admin', password: process.env.TF_ADMIN_PASSWORD ?? 'TfAdmin@2026' }
const CHROME = [
  'C:/Program Files/Google/Chrome/Application/chrome.exe',
  'C:/Program Files (x86)/Google/Chrome/Application/chrome.exe'
].find((p) => existsSync(p))
if (!CHROME) {
  console.error('找不到 Chrome')
  process.exit(2)
}

let failed = 0
const ok = (label, pass, detail = '') => {
  if (!pass) failed++
  console.log(`  ${pass ? '✓' : '✗'} ${label}${detail ? ` — ${detail}` : ''}`)
}

/* 服务端那份「真数」，一会儿和界面比 */
const login = await fetch(`${API}/base/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(ADMIN)
}).then((r) => r.json())
if (login.code !== 0) {
  console.error('后台登录失败：', login.msg)
  process.exit(2)
}
const token = login.data.token
const overview = await fetch(`${API}/mall/dashboard/overview`, {
  headers: { 'x-token': token }
}).then((r) => r.json())
console.log(`接口真数：${JSON.stringify(overview.data)}\n`)

const browser = await puppeteer.launch({
  executablePath: CHROME,
  headless: 'new',
  args: ['--no-first-run']
})
const page = await browser.newPage()
await page.setViewport({ width: 1440, height: 900 })
const errors = []
page.on('pageerror', (e) => errors.push(String(e).slice(0, 200)))

/** 点一个「文字正好/包含它」的**最内层**元素（找不到就抛，别静默点空）
 *
 * ★ 一定要取「最内层」：`html` / `body` / 外层 div 的 innerText 里也有这段文字，
 *   拿 `querySelectorAll('*')` 随便挑一个，很可能点的是 `body` 的正中间 ——
 *   什么都不会发生，而断言会红在下一步（「登录没进去」），完全看不出是这个原因。
 *   判据：**没有任何子元素**的 innerText 也包含这段文字。
 */
async function clickByText(text, selector = 'button, a, li, span, div') {
  const hit = await page.evaluate(
    (t, sel) => {
      const all = [...document.querySelectorAll(sel)].filter((el) => {
        const s = (el.innerText || '').trim()
        return s === t || s.includes(t)
      })
      /* ① 只要最内层（外层容器的 innerText 也含这段文字）；
         ② **精确匹配优先于包含匹配** —— 「刷新」按钮旁边那句提示
            「点右上角刷新重新统计」也含「刷新」，先挑包含匹配就会点到提示文字上，
            按钮一次都没按，断言红成「点了没反应」。 */
      const innermost = (arr) =>
        arr.filter((el) => ![...el.children].some((c) => ((c.innerText || '').trim() || '').includes(t)))
      const exact = innermost(all.filter((el) => (el.innerText || '').trim() === t))
      const near = innermost(all)
      const el = exact[0] ?? near[0]
      if (!el) return null
      el.scrollIntoView({ block: 'center' })
      const r = el.getBoundingClientRect()
      if (!r.width || !r.height) return null
      return {
        cx: Math.round(r.x + r.width / 2),
        cy: Math.round(r.y + r.height / 2),
        tag: el.tagName,
        txt: (el.innerText || '').trim().slice(0, 16)
      }
    },
    text,
    selector
  )
  if (!hit) throw new Error(`找不到「${text}」`)
  /* 先移过去再点：侧边栏的子菜单是**悬停展开**的（`el-menu` 在 GVA 里配的是
     hover 触发），只 `click` 不 `move` 的话鼠标还停在上一处 —— 展开不了，
     而断言会红成「菜单没建」。 */
  await page.mouse.move(hit.cx, hit.cy)
  await new Promise((r) => setTimeout(r, 300))
  await page.mouse.click(hit.cx, hit.cy)
  return hit
}

try {
  /* ============ ① 登录 ============ */
  console.log('① 登录后台')
  await page.goto(URL_, { waitUntil: 'domcontentloaded', timeout: 60000 })
  await page.waitForSelector('input[placeholder="请输入用户名"]', { timeout: 60000 })
  /* ★ 两个框都**可能是预填的**（GVA 会把上次登录的用户名留在表单里），
     直接 type 会拼成 `adminadmin`，然后红在「用户名或密码错误」上 ——
     看着像密码不对，其实是填错了。所以先 Ctrl+A 全选再打。 */
  const fill = async (sel, text) => {
    await page.focus(sel)
    await page.keyboard.down('Control')
    await page.keyboard.press('KeyA')
    await page.keyboard.up('Control')
    await page.type(sel, text)
    const v = await page.$eval(sel, (el) => (el.type === 'password' ? el.value.length : el.value))
    if (v !== (sel.includes('密码') ? text.length : text)) {
      throw new Error(`${sel} 没填对：${JSON.stringify(v)}`)
    }
  }
  await fill('input[placeholder="请输入用户名"]', ADMIN.username)
  await fill('input[placeholder="请输入密码"]', ADMIN.password)
  /* 验证码是「失败 5 次之后才要」，但配置可能被改过 —— 出现了就报出来，
     别装作没看见（那会让后面所有断言都红在一个莫名其妙的地方）。 */
  const captcha = await page.evaluate(
    () => !!document.querySelector('input[placeholder="请输入验证码"]')
  )
  ok('没有要求验证码（open-captcha 阈值内）', !captcha, captcha ? '出现了验证码输入框' : '')
  await clickByText('登 录')
  await page.waitForFunction(() => !location.hash.includes('login'), { timeout: 60000 })
  ok('登录成功，进了后台', true)

  /* ============ ② 菜单里要有「经营看板」 ============ */
  console.log('\n② 菜单')
  /* ★ 先展开「商城运营」再看子项：`el-menu` 的子菜单**没展开时是 display:none**，
     而 `innerText` 不含隐藏元素 —— 直接查会得出「菜单没建」，其实是没展开。 */
  const group = await page.evaluate(() => document.body.innerText.includes('商城运营'))
  ok('「商城运营」这一组在', group)
  await clickByText('商城运营')
  await new Promise((r) => setTimeout(r, 600))
  const menuText = await page.evaluate(() => document.body.innerText)
  ok('★ 展开后能看到「经营看板」', menuText.includes('经营看板'), menuText.slice(0, 80))

  /* ============ ③ 点进去 ============ */
  console.log('\n③ 打开看板')
  await clickByText('经营看板')
  await page.waitForFunction(() => location.hash.includes('dashboard'), { timeout: 20000 })
  await page.waitForFunction(
    () => document.body.innerText.includes('钱包里的积分'),
    { timeout: 20000 }
  )
  ok('★ 页面渲染出来了（不是空白页）', true)

  /* ============ ④ 数字要和接口一致 ============ */
  console.log('\n④ 数字')
  const shown = await page.evaluate(() => {
    /* 卡片结构：.k 是标题、.v 是数字。按标题取数字，比按位置取稳。 */
    const out = {}
    for (const card of document.querySelectorAll('.card')) {
      const k = card.querySelector('.k')?.innerText?.trim() ?? ''
      const v = card.querySelector('.v')?.innerText?.trim() ?? ''
      if (k) out[k] = v
    }
    const facts = {}
    for (const f of document.querySelectorAll('.fact')) {
      const k = f.querySelector('.fk')?.innerText?.trim() ?? ''
      const v = f.querySelector('.fv')?.innerText?.trim() ?? ''
      if (k) facts[k] = v
    }
    return { cards: out, facts }
  })

  const want = overview.data ?? {}
  const num = (s) => Number(String(s).replace(/[^\d-]/g, ''))
  ok(
    '★ 「钱包里的积分」= 接口的 points_in_wallets',
    num(shown.cards['💰钱包里的积分']) === want.points_in_wallets,
    `界面 ${shown.cards['💰钱包里的积分']} / 接口 ${want.points_in_wallets}`
  )
  ok(
    '「账号总数」= 接口的 user_total',
    num(shown.facts['账号总数']) === want.user_total,
    `界面 ${shown.facts['账号总数']} / 接口 ${want.user_total}`
  )
  ok(
    '「游戏总数」= 接口的 game_total',
    num(shown.facts['游戏总数']) === want.game_total,
    `界面 ${shown.facts['游戏总数']} / 接口 ${want.game_total}`
  )

  /* ④' 数字不能全是 0 —— 全 0 通常意味着接口 403 了，而页面把 undefined 显示成 0 */
  const allZero =
    Object.values(shown.cards).every((v) => num(v) === 0) &&
    Object.values(shown.facts).every((v) => num(v) === 0)
  ok('★ 不是「一片 0」（全 0 多半是接口 403 被显示成了 0）', !allZero, JSON.stringify(shown.cards))

  /* ============ ⑤ 刷新按钮真的会重新打接口 ============ */
  console.log('\n⑤ 刷新')
  let calls = 0
  page.on('request', (r) => {
    if (r.url().includes('/mall/dashboard/overview')) calls++
  })
  await clickByText('刷新')
  await new Promise((r) => setTimeout(r, 1500))
  ok('点「刷新」会重新请求看板接口', calls >= 1, `打了 ${calls} 次`)

  ok('没有页面级报错', errors.length === 0, errors.slice(0, 2).join(' | '))

  /* 留一张图给人工看：断言只验数字，验不了「卡片挤在一起 / 字被截断」。
     目录先建好 —— 别指望 screenshot 自己 mkdir（换个环境就是 ENOENT）。 */
  mkdirSync('.e2e-scratch', { recursive: true })
  await page.screenshot({ path: '.e2e-scratch/dashboard.png' })
  console.log('  截图：.e2e-scratch/dashboard.png')
} catch (e) {
  failed++
  console.error('\n✗ 探针中断：', String(e).slice(0, 400))
  try {
    const txt = await page.evaluate(() => document.body.innerText.replace(/\s+/g, ' ').slice(0, 300))
    console.error('  页面文字：', txt)
    await page.screenshot({ path: '.e2e-scratch/dashboard-fail.png', fullPage: false })
    console.error('  截图：.e2e-scratch/dashboard-fail.png')
  } catch {}
} finally {
  await browser.close()
  console.log(failed === 0 ? '\n全部通过' : `\n${failed} 条失败`)
  process.exit(failed === 0 ? 0 : 1)
}
