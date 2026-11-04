/**
 * 探针：后台「心愿商城 → 激活码」—— 生成 / 明文只此一次 / 列表 / 作废
 * ------------------------------------------------------------
 * 这一页最要紧的不是「有没有按钮」，而是**明文只出现一次**这件事有没有做到：
 *
 *   · 表里只存 sha256。生成接口把明文返回给运营去印卡片，**之后再也拿不回来**。
 *     所以结果面板必须：① 当场把码列出来；② 有「复制全部」；③ 关之前拦一下。
 *     少任何一条，运营点一下叉号就丢了一整批码 —— 只能作废重发，
 *     而界面上不会有任何报错。
 *   · 明文码必须**原样显示 6 位**：加分隔符 / 小写 / 截断都会让印出来的卡
 *     和数据库里的哈希对不上（三处 normalize 只认 Crockford Base32 那 32 个字符）。
 *   · 只能选「游戏」分类的商品：挂在实物或盲盒上，核销会「成功」却什么都不解锁。
 *
 * 探针会**真的生成一批码**（批次名带 probe 前缀），结束时把没用过的整批作废，
 * 所以可以反复跑。已经用掉的码不能作废，所以它不会污染核销记录。
 *
 * 用法：
 *   1) 后台：tf-admin/server 里的 tf-admin.exe（8890）
 *   2) 前端：tf-admin/web 的 dev/preview（8891）
 *   3) node scripts/_probe-wishcode-ui.mjs [url]
 */
import { adminApi, openAdmin } from './_admin-lib.mjs'

const URL_ = process.argv[2] ?? 'http://localhost:8891/'
const BATCH = `PROBEUI-${Date.now().toString(36).toUpperCase()}`
const ALPHABET = '0123456789ABCDEFGHJKMNPQRSTVWXYZ'
const COUNT = 2

let failed = 0
const ok = (label, pass, detail = '') => {
  if (!pass) failed++
  console.log(`  ${pass ? '✓' : '✗'} ${label}${detail ? ` — ${detail}` : ''}`)
}

const a = await openAdmin({ url: URL_ })
const { page, errors } = a

/** 按 form-item 的 label 找输入框（不按索引：表单顺序一变就静默填错框） */
const inputByLabel = async (label) => {
  const sel = await page.evaluate((t) => {
    /* ★ 上一轮留下的标记必须先清掉。
       两次调用都把 `data-probe="in"` 挂在各自的 form-item 上时，
       `document.querySelector` 只返回**第一个** ⇒ 第二次的输入被写进
       第一个字段，读回来的却是 ""（「数量」是 el-input-number，
       非数字会被它自己抹掉），报错看着像「批次号填不进去」。
       判据：错误信息里「是 」后面是空的。 */
    document.querySelectorAll('[data-probe]').forEach((el) => el.removeAttribute('data-probe'))
    const items = [...document.querySelectorAll('.el-dialog .el-form-item')].filter(
      (el) => el.offsetParent !== null
    )
    const it = items.find(
      (el) => (el.querySelector('.el-form-item__label')?.innerText || '').trim().replace(/[:：]\s*$/, '') === t
    )
    if (!it) return ''
    const inp = it.querySelector('input')
    if (!inp) return ''
    it.setAttribute('data-probe', 'in')
    return '.el-dialog .el-form-item[data-probe="in"] input'
  }, label)
  return sel
}

const setInput = async (label, text) => {
  const sel = await inputByLabel(label)
  if (!sel) throw new Error(`弹窗里找不到「${label}」输入框`)
  await page.evaluate((s) => {
    const el = document.querySelector(s)
    el.focus()
    el.select()
  }, sel)
  await page.keyboard.type(String(text))
  await page.evaluate((s) => document.querySelector(s).blur(), sel)
  await new Promise((r) => setTimeout(r, 200))
  const got = await page.$eval(sel, (el) => el.value)
  if (got !== String(text)) throw new Error(`「${label}」没填成 ${text}（是 ${got}）`)
}

