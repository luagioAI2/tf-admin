<template>
  <div>
    <el-card shadow="never" class="panel" v-loading="loading">
      <template #header>
        <div class="panel-head">
          <span>入驻状态</span>
          <el-tag v-if="app" :type="tagType" size="small">{{ statusText }}</el-tag>
        </div>
      </template>

      <!-- 没申请过：`data` 是 null，不是错误。
           前端拿到 null 就说明「这个人通过了 App 的登录，但没有商家身份」——
           正常情况下登录那一步就会被拒，走到这里说明审核被撤销过。 -->
      <el-empty
        v-if="!loading && !app"
        description="这个账号还没有商家申请记录"
        :image-size="90"
      >
      </el-empty>

      <template v-else-if="app">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="店铺名">
            {{ app.name || '—' }}
          </el-descriptions-item>
          <el-descriptions-item label="联系方式">
            {{ app.contact || '—' }}
          </el-descriptions-item>
          <el-descriptions-item label="申请说明" :span="2">
            {{ app.note || '—' }}
          </el-descriptions-item>
          <el-descriptions-item label="审核时间">
            {{ formatTs(app.reviewed_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="后台地址">
            <el-link
              v-if="app.admin_url"
              type="primary"
              :href="app.admin_url"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ app.admin_url }} ↗
            </el-link>
            <span v-else class="muted">—</span>
          </el-descriptions-item>
          <el-descriptions-item
            v-if="app.review_note"
            label="审核备注"
            :span="2"
          >
            {{ app.review_note }}
          </el-descriptions-item>
        </el-descriptions>

        <el-alert
          v-if="app.status === 'rejected'"
          type="error"
          :closable="false"
          class="hint"
          :title="'申请被驳回：' + (app.review_note || '没有写原因')"
        />
      </template>
    </el-card>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { getMerchantStatus } from '@/api/portal'
  import { formatDate } from '@/utils/format'

  defineOptions({ name: 'MerchantStatus' })

  const loading = ref(false)
  const app = ref(null)

  const STATUS_TEXT = {
    pending: '审核中',
    approved: '已通过',
    rejected: '已驳回'
  }
  const statusText = computed(() => STATUS_TEXT[app.value?.status] ?? app.value?.status ?? '—')
  const tagType = computed(
    () =>
      ({ approved: 'success', pending: 'warning', rejected: 'danger' })[app.value?.status] ??
      'info'
  )

  /* 平台的时间戳是 **Unix 秒**，`formatDate` 吃毫秒。传秒会得到 1970 年且不报错。 */
  const formatTs = (ts) => (ts ? formatDate(ts * 1000) : '—')

  const load = async () => {
    loading.value = true
    try {
      const res = await getMerchantStatus()
      if (res.code === 0) app.value = res.data ?? null
    } catch (e) {
      ElMessage.error('状态没拉到：' + (e?.message ?? e))
    } finally {
      loading.value = false
    }
  }

  onMounted(load)
</script>

<style scoped>
  .panel-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .muted {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .center {
    text-align: center;
  }
  .hint {
    margin-top: 12px;
  }
</style>
