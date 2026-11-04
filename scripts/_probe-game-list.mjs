/**
 * 探针：后台「游戏 → 游戏列表」—— 菜单能点开、界面 == 服务端、改排序真落库
 * ------------------------------------------------------------
 * #16 新建的一级菜单 + 新建页面。只跑 build 不够：
 *   · 菜单建了但 component 指错文件 ⇒ 点进去空白页，前端只报 404 chunk；
 *   · 列表接口通、界面却 0 行（分页参数 / 字段名对不上）⇒ 不报错；
 *   · 分类列显示 chill / play / think 原值 ⇒ 运营对不上 App 那边的分类名；
 *   · 改了排序只弹「已保存」，接口读回来没变 ⇒ 和兑换商城踩过的是同一类坑。
 *
 * 探针改一条真实游戏的「排序」，**测完改回去**，两步都断言，
 * 所以可以反复跑（只测改成功不测改回来，跑两次就把排序推上去两格）。
 *
 * 用法：
 *   1) 后台 8890、前端 dev 8891 都在跑
 *   2) node scripts/_probe-game-list.mjs [url]
 */
import { adminApi, openAdmin } from './_admin-lib.mjs'

const URL_ = process.argv[2] ?? 'http://localhost:8891/'

let failed = 0
const ok = (label, pass, detail = '') => {
  if (!pass) failed++
  console.log(`  ${pass ? '✓' : '✗'} ${label}${detail ? ` — ${detail}` : ''}`)
}

const listGames = (body = { page: 1, pageSize: 200 }) => adminApi('/mall/game/list', body)

const before = await listGames()
if (before.code !== 0) {
  console.error('游戏列表接口不通：', before.msg)
  process.exit(1)
}
const total = before.data?.total ?? 0
const rows = before.data?.list ?? []
if (!rows.length) {
  console.error('游戏表里一行都没有，探针没法验渲染')
  process.exit(1)
}
const target = rows[0] // sort DESC 第一条，改完仍稳稳落在第一页
const originSort = target.sort ?? 0

const a = await openAdmin({ url: URL_ })
const { page, errors, reqFails } = a

