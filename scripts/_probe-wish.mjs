/**
 * 探针：后台「心愿商城」—— 菜单能看见、两个页签都有数、口径没串、改外链真落库
 * ------------------------------------------------------------
 * 这一页和「兑换商城」长得像，但**口径正好相反**，所以静默错法也不一样：
 *   · `price_cents` 是**人民币分**（6900 = 69 元），不是券。
 *     照着兑换页的直觉把它当券数显示，会写成「6900 券」—— 不报错、
 *     不越界、就是错的，而且只有人肉看才发现；
 *   · 心愿商品是**全站一份**（没有 uid）。照抄兑换页会长出一个「所属」筛选，
 *     点下去永远筛不到东西（接口不认这个参数）；
 *   · 排序要和 C 端一致（盲盒永远最前）。后台按 id 排的话，
 *     运营在这里调整顺序、孩子在那边看到的却是另一个顺序；
 *   · 订单状态是 wished → committed → confirmed → done，`vouchers_held`
 *     决定了券押在平台还是回了孩子钱包。只显示状态不显示这一列，
 *     对账时会把「已扣券未兑现」的券算成孩子的余额。
 *
 * 探针会改一条真实商品的「外链文字」，**测完改回去**（改前改后都断言），
 * 所以可以反复跑。
 *
 * 用法：
 *   1) 后台：tf-admin/server 里的 tf-admin.exe（8890）
 *   2) 前端：tf-admin/web 的 pnpm dev（8891）
 *   3) node scripts/_probe-wish.mjs [url]
 */
import { adminApi, openAdmin } from './_admin-lib.mjs'

const URL_ = process.argv[2] ?? 'http://localhost:8891/'

let failed = 0
const ok = (label, pass, detail = '') => {
  if (!pass) failed++
  console.log(`  ${pass ? '✓' : '✗'} ${label}${detail ? ` — ${detail}` : ''}`)
}

const num = (s) => Number(String(s ?? '').replace(/[^\d-]/g, ''))

/* 服务端那份「真数」：列表顺序和 C 端一致（盲盒最前，其次 sort_order） */
const itemApi = await adminApi('/mall/wish/item/list', { page: 1, pageSize: 10 })
if (itemApi.code !== 0) {
  console.error('心愿商品接口不通：', itemApi.msg)
  process.exit(2)
}
const apiItems = itemApi.data.list ?? []
if (!apiItems.length) {
  console.error('库里一个心愿商品都没有')
  process.exit(2)
}
console.log(
  `接口第一行：${apiItems[0].emoji} ${apiItems[0].name}（${apiItems[0].kind}，参考价 ${apiItems[0].price_cents} 分）\n`
)

const a = await openAdmin({ url: URL_ })
const { page, errors, reqFails } = a

/** 取当前**可见**页签里的表格行（两个页签都在 DOM 里，隐藏的那个也有行） */
const readRows = () =>
  page.evaluate(() => {
    const t = [...document.querySelectorAll('.el-table')].find((x) => x.offsetParent !== null)
    if (!t) return []
    return [...t.querySelectorAll('tbody tr.el-table__row')].map((tr) =>
      [...tr.querySelectorAll('td')].map((td) => (td.innerText || '').trim())
    )
  })

