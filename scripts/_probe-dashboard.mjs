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
import { adminApi, adminGet, openAdmin } from './_admin-lib.mjs'

const URL_ = process.argv[2] ?? 'http://localhost:8891/'

let failed = 0
const ok = (label, pass, detail = '') => {
  if (!pass) failed++
  console.log(`  ${pass ? '✓' : '✗'} ${label}${detail ? ` — ${detail}` : ''}`)
}

/* 服务端那份「真数」，一会儿和界面比。★ 看板是 **GET** 接口。 */
const overview = await adminGet('/mall/dashboard/overview')
if (overview.code !== 0) {
  console.error('看板接口不通：', overview.msg)
  process.exit(2)
}
console.log(`接口真数：${JSON.stringify(overview.data)}\n`)

const a = await openAdmin({ url: URL_ })
const { page, errors, reqFails } = a

try {
  /* ============ ① 登录 ============ */
  console.log('① 登录后台')
  const { captchaRequired } = await a.login()
  ok('没有要求验证码（open-captcha 阈值内）', !captchaRequired)
  ok('登录成功，进了后台', true)

  /* ============ ② 菜单 ============ */
  console.log('\n② 菜单')
  const group = await page.evaluate(() => document.body.innerText.includes('商城运营'))
  ok('「商城运营」这一组在', group)
  await a.openMallGroup()
  const menuText = await page.evaluate(() => document.body.innerText)
  ok('★ 展开后能看到「经营看板」', menuText.includes('经营看板'))

  /* ============ ③ 点进去 ============ */
  console.log('\n③ 打开看板')
  await a.clickByText('经营看板')
  /* ★ 要等 `mall/dashboard`，不能只等 `dashboard`：登录后落地页的 hash 本来就是
     `#/layout/dashboard`，条件立刻满足 ⇒ 等于没等。 */
  await page.waitForFunction(() => location.hash.includes('mall/dashboard'), { timeout: 20000 })
  await page.waitForFunction(() => document.body.innerText.includes('钱包里的积分'), {
    timeout: 20000
  })
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

  /* ============ ④'' 两个「待处理」数和心愿单列表对账 ============
     ★ 为什么单独对一次账：看板那两个数是**另一条代码路径**算的
       （`Overview` 里两句 Count），和 `wish/order/list` 的筛选完全独立。
       原来那两句写的是 `_ = ....Error`，把错误吞掉了 —— 列名一改，
       看板永远显示 0，而 0 看起来像「今天没有待办」，没人会去查。
       所以判据必须是「两条路径算出来一样」，不能是「界面显示了个数」。 */
  console.log('\n④′ 待处理数对账')
  const countWish = async (status) => {
    const r = await adminApi('/mall/wish/order/list', { page: 1, pageSize: 1, status })
    return r.code === 0 ? r.data.total : NaN
  }
  const [wished, committed, confirmed] = await Promise.all([
    countWish('wished'),
    countWish('committed'),
    countWish('confirmed')
  ])
  ok(
    '★ 「心愿待处理·家长没处理」= wish/order/list?status=wished 的 total',
    want.wish_pending_wished === wished,
    `看板 ${want.wish_pending_wished} / 列表 ${wished}`
  )
  ok(
    '★ 「心愿待处理·已扣券待兑现」= committed + confirmed 的 total',
    want.wish_pending_action === committed + confirmed,
    `看板 ${want.wish_pending_action} / 列表 ${committed}+${confirmed}=${committed + confirmed}`
  )

  /* ============ ⑤ 刷新按钮真的会重新打接口 ============ */
  console.log('\n⑤ 刷新')
  let calls = 0
  page.on('request', (r) => {
    /* 只数 GET（看板是 GET）。把预检 OPTIONS 也数进去的话，
       同一次点击有时 1、有时 2，看着像多打了一轮。 */
    if (r.method() === 'GET' && r.url().includes('/mall/dashboard/overview')) calls++
  })
  await a.clickByText('刷新')
  await new Promise((r) => setTimeout(r, 1500))
  ok('点「刷新」会重新请求看板接口', calls >= 1, `打了 ${calls} 次`)

  ok(
    '没有页面级报错',
    errors.length === 0,
    [...errors, ...reqFails].slice(0, 3).join(' | ')
  )
  console.log('  截图：' + (await a.shot('dashboard')))
} catch (e) {
  failed++
  console.error('\n✗ 探针中断：', String(e).slice(0, 400))
  try {
    const txt = await page.evaluate(() => document.body.innerText.replace(/\s+/g, ' ').slice(0, 300))
    console.error('  页面文字：', txt)
    console.error('  截图：' + (await a.shot('dashboard-fail')))
  } catch {}
} finally {
  await a.close()
  console.log(failed === 0 ? '\n全部通过' : `\n${failed} 条失败`)
  process.exit(failed === 0 ? 0 : 1)
}