try {
  /* ============ ① 登录 ============ */
  const { captchaRequired } = await a.login()
  ok('没有要求验证码', !captchaRequired)

  /* ============ ② 菜单结构 ============ */
  console.log('\n② 游戏菜单')
  await a.openGameGroup()
  const menuText = await page.evaluate(() => document.body.innerText)
  ok(
    '★ 展开「游戏」后能看到「游戏列表」「游戏激活码」两个下级',
    menuText.includes('游戏列表') && menuText.includes('游戏激活码'),
    menuText.slice(0, 160).replace(/\s+/g, ' ')
  )

  /* ============ ③ 打开页面、界面 == 服务端 ============ */
  console.log('\n③ 列表渲染')
  await a.clickByText('游戏列表')
  await page.waitForFunction(() => location.hash.includes('game'), { timeout: 20000 })
  await page.waitForFunction(
    () => [...document.querySelectorAll('tbody tr.el-table__row')].length > 0,
    { timeout: 20000 }
  )

  const shell = await page.evaluate(() => ({
    rowCount: [...document.querySelectorAll('tbody tr.el-table__row')].length,
    text: document.body.innerText,
    tableText: [...document.querySelectorAll('tbody tr.el-table__row')]
      .map((r) => r.innerText || '')
      .join('\n')
  }))
  ok('★ 表格真渲染出行（不是空白页 / 404 chunk）', shell.rowCount > 0, `${shell.rowCount} 行`)

  const m = shell.text.match(/共\s*(\d+)\s*款游戏/)
  ok(
    '★「共 N 款游戏」的 N == 接口 total',
    m !== null && Number(m[1]) === total,
    `界面=${m ? m[1] : '没找到'} / 接口=${total}`
  )
  ok('★ 首屏能看到探针要改的那款游戏', shell.tableText.includes(target.game_id), target.game_id)

  /* ============ ④ 分类口径 ============ */
  console.log('\n④ 分类显示')
  const kindCn = ['动脑 · 有产出', '纯玩 · 图个乐', '纯玩 · 不产出']
  ok(
    '★ 分类列用 App 那套中文名（GAME_KINDS），不是原值',
    kindCn.some((k) => shell.tableText.includes(k)),
    shell.tableText.slice(0, 160).replace(/\s+/g, ' ')
  )
  ok('★ 表格里没有裸露的 kind 原值', !/\b(chill|play|think)\b/.test(shell.tableText))

  /* ============ ⑤ 编辑弹窗可达 ============ */
  console.log('\n⑤ 编辑弹窗')
  /* ★ v-loading 的遮罩有离场动画：表格行已经渲染出来了，遮罩还盖在上面，
     elementFromPoint 命中的是 .el-loading-mask 而不是「调整」按钮 ⇒
     点下去什么都没发生、页面也不报错（实测过一次）。
     ★ 判「点得到」只能用 elementFromPoint，不能看 rect 有没有尺寸。 */
  await page.waitForFunction(() => {
    const btn = [...document.querySelectorAll('button')].find(
      (b) => (b.innerText || '').trim() === '调整'
    )
    if (!btn) return false
    const r = btn.getBoundingClientRect()
    const hit = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2)
    return !!hit && (btn === hit || btn.contains(hit))
  }, { timeout: 20000 })
  let dlgOpen = false
  for (let i = 0; i < 4 && !dlgOpen; i++) {
    await a.clickByText('调整')
    await new Promise((r) => setTimeout(r, 500))
    dlgOpen = (await page.evaluate(() => document.querySelectorAll('.el-dialog').length)) > 0
  }
  await page.waitForSelector('.el-dialog', { timeout: 20000 })
  const dlg = await page.evaluate(() => document.querySelector('.el-dialog')?.innerText ?? '')
  ok(
    '★ 弹窗列出可改字段（积分 / 钻石定价、默认拥有、状态、排序）',
    ['积分定价', '钻石定价', '默认拥有', '状态', '排序'].every((k) => dlg.includes(k)),
    dlg.slice(0, 160).replace(/\s+/g, ' ')
  )
  await a.clickByText('取消')
  await new Promise((r) => setTimeout(r, 400))

  /* ============ ⑥ 改排序：落库 → 界面读到 → 改回来 ============ */
  console.log('\n⑥ 改排序真的落库')
  const newSort = originSort + 7
  const up = await adminApi('/mall/game/update', { game_id: target.game_id, sort: newSort })
  ok('★ /mall/game/update 返回成功', up.code === 0, up.msg)

  const after = await listGames()
  const changed = (after.data?.list ?? []).find((x) => x.game_id === target.game_id)
  ok(
    '★ 接口读回 sort 已变（不是只弹了个「已保存」）',
    changed?.sort === newSort,
    `期望 ${newSort} / 实得 ${changed?.sort}`
  )

  await a.clickByText('查询')
  await page.waitForFunction(
    (g, s) => {
      const tr = [...document.querySelectorAll('tbody tr.el-table__row')].find((r) =>
        (r.innerText || '').includes(g)
      )
      if (!tr) return false
      return [...tr.querySelectorAll('td')].some(
        (c) => (c.innerText || '').trim() === String(s)
      )
    },
    { timeout: 20000 },
    target.game_id,
    newSort
  )
  ok('★ 刷新后界面也读到新排序（界面 == 服务端）', true, `${target.game_id} → ${newSort}`)

  const back = await adminApi('/mall/game/update', { game_id: target.game_id, sort: originSort })
  ok('★ 改回原排序成功', back.code === 0, back.msg)
  const restored = await listGames()
  ok(
    '★ 原排序已恢复（反复跑不会把排序一路推上去）',
    (restored.data?.list ?? []).find((x) => x.game_id === target.game_id)?.sort === originSort
  )

  ok(
    '没有页面级报错',
    errors.length === 0,
    [...errors, ...reqFails].slice(0, 3).join(' | ')
  )
  console.log('  截图：' + (await a.shot('game-list')))
} catch (e) {
  failed++
  console.error('\n✗ 探针中断：', String(e).slice(0, 400))
  try {
    const txt = await page.evaluate(() =>
      document.body.innerText.replace(/\s+/g, ' ').slice(0, 300)
    )
    console.error('  页面文字：', txt)
    console.error('  截图：' + (await a.shot('game-list-fail')))
  } catch {}
} finally {
  await a.close()
  console.log(failed === 0 ? '\n全部通过' : `\n${failed} 条失败`)
  process.exit(failed === 0 ? 0 : 1)
}
