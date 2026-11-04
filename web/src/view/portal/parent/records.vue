<template>
  <div>
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-input
          v-model="keyword"
          placeholder="搜兑换的东西"
          clearable
          class="kw"
          @keyup.enter="search"
          @clear="search"
        />
        <el-button type="primary" @click="search">查询</el-button>
      </div>

      <el-table :data="rows" v-loading="loading" row-key="record_id" class="tbl">
        <el-table-column label="时间" width="160">
          <template #default="{ row }">{{ formatTs(row.created_at) }}</template>
        </el-table-column>

        <el-table-column label="兑换的东西" min-width="200">
          <template #default="{ row }">
            <span class="emoji">{{ row.emoji || '🎁' }}</span>
            <span class="name">{{ row.name }}</span>
            <span v-if="row.quantity > 1" class="muted">×{{ row.quantity }}</span>
          </template>
        </el-table-column>

        <el-table-column label="花了" width="150">
          <template #default="{ row }">
            <!-- 兑换可能是「券 + 积分」一起付，所以两栏分开显示；
                 只显示合计的话，对不上账时看不出是哪一种少了 -->
            <span v-if="row.coupon_paid" class="muted">券 {{ row.coupon_paid }}</span>
            <span v-if="row.points_paid" class="muted ml">
              积分 {{ row.points_paid }}
            </span>
            <span v-if="!row.coupon_paid && !row.points_paid" class="muted">—</span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.cancelled" type="info" size="small">已取消</el-tag>
            <el-tag v-else-if="row.fulfilled" type="success" size="small">已发放</el-tag>
            <el-tag v-else type="warning" size="small">待发放</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="备注" min-width="160">
          <template #default="{ row }">
            <span class="muted">
              {{ row.cancelled ? row.cancel_reason || '已取消' : row.fulfilled_note || '—' }}
            </span>
          </template>
        </el-table-column>
      </el-table>

      <el-empty
        v-if="!loading && !rows.length"
        description="还没有兑换记录。"
        :image-size="80"
      />

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
  import { getParentRecords } from '@/api/portal'
  import { formatDate } from '@/utils/format'

  defineOptions({ name: 'ParentRecords' })

  const rows = ref([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(10)
  const keyword = ref('')
  const loading = ref(false)

  /* 平台的时间戳是 **Unix 秒**，`formatDate` 吃毫秒。传秒会得到 1970 年且不报错。 */
  const formatTs = (ts) => (ts ? formatDate(ts * 1000) : '—')

  const load = async () => {
    loading.value = true
    try {
      const res = await getParentRecords({
        page: page.value,
        pageSize: pageSize.value,
        keyword: keyword.value
      })
      if (res.code === 0) {
        rows.value = res.data?.list ?? []
        total.value = res.data?.total ?? 0
      }
    } catch (e) {
      ElMessage.error('兑换记录没拉到：' + (e?.message ?? e))
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
  .ml {
    margin-left: 6px;
  }
  .tbl {
    width: 100%;
  }
</style>
