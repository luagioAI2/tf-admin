<template>
  <div>
    <div class="gva-search-box">
      <el-alert
        type="info"
        :closable="false"
        show-icon
        style="margin-bottom: 12px"
      >
        <template #title>
          这一页是唯一能把申请推到「已通过」的地方。
          通过时必须填后台地址 —— 对方 App 里那一屏写的是
          「审核通过后，这里会出现一个后台链接」，不给地址他就永远拿不到入口。
        </template>
      </el-alert>
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="角色">
          <el-select
            v-model="searchInfo.role"
            placeholder="全部"
            clearable
            style="width: 140px"
          >
            <el-option label="家长申请" value="parent" />
            <el-option label="商家申请" value="merchant" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select
            v-model="searchInfo.status"
            placeholder="全部"
            clearable
            style="width: 140px"
          >
            <el-option label="待审核" value="pending" />
            <el-option label="已通过" value="approved" />
            <el-option label="已拒绝" value="rejected" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键词">
          <el-input
            v-model="searchInfo.keyword"
            placeholder="名字 / 联系方式 / 备注"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSubmit">
            查询
          </el-button>
          <el-button icon="refresh" @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <div class="gva-btn-list">
        <span class="pending-hint">
          待审核 <b>{{ pendingCount }}</b> 条
        </span>
      </div>
      <el-table :data="tableData" style="width: 100%" row-key="id">
        <el-table-column align="left" label="ID" prop="id" width="70" />
        <el-table-column align="left" label="角色" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.role === 'parent' ? '' : 'warning'">
              {{ roleText(scope.row.role) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column align="left" label="称呼" prop="name" width="140" />
        <el-table-column align="left" label="联系方式" width="150">
          <template #default="scope">
            <!-- 这里是**明文**手机号（C 端看到的是 138****8000）。
                 明文只给有权限的运营看，别复制到日志/工单里。 -->
            <span class="mono">{{ scope.row.contact || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="说明" prop="note" min-width="160">
          <template #default="scope">
            <span class="note">{{ scope.row.note || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="状态" width="100">
          <template #default="scope">
            <el-tag :type="statusTagType(scope.row.status)">
              {{ statusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column align="left" label="提交时间" width="170">
          <template #default="scope">
            {{ formatTs(scope.row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column align="left" label="审核时间" width="170">
          <template #default="scope">
            {{ formatTs(scope.row.reviewed_at) }}
          </template>
        </el-table-column>
        <el-table-column align="left" label="后台地址" min-width="220">
          <template #default="scope">
            <a
              v-if="scope.row.admin_url"
              :href="scope.row.admin_url"
              target="_blank"
              rel="noopener noreferrer"
              class="mono link"
            >
              {{ scope.row.admin_url }}
            </a>
            <span v-else class="dim">—</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="操作" width="150" fixed="right">
          <template #default="scope">
            <template v-if="scope.row.status === 'pending'">
              <el-button type="success" link @click="openReview(scope.row, true)">
                通过
              </el-button>
              <el-button type="danger" link @click="openReview(scope.row, false)">
                拒绝
              </el-button>
            </template>
            <span v-else class="dim">已处理</span>
          </template>
        </el-table-column>
      </el-table>

      <div class="gva-pagination">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :page-sizes="[10, 30, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handleCurrentChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 审核弹层 -->
    <el-dialog
      v-model="dialogVisible"
      :title="reviewForm.approve ? '通过申请' : '拒绝申请'"
      width="560px"
      @closed="resetReview"
    >
      <el-form :model="reviewForm" label-width="96px">
        <el-form-item label="申请">
          <span>
            #{{ reviewForm.id }} · {{ roleText(reviewForm.role) }} ·
            {{ reviewForm.name }}
          </span>
        </el-form-item>
        <el-form-item v-if="reviewForm.approve" label="后台地址" required>
          <el-input
            v-model="reviewForm.admin_url"
            placeholder="https://..."
            clearable
          />
          <div class="hint">
            对方点「打开后台」就跳这里。地址写错的话，他点进去是 404，
            而界面上写着「已通过」—— 看着像我们把他账号弄丢了。
          </div>
        </el-form-item>
        <el-form-item :label="reviewForm.approve ? '备注' : '拒绝原因'">
          <el-input
            v-model="reviewForm.note"
            type="textarea"
            :rows="2"
            maxlength="200"
            show-word-limit
            :placeholder="
              reviewForm.approve
                ? '可不填'
                : '会显示在对方 App 里（不填就只有「这次没通过」）'
            "
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button
          :type="reviewForm.approve ? 'success' : 'danger'"
          :loading="submitting"
          @click="submitReview"
        >
          {{ reviewForm.approve ? '确认通过' : '确认拒绝' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
  import { ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    getMerchantList,
    getMerchantPendingCount,
    reviewMerchant
  } from '@/api/mallMerchant'
  import { formatDate } from '@/utils/format'

  defineOptions({ name: 'MallMerchant' })

  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const pendingCount = ref(0)
  const searchInfo = ref({ role: '', status: '', keyword: '' })

  const dialogVisible = ref(false)
  const submitting = ref(false)
  const reviewForm = ref({
    id: 0,
    role: '',
    name: '',
    approve: true,
    admin_url: '',
    note: ''
  })

  const ROLE_TEXT = { parent: '家长', merchant: '商家' }
  const STATUS_TEXT = { pending: '待审核', approved: '已通过', rejected: '已拒绝' }
  const roleText = (v) => ROLE_TEXT[v] ?? v ?? '—'
  const statusText = (v) => STATUS_TEXT[v] ?? v ?? '—'
  const statusTagType = (v) =>
    v === 'approved' ? 'success' : v === 'rejected' ? 'danger' : 'warning'

  /**
   * 平台的时间戳是 **Unix 秒**，而 `formatDate` 吃的是毫秒。
   * 直接传秒会得到 1970 年（`new Date(1793284301)`），而且不报错。
   */
  const formatTs = (ts) => (ts ? formatDate(ts * 1000) : '—')

  const getTableData = async () => {
    const res = await getMerchantList({
      page: page.value,
      pageSize: pageSize.value,
      ...searchInfo.value
    })
    if (res.code === 0) {
      tableData.value = res.data.list ?? []
      total.value = res.data.total ?? 0
      refreshPending()
    }
  }

  const refreshPending = async () => {
    const res = await getMerchantPendingCount()
    if (res.code === 0) pendingCount.value = res.data ?? 0
  }

  const onSubmit = () => {
    page.value = 1
    getTableData()
  }
  const onReset = () => {
    searchInfo.value = { role: '', status: '', keyword: '' }
    page.value = 1
    getTableData()
  }
  const handleSizeChange = (val) => {
    pageSize.value = val
    getTableData()
  }
  const handleCurrentChange = (val) => {
    page.value = val
    getTableData()
  }

  const openReview = (row, approve) => {
    reviewForm.value = {
      id: row.id,
      role: row.role,
      name: row.name,
      approve,
      admin_url: row.admin_url ?? '',
      note: ''
    }
    dialogVisible.value = true
  }
  const resetReview = () => {
    reviewForm.value = {
      id: 0,
      role: '',
      name: '',
      approve: true,
      admin_url: '',
      note: ''
    }
  }

  const submitReview = async () => {
    const f = reviewForm.value
    if (f.approve) {
      const url = f.admin_url.trim()
      // 服务端也会拒，但让用户等一个来回才知道「少了什么」是没必要的。
      // 两条都要：只判空的话，填了 `merchant.example.com` 会先提交、
      // 再弹一个服务端的错，而输入框还留在那儿。
      if (!url) {
        ElMessage.warning('通过审核必须填后台地址')
        return
      }
      if (!/^https?:\/\//i.test(url)) {
        ElMessage.warning('后台地址要以 http:// 或 https:// 开头')
        return
      }
    }
    submitting.value = true
    try {
      const res = await reviewMerchant({
        id: f.id,
        approve: f.approve,
        admin_url: f.approve ? f.admin_url.trim() : '',
        note: f.note.trim()
      })
      if (res.code === 0) {
        ElMessage.success('审核完成')
        dialogVisible.value = false
        getTableData()
      }
    } finally {
      // ⚠️ 放 finally：失败时按钮必须恢复，否则「提交中…」永远转下去
      submitting.value = false
    }
  }

  getTableData()
</script>

<style scoped>
  .pending-hint {
    font-size: 13px;
    color: #606266;
  }
  .pending-hint b {
    color: #e6a23c;
    font-size: 15px;
  }
  .mono {
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 12px;
  }
  .link {
    color: var(--el-color-primary);
    word-break: break-all;
  }
  .dim {
    color: #c0c4cc;
  }
  .note {
    color: #606266;
  }
  .hint {
    margin-top: 6px;
    font-size: 12px;
    line-height: 1.6;
    color: #909399;
  }
</style>
