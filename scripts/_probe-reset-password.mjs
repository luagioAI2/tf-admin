/**
 * 探针：后台「重置登录密码」—— 忘了密码的唯一兜底
 * ------------------------------------------------------------
 * 为什么要有这个功能：生产环境**没有短信通道**（平台 `DevSmsSender` 在
 * `IsProduction()` 下直接返回错误），C 端「忘记密码」那条路走不通。
 * 用户忘了密码 = 永久进不去，而他的积分、兑换券、农场、存档全在那个账号里。
 *
 * 这个探针守的是**四件容易静默做坏的事**：
 *
 *   ① 写进库的必须是 bcrypt 哈希，不能是明文。写明文的话平台端
 *      `bcrypt.CompareHashAndPassword` 永远不匹配 —— 后台显示「已重置」，
 *      用户拿新密码登不上，而且没有任何报错；
 *   ② 旧密码必须**立刻失效**、新密码必须**真的能登**（这是全部意义所在）；
 *   ③ 旧设备的 refresh token 必须被撤掉 —— 否则「账号被别人拿走」的场景里，
 *      对方还能续命 7 天；
 *   ④ 游客不能被重置：他没有手机号，设了密码也登不进来。
 *      静默给他设一个密码 = 后台显示成功、用户永远登不上。
 *
 * ★ 不碰真实用户：探针自己用平台接口注册一个专用账号（昵称「探针·重置密码」），
 *   所有操作都只发生在它身上。
 *
 * 用法：
 *   1) 平台：tf-platform-api 的 tf-api.exe（8090）
 *   2) 后台：tf-admin/server 的 tf-admin.exe（8890）
 *   3) 前端：tf-admin/web 的 pnpm dev（8891）
 *   4) node scripts/_probe-reset-password.mjs [url]
 */
import { adminApi, openAdmin } from './_admin-lib.mjs'

const URL_ = process.argv[2] ?? 'http://localhost:8891/'
const PLATFORM = process.env.TF_PLATFORM_API ?? 'http://127.0.0.1:8090'

let failed = 0
const ok = (label, pass, detail = '') => {
  if (!pass) failed++
  console.log(`  ${pass ? '✓' : '✗'} ${label}${detail ? ` — ${detail}` : ''}`)
}
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

