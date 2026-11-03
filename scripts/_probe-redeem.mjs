/**
 * 探针：后台「兑换商城」—— 菜单能看见、两个页签都有数、改价真的落库
 * ------------------------------------------------------------
 * 这条探针要抓的是**兑换商城独有**的那几类静默坏法：
 *   · 「所属」列显示成 `#3`：`uid` 没换成昵称。页面不报错、数字也对，
 *     但运营看到一列 `#3` `#4` 根本不知道是谁的东西 —— 而这一页的全部意义
 *     就是「这是谁家的兑换清单」；
 *   · 库存 `-1` 被原样显示：-1 是「无限」，直接打出来像「欠一个」；
 *   · 改价点保存**什么也没发生**：`item/update` 没登记 casbin ⇒ 返回
 *     `code=7`，而前端只看 `code===0` 才提示成功 —— 不提示、不报错、
 *     弹窗关掉，人以为改成了。所以断言必须是「**接口读回来的价**变了」，
 *     不是「弹窗关了」；
 *   · 「兑换记录」页签是空的：两个页签各打一个接口，只写了一个。
 *
 * 探针会改一条真实商品的价，**测完改回去**（改价前后都断言），
 * 所以可以反复跑。改回去那一步也断言 —— 只测「改成功」不测「改回来」的话，
 * 跑两次就把价格推上去两格。
 *
 * 用法：
 *   1) 后台：tf-admin/server 里的 tf-admin.exe（8890）
 *   2) 前端：tf-admin/web 的 pnpm dev（8891）
 *   3) node scripts/_probe-redeem.mjs [url]
 */
import { adminApi, openAdmin } from './_admin-lib.mjs'

const URL_ = process.argv[2] ?? 'http://localhost:8891/'

let failed = 0
const ok = (label, pass, detail = '') => {
  if (!pass) failed++
  console.log(`  ${pass ? '✓' : '✗'} ${label}${detail ? ` — ${detail}` : ''}`)
}

const num = (s) => Number(String(s ?? '').replace(/[^\d-]/g, ''))

/** 服务端那份「真数」：列表顺序是 id DESC，和界面第一页第一行是同一行 */
const listItems = (pageSize = 10) =>
  adminApi('/mall/redeem/item/list', { page: 1, pageSize })

const first = await listItems()
if (first.code !== 0) {
  console.error('兑换商品接口不通：', first.msg)
  process.exit(2)
}
const apiRow = first.data.list?.[0]
if (!apiRow) {
  console.error('库里一个兑换商品都没有，先把商品建出来再跑这条探针')
  process.exit(2)
}
console.log(
  `接口第一行：${apiRow.emoji} ${apiRow.name} / ${apiRow.price} 券 / 库存 ${apiRow.stock} / uid ${apiRow.uid}\n`
)

const a = await openAdmin({ url: URL_ })
const { page, errors, reqFails } = a

/** 取当前**可见**页签里的表格行（两个页签都在 DOM 里，隐藏的那个也有行） */
const readRows = () =>
  page.evaluate(() => {
    const tables = [...document.querySelectorAll('.el-table')].filter(
      (t) => t.offsetParent !== null
    )
    const t = tables[0]
    if (!t) return []
    return [...t.querySelectorAll('tbody tr.el-table__row')].map((tr) =>
      [...tr.querySelectorAll('td')].map((td) => (td.innerText || '').trim())
    )
  })

