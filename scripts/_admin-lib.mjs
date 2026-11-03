/**
 * 后台探针的公共件：起浏览器、登录、按文字点、直连接口。
 *
 * 为什么抽出来：后台的每条探针都要「登录 → 展开菜单 → 点进去」这三步，
 * 而这三步各有坑（登录框预填、子菜单悬停展开、clickByText 要点最内层）。
 * 抄成 N 份的话，改一处、其余几份还是老样子 —— 那种不一致最难查。
 *
 * 用法：
 *   import { openAdmin } from './_admin-lib.mjs'
 *   const a = await openAdmin()
 *   await a.clickByText('兑换商城')
 *   a.api('/mall/redeem/item/list', { page: 1, pageSize: 3 })
 *   await a.close()
 */
import { createRequire } from 'node:module'
import { existsSync, mkdirSync } from 'node:fs'
import { pathToFileURL } from 'node:url'
import { resolve } from 'node:path'

const CHROME = [
  'C:/Program Files/Google/Chrome/Application/chrome.exe',
  'C:/Program Files (x86)/Google/Chrome/Application/chrome.exe'
].find((p) => existsSync(p))

/* `puppeteer-core` 不在本工程的依赖里（后台前端不需要它），而 ESM **不认
   NODE_PATH** —— `import 'puppeteer-core'` 直接 ERR_MODULE_NOT_FOUND。
   所以按候选目录用 CJS 解析去找；找不到就报清楚要装在哪。
   ⚠️ `createRequire()` 不会因为目录不存在就抛，必须自己先看包在不在。 */
function loadPuppeteer() {
  const cands = [
    process.env.PUPPETEER_FROM,
    '../kid-quest-farm-online/node_modules',
    './web/node_modules',
    './node_modules'
  ].filter(Boolean)
  const found = cands.find((d) => existsSync(resolve(d, 'puppeteer-core', 'package.json')))
  if (!found) {
    throw new Error(
      '找不到 puppeteer-core。用 PUPPETEER_FROM 指到某个装了它的 node_modules：\n' +
        cands.map((d) => '  ' + resolve(d)).join('\n')
    )
  }
  return createRequire(pathToFileURL(resolve(found, 'index.js')).href)('puppeteer-core')
}

export const ADMIN = {
  username: process.env.TF_ADMIN_USER ?? 'admin',
  password: process.env.TF_ADMIN_PASSWORD ?? 'TfAdmin@2026'
}
export const API = process.env.TF_ADMIN_API ?? 'http://127.0.0.1:8890'

let cachedToken = null