/** 打平台接口（本机代理会截 127.0.0.1，node 的 fetch 不受影响，但这里显式说明） */
const platform = async (path, body) => {
  const res = await fetch(`${PLATFORM}/api${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body ?? {})
  })
  return res.json()
}

/* ============ 准备：造一个专用账号 ============ */
const mobile = '137' + String(Date.now()).slice(-8)
const oldPwd = 'oldpass123'
const reg = await platform('/auth/register', {
  mobile,
  password: oldPwd,
  nickname: '探针·重置密码'
})
if (reg.code !== 200) {
  console.error(
    `注册测试账号失败（code=${reg.code} msg=${reg.msg}）——\n` +
      `  8090 上的 tf-platform-api 起来了吗？（${PLATFORM}）`
  )
  process.exit(2)
}
const uid = reg.data.user.uid
const oldRefresh = reg.data.refresh_token
console.log(`测试账号：uid=${uid} mobile=${mobile}\n`)

/* ============ ⓪ 接口层：游客 / 空备注 必须被拒 ============ */
console.log('⓪ 接口层守卫')
const guest = await platform('/auth/guest', { device_id: 'probe-reset-' + Date.now() })
if (guest.code === 200) {
  const r = await adminApi('/mall/user/reset-password', {
    uid: guest.data.user.uid,
    memo: '探针：游客不该能重置'
  })
  ok('★ 游客被拒（他没有手机号，设了也登不进来）', r.code !== 0, `code=${r.code} msg=${r.msg}`)
  ok('拒绝原因说人话（提到「游客」或「手机号」）', /游客|手机号/.test(r.msg ?? ''), r.msg)
} else {
  ok('游客账号建得出来（前置条件）', false, JSON.stringify(guest).slice(0, 120))
}

const noMemo = await adminApi('/mall/user/reset-password', { uid, memo: '' })
ok('空备注被拒（备注是唯一的追溯线索）', noMemo.code !== 0, `code=${noMemo.code}`)

/* ============ ① 界面：走完整条路 ============ */
const a = await openAdmin({ url: URL_ })
const { page } = a

/**
 * 在**可见的**弹窗里按文字点按钮。
 *
 * ★ 不能用 `document.querySelector('.el-dialog')`：这一页有三个弹窗
 *   （详情 / 调余额 / 重置密码），`querySelector` 拿的是 **DOM 里第一个**，
 *   也就是详情那个 —— 于是「确认重置」永远找不到，而报错看起来像
 *   「按钮没渲染」。必须遍历所有可见弹窗。
 * ★ 也不能用 `offsetParent` 之外的可见性判断：`position:fixed` 的
 *   offsetParent 恒为 null，而 Element Plus 的弹窗是 fixed 的 ——
 *   这里用的是 `getBoundingClientRect().width`（0 = 隐藏）。
 */
const clickIn = async (text) => {
  const hit = await page.evaluate((t) => {
    const dialogs = [...document.querySelectorAll('.el-dialog')].filter(
      (d) => d.getBoundingClientRect().width > 0
    )
    for (const d of dialogs) {
      const btn = [...d.querySelectorAll('button')].find((b) => (b.innerText || '').includes(t))
      if (!btn) continue
      const r = btn.getBoundingClientRect()
      if (!r.width || !r.height) continue
      return { cx: Math.round(r.x + r.width / 2), cy: Math.round(r.y + r.height / 2) }
    }
    return null
  }, text)
  if (!hit) throw new Error(`可见弹窗里找不到「${text}」`)
  await page.mouse.click(hit.cx, hit.cy)
}

try {
  console.log('\n① 登录后台并进 App 用户')
  const { captchaRequired } = await a.login()
  ok('没有要求验证码', !captchaRequired)

  await a.openMallGroup()
  await a.clickByText('App 用户')
  await page.waitForFunction(() => location.hash.includes('mall/user'), { timeout: 20000 })
  /* ★ 等**这一页特有**的文字：仪表盘也有表格，等「表格里有行」会读到上一页 */
  await page.waitForFunction(() => document.body.innerText.includes('个账号'), { timeout: 20000 })

  /* 搜手机号定位到刚建的那个账号（列表按 id DESC，它本来就在第一页，但搜索更稳） */
  await page.type('input[placeholder="支持片段，如 138"]', mobile)
  await a.clickByText('查询')
  await page.waitForFunction(
    (m) => [...document.querySelectorAll('tbody tr.el-table__row')].some((r) =>
      (r.innerText || '').includes(m)
    ),
    { timeout: 20000 },
    mobile
  )
  ok('★ 搜得到刚建的账号', true)

  console.log('\n② 详情 → 重置登录密码')
  /* ★★ 这一步是**间歇性**点不中的：坐标算好之后表格还会重渲染一次
     （查询回包后 el-table 重排），点下去落在空白上，什么都不会发生，
     然后 20 秒后报 `Waiting failed: 20000ms exceeded` ——
     和「按钮没渲染」「弹窗没开」长得一模一样，实测红了两次、第三次自己好了。
     所以：点 → 短等 → 没开就重来，最多 3 次；三次都没开就**带着当时的
     弹窗 / 表格状态**报错（`a.waitFor` 会把快照写进错误里）。 */
  const detailReady = () => !!document.querySelector('[data-reset-open]')
  let opened = false
  for (let attempt = 0; attempt < 3 && !opened; attempt++) {
    const rowHit = await page.evaluate((m) => {
      const tr = [...document.querySelectorAll('tbody tr.el-table__row')].find((r) =>
        (r.innerText || '').includes(m)
      )
      if (!tr) return null
      const btn = [...tr.querySelectorAll('button')].find((b) => (b.innerText || '').includes('详情'))
      if (!btn) return null
      const r = btn.getBoundingClientRect()
      return { cx: Math.round(r.x + r.width / 2), cy: Math.round(r.y + r.height / 2) }
    }, mobile)
    if (!rowHit) throw new Error('那一行里找不到「详情 / 流水」按钮')
    await page.mouse.click(rowHit.cx, rowHit.cy)
    opened = await a.waitFor(detailReady, '详情弹窗里出现「重置登录密码」', {
      timeout: 6000,
      soft: true
    })
  }
  /* 三次都没开：再等 1.5 秒让它超时 —— 只为拿到带快照的那条错误 */
  if (!opened) await a.waitFor(detailReady, '详情弹窗里出现「重置登录密码」', { timeout: 1500 })
  ok('详情弹窗里出现「重置登录密码」', true)

  await clickIn('重置登录密码')
  await a.waitFor(() => !!document.querySelector('[data-reset-memo]'), '重置弹窗里出现备注框', {
    timeout: 10000
  })
  await sleep(500)
  ok('弹窗打开了（有备注框）', true)

  /* 先验「不填备注不让走」—— 服务端也拒，但不该让人等一个来回才知道 */
  await clickIn('确认重置')
  await sleep(600)
  const memoGuard = await page.evaluate(() => document.body.innerText.includes('备注必须填'))
  ok('★ 不填备注点确认会被拦住（本地就拦）', memoGuard)
  ok(
    '被拦住时**没有**真的调接口（新密码框还没出现）',
    await page.evaluate(() => !document.querySelector('[data-reset-password]'))
  )

  /* ★ `data-reset-memo` 直接落在 <textarea> 上（Element Plus 的 el-input
     把 $attrs 透传给内层控件，不是包一层 div）——写成 `[data-reset-memo] textarea`
     会一个都选不到。 */
  await page.type('[data-reset-memo]', '工单 #探针 家长忘了密码')
  await clickIn('确认重置')
  /* ElMessageBox 的确认框：不点它什么都不会发生 */
  await page.waitForFunction(
    () => !!document.querySelector('.el-message-box__btns'),
    { timeout: 10000 }
  )
  await sleep(300)
  const confirmHit = await page.evaluate(() => {
    const box = document.querySelector('.el-message-box__btns')
    const btn = [...box.querySelectorAll('button')].find((b) => (b.innerText || '').includes('确定'))
    if (!btn) return null
    const r = btn.getBoundingClientRect()
    return { cx: Math.round(r.x + r.width / 2), cy: Math.round(r.y + r.height / 2) }
  })
  if (!confirmHit) throw new Error('确认框里找不到「确定」')
  await page.mouse.click(confirmHit.cx, confirmHit.cy)

  await page.waitForFunction(() => !!document.querySelector('[data-reset-password]'), {
    timeout: 20000
  })
  const newPwd = await page.evaluate(
    () => (document.querySelector('[data-reset-password]')?.innerText || '').trim()
  )
  ok('★ 界面把新密码显示出来了', newPwd.length > 0, newPwd)
  /* 生成器的字母表故意去掉了 0/O/1/l/I —— 这串要被人念出来 */
  ok(
    '新密码不含易混字符（0 O 1 l I）',
    /^[abcdefghjkmnpqrstuvwxyz23456789]{8}$/.test(newPwd),
    newPwd
  )
  await a.shot('reset-password')

  await clickIn('我记下了')
  await sleep(600)
  ok(
    '关掉之后新密码不再显示（只给一次）',
    await page.evaluate(() => !document.querySelector('[data-reset-password]'))
  )

  /* ============ ③ 真的能登 / 真的不能登 ============ */
  console.log('\n③ 新密码真的能用、旧密码真的废了')
  const withOld = await platform('/auth/login', { mobile, password: oldPwd })
  ok('★ 旧密码登不上了', withOld.code !== 200, `code=${withOld.code}`)

  const withNew = await platform('/auth/login', { mobile, password: newPwd })
  ok('★ 新密码能登上（这是全部意义）', withNew.code === 200, `code=${withNew.code}`)
  ok(
    '登回来还是同一个 uid（没被重置成新号）',
    withNew.data?.user?.uid === uid,
    `${withNew.data?.user?.uid} vs ${uid}`
  )

  /* ============ ④ 旧设备的刷新令牌必须作废 ============ */
  const refreshed = await platform('/auth/refresh', { refresh_token: oldRefresh })
  ok(
    '★ 旧设备的 refresh token 已作废（不然被拿走的号还能续 7 天）',
    refreshed.code !== 200,
    `code=${refreshed.code}`
  )

  ok('控制台没有报错', a.errors.length === 0, a.errors.slice(0, 2).join(' | '))
  ok('没有本机 4xx/5xx 请求', a.reqFails.length === 0, a.reqFails.slice(0, 2).join(' | '))
} catch (e) {
  failed++
  console.log(`  ✗ 探针中断：${e.message}`)
} finally {
  await a.close()
}

console.log(`\n===== ${failed === 0 ? '全部通过' : `${failed} 条失败`} =====`)
process.exit(failed === 0 ? 0 : 1)
