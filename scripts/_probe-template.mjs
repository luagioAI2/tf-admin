/**
 * 探针：后台「模板配置 → 任务模板 / 兑换模板」
 * ------------------------------------------------------------
 * 这一页是新用户首次启动铺的内容（每日任务 / 签到 / 长期任务 / 兑换品）。
 * 只跑 build 不够，四种「静默坏」都只在界面上看得出来：
 *
 *   1. 菜单建了但 component 指错文件 ⇒ 点进去空白页，前端只报 404 chunk；
 *   2. 列表接口通、界面 0 行（分页参数 / 字段名对不上）⇒ 不报错；
 *   3. 「条目数」那一列和详情里的条数对不上 ⇒ 运营看着 16 条、点开只有 3 条；
 *   4. ★ 条目保存是**整条覆盖**：表单里没放的玩法参数（checkInEnabled /
 *      checkInTargetCount / rewardItemIds）如果没带回去，会静默消失，
 *      而保存还报成功 —— 所以这里改完分值要回读 extra。
 *
 * 改的是「练字签到」的基础积分（+1 → 回读 → 改回），两步都断言，
 * 可以反复跑。
 *
 * 用法：
 *   1) 后台 8890、前端 dev 8891 都在跑
 *   2) node scripts/_probe-template.mjs [url]
 */
import { adminApi, openAdmin } from './_admin-lib.mjs'

const URL_ = process.argv[2] ?? 'http://localhost:8891/'

let failed = 0
const ok = (label, pass, detail = '') => {
  if (!pass) failed++
  console.log(`  ${pass ? '✓' : '✗'} ${label}${detail ? ` — ${detail}` : ''}`)
}
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

const listTpl = (kind) => adminApi('/mall/template/list', { page: 1, pageSize: 100, kind })
const detailTpl = (id) => adminApi('/mall/template/detail', { id })

const taskList = await listTpl('task')
const redeemList = await listTpl('redeem')
if (taskList.code !== 0 || redeemList.code !== 0) {
  console.error('模板列表接口不通：', taskList.msg || redeemList.msg)
  process.exit(1)
}
const taskTpl = (taskList.data?.list ?? [])[0]
const redeemTpl = (redeemList.data?.list ?? [])[0]
if (!taskTpl || !redeemTpl) {
  console.error('库里没有官方模板，先跑一次平台端种子（seed.EnsureTemplates）')
  process.exit(1)
}
const taskDetail = await detailTpl(taskTpl.id)
const redeemDetail = await detailTpl(redeemTpl.id)
const taskItems = taskDetail.data?.items ?? []
const redeemItems = redeemDetail.data?.items ?? []

/* 找一条「带签到参数」的任务：改它才能证明 extra 没被表单吞掉 */
const victim = taskItems.find((it) => JSON.parse(it.extra || '{}').checkInEnabled === true)
if (!victim) {
  console.error('官方任务模板里没有带 checkInEnabled 的条目，探针失去目标')
  process.exit(1)
}
const victimExtra = JSON.parse(victim.extra || '{}')
const originPoints = victim.base_points

const a = await openAdmin({ url: URL_ })
const { page, errors, reqFails } = a

/** 页签和左侧菜单同名（「任务模板」「兑换模板」两边都有），
    只能点 `.el-tabs__item`，不能按文字全页找。 */
const clickTab = async (label) => {
  const hit = await page.evaluate((t) => {
    const el = [...document.querySelectorAll('.el-tabs__item')].find(
      (x) => (x.innerText || '').trim() === t
    )
    if (!el) return false
    el.click()
    return true
  }, label)
  if (!hit) throw new Error(`点不到页签「${label}」`)
  await sleep(700)
}

/** ★ 布局里还有一个「系统配置」抽屉常驻 DOM ⇒ `querySelector('.el-drawer')`
    拿到的是它（里面只有「重置配置」），不是模板这一屏。
    按「里面有表格行」认。 */
const DRAWER = `[...document.querySelectorAll('.el-drawer')].find(d => d.querySelector('tbody tr.el-table__row'))`

const drawerRows = () =>
  page.evaluate(`document.querySelectorAll('.el-drawer tbody tr.el-table__row').length`)
const drawerText = () =>
  page.evaluate(`(${DRAWER})?.innerText ?? ''`)

/** 点某一行里的某个按钮（按行内文字定位，避免点错行） */
const clickRowBtn = async (rowText, btnText) =>
  page.evaluate(
    (rt, bt) => {
      const tr = [...document.querySelectorAll('tbody tr.el-table__row')].find((r) =>
        (r.innerText || '').includes(rt)
      )
      if (!tr) return false
      const btn = [...tr.querySelectorAll('button')].find(
        (b) => (b.innerText || '').trim() === bt
      )
      if (!btn) return false
      btn.click()
      return true
    },
    rowText,
    btnText
  )