try {
  /* ============ ① 登录 ============ */
  console.log('① 登录后台')
  const { captchaRequired } = await a.login()
  ok('没有要求验证码', !captchaRequired)

  /* ============ ② 菜单 ============ */
  console.log('\n② 菜单')
  await a.openMallGroup()
  const menuText = await page.evaluate(() => document.body.innerText)
  ok('★ 展开「商城运营」后能看到「心愿商城」', menuText.includes('心愿商城'))

  /* ============ ③ 打开页面 ============ */
  console.log('\n③ 打开心愿商城')
  await a.clickByText('心愿商城')
  await page.waitForFunction(() => location.hash.includes('wish'), { timeout: 20000 })
  await page.waitForFunction(() => document.body.innerText.includes('心愿商品'), { timeout: 20000 })
  ok('★ 页面渲染出来了（不是空白页）', true)
  /* ★ 这一页不该有「所属」筛选：心愿商品是全站一份。有的话就是照抄兑换页抄错了 */
  ok(
    '★ 商品页签没有「所属」筛选（心愿商品是全站一份，不是每人一份）',
    !(await page.evaluate(() => {
      const t = [...document.querySelectorAll('.el-form-item')].filter(
        (el) => el.offsetParent !== null
      )
      return t.some((el) =>
        (el.querySelector('.el-form-item__label')?.innerText || '').includes('所属')
      )
    }))
  )

  /* ============ ④ 商品表口径 ============ */
  console.log('\n④ 商品表')
  await page.waitForFunction(() => document.querySelectorAll('tbody tr.el-table__row').length > 0, {
    timeout: 20000
  })
  let rows = await readRows()
  ok('商品表有行', rows.length > 0, `${rows.length} 行`)

  /* 列顺序：商品 / 分类 / 参考价 / 合作方 / 外链 / 排序 / 状态 / 更新于 / 操作 */
  ok(
    '★ 第一行是接口第一行（盲盒排最前，和 C 端排序一致）',
    (rows[0]?.[0] ?? '').includes(apiItems[0].name),
    rows[0]?.[0]
  )

  /* ★ 参考价的单位：接口是「分」，界面必须是 ¥ 元，不能是券 */
  const priced = apiItems.find((x) => x.price_cents > 0)
  if (priced) {
    const row = rows.find((r) => (r[0] ?? '').includes(priced.name))
    const wantYuan = (priced.price_cents / 100).toFixed(2)
    ok(
      '★ 参考价按「元」显示（接口给的是**分**，不是券）',
      (row?.[2] ?? '').includes(`¥${wantYuan}`),
      `界面 ${row?.[2]} / 接口 ${priced.price_cents} 分 = ¥${wantYuan}`
    )
    ok(
      '★ 参考价没被当成券（不能出现「券」字）',
      !(row?.[2] ?? '').includes('券'),
      row?.[2]
    )
  } else {
    ok('★ 参考价按「元」显示（接口给的是**分**，不是券）', false, '第一页没有带价的商品，无法验证')
  }
  /* 没标价的（游戏类 price_cents=0）要显示「—」，不是 ¥0.00 */
  const free = apiItems.find((x) => x.price_cents === 0)
  if (free) {
    const row = rows.find((r) => (r[0] ?? '').includes(free.name))
    ok('没标价的显示「—」（不是 ¥0.00）', (row?.[2] ?? '') === '—', row?.[2])
  }

  /* ============ ⑤ 改外链文字 → 落库 → 改回 ============ */
  console.log('\n⑤ 改外链（真改真验真改回）')
  const target = apiItems.find((x) => x.buy_url && x.buy_label) ?? apiItems[1] ?? apiItems[0]
  const newLabel = `探针外链${Date.now() % 10000}`

  const openEditAndSetLabel = async (label) => {
    /* 找到目标那一行的「改外链」按钮：按行定位，不能点第一个（那是盲盒） */
    const hit = await page.evaluate((name) => {
      const t = [...document.querySelectorAll('.el-table')].find((x) => x.offsetParent !== null)
      if (!t) return null
      const tr = [...t.querySelectorAll('tbody tr.el-table__row')].find((r) =>
        (r.innerText || '').includes(name)
      )
      if (!tr) return null
      const btn = [...tr.querySelectorAll('button')].find((b) =>
        (b.innerText || '').includes('改外链')
      )
      if (!btn) return null
      const r = btn.getBoundingClientRect()
      return { cx: Math.round(r.x + r.width / 2), cy: Math.round(r.y + r.height / 2) }
    }, target.name)
    if (!hit) throw new Error(`找不到「${target.name}」那一行的「改外链」按钮`)
    await page.mouse.click(hit.cx, hit.cy)
    await page.waitForSelector('.el-dialog input', { timeout: 20000 })
    await new Promise((r) => setTimeout(r, 400))

    /* 弹窗里有好几个输入框（外链、外链文字）和一个数字框（排序）——
       按 label 找，别按索引，否则改了表单顺序就静默填错框。 */
    const realSel = await page.evaluate(() => {
      const items = [...document.querySelectorAll('.el-dialog .el-form-item')]
      const it = items.find((el) =>
        (el.querySelector('.el-form-item__label')?.innerText || '').includes('外链文字')
      )
      if (!it) return ''
      it.setAttribute('data-probe', 'label')
      return '.el-dialog .el-form-item[data-probe="label"] input'
    })
    if (!realSel) throw new Error('弹窗里找不到「外链文字」输入框')

    await page.evaluate((s) => {
      const el = document.querySelector(s)
      el.focus()
      el.select()
    }, realSel)
    await page.keyboard.type(label)
    await page.evaluate((s) => document.querySelector(s).blur(), realSel)
    await new Promise((r) => setTimeout(r, 200))
    const got = await page.$eval(realSel, (el) => el.value)
    if (got !== label) throw new Error(`外链文字没填成 ${label}（是 ${got}）`)

    await a.clickByText('保存')
    await page.waitForFunction(() => !document.querySelector('.el-dialog'), { timeout: 20000 })
    await new Promise((r) => setTimeout(r, 1200))
  }

  const readLabelFromApi = async (itemId) => {
    const r = await adminApi('/mall/wish/item/list', { page: 1, pageSize: 100 })
    return (r.data?.list ?? []).find((x) => x.item_id === itemId)?.buy_label
  }

  await openEditAndSetLabel(newLabel)
  const afterLabel = await readLabelFromApi(target.item_id)
  ok(
    '★ 点「保存」后**接口里的外链文字真的变了**（不是只关了个弹窗）',
    afterLabel === newLabel,
    `接口 ${JSON.stringify(afterLabel)} / 期望 ${JSON.stringify(newLabel)}`
  )

  rows = await readRows()
  const editedRow = rows.find((r) => (r[0] ?? '').includes(target.name)) ?? []
  ok('表格刷新后外链列显示新文字', (editedRow[4] ?? '').includes(newLabel), editedRow[4])

  /* 改回去 —— 探针必须能反复跑 */
  await openEditAndSetLabel(target.buy_label)
  const restored = await readLabelFromApi(target.item_id)
  ok(
    '★ 已改回原文案（探针可反复跑）',
    restored === target.buy_label,
    `接口 ${JSON.stringify(restored)} / 原文 ${JSON.stringify(target.buy_label)}`
  )
  console.log('  截图（商品页签）：' + (await a.shot('wish-items')))

  /* ============ ⑥ 心愿单页签 ============ */
  console.log('\n⑥ 心愿单')
  const ordApi = await adminApi('/mall/wish/order/list', { page: 1, pageSize: 10 })
  ok('心愿单接口通', ordApi.code === 0, ordApi.msg)
  await a.clickByText('心愿单')
  await new Promise((r) => setTimeout(r, 1200))
  let ordRows = await readRows()
  ok(
    '★ 切到「心愿单」后表格有行（两个页签各打一个接口，别只写一个）',
    ordRows.length > 0,
    `${ordRows.length} 行 / 接口 total ${ordApi.data?.total}`
  )
  ok(
    '★ 总数 = 接口 total',
    num((await page.evaluate(() => document.body.innerText)).match(/共\s*(\d+)\s*条/)?.[1]) ===
      ordApi.data?.total,
    `界面 ${(await page.evaluate(() => document.body.innerText)).match(/共\s*\d+\s*条/)?.[0]} / 接口 ${ordApi.data?.total}`
  )

  /* 状态必须是中文，不能把 `done` / `cancelled` 原样打出来 */
  const rawStatus = ordRows.filter((r) => /^(done|cancelled|wished|committed|confirmed)$/.test(r[4] ?? ''))
  ok('★ 状态显示成中文（没把 done/cancelled 原样打出来）', rawStatus.length === 0, ordRows[0]?.[4])

  /* ★ 「券在谁手里」要和接口的 vouchers_held 口径一致：
     committed / confirmed 才算平台押着（= WishStatus.HasHeldVouchers） */
  const apiHeld = (ordApi.data?.list ?? []).filter(
    (o) => o.status === 'committed' || o.status === 'confirmed'
  ).length
  const shownHeld = ordRows.filter((r) => (r[5] ?? '').includes('平台押着')).length
  ok(
    '★ 「券在谁手里」的条数和接口口径一致',
    shownHeld === apiHeld,
    `界面 ${shownHeld} 条押着 / 接口 ${apiHeld} 条`
  )
  ok(
    '★ 券没押着的显示「孩子钱包」',
    ordRows.filter((r) => (r[5] ?? '').includes('孩子钱包')).length === ordRows.length - shownHeld
  )

  /* ============ ⑦ 状态筛选真的会生效 ============ */
  console.log('\n⑦ 状态筛选')
  const cancelApi = await adminApi('/mall/wish/order/list', {
    page: 1,
    pageSize: 10,
    status: 'cancelled'
  })
  /* ★ 这里不能用 clickByText('已取消')：表格里本来就有一堆「已取消」标签，
     而且它们在 DOM 里排在 body 末尾的 popper 前面 ⇒ 点到表格里去了，
     什么都不会发生，断言会红成「筛选没生效」。 */
  await a.openSelectByLabel('状态')
  await a.pickOption('已取消')
  /* ★ 这一页的筛选**不是即时的**：选完只是改了 `ordSearch.status`，
     要再点「查询」才会发请求（GVA 的惯例，和「重置」配对）。
     不点查询的话请求体一个都不会发，断言会红成「筛选没生效」——
     而页面上看着那个下拉已经显示「已取消」了，非常像后端坏了。 */
  await a.clickByText('查询')
  await new Promise((r) => setTimeout(r, 1200))
  ordRows = await readRows()
  const totalShown = num(
    (await page.evaluate(() => document.body.innerText)).match(/共\s*(\d+)\s*条/)?.[1]
  )
  ok(
    '★ 选「已取消」后总数 = 接口 cancelled 的 total',
    totalShown === cancelApi.data?.total,
    `界面 ${totalShown} / 接口 ${cancelApi.data?.total}`
  )
  ok(
    '筛选后每一行都是「已取消」',
    ordRows.length > 0 && ordRows.every((r) => (r[4] ?? '').includes('已取消')),
    ordRows.map((r) => r[4]).join(',')
  )

  ok(
    '没有页面级报错',
    errors.length === 0,
    [...errors, ...reqFails].slice(0, 3).join(' | ')
  )
  console.log('  截图（心愿单）：' + (await a.shot('wish-orders')))
} catch (e) {
  failed++
  console.error('\n✗ 探针中断：', String(e).slice(0, 400))
  try {
    const txt = await page.evaluate(() => document.body.innerText.replace(/\s+/g, ' ').slice(0, 300))
    console.error('  页面文字：', txt)
    console.error('  截图：' + (await a.shot('wish-fail')))
  } catch {}
} finally {
  await a.close()
  console.log(failed === 0 ? '\n全部通过' : `\n${failed} 条失败`)
  process.exit(failed === 0 ? 0 : 1)
}
