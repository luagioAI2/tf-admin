/**
 * 一次跑完后端的所有页面探针（顺序执行，中间留间隔）。
 *
 * ★ 为什么不能并行 / 不能连着跑：后台登录和几个列表接口都有频率限制，
 *   12+ 次请求连着发会返回 `too many requests`，探针会红在
 *   「有页面级报错 — AxiosError$1」这种完全没有线索的地方 ——
 *   单独跑一遍又全绿，非常难判断到底是不是自己改坏了。
 *
 * 用法：
 *   node scripts/_run-all-probes.mjs            # 全跑
 *   node scripts/_run-all-probes.mjs wish redeem  # 只跑名字里含这些词的
 */
import { spawnSync } from 'node:child_process'
import { readdirSync } from 'node:fs'

const GAP_MS = 6000
const filters = process.argv.slice(2)

const files = readdirSync('scripts')
  .filter((f) => /^_probe-.*\.mjs$/.test(f))
  .filter((f) => !filters.length || filters.some((k) => f.includes(k)))
  .sort()

if (!files.length) {
  console.error('没有匹配的探针文件')
  process.exit(2)
}

const results = []
for (const [i, f] of files.entries()) {
  console.log(`\n${'='.repeat(60)}\n▶ ${f}\n${'='.repeat(60)}`)
  const r = spawnSync(process.execPath, [`scripts/${f}`], { stdio: 'inherit' })
  results.push({ f, code: r.status })
  /* 最后一个不用等 */
  if (i < files.length - 1) {
    console.log(`\n… 等 ${GAP_MS / 1000}s（避开接口限流）`)
    await new Promise((res) => setTimeout(res, GAP_MS))
  }
}

console.log(`\n${'='.repeat(60)}\n汇总`)
for (const { f, code } of results) {
  console.log(`  ${code === 0 ? '✓' : '✗'} ${f}${code === 0 ? '' : `（退出码 ${code}）`}`)
}
const bad = results.filter((r) => r.code !== 0).length
console.log(bad === 0 ? '\n全部通过' : `\n${bad} 个探针失败`)
process.exit(bad === 0 ? 0 : 1)
