<template>
  <div>
    <!-- ============ 顶部：现在要动手的事 ============ -->
    <el-row :gutter="12" class="row">
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="card" :class="{ warn: pendingMerchant > 0 }">
          <div class="k">
            <span class="ico">📮</span>待审核招商
          </div>
          <div class="v">{{ pendingMerchant }}</div>
          <div class="sub">
            <el-link v-if="pendingMerchant > 0" type="primary" @click="go('/mall/merchant')">
              去审核 →
            </el-link>
            <span v-else>没有积压</span>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="card" :class="{ warn: wishWaiting > 0 }">
          <div class="k">
            <span class="ico">🎁</span>心愿待处理
          </div>
          <div class="v">{{ wishWaiting }}</div>
          <div class="sub">
            <!-- 这里**不能**放跳转：心愿后台页还没写，点进去是空白页，
                 而人的第一反应是「权限配错了」。 -->
            已许愿 {{ n.wish_pending_wished ?? 0 }} · 待兑现
            {{ n.wish_pending_action ?? 0 }}
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="card">
          <div class="k">
            <span class="ico">🧒</span>近 {{ n.days ?? 7 }} 天新增
          </div>
          <div class="v">{{ n.user_new ?? 0 }}</div>
          <div class="sub">共 {{ n.user_total ?? 0 }} 个账号</div>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="card">
          <div class="k">
            <span class="ico">💰</span>钱包里的积分
          </div>
          <div class="v">{{ n.points_in_wallets ?? 0 }}</div>
          <div class="sub">这是<b>还没花掉</b>的部分</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- ============ 下面：家底 ============ -->
    <el-card shadow="never" class="panel">
      <template #header>
        <div class="panel-head">
          <span>家底</span>
          <span class="dim">点右上角刷新重新统计</span>
          <el-button
            class="refresh"
            icon="refresh"
            :loading="loading"
            size="small"
            @click="load"
          >
            刷新
          </el-button>
        </div>
      </template>

      <el-row :gutter="12">
        <el-col v-for="it in facts" :key="it.key" :xs="12" :sm="8" :md="4">
          <div class="fact">
            <div class="fk">{{ it.label }}</div>
            <div class="fv">{{ n[it.key] ?? 0 }}</div>
            <div class="fh">{{ it.hint }}</div>
          </div>
        </el-col>
      </el-row>
    </el-card>

    <!-- ============ 说明 ============ -->
    <el-alert type="info" :closable="false" class="note">
      <template #title>
        这些数是 <b>App 那边的真实数据</b>（<code>user</code> /
        <code>wallet</code> / <code>redeem_records</code> /
        <code>wish_orders</code> / <code>task_instances</code> /
        <code>games</code>），和左边「用户管理」（<code>sys_users</code>，
        后台管理员）<b>不是一回事</b>。<br />
        「钱包里的积分」是<b>余额之和</b>，不是发放总量 —— 孩子花掉的已经不在里面了。
        要看发放和消耗的来龙去脉，去「App 用户 → 账本」。
      </template>
    </el-alert>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { ElMessage } from 'element-plus'
  import { getMallOverview, getMerchantPendingCount } from '@/api/mallDashboard'

  defineOptions({ name: 'MallDashboard' })

  const router = useRouter()
  const loading = ref(false)
  const n = ref({})
  const pendingMerchant = ref(0)

  /* 心愿待处理 = 两种状态的合计。分开留着是因为它们的含义不同：
     `wished` = 孩子刚许愿、还没被受理；`action` = 已经受理、等兑现。 */
  const wishWaiting = computed(
    () => (n.value.wish_pending_wished ?? 0) + (n.value.wish_pending_action ?? 0)
  )

  const facts = [
    { key: 'user_total', label: '账号总数', hint: '含游客' },
    { key: 'game_total', label: '游戏总数', hint: '已入库' },
    /* ⚠️ 这个数**现在一般是 0，而且是正常的**：本版客户端把游戏**打包在 App 里**
       （`src/games/localGames.ts` + 哈希），不走服务端下发。`game_release`
       是「云端游戏包」那套的遗留，将来真做下发才会用上。
       不写这句提示的话，运营看到 0 会以为「游戏全下架了」。 */
    { key: 'game_release_active', label: '在架版本', hint: '云端下发用；本版打包在 App 里，一般是 0' },
    { key: 'task_instances', label: '任务实例', hint: '历史累计' },
    { key: 'redeem_records', label: '兑换记录', hint: '历史累计' },
    { key: 'wish_orders', label: '心愿单', hint: '历史累计' }
  ]

  const load = async () => {
    loading.value = true
    try {
      const [ov, pc] = await Promise.all([getMallOverview(), getMerchantPendingCount()])
      if (ov.code === 0) n.value = ov.data ?? {}
      if (pc.code === 0) pendingMerchant.value = pc.data ?? 0
    } catch (e) {
      /* 失败要说话：看板是「一眼看家底」的地方，安静地显示 0
         比报错更坏 —— 0 看起来像一个真实的数。 */
      ElMessage.error('看板没拉到数据：' + (e?.message ?? e))
    } finally {
      loading.value = false
    }
  }

  /* 只跳**已经存在**的后台页。跳到一个没有 .vue 的菜单是空白页，
     前端只报一个 404 chunk，看着像权限配错了。 */
  const go = (path) => router.push(path)

  onMounted(load)
</script>

<style scoped>
  .row {
    margin-bottom: 4px;
  }
  .card {
    margin-bottom: 12px;
  }
  .card :deep(.el-card__body) {
    padding: 16px 18px;
  }
  /* 有积压才变色：看板的用处是「一眼看出哪件事要动手」，
     四个卡片全一个颜色就等于没说。 */
  .card.warn {
    border-left: 4px solid var(--el-color-warning);
  }
  .k {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
  .ico {
    margin-right: 6px;
  }
  .v {
    margin: 6px 0 2px;
    font-size: 30px;
    font-weight: 700;
    line-height: 1.2;
    font-variant-numeric: tabular-nums;
  }
  .sub {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    min-height: 18px;
  }
  .panel {
    margin-top: 4px;
  }
  .panel-head {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .panel-head .dim {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .panel-head .refresh {
    margin-left: auto;
  }
  .fact {
    padding: 10px 12px;
    border-radius: 8px;
    background: var(--el-fill-color-lighter);
    margin-bottom: 8px;
  }
  .fk {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .fv {
    font-size: 22px;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .fh {
    color: var(--el-text-color-placeholder);
    font-size: 11px;
  }
  .note {
    margin-top: 12px;
  }
</style>
