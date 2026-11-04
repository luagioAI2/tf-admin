<template>
  <div>
    <el-row :gutter="12" class="row">
      <el-col :xs="24" :sm="8">
        <el-card shadow="hover" class="card">
          <div class="k"><span class="ico">💰</span>积分</div>
          <div class="v">{{ n.points ?? 0 }}</div>
          <div class="sub">孩子做任务攒下来的</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="8">
        <el-card shadow="hover" class="card">
          <div class="k"><span class="ico">🎟️</span>兑换券</div>
          <div class="v">{{ n.coupon ?? 0 }}</div>
          <div class="sub">心愿单里扣的是这个</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="8">
        <el-card shadow="hover" class="card">
          <div class="k"><span class="ico">💎</span>钻石</div>
          <div class="v">{{ n.diamond ?? 0 }}</div>
          <div class="sub">特殊奖励</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="panel">
      <template #header>
        <div class="panel-head">
          <span>心愿单</span>
          <el-link type="primary" @click="go('parentWishes')">看全部 →</el-link>
        </div>
      </template>

      <div class="stat-line">
        <span v-for="s in wishStats" :key="s.key" class="stat">
          <b>{{ s.n }}</b>
          <i>{{ s.label }}</i>
        </span>
      </div>

      <el-table :data="n.recent ?? []" v-loading="loading" size="small" class="tbl">
        <el-table-column label="时间" width="160">
          <template #default="{ row }">{{ formatTs(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="心愿" min-width="200">
          <template #default="{ row }">
            <span class="emoji">{{ row.item_emoji || '🎁' }}</span>
            {{ row.item_name }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="130">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">
              {{ statusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="券" width="80" align="right">
          <template #default="{ row }">{{ row.voucher_price ?? 0 }}</template>
        </el-table-column>
      </el-table>

      <el-empty
        v-if="!loading && !(n.recent ?? []).length"
        description="还没有心愿。孩子可以在 App 里许愿。"
        :image-size="80"
      />
    </el-card>

    <el-alert type="info" :closable="false" class="hint">
      <template #title>
        这一屏是<b>只读</b>的。确认、回绝、取消都在 App 里做 —— 那里有完整的
        托管和退款规则；后台再实现一遍，两份规则迟早对不上。
      </template>
    </el-alert>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { ElMessage } from 'element-plus'
  import { getParentOverview } from '@/api/portal'
  import { formatDate } from '@/utils/format'

  defineOptions({ name: 'ParentOverview' })

  const router = useRouter()
  const loading = ref(false)
  const n = ref({})

  /* 状态机和后台「心愿商城」页共用一套口径（`entities/wish.go`）：
     wished（孩子想要）→ committed（确定兑换，扣券）→ confirmed（家长已下单）→ done */
  const STATUS_TEXT = {
    wished: '孩子想要',
    committed: '已兑换待兑现',
    confirmed: '已下单',
    done: '已完成',
    cancelled: '已取消'
  }
  const statusText = (v) => STATUS_TEXT[v] ?? v ?? '—'
  const statusTagType = (v) =>
    v === 'done' ? 'success' : v === 'cancelled' ? 'info' : v === 'wished' ? 'warning' : 'primary'

  /* 各状态的计数来自接口的 `wish` 字典（`{wished: n, committed: n, ...}`）。
     ⚠️ 不能写成静态数组 + `s.n` —— 那个 `n` 在静态数组里根本不存在，
     渲染出来是「想要 待兑现 已下单 已完成」四个字、后面一个数都没有，
     而且**不报错**。 */
  const wishStats = computed(() =>
    [
      { key: 'wished', label: '想要' },
      { key: 'committed', label: '待兑现' },
      { key: 'confirmed', label: '已下单' },
      { key: 'done', label: '已完成' }
    ].map((s) => ({ ...s, n: n.value.wish?.[s.key] ?? 0 }))
  )

  /* 平台的时间戳是 **Unix 秒**，`formatDate` 吃毫秒。
     直接传秒会显示 1970 年，而且不报错。 */
  const formatTs = (ts) => (ts ? formatDate(ts * 1000) : '—')

  const go = (name) => router.push({ name })

  const load = async () => {
    loading.value = true
    try {
      const res = await getParentOverview()
      if (res.code === 0) n.value = res.data ?? {}
    } catch (e) {
      // 静默显示 0 比报错更坏：0 看起来像一个真实的余额
      ElMessage.error('概览没拉到：' + (e?.message ?? e))
    } finally {
      loading.value = false
    }
  }

  onMounted(load)
</script>

<style scoped>
  .row {
    margin-bottom: 12px;
  }
  .card .k {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }
  .card .ico {
    margin-right: 4px;
  }
  .card .v {
    font-size: 30px;
    font-weight: 700;
    line-height: 1.3;
    margin: 4px 0 2px;
  }
  .card .sub {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .panel-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .stat-line {
    display: flex;
    gap: 24px;
    padding: 4px 0 12px;
  }
  .stat {
    display: flex;
    align-items: baseline;
    gap: 6px;
  }
  .stat b {
    font-size: 20px;
  }
  .stat i {
    font-style: normal;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .emoji {
    margin-right: 4px;
  }
  .tbl {
    width: 100%;
  }
  .hint {
    margin-top: 12px;
  }
</style>
