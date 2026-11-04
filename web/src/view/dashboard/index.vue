<template>
  <!-- 登录后的第一屏。
       原来这一屏是框架自己的宣传页（访问人数 / 新增客户 / 最新插件 /
       授权版演示 / 购买商业授权），家长和商家看到只会以为走错了地方。
       现在只回答两个问题：**现在有什么要处理的**、**常用入口在哪**。 -->
  <div class="min-h-full bg-[#fffcf3] p-4 lg:p-6">
    <!-- ============ 抬头 ============ -->
    <section
      class="flex items-center gap-3 rounded-2xl border border-[#e6efe6] bg-white px-5 py-4 shadow-md"
    >
      <div
        class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-[#f0f8f2] shadow-sm"
      >
        <img
          src="/logo.png"
          alt="小任务农场"
          class="h-8 w-8 rounded-lg object-contain"
        />
      </div>
      <div class="min-w-0">
        <p class="truncate text-base font-semibold text-[#14532d]">
          {{ greeting }}，{{ nickName }}
        </p>
        <p class="mt-1 text-xs text-[#6b7c6f]">
          {{ today }} · {{ roleName }}
        </p>
      </div>
      <span
        class="ml-auto hidden h-2 w-2 rounded-full bg-[#22c55e] shadow-[0_0_0_4px_rgba(34,197,94,0.16)] sm:block"
      ></span>
    </section>

    <!-- ============ 现在要处理的四件事 ============ -->
    <div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <div
        v-for="k in kpis"
        :key="k.key"
        :data-kpi="k.key"
        class="rounded-2xl border bg-white px-5 py-4 shadow-md transition hover:-translate-y-0.5 hover:shadow-lg"
        :class="k.warn ? 'border-[#e5b9bd]' : 'border-[#e6efe6]'"
      >
        <div class="flex items-center gap-2">
          <span
            class="h-1.5 w-1.5 shrink-0 rounded-full"
            :class="k.warn ? 'bg-[#d97757]' : 'bg-[#15803d]'"
          ></span>
          <span class="text-sm text-slate-500">{{ k.label }}</span>
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
    <section class="mt-4 rounded-2xl border border-[#e6efe6] bg-white p-5 shadow-md">
      <div class="flex items-baseline justify-between">
        <h2 class="text-base font-semibold text-[#14532d]">常用入口</h2>
      </div>
      <div class="mt-4 grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
        <button
          v-for="s in shortcuts"
          :key="s.name"
          type="button"
          class="flex items-center gap-3 rounded-xl border border-[#e6efe6] bg-[#fbfdf9] px-4 py-3 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-[#15803d] hover:shadow-md"
          @click="go(s.name)"
        >
          <span class="min-w-0">
            <span class="block truncate text-sm font-medium text-[#14532d]">
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
      class="mt-4 rounded-2xl border border-[#e5b9bd] bg-[#fff6f6] px-4 py-3 text-sm text-[#a32d2d] shadow-sm"
    >
      上面的数字没拉到：{{ errMsg }}。稍后点右上角刷新再试。
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
      to: 'merchant'
    },
    {
      key: 'wish',
      icon: '🎁',
      label: '心愿待处理',
      value: num(raw.value.wish_pending_action),
      warn: state.value === 'ok' && (raw.value.wish_pending_action ?? 0) > 0,
      hint: `已许愿 ${num(raw.value.wish_pending_wished)} 个 · 待兑现`,
      to: 'mallWish'
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
     写死一个没挂上的路径，点下去是空白页，而人的第一反应是「权限配错了」。
     ★ 而且**只能按 name 跳**：动态路由整体挂在 `/layout` 底下，真实路径是
       `/layout/mall/merchant`；写 `/mall/merchant` 会落到 catchAll 的 404 页
       （那一屏还带着框架的项目地址）。`hasRoute` 按 name 查得通过，path 却
       对不上 —— 所以「卡片显示出来了」根本不等于「点得动」。 */
  const ALL_SHORTCUTS = [
    { name: 'merchant', icon: '📮', title: '招商审核', desc: '通过 / 拒绝入驻' },
    { name: 'mallWish', icon: '🎁', title: '心愿商城', desc: '孩子许的愿' },
    { name: 'mallRedeem', icon: '🛍️', title: '兑换商城', desc: '商品与定价' },
    { name: 'appUser', icon: '👨‍👩‍👧', title: 'App 用户', desc: '家长与孩子的账号' },
    { name: 'mallDashboard', icon: '📊', title: '经营看板', desc: '更细的盘子' }
  ]
  const shortcuts = computed(() =>
    ALL_SHORTCUTS.filter((s) => router.hasRoute(s.name))
  )

  const go = (name) => {
    if (!router.hasRoute(name)) return
    router.push({ name })
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