/** 按索引设弹窗里第 i 个数字框（0 = 价格、1 = 库存） */
const setNumAt = async (i, value) => {
  const sel = '.el-dialog .el-input-number input'
  const n = await page.$$eval(sel, (els) => els.length)
  if (n <= i) throw new Error(`弹窗里只有 ${n} 个数字框，取不到第 ${i} 个`)
  /* 用 JS 的 focus() + select()，不走键盘 Ctrl+A：弹窗刚出现时焦点还在
     遮罩上，盲打会把数字打进上一个框。 */
  await page.evaluate(
    (s, idx) => {
      const el = document.querySelectorAll(s)[idx]
      el.focus()
      el.select()
    },
    sel,
    i
  )
  await page.keyboard.type(String(value))
  /* el-input-number 在 blur / change 时才提交，不 blur 的话点保存读到的还是旧值 */
  await page.evaluate(
    (s, idx) => document.querySelectorAll(s)[idx].blur(),
    sel,
    i
  )
  await new Promise((r) => setTimeout(r, 200))
  const got = await page.$$eval(sel, (els) => els.map((e) => e.value))
  if (Number(got[i]) !== Number(value)) throw new Error(`第 ${i} 个框没填成 ${value}（是 ${got[i]}）`)
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
  ok('★ 展开「商城运营」后能看到「兑换商城」', menuText.includes('兑换商城'))

  /* ============ ③ 打开页面 ============ */
  console.log('\n③ 打开兑换商城')
  await a.clickByText('兑换商城')
  await page.waitForFunction(() => location.hash.includes('redeem'), { timeout: 20000 })
  await page.waitForFunction(() => document.body.innerText.includes('兑换商品'), {
    timeout: 20000
  })
  ok('★ 页面渲染出来了（不是空白页）', true)

  /* ============ ④ 商品表第一行 == 接口第一行 ============ */
  console.log('\n④ 商品表')
  await page.waitForFunction(() => document.querySelectorAll('tbody tr.el-table__row').length > 0, {
    timeout: 20000
  })
  let rows = await readRows()
  ok('商品表有行', rows.length > 0, `${rows.length} 行`)

  const r0 = rows[0] ?? []
  /* 列顺序：商品 / 所属 / 分类 / 价格 / 库存 / 每天限 / 一次性 / 状态 / 更新于 / 操作 */
  ok('★ 第一行是接口第一行那个商品', (r0[0] ?? '').includes(apiRow.name), r0[0])
  ok(
    '★ 价格 = 接口的 price',
    num(r0[3]) === apiRow.price,
    `界面 ${r0[3]} / 接口 ${apiRow.price}`
  )

  /* 库存 -1 = 无限。直接打 -1 看着像「欠一个」。
     ★ 断言要数**个数**对上，不能只判「界面里有没有『无限』」——
       那样只要有一条 -1 就恒绿，剩下的全错也发现不了。 */
  const apiInfinite = first.data.list.filter((x) => x.stock < 0).length
  const shownInfinite = rows.filter((r) => (r[4] ?? '') === '无限').length
  ok(
    '★ 库存 -1 显示成「无限」（不是 -1）',
    shownInfinite === apiInfinite && !rows.some((r) => (r[4] ?? '') === '-1'),
    `界面 ${shownInfinite} 个无限 / 接口 ${apiInfinite} 个`
  )
  /* 有限库存必须显示成数字，且等于接口 */
  const finite = first.data.list.filter((x) => x.stock >= 0)
  const shownFinite = rows.filter((r) => (r[4] ?? '') !== '无限').map((r) => num(r[4]))
  ok(
    '有限库存显示成数字，和接口一致',
    finite.every((x) => shownFinite.includes(x.stock)),
    `界面 ${shownFinite.join(',')} / 接口 ${finite.map((x) => x.stock).join(',')}`
  )

  /* ============ ⑤ 「所属」必须是昵称，不能是 #uid ============ */
  console.log('\n⑤ 所属列')
  const ownerCells = rows.map((r) => r[1] ?? '')
  const hashed = ownerCells.filter((s) => /^#\d+$/.test(s))
  ok(
    '★ 没有一行显示成 `#uid`（uid 换成昵称了）',
    hashed.length === 0,
    hashed.length ? `有 ${hashed.length} 行是 ${hashed.slice(0, 3).join(' ')}` : ownerCells[0]
  )
  ok('所属列非空', ownerCells.every((s) => s.length > 0))
  console.log('  截图（商品页签）：' + (await a.shot('redeem-items')))

  /* ============ ⑥ 改价 → 落库 → 改回 ============ */
  console.log('\n⑥ 改价（真改真验真改回）')
  const newPrice = apiRow.price + 7

  const openAndSetPrice = async (price) => {
    await a.clickByText('改价 / 库存')
    await page.waitForSelector('.el-dialog input.el-input__inner', { timeout: 20000 })
    await new Promise((r) => setTimeout(r, 400)) // 弹窗有过渡动画，太早点会点空
    /* 第一个是价格、第二个是库存 —— 按顺序取，别按 label 找（label 是中文，容易改） */
    await setNumAt(0, price)
    await a.clickByText('保存')
    await page.waitForFunction(() => !document.querySelector('.el-dialog'), { timeout: 20000 })
    await new Promise((r) => setTimeout(r, 1200)) // 等列表重新拉
  }

  await openAndSetPrice(newPrice)
  const afterEdit = await listItems()
  const apiAfter = afterEdit.data.list.find((x) => x.item_id === apiRow.item_id)
  ok(
    '★ 点「保存」后**接口里的价真的变了**（不是只关了个弹窗）',
    apiAfter?.price === newPrice,
    `接口 ${apiAfter?.price} / 期望 ${newPrice}`
  )

  rows = await readRows()
  const sameRow = rows.find((r) => (r[0] ?? '').includes(apiRow.name)) ?? rows[0] ?? []
  ok('表格刷新后显示新价', num(sameRow[3]) === newPrice, `界面 ${sameRow[3]}`)

  /* 改回去 —— 探针必须能反复跑，否则每跑一次价就涨 7 */
  await openAndSetPrice(apiRow.price)
  const restored = await listItems()
  const apiRestored = restored.data.list.find((x) => x.item_id === apiRow.item_id)
  ok(
    '★ 已改回原价（探针可反复跑）',
    apiRestored?.price === apiRow.price,
    `接口 ${apiRestored?.price} / 原价 ${apiRow.price}`
  )

  /* ============ ⑦ 兑换记录页签 ============ */
  console.log('\n⑦ 兑换记录')
  const recApi = await adminApi('/mall/redeem/record/list', { page: 1, pageSize: 10 })
  ok('记录接口通', recApi.code === 0, recApi.msg)
  await a.clickByText('兑换记录')
  await new Promise((r) => setTimeout(r, 1200))
  const recRows = await readRows()
  ok(
    '★ 切到「兑换记录」后表格有行（两个页签各打一个接口，别只写一个）',
    recRows.length > 0,
    `${recRows.length} 行 / 接口 total ${recApi.data?.total}`
  )
  ok(
    '★ 记录总数 = 接口 total',
    num((await page.evaluate(() => document.body.innerText)).match(/共\s*(\d+)\s*条/)?.[1]) ===
      recApi.data?.total,
    `界面 ${(await page.evaluate(() => document.body.innerText)).match(/共\s*(\d+)\s*条/)?.[0]} / 接口 ${recApi.data?.total}`
  )
  const recOwner = (recRows[0] ?? [])[1] ?? ''
  ok('★ 记录里的「孩子」也是昵称', recOwner.length > 0 && !/^#\d+$/.test(recOwner), recOwner)

  ok(
    '没有页面级报错',
    errors.length === 0,
    [...errors, ...reqFails].slice(0, 3).join(' | ')
  )
  console.log('  截图：' + (await a.shot('redeem')))
} catch (e) {
  failed++
  console.error('\n✗ 探针中断：', String(e).slice(0, 400))
  try {
    const txt = await page.evaluate(() => document.body.innerText.replace(/\s+/g, ' ').slice(0, 300))
    console.error('  页面文字：', txt)
    console.error('  截图：' + (await a.shot('redeem-fail')))
  } catch {}
} finally {
  await a.close()
  console.log(failed === 0 ? '\n全部通过' : `\n${failed} 条失败`)
  process.exit(failed === 0 ? 0 : 1)
}
