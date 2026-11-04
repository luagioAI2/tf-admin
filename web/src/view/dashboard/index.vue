<template>
  <!-- 登录后的第一屏。
       原来这一屏是框架自己的宣传页（访问人数 / 新增客户 / 最新插件 /
       授权版演示 / 购买商业授权），家长和商家看到只会以为走错了地方。
       现在只回答两个问题：**现在有什么要处理的**、**常用入口在哪**。 -->
  <div class="min-h-full bg-[#fffdf7] p-4 lg:p-6">
    <!-- ============ 抬头 ============ -->
    <section
      class="rounded-2xl bg-gradient-to-br from-[#22c55e] to-[#15803d] px-5 py-6 text-white shadow-sm lg:px-7"
    >
      <div class="flex items-center gap-4">
        <img
          src="/logo.png"
          alt="小任务农场"
          class="h-14 w-14 rounded-xl bg-white/20 object-contain p-1"
        />
        <div class="min-w-0">
          <p class="truncate text-lg font-semibold lg:text-xl">
            {{ greeting }}，{{ nickName }}
          </p>
          <p class="mt-1 text-sm text-white/85">
            {{ today }} · 当前角色：{{ roleName }}
          </p>
        </div>
      </div>
      <p class="mt-4 text-sm leading-6 text-white/85">
        这里是「小任务农场」的后台：家长看孩子的心愿和兑换，商家看自己的商品，运营管审核和上架。
        家长 / 商家用的就是 App 的账号，能看到什么由权限决定。
      </p>
    </section>

    <!-- ============ 现在要处理的四件事 ============ -->
    <div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <div
        v-for="k in kpis"
        :key="k.key"
        :data-kpi="k.key"
        class="rounded-2xl border bg-white px-5 py-4 shadow-sm transition"
        :class="k.warn ? 'border-[#f5c2c7] ring-1 ring-[#f5c2c7]/50' : 'border-[#f1e7cf]'"
      >
        <div class="flex items-center justify-between">
          <span class="text-sm text-slate-500">{{ k.label }}</span>
          <span class="text-lg">{{ k.icon }}</span>
        </div>
        <!-- `kpi-v` 是探针的锚点：数字和接口对不对得上，只认这一个 class -->
        <div class="kpi-v mt-2 text-3xl font-semibold text-[#14532d]">
          {{ k.value }}
        </div>
        <div class="mt-1 text-xs text-slate-400">
          <el-link
            v-if="k.to && k.warn"
            type="primary"
            class="!text-xs"
            @click="go(k.to)"
          >
            {{ k.hint }} →
          </el-link>
          <template v-else>{{ k.hint }}</template>
        </div>
      </div>
    </div>

    <!-- ============ 常用入口 ============ -->
    <section class="mt-4 rounded-2xl border border-[#f1e7cf] bg-white p-5 shadow-sm">
      <div class="flex items-baseline justify-between">
        <h2 class="text-base font-semibold text-slate-800">常用入口</h2>
        <span class="text-xs text-slate-400">只显示这个账号有权限的页面</span>
      </div>
      <div class="mt-4 grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
        <button
          v-for="s in shortcuts"
          :key="s.name"
          type="button"
          class="flex items-center gap-3 rounded-xl border border-[#f1e7cf] bg-[#fffdf7] px-4 py-3 text-left transition hover:border-[#22c55e] hover:bg-[#f0fdf4]"
          @click="go(s.path)"
        >
          <span class="text-xl">{{ s.icon }}</span>
          <span class="min-w-0">
            <span class="block truncate text-sm font-medium text-slate-800">
              {{ s.title }}
            </span>
            <span class="block truncate text-xs text-slate-400">
              {{ s.desc }}
            </span>
          </span>
        </button>
      </div>
      <p v-if="!shortcuts.length" class="mt-3 text-sm text-slate-400">
        这个账号还没有可进入的页面。
      </p>
    </section>

    <!-- 拉不到数据时**必须说话**：安静地显示 0 比报错更坏 ——
         0 看起来像一个真实的数，运营会以为「今天真的没人申请」。 -->
    <p
      v-if="state === 'fail'"
      class="mt-4 rounded-xl border border-[#f5c2c7] bg-[#fff1f2] px-4 py-3 text-sm text-[#b91c1c]"
    >
      上面的数字没拉到：{{ errMsg }}。稍后点右上角刷新再试。
    </p>

    <p class="mt-4 px-1 text-xs leading-6 text-slate-400">
      上面的数字来自 App 的真实数据（用户、钱包、兑换、心愿、任务），
      和「超级管理员 → 用户管理」里那张表不是一回事：那张表是后台自己的账号。
    </p>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { useUserStore } from '@/pinia/modules/user'
  import { getMallOverview, getMerchantPendingCount } from '@/api/mallDashboard'

  defineOptions({
    name: 'Dashboard'
  })

  const router = useRouter()
  const userStore = useUserStore()

  const nickName = computed(() => userStore.userInfo?.nickName || '你好')
  const roleName = computed(
    () => userStore.userInfo?.authority?.authorityName || '管理员'
  )

  const greeting = computed(() => {
    const h = new Date().getHours()
    if (h < 6) return '夜深了'
    if (h < 11) return '早上好'
    if (h < 14) return '中午好'
    if (h < 18) return '下午好'
    return '晚上好'
  })

  const today = computed(() => {
    const d = new Date()
    const week = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][
      d.getDay()
    ]
    return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日 ${week}`
  })

  /* 门户角色（家长 / 商家）没有这一屏的权限，也不该去请求运营接口 ——
     否则打开就是一条 403 弹窗，用户会以为自己账号坏了。 */
  const PORTAL_AUTHORITIES = [1001, 1002, 1003]
  const isStaff = computed(
    () => !PORTAL_AUTHORITIES.includes(Number(userStore.userInfo?.authorityId))
  )

  /* loading / ok / fail 三态。**不用 `loaded` 布尔**：
     布尔只有两态，失败只能混进「已加载」，于是 0 被当成真数。 */
  const state = ref('loading')
  const errMsg = ref('')
  const raw = ref({})
  const pendingMerchant = ref(0)

  const num = (v, d = 0) => (state.value === 'ok' ? (v ?? d) : '—')

  const kpis = computed(() => [
    {
      key: 'merchant',
      icon: '📮',
      label: '待审核招商',
      value: num(pendingMerchant.value),
      warn: state.value === 'ok' && pendingMerchant.value > 0,
      hint:
        state.value !== 'ok'
          ? '还没拉到数据'
          : pendingMerchant.value > 0
            ? '有商家在等，去审核'
            : '没有积压',
      to: '/mall/merchant'
    },
    {
      key: 'wish',
      icon: '🎁',
      label: '心愿待处理',
      value: num(raw.value.wish_pending_action),
      warn: state.value === 'ok' && (raw.value.wish_pending_action ?? 0) > 0,
      hint: `已许愿 ${num(raw.value.wish_pending_wished)} 个 · 待兑现`,
      to: '/mall/wish'
    },
    {
      key: 'user',
      icon: '🧒',
      label: `近 ${raw.value.days ?? 7} 天新增`,
      value: num(raw.value.user_new),
      warn: false,
      // ⚠️ 别写「个账号」：「App 用户」页的探针拿「个账号」当**这一页特有**的锚点，
      //    首页也出现这四个字，就等于把那个锚点弄成了恒真。
      hint: `累计 ${num(raw.value.user_total)} 位用户`
    },
    {
      key: 'coin',
      icon: '🪙',
      label: '钱包里的积分',
      value: num(raw.value.points_in_wallets),
      warn: false,
      hint: '这是还没花掉的部分'
    }
  ])

  /* 只列**真的存在**的路由：菜单是按角色动态挂的，
     写死一个没挂上的路径，点下去是空白页，而人的第一反应是「权限配错了」。 */
  const ALL_SHORTCUTS = [
    { name: 'merchant', path: '/mall/merchant', icon: '📮', title: '招商审核', desc: '通过 / 拒绝入驻' },
    { name: 'mallWish', path: '/mall/wish', icon: '🎁', title: '心愿商城', desc: '孩子许的愿' },
    { name: 'mallRedeem', path: '/mall/redeem', icon: '🛍️', title: '兑换商城', desc: '商品与定价' },
    { name: 'appUser', path: '/mall/user', icon: '👨‍👩‍👧', title: 'App 用户', desc: '家长与孩子的账号' },
    { name: 'mallDashboard', path: '/mall/dashboard', icon: '📊', title: '经营看板', desc: '更细的盘子' }
  ]
  const shortcuts = computed(() =>
    ALL_SHORTCUTS.filter((s) => router.hasRoute(s.name))
  )

  const go = (path) => {
    router.push(path)
  }

  onMounted(async () => {
    if (!isStaff.value) {
      state.value = 'ok'
      return
    }
    try {
      const [ov, pc] = await Promise.all([
        getMallOverview(),
        getMerchantPendingCount()
      ])
      if (ov?.code !== 0) throw new Error(ov?.msg || '看板接口返回异常')
      raw.value = ov.data || {}
      // `pending-count` 的 data 就是**一个数字**，不是 { count } ——
      // 按对象取会静默拿到 undefined，界面上永远是 0。
      pendingMerchant.value = Number(pc?.data ?? 0)
      state.value = 'ok'
    } catch (e) {
      errMsg.value = e?.message || '网络异常'
      state.value = 'fail'
    }
  })
</script>

<style lang="scss" scoped></style>
