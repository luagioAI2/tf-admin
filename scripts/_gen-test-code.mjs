/* 一次性：用后台接口生成两张真码，给「商品详情页输码」走查用（2026-11-05）。
   验完就删。 */
const API = process.env.TF_ADMIN_API ?? 'http://127.0.0.1:8890'
const ADMIN = {
  username: process.env.TF_ADMIN_USER ?? 'admin',
  password: process.env.TF_ADMIN_PASSWORD ?? 'TfAdmin@2026',
}

const login = await fetch(`${API}/base/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(ADMIN),
}).then((r) => r.json())
if (login?.code !== 0) {
  console.error('登录失败', JSON.stringify(login).slice(0, 200))
  process.exit(1)
}
const token = login.data.token

const gen = await fetch(`${API}/mall/wish/code/generate`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json', 'x-token': token },
  body: JSON.stringify({
    item_id: 'wish-game-renttower',
    count: 2,
    batch: 'probe-1105',
    expire_days: 7,
  }),
}).then((r) => r.json())

if (gen?.code !== 0) {
  console.error('生成失败', JSON.stringify(gen).slice(0, 300))
  process.exit(1)
}
console.log(JSON.stringify({ batch: gen.data.batch, codes: gen.data.codes }))