async function adminToken() {
  if (cachedToken) return cachedToken
  const login = await fetch(`${API}/base/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(ADMIN)
  }).then((r) => r.json())
  if (login.code !== 0) throw new Error('后台登录失败：' + login.msg)
  cachedToken = login.data.token
  return cachedToken
}

/**
 * 直连接口（不经过浏览器）：拿 token、核对「服务端真数」都靠它。
 *
 * ★ 方法必须传对：GVA 里读接口有的是 **GET**（`mall/dashboard/overview`、
 *   `mall/user/detail`、`mall/merchant/pending-count`），其余是 POST。
 *   方法错了 Gin 直接走 NoRoute，回的是纯文本 `404 page not found` ——
 *   以前这里无脑 `res.json()`，报错会红在四层之外的
 *   `Unexpected non-whitespace character after JSON at position 4`，
 *   完全看不出是「方法不对」还是「路由不存在」。所以下面先看 content-type，
 *   不是 JSON 就带着 status / 方法 / 路径 / 前 120 字节自己报出来。
 */
export async function adminApi(path, body, method = 'POST') {
  const token = await adminToken()
  const res = await fetch(`${API}${path}`, {
    method,
    headers: { 'Content-Type': 'application/json', 'x-token': token },
    body: method === 'GET' ? undefined : JSON.stringify(body ?? {})
  })
  const text = await res.text()
  if (!/json/i.test(res.headers.get('content-type') ?? '')) {
    throw new Error(
      `${method} ${path} 没回 JSON（status=${res.status}）：${JSON.stringify(text.slice(0, 120))}\n` +
        '  → 方法写错会 404；路由没注册也会 404；权限不足则是 JSON 里的 code=7'
    )
  }
  return JSON.parse(text)
}

/** 读接口的简写（GVA 里读接口是 GET，写接口才是 POST） */
export const adminGet = (path) => adminApi(path, undefined, 'GET')

export async function openAdmin({ url = 'http://localhost:8891/', viewport } = {}) {
  const puppeteer = loadPuppeteer()
  if (!CHROME) throw new Error('找不到 Chrome')
  const browser = await puppeteer.launch({
    executablePath: CHROME,
    headless: 'new',
    args: ['--no-first-run']
  })
  const page = await browser.newPage()
  await page.setViewport(viewport ?? { width: 1440, height: 900 })
  const errors = []
  /* ★ `String(e)` 对 axios 的错只会给出 `AxiosError$1` —— 一个字的线索都没有。
     必须取 message（里面有 status 和 URL），再把失败的请求也记下来，
     否则「有页面报错」这条断言红了等于没红。 */
  const reqFails = []
  /* 只看**本机**的请求：GVA 默认头像指向外部 CDN
     （`qmplusimg.henrongyi.top`），离线环境每次都被 ORB 拦掉 ——
     这种「预期内的失败」混进来，真正的 4xx 就看不见了。 */
  const isLocal = (u) => /^https?:\/\/(localhost|127\.0\.0\.1)(:|\/)/.test(u)
  page.on('pageerror', (e) => {
    const msg = e?.message ?? String(e)
    const where = (e?.stack ?? '').split('\n').find((l) => l.includes('/src/')) ?? ''
    errors.push(`${msg}${where ? ' @ ' + where.trim() : ''}`.slice(0, 300))
  })
  page.on('requestfailed', (r) => {
    if (isLocal(r.url())) reqFails.push(`${r.method()} ${r.url()} — ${r.failure()?.errorText}`)
  })
  page.on('response', (r) => {
    if (r.status() >= 400 && isLocal(r.url())) {
      reqFails.push(`${r.status()} ${r.request().method()} ${r.url()}`)
    }
  })

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
    const want = sel.includes('密码') ? text.length : text
    if (v !== want) throw new Error(`${sel} 没填对：${JSON.stringify(v)}`)
  }

  /**
   * 点一个「文字正好/包含它」的**最内层**元素（找不到就抛，别静默点空）。
   *
   * ★ 一定要取「最内层」：`html` / `body` / 外层 div 的 innerText 里也有这段文字，
   *   拿 `querySelectorAll('*')` 随便挑一个，很可能点的是 `body` 的正中间 ——
   *   什么都不会发生，而断言会红在下一步，完全看不出是这个原因。
   * ★ **精确匹配优先于包含匹配**：「刷新」按钮旁边那句提示
   *   「点右上角刷新重新统计」也含「刷新」，先挑包含匹配就点到提示文字上。
   */
  const clickByText = async (text, selector = 'button, a, li, span, div') => {
    const hit = await page.evaluate(
      (t, sel) => {
        const withRect = [...document.querySelectorAll(sel)].map((el) => ({
          el,
          r: el.getBoundingClientRect()
        }))
        /* ★ 先按「占面积」过滤：`el-tab-pane` 是 v-show 隐藏的，
           隐藏页签里的按钮**照样在 DOM 里、innerText 照样有字**，
           但 rect 是 0×0。不过滤的话会挑到隐藏的那个，
           点了什么也不发生（或者直接报「找不到」）。
           注意不能改用 `offsetParent`：`position:fixed` 的元素的
           offsetParent 恒为 null，会把顶栏按钮全排除掉。 */
        const all = withRect.filter((x) => x.r.width > 0 && x.r.height > 0).map((x) => x.el)
        const match = (el) => {
          const s = (el.innerText || '').trim()
          return s === t || s.includes(t)
        }
        const innermost = (arr) =>
          arr.filter(
            (el) => ![...el.children].some((c) => ((c.innerText || '').trim() || '').includes(t))
          )
        const hits = all.filter(match)
        const exact = innermost(hits.filter((el) => (el.innerText || '').trim() === t))
        const el = exact[0] ?? innermost(hits)[0]
        if (!el) return null
        el.scrollIntoView({ block: 'center' })
        const r = el.getBoundingClientRect()
        if (!r.width || !r.height) return null
        return {
          cx: Math.round(r.x + r.width / 2),
          cy: Math.round(r.y + r.height / 2),
          tag: el.tagName
        }
      },
      text,
      selector
    )
    if (!hit) throw new Error(`找不到「${text}」`)
    /* 先移过去再点：侧边栏子菜单是**悬停展开**的，只 click 不 move 展不开，
       而断言会红成「菜单没建」。 */
    await page.mouse.move(hit.cx, hit.cy)
    await new Promise((r) => setTimeout(r, 300))
    await page.mouse.click(hit.cx, hit.cy)
    return hit
  }

  const login = async () => {
    await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 60000 })
    await page.waitForSelector('input[placeholder="请输入用户名"]', { timeout: 60000 })
    await fill('input[placeholder="请输入用户名"]', ADMIN.username)
    await fill('input[placeholder="请输入密码"]', ADMIN.password)
    const captcha = await page.evaluate(
      () => !!document.querySelector('input[placeholder="请输入验证码"]')
    )
    await clickByText('登 录')
    await page.waitForFunction(() => !location.hash.includes('login'), { timeout: 60000 })
    return { captchaRequired: captcha }
  }

  /** 展开「商城运营」这一组（子菜单没展开时 `display:none`，innerText 看不到） */
  const openMallGroup = async () => {
    await clickByText('商城运营')
    await new Promise((r) => setTimeout(r, 600))
  }

  /**
   * 点开某个 label 的表单项里的下拉框（只找**可见**的表单项）。
   *
   * ★ 必须按 label 定位：同一屏常常有好几个 `placeholder="全部"` 的下拉
   *   （分类 / 状态），按 placeholder 找会点到隔壁那个。
   * ★ 也必须限定「可见」：Element Plus 的 `el-tab-pane` 用 v-show，
   *   隐藏页签里的表单项照样在 DOM 里。
   */
  const openSelectByLabel = async (label) => {
    const hit = await page.evaluate((t) => {
      const items = [...document.querySelectorAll('.el-form-item')].filter(
        (el) => el.offsetParent !== null
      )
      const it = items.find((el) => {
        const l = (el.querySelector('.el-form-item__label')?.innerText || '').trim()
        return l.replace(/[:：]\s*$/, '') === t
      })
      if (!it) return null
      const box = it.querySelector('.el-select__wrapper') ?? it.querySelector('input')
      if (!box) return null
      const r = box.getBoundingClientRect()
      return { cx: Math.round(r.x + r.width / 2), cy: Math.round(r.y + r.height / 2) }
    }, label)
    if (!hit) throw new Error(`找不到「${label}」这个下拉框`)
    await page.mouse.click(hit.cx, hit.cy)
    await new Promise((r) => setTimeout(r, 400))
  }

  /**
   * 在**已经展开的下拉**里选一项。
   *
   * ★★ 绝对不能用 `clickByText` 选下拉项：下拉选项的文字和页面正文里的
   *   文字**经常一模一样** —— 比如筛选「已取消」时，表格里本来就有一堆
   *   「已取消」标签，而标签在 DOM 里排在 body 末尾的 popper **前面**，
   *   于是点到了表格里的标签：什么都不会发生，断言会红成「筛选没生效」，
   *   看着像后端 / 前端筛选逻辑坏了。
   */
  const pickOption = async (text) => {
    const hit = await page.evaluate((t) => {
      const dds = [...document.querySelectorAll('.el-select-dropdown')].filter(
        (d) => d.offsetParent !== null
      )
      for (const dd of dds) {
        const it = [...dd.querySelectorAll('.el-select-dropdown__item')].find(
          (el) => (el.innerText || '').trim() === t
        )
        if (!it) continue
        const r = it.getBoundingClientRect()
        return { cx: Math.round(r.x + r.width / 2), cy: Math.round(r.y + r.height / 2) }
      }
      return null
    }, text)
    if (!hit) throw new Error(`展开的下拉里找不到选项「${text}」`)
    await page.mouse.move(hit.cx, hit.cy)
    await page.mouse.click(hit.cx, hit.cy)
    await new Promise((r) => setTimeout(r, 600))
  }

  const shot = async (name) => {
    mkdirSync('.e2e-scratch', { recursive: true })
    await page.screenshot({ path: `.e2e-scratch/${name}.png` })
    return `.e2e-scratch/${name}.png`
  }

  return {
    browser,
    page,
    errors,
    reqFails,
    fill,
    clickByText,
    openSelectByLabel,
    pickOption,
    login,
    openMallGroup,
    shot,
    close: () => browser.close()
  }
}
