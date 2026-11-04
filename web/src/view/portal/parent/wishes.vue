<template>
  <div>
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-select v-model="status" placeholder="全部状态" clearable class="sel" @change="search">
          <el-option
            v-for="(label, key) in STATUS_TEXT"
            :key="key"
            :label="label"
            :value="key"
          />
        </el-select>
        <el-input
          v-model="keyword"
          placeholder="搜心愿名字"
          clearable
          class="kw"
          @keyup.enter="search"
          @clear="search"
        />
        <el-button type="primary" @click="search">查询</el-button>
      </div>

      <el-table :data="rows" v-loading="loading" row-key="order_id" class="tbl">
        <el-table-column label="时间" width="160">
          <template #default="{ row }">{{ formatTs(row.created_at) }}</template>
        </el-table-column>

        <el-table-column label="心愿" min-width="200">
          <template #default="{ row }">
            <span class="emoji">{{ row.item_emoji || '🎁' }}</span>
            <span class="name">{{ row.item_name }}</span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="130">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">
              {{ statusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="券" width="70" align="right">
          <template #default="{ row }">{{ row.voucher_price ?? 0 }}</template>
        </el-table-column>

        <el-table-column label="说明" min-width="180">
          <template #default="{ row }">
            <!-- 取消要看理由，已发货要看单号；两样都没有就说明还在等家长动手 -->
            <span v-if="row.status === 'cancelled'" class="muted">
              {{ row.cancel_reason || '已取消' }}
            </span>
            <span v-else-if="row.tracking_no" class="muted">
              快递 {{ row.tracking_no }}
            </span>
            <span v-else-if="row.opened_game_id" class="muted">
              开出 {{ row.opened_game_id }}
            </span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <!-- 「去下单」只在这两个状态下有意义：券已经扣了，等家长去买。
                 链接来自商品表（订单本身不存 URL）。 -->
            <el-link
              v-if="needsBuy(row) && row.buy_url"
              type="primary"
              :href="row.buy_url"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ row.buy_label || '去下单' }} ↗
            </el-link>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
      </el-table>

      <div class="gva-pagination">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="total"
          layout="total, sizes, prev, pager, next"
          @current-change="(v) => { page = v; load() }"
          @size-change="(v) => { pageSize = v; page = 1; load() }"
        />
      </div>
    </div>

  </div>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { getParentWishes } from '@/api/portal'
  import { formatDate } from '@/utils/format'

  defineOptions({ name: 'ParentWishes' })

  const rows = ref([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(10)
  const status = ref('')
  const keyword = ref('')
  const loading = ref(false)

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

  // 券已经押在平台、等家长去买的两个状态（和 `WishStatus.HasHeldVouchers()` 一致）
  const needsBuy = (row) => row.status === 'committed' || row.status === 'confirmed'

  /* 平台的时间戳是 **Unix 秒**，`formatDate` 吃毫秒。传秒会得到 1970 年且不报错。 */
  const formatTs = (ts) => (ts ? formatDate(ts * 1000) : '—')

  const load = async () => {
    loading.value = true
    try {
      const res = await getParentWishes({
        page: page.value,
        pageSize: pageSize.value,
        status: status.value || '',
        keyword: keyword.value
      })
      if (res.code === 0) {
        rows.value = res.data?.list ?? []
        total.value = res.data?.total ?? 0
      }
    } catch (e) {
      ElMessage.error('心愿单没拉到：' + (e?.message ?? e))
    } finally {
      loading.value = false
    }
  }

  const search = () => {
    page.value = 1
    load()
  }

  onMounted(load)
</script>

<style scoped>
  .sel {
    width: 150px;
    margin-right: 8px;
  }
  .kw {
    width: 200px;
    margin-right: 8px;
  }
  .emoji {
    margin-right: 4px;
  }
  .name {
    font-weight: 500;
  }
  .muted {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .tbl {
    width: 100%;
  }
  .hint {
    margin-top: 12px;
  }
</style>