/** 当前可见表格的行 */
const readRows = () =>
  page.evaluate(() => {
    const t = [...document.querySelectorAll('.el-table')].find((x) => x.offsetParent !== null)
    if (!t) return []
    return [...t.querySelectorAll('tbody tr.el-table__row')].map((tr) =>
      [...tr.querySelectorAll('td')].map((td) => (td.innerText || '').trim())
    )
  })

/** 生成结果面板里的明文码 */
const readCodes = () =>
  page.evaluate(() =>
    [...document.querySelectorAll('.codebox .codeline')].map((el) => {
      const spans = el.querySelectorAll('span')
      return (spans[1]?.innerText ?? '').trim()
    })
  )

let codes = []

try {
  /* ============ ① 登录 ============ */
  console.log('① 登录后台')
  const { captchaRequired } = await a.login()
  ok('没有要求验证码', !captchaRequired)

  /* ============ ② 进页面、找页签 ============ */
  console.log('\n② 游戏 → 游戏激活码')
  await a.openGameGroup()
  await a.clickByText('游戏激活码')
  await page.waitForFunction(() => location.hash.includes('game'), { timeout: 20000 })
  await page.waitForFunction(() => document.body.innerText.includes('明文只在'), { timeout: 20000 })

  /* ★ GVA 顶栏的多标签栏也用 .el-tabs__item，「首页 / 游戏激活码」会混进来。
     只认页面那一组三个页签，否则 querySelector 命中的 active 是顶栏标签，
     断言恒红（实测过一次：tabs = 首页/游戏激活码/心愿商品/心愿单/激活码）。 */
  const PAGE_TABS = ['心愿商品', '心愿单', '激活码']
  const tabs = await page.evaluate((names) =>
    [...document.querySelectorAll('.el-tabs__item')]
      .map((el) => (el.innerText || '').trim())
      .filter((t) => names.includes(t)), PAGE_TABS)
  ok('★ 有第三个页签「激活码」', tabs.includes('激活码'), tabs.join(' / '))
  ok(
    '★ 从「游戏激活码」菜单进来默认就停在激活码页签（MODE_BY_NAME 生效）',
    await page.evaluate((names) => {
      const active = [...document.querySelectorAll('.el-tabs__item.is-active')]
        .map((el) => (el.innerText || '').trim())
        .filter((t) => names.includes(t))
      return active.length === 1 && active[0] === '激活码'
    }, PAGE_TABS)
  )

  await a.clickByText('激活码')
  await new Promise((r) => setTimeout(r, 1200))
  const codeTabText = await page.evaluate(() => document.body.innerText)
  ok(
    '★ 页签里写清了「明文只给一次」（不然运营会以为随时能查回来）',
    codeTabText.includes('明文只在') && codeTabText.includes('拿不回来'),
    codeTabText.slice(0, 80)
  )

  /* ============ ③ 生成弹窗：只能选「游戏」分类的商品 ============ */
  console.log('\n③ 生成弹窗')
  await a.clickByText('生成激活码')
  await page.waitForSelector('.el-dialog', { timeout: 20000 })
  await new Promise((r) => setTimeout(r, 500))

  const genText = await page.evaluate(() => document.querySelector('.el-dialog')?.innerText ?? '')
  ok('弹窗打开', genText.includes('数量'), genText.slice(0, 60))
  ok(
    '★ 说明了「只能挂在游戏分类上」（挂错商品 = 卡白卖）',
    genText.includes('只解锁') || genText.includes('只能挂在'),
    genText.slice(0, 120)
  )

  /* 商品下拉里**只能**出现 gamebox 且带 game_id 的商品 */
  const gameboxApi = await adminApi('/mall/wish/item/list', { page: 1, pageSize: 200 })
  const allowed = (gameboxApi.data?.list ?? []).filter((x) => x.kind === 'gamebox' && x.game_id)
  const forbidden = (gameboxApi.data?.list ?? []).filter(
    (x) => x.kind !== 'gamebox' || !x.game_id
  )
  ok('接口里有可挂码的商品', allowed.length > 0, `${allowed.length} 个`)

  /* 选第一个可挂码的商品。
     ⚠️ 下拉是异步渲染的：点开之后**立刻**查 `.el-select-dropdown__item`
        会拿到 0 项（元素还没插进去），看起来像「下拉是空的」。
        所以这里轮询等，而不是 `sleep 固定值`。 */
  const readOptions = () =>
    page.evaluate(() => {
      const dds = [...document.querySelectorAll('.el-select-dropdown')].filter(
        (d) => d.offsetParent !== null
      )
      const out = []
      for (const dd of dds) {
        for (const el of dd.querySelectorAll('.el-select-dropdown__item')) {
          const t = (el.innerText || '').trim()
          if (t) out.push(t)
        }
      }
      return out
    })

  /* ⚠️ 必须限定在弹窗里找：筛选栏也有一个 label 叫「商品」，
     按 DOM 顺序会命中那一个，鼠标点在遮罩上 ⇒ 读出来「下拉 0 项」。 */
  await a.openSelectByLabel('商品', 'dialog')
  let options = []
  for (let i = 0; i < 20 && options.length === 0; i++) {
    await new Promise((r) => setTimeout(r, 200))
    options = await readOptions()
  }

  /* ★ 判据：下拉里的每一项都必须对应一件「游戏」分类的商品。
     用名字比对（界面上没有 item_id），并且**数量也要对得上** ——
     少了就是筛漏了，多了就是把不能挂码的也列出来了。 */
  const allowedNames = new Set(allowed.map((x) => x.name))
  const optionNames = options.map((t) => t.replace(/^\S+\s+/, '').trim())
  ok(
    '★ 下拉里只有「游戏」分类的商品（实物 / 盲盒挂不了码）',
    options.length > 0 && optionNames.every((n) => allowedNames.has(n)),
    `下拉 ${options.length} 项 / 可挂码 ${allowed.length} 个${forbidden.length ? ` / 不可挂码 ${forbidden.length} 个` : ''}`
  )
  if (!options.length) throw new Error('商品下拉是空的 —— 没有「游戏」分类的商品')
  await a.pickOption(options[0])

  await setInput('数量', COUNT)
  await setInput('批次号', BATCH)

  /* ============ ④ 生成 → 结果面板 ============ */
  console.log('\n④ 生成结果（明文只此一次）')
  await a.clickByText('生成')
  await a.waitFor(
    () => !!document.querySelector('.codebox .codeline'),
    '生成结果面板出现（.codebox 里有行）'
  )
  await new Promise((r) => setTimeout(r, 300))

  codes = await readCodes()
  ok('★ 结果面板列出了明文码', codes.length === COUNT, `${codes.length} 个：${codes.join(',')}`)
  ok('每个码都是 6 位', codes.every((c) => c.length === 6), codes.join(','))
  ok(
    '★ 字符都在 Crockford Base32 里（没有 I / L / O / U —— 手抄不会错认）',
    codes.every((c) => [...c].every((ch) => ALPHABET.includes(ch))),
    codes.join(',')
  )

  /* ★ 必须按「含 .codebox 的那个弹窗」找。
     页面上同时有两个 `el-dialog`（生成 / 结果），`querySelector` 取第一个，
     而生成弹窗关掉时还有一段退场动画、`destroy-on-close` 也还没把它摘掉
     ⇒ 读到的是**生成弹窗**的标题「生成激活码」，断言红成
     「警告文案不见了」。单独跑时赢在动画刚好结束，所以时绿时红。 */
  const readResultDialogText = () =>
    page.evaluate(() => {
      const d = [...document.querySelectorAll('.el-dialog')].find((x) =>
        x.querySelector('.codebox')
      )
      return d?.innerText ?? ''
    })
  const resultText = await readResultDialogText()
  ok(
    '★ 结果面板警告「关掉就拿不回来」',
    resultText.includes('拿不回来') && resultText.includes('哈希'),
    resultText.slice(0, 90)
  )

  /* 复制：按钮要变成「已复制 ✓」（复制失败会停在「复制全部」，不报错才最坑） */
  await a.clickByText('复制全部')
  await new Promise((r) => setTimeout(r, 500))
  const copiedText = await readResultDialogText()
  ok('★ 点「复制全部」后按钮变成「已复制」', copiedText.includes('已复制'), copiedText.slice(0, 60))

  await a.clickByText('我已复制')
  /* ⚠️ 判据是「**看不见**了」，不是「DOM 里没有」：
     结果弹窗没加 `destroy-on-close`，关掉之后 `.codebox` 仍然在 DOM 里
     （只是父层 `display:none`）。原来等的是 `!querySelector('.codebox')`，
     面板明明关了却等到 20 秒超时 —— 报的是「Waiting failed」，
     看起来像「按钮点不动」，其实界面早就对了。 */
  await page.waitForFunction(
    () => {
      const box = document.querySelector('.codebox')
      return !box || box.offsetParent === null
    },
    { timeout: 20000 }
  )
  ok('点「我已复制」后结果面板关闭', true)

  /* ============ ⑤ 列表里能按批次查到，且码尾对得上 ============ */
  console.log('\n⑤ 列表')
  await page.waitForFunction(
    (b) => document.body.innerText.includes(b) || true,
    { timeout: 5000 },
    BATCH
  )
  /* 按批次筛（不筛的话新批次可能不在第一页） */
  await a.openSelectByLabel('批次')
  await new Promise((r) => setTimeout(r, 400))
  const batchOptions = await page.evaluate(() => {
    const dd = [...document.querySelectorAll('.el-select-dropdown')].find(
      (d) => d.offsetParent !== null
    )
    return [...(dd?.querySelectorAll('.el-select-dropdown__item') ?? [])].map((el) =>
      (el.innerText || '').trim()
    )
  })
  ok('★ 批次下拉里出现了刚生成的批次（否则「作废整批」永远用不了）',
    batchOptions.includes(BATCH), batchOptions.slice(0, 5).join(' / '))
  await a.pickOption(BATCH)
  await a.clickByText('查询')
  await new Promise((r) => setTimeout(r, 1200))

  const rows = await readRows()
  ok('按批次筛出 2 行', rows.length === COUNT, `${rows.length} 行`)
  const tails = rows.map((r) => (r[2] ?? '').replace(/[^\w]/g, ''))
  ok(
    '★ 列表里的码尾 == 明文后 4 位（家长拿卡来问「是不是这张」就靠它）',
    codes.every((c) => tails.some((t) => t.endsWith(c.slice(-4)))),
    `界面 ${tails.join(',')} / 明文 ${codes.map((c) => c.slice(-4)).join(',')}`
  )
  ok('新生成的码状态是「未使用」', rows.every((r) => (r[3] ?? '').includes('未使用')), rows[0]?.[3])

  console.log('  截图：' + (await a.shot('wish-codes')))

  /* ============ ⑥ 收尾：整批作废（探针可反复跑） ============ */
  console.log('\n⑥ 收尾')
  const voided = await adminApi('/mall/wish/code/void', { batch: BATCH })
  ok('整批作废成功（没用过的都清掉）', voided.code === 0 && (voided.data?.voided ?? 0) === COUNT,
    JSON.stringify(voided.data ?? voided.msg))

  ok('没有页面级报错', errors.length === 0, errors.slice(0, 2).join(' | '))
} catch (e) {
  failed++
  console.log(`\n✗ 探针中断： ${e.message}`)
  try {
    const snap = await page.evaluate(() => document.body.innerText.slice(0, 600))
    console.log('  页面文字：' + snap.replace(/\s+/g, ' '))
    console.log('  截图：' + (await a.shot('wishcode-fail')))
  } catch {
    /* 页面可能已经关了 */
  }
} finally {
  /* 中断了也要把码作废掉，否则库里留一批「没人知道明文」的码 */
  if (codes.length) {
    await adminApi('/mall/wish/code/void', { batch: BATCH }).catch(() => null)
  }
  await a.close()
}

console.log(failed ? `\n${failed} 条失败` : '\n全部通过')
process.exit(failed ? 1 : 0)