/** 弹窗里某个 label 的 input 当前值 */
const dialogInputValue = (label) =>
  page.evaluate((l) => {
    const item = [...document.querySelectorAll('.el-dialog .el-form-item')].find(
      (fi) => (fi.querySelector('.el-form-item__label')?.innerText || '').trim() === l
    )
    return item?.querySelector('input')?.value ?? null
  }, label)

try {
  /* ============ ① 登录 ============ */
  const { captchaRequired } = await a.login()
  ok('没有要求验证码', !captchaRequired)

  /* ============ ② 菜单 ============ */
  console.log('\n② 模板配置菜单')
  await a.clickByText('模板配置')
  await sleep(600)
  const menuText = await page.evaluate(() => document.body.innerText)
  ok(
    '★ 展开「模板配置」后能看到「任务模板」「兑换模板」两个下级',
    menuText.includes('任务模板') && menuText.includes('兑换模板')
  )

  /* ============ ③ 任务模板列表 == 服务端 ============ */
  console.log('\n③ 任务模板列表')
  await a.clickByText('任务模板')
  await page.waitForFunction(() => location.hash.includes('template'), { timeout: 20000 })
  await page.waitForFunction(
    () => document.querySelectorAll('tbody tr.el-table__row').length > 0,
    { timeout: 20000 }
  )

  const shell = await page.evaluate(() => ({
    rows: [...document.querySelectorAll('tbody tr.el-table__row')].map((r) => r.innerText || ''),
    text: document.body.innerText,
    names: [...document.querySelectorAll('[data-probe="tpl-name"]')].map((e) => e.innerText.trim()),
    counts: [...document.querySelectorAll('[data-probe="tpl-count"]')].map((e) =>
      Number(e.innerText.trim())
    )
  }))
  ok('★ 表格真渲染出行（不是空白页 / 404 chunk）', shell.rows.length > 0, `${shell.rows.length} 行`)
  ok(
    '★ 行数 == 接口 total',
    shell.rows.length === (taskList.data?.total ?? 0),
    `界面=${shell.rows.length} / 接口=${taskList.data?.total}`
  )
  ok(
    '★ 第一份就是接口里的第一份（排序口径一致）',
    shell.names[0] === taskTpl.name,
    `界面=${shell.names[0]} / 接口=${taskTpl.name}`
  )
  ok(
    '★「条目数」列 == 详情里的条数（不是 0、也不是总数）',
    shell.counts[0] === taskItems.length,
    `界面=${shell.counts[0]} / 详情=${taskItems.length}`
  )
  ok(
    '官方模板标了「官方」，不是裸露的 official',
    shell.rows[0].includes('官方') && !shell.rows[0].includes('official'),
    shell.rows[0].replace(/\s+/g, ' ').slice(0, 120)
  )
  ok('默认模板标了「默认」', shell.rows[0].includes('默认'))

  /* ============ ④ 条目抽屉 == 详情 ============ */
  console.log('\n④ 条目抽屉')
  ok('点得到「编辑内容」', await clickRowBtn(taskTpl.name, '编辑内容'))
  await page.waitForFunction(
    () => document.querySelectorAll('.el-drawer tbody tr.el-table__row').length > 0,
    { timeout: 20000 }
  )
  const drawer = {
    rows: await drawerRows(),
    text: await drawerText()
  }
  ok(
    '★ 抽屉里的行数 == 详情条数（16 条任务一条不少）',
    drawer.rows === taskItems.length,
    `抽屉=${drawer.rows} / 详情=${taskItems.length}`
  )
  ok('能看到签到那条（练字签到）', drawer.text.includes('练字签到'))
  ok(
    '★ 签到列显示「打卡 5 次」而不是空白（extra 里的 checkInTargetCount 读出来了）',
    /打卡\s*5\s*次/.test(drawer.text),
    drawer.text.replace(/\s+/g, ' ').slice(0, 160)
  )
  ok(
    '分类 / 周期都译成中文（没有裸露的 study / weekly）',
    !/\b(study|chore|art|reading|habit|weekly|daily)\b/.test(drawer.text)
  )

  /* ============ ⑤ 改一条条目：落库 + extra 不丢 ============ */
  console.log('\n⑤ 改基础积分（同时验 extra 不被表单吞掉）')
  ok('点得到「练字签到」那一行的编辑', await clickRowBtn('练字签到', '编辑'))
  await page.waitForFunction(
    () => document.querySelectorAll('.el-dialog').length > 0,
    { timeout: 20000 }
  )
  await sleep(400)

  /* 先点 input-number 的加号；★ 点完必须**读回 input 的值**才知道有没有生效
     （按钮在，但点不动的情况真的存在）。没变就退化成键盘输入。 */
  await page.evaluate(() => {
    const item = [...document.querySelectorAll('.el-dialog .el-form-item')].find(
      (fi) => (fi.querySelector('.el-form-item__label')?.innerText || '').trim() === '基础积分'
    )
    item?.querySelector('.el-input-number__increase')?.click()
  })
  await sleep(300)
  let shown = await dialogInputValue('基础积分')
  if (Number(shown) !== originPoints + 1) {
    /* 键盘输入：全选 → 覆盖 → Tab 触发 change。
       ★ 直接改 `input.value` 不触发 Vue 的 v-model，保存时还是旧值。 */
    const box = await page.evaluate(() => {
      const item = [...document.querySelectorAll('.el-dialog .el-form-item')].find(
        (fi) => (fi.querySelector('.el-form-item__label')?.innerText || '').trim() === '基础积分'
      )
      const r = item?.querySelector('input')?.getBoundingClientRect()
      return r ? { x: r.x + r.width / 2, y: r.y + r.height / 2 } : null
    })
    if (!box) throw new Error('找不到「基础积分」输入框')
    await page.mouse.click(box.x, box.y)
    await page.keyboard.down('Control')
    await page.keyboard.press('KeyA')
    await page.keyboard.up('Control')
    await page.keyboard.type(String(originPoints + 1))
    await page.keyboard.press('Tab')
    await sleep(300)
    shown = await dialogInputValue('基础积分')
  }
  ok(
    '★ 弹窗里的分值真的变成 +1（不是按钮点了但没生效）',
    Number(shown) === originPoints + 1,
    `框里=${shown} / 期望=${originPoints + 1}`
  )

  await a.clickByText('保存')
  await page.waitForFunction(
    () => document.querySelectorAll('.el-dialog').length === 0,
    { timeout: 20000 }
  )
  await sleep(800)

  const afterSave = await detailTpl(taskTpl.id)
  const saved = (afterSave.data?.items ?? []).find((it) => it.id === victim.id)
  ok(
    '★ 接口读回：基础积分 +1（不是只弹了个「已保存」）',
    saved?.base_points === originPoints + 1,
    `期望 ${originPoints + 1} / 实得 ${saved?.base_points}`
  )
  const savedExtra = JSON.parse(saved?.extra || '{}')
  ok(
    '★★ 玩法参数没被吞（checkInEnabled / checkInTargetCount 都还在）',
    savedExtra.checkInEnabled === true && Number(savedExtra.checkInTargetCount) === Number(victimExtra.checkInTargetCount),
    JSON.stringify(savedExtra)
  )
  ok(
    '★★ 其余参数也原样保留（remindAt / rewardItemIds / allowOvertime）',
    savedExtra.allowOvertime === victimExtra.allowOvertime &&
      savedExtra.remindAt === victimExtra.remindAt,
    JSON.stringify(savedExtra)
  )

  /* 抽屉里那一行要跟着变 —— 不刷新抽屉的话运营会以为没保存上 */
  await page.waitForFunction(
    (name, v) => {
      const tr = [...document.querySelectorAll('.el-drawer tbody tr.el-table__row')].find((r) =>
        (r.innerText || '').includes(name)
      )
      if (!tr) return false
      /* ⚠️ 单元格里是「6」+「分」两个 span ⇒ innerText 是「6分」，
         不能和 "6" 全等比较（那样永远等不到，报成「界面没刷新」）。 */
      return [...tr.querySelectorAll('td')].some((c) =>
        (c.innerText || '').trim().startsWith(String(v))
      )
    },
    { timeout: 20000 },
    '练字签到',
    originPoints + 1
  )
  ok('★ 抽屉里也读到了新分值（界面 == 服务端）', true, `${originPoints} → ${originPoints + 1}`)

  /* 改回去：反复跑不会把积分一路推上去 */
  const restore = await adminApi('/mall/template/item/save', {
    id: victim.id,
    template_id: taskTpl.id,
    item_id: victim.item_id,
    name: victim.name,
    emoji: victim.emoji,
    category: victim.category,
    cycle: victim.cycle,
    planned_minutes: victim.planned_minutes,
    base_points: originPoints,
    price: victim.price,
    limit_per_day: victim.limit_per_day,
    stock: victim.stock,
    once: victim.once,
    note: victim.note,
    sort: victim.sort,
    extra: victim.extra
  })
  ok('★ 改回原分值成功', restore.code === 0, restore.msg)
  const back = await detailTpl(taskTpl.id)
  const restored = (back.data?.items ?? []).find((it) => it.id === victim.id)
  ok('★ 原分值已恢复', restored?.base_points === originPoints, String(restored?.base_points))
  ok(
    '★ 恢复后 extra 仍与原来一致',
    JSON.stringify(JSON.parse(restored?.extra || '{}')) === JSON.stringify(victimExtra)
  )

  /* ============ ⑤·2 原样保存一条「签到关着但留着目标次数」的任务 ============ */
  /* 官方种子里「背诵」是 `checkInEnabled:false` + `checkInTargetCount:2`。
     运营只想改个积分、连弹窗里的开关都没碰，保存后目标次数**不该消失** ——
     这是「整条覆盖」最容易顺手做坏的一处。 */
  const keep = taskItems.find((it) => {
    const e = JSON.parse(it.extra || '{}')
    return e.checkInEnabled === false && Number(e.checkInTargetCount) > 0
  })
  if (keep) {
    const before = JSON.parse(keep.extra || '{}')
    ok(`点得到「${keep.name}」那一行的编辑`, await clickRowBtn(keep.name, '编辑'))
    await page.waitForFunction(() => document.querySelectorAll('.el-dialog').length > 0, {
      timeout: 20000
    })
    await sleep(400)
    await a.clickByText('保存')
    await page.waitForFunction(() => document.querySelectorAll('.el-dialog').length === 0, {
      timeout: 20000
    })
    await sleep(600)
    const d2 = await detailTpl(taskTpl.id)
    const kept = (d2.data?.items ?? []).find((it) => it.id === keep.id)
    ok(
      '★★ 什么都没改直接保存：extra 逐键不变（签到关着也要留着目标次数）',
      JSON.stringify(JSON.parse(kept?.extra || '{}')) === JSON.stringify(before),
      `前=${JSON.stringify(before)} 后=${kept?.extra}`
    )
    ok('★ 分值和时长也没被清掉', kept?.base_points === keep.base_points && kept?.planned_minutes === keep.planned_minutes)
  } else {
    console.log('  · 跳过 ⑤·2：官方模板里没有「签到关着但留着目标次数」的条目')
  }

  /* ============ ⑥ 兑换模板页签 ============ */
  console.log('\n⑥ 兑换模板页签')
  /* ★ 抽屉还开着：不关掉的话页签虽然在背后切了，读到的还是上一屏，
     而且「编辑内容」会点到底下那张表 —— 看着像页签没生效。 */
  await page.evaluate(() => document.querySelector('.el-drawer__close-btn')?.click())
  await sleep(600)
  await clickTab('兑换模板')
  await page.waitForFunction(
    (n) => document.querySelectorAll('tbody tr.el-table__row').length === n,
    { timeout: 20000 },
    (redeemList.data?.list ?? []).length
  )
  const rd = await page.evaluate(() => ({
    names: [...document.querySelectorAll('[data-probe="tpl-name"]')].map((e) => e.innerText.trim()),
    counts: [...document.querySelectorAll('[data-probe="tpl-count"]')].map((e) =>
      Number(e.innerText.trim())
    )
  }))
  ok(
    '★ 切到兑换模板：第一份 == 接口里的第一份',
    rd.names[0] === redeemTpl.name,
    `界面=${rd.names[0]} / 接口=${redeemTpl.name}`
  )
  ok(
    '★ 兑换模板的条目数 == 详情条数（15 条兑换品）',
    rd.counts[0] === redeemItems.length,
    `界面=${rd.counts[0]} / 详情=${redeemItems.length}`
  )

  ok('点得到「编辑内容」', await clickRowBtn(redeemTpl.name, '编辑内容'))
  await page.waitForFunction(
    () => document.querySelectorAll('.el-drawer tbody tr.el-table__row').length > 0,
    { timeout: 20000 }
  )
  const rdrawer = { rows: await drawerRows(), text: await drawerText() }
  ok('★ 兑换抽屉行数 == 详情条数', rdrawer.rows === redeemItems.length, `${rdrawer.rows}`)
  ok('能看到「一份小零食」', rdrawer.text.includes('一份小零食'))
  ok(
    '价格列带「券」单位（30 券），不是裸数字',
    /30\s*券/.test(rdrawer.text),
    rdrawer.text.replace(/\s+/g, ' ').slice(0, 160)
  )

  ok(
    '没有页面级报错',
    errors.length === 0,
    [...errors, ...reqFails].slice(0, 3).join(' | ')
  )
  console.log('  截图：' + (await a.shot('template-page')))
} catch (e) {
  failed++
  console.error('\n✗ 探针中断：', String(e).slice(0, 400))
  try {
    const txt = await page.evaluate(() =>
      document.body.innerText.replace(/\s+/g, ' ').slice(0, 300)
    )
    console.error('  页面文字：', txt)
    console.error('  截图：' + (await a.shot('template-page-fail')))
  } catch {}
} finally {
  await a.close()
  console.log(failed === 0 ? '\n全部通过' : `\n${failed} 条失败`)
  process.exit(failed === 0 ? 0 : 1)
}
