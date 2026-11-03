<template>
  <div>
    <div class="gva-search-box">
      <el-alert type="info" :closable="false" show-icon style="margin-bottom: 12px">
        <template #title>
          这里是 <b>App 的用户</b>（表 <code>user</code>）。<b>家长和孩子是同一个账号</b> ——
          家长只是在这行上多设了一个 4 位密码（进家长区用），所以没有「家长列表」，
          只有一张用户表，最多多一列「有没有设家长锁」。
          菜单里另一个「用户管理」管的是 <code>sys_users</code>（后台管理员），
          改错那边等于把自己的管理员踢掉。
        </template>
      </el-alert>
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="手机号">
          <el-input
            v-model="searchInfo.mobile"
            placeholder="支持片段，如 138"
            clearable
            style="width: 180px"
          />
        </el-form-item>
        <el-form-item label="昵称">
          <el-input
            v-model="searchInfo.nickname"
            placeholder="支持片段"
            clearable
            style="width: 160px"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select
            v-model="searchInfo.status"
            placeholder="全部"
            clearable
            style="width: 120px"
          >
            <el-option label="正常" value="1" />
            <el-option label="禁用" value="0" />
            <el-option label="封禁" value="2" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSubmit">查询</el-button>
          <el-button icon="refresh" @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <div class="gva-btn-list">
        <span class="dim">共 {{ total }} 个账号</span>
      </div>
      <el-table :data="tableData" style="width: 100%" row-key="id">
        <el-table-column align="left" label="ID" prop="id" width="70" />
        <el-table-column align="left" label="头像" width="70">
          <template #default="scope">
            <span class="avatar">{{ scope.row.avatar || '👤' }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="昵称" min-width="130">
          <template #default="scope">
            <span>{{ scope.row.nickname || '—' }}</span>
            <el-tag v-if="scope.row.is_guest" size="small" type="info" style="margin-left: 6px">
              游客
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column align="left" label="手机号" width="150">
          <template #default="scope">
            <!-- 这里是**明文**手机号。C 端只显示 138****8000，
                 明文只给有权限的运营看，别复制到工单 / 群里。 -->
            <span class="mono">{{ scope.row.mobile || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="状态" width="90">
          <template #default="scope">
            <el-tag :type="statusTagType(scope.row.status)">
              {{ statusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column align="left" label="积分" width="90">
          <template #default="scope">
            <span class="num">{{ scope.row.points ?? 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="兑换券" width="90">
          <template #default="scope">
            <span class="num">{{ scope.row.coupon ?? 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="钻石" width="90">
          <template #default="scope">
            <span class="num">{{ scope.row.diamond ?? 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="家长密码" width="100">
          <template #default="scope">
            <!-- 家长不是另一种账号：同一个用户多设一个 4 位密码就能进家长区。
                 pin_known=false 表示 `user_settings` 表还没建（老库），
                 这时画「—」而不是「未设」—— 把「不知道」画成「没有」是在撒谎。 -->
            <el-tag v-if="scope.row.pin_known === false" size="small" type="info">—</el-tag>
            <el-tag v-else-if="scope.row.has_pin" size="small" type="success">已设置</el-tag>
            <el-tag v-else size="small" type="info" effect="plain">未设</el-tag>
          </template>
        </el-table-column>
        <el-table-column align="left" label="注册时间" width="170">
          <template #default="scope">
            {{ formatTs(scope.row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column align="left" label="最近登录" width="170">
          <template #default="scope">
            {{ formatTs(scope.row.last_login_at) }}
          </template>
        </el-table-column>
        <el-table-column align="left" label="操作" width="150" fixed="right">
          <template #default="scope">
            <el-button type="primary" link @click="openDetail(scope.row)">
              详情 / 流水
            </el-button>
            <el-button type="warning" link @click="openAdjust(scope.row)">
              调余额
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="gva-pagination">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handleCurrentChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 详情 + 账本 -->
    <el-dialog v-model="detailVisible" title="用户详情" width="760px" @closed="resetDetail">
      <div v-if="detail.user" class="detail">
        <div class="kv">
          <span class="k">uid</span><span class="mono">{{ detail.user.id }}</span>
          <span class="k">昵称</span><span>{{ detail.user.nickname || '—' }}</span>
          <span class="k">手机号</span><span class="mono">{{ detail.user.mobile || '—' }}</span>
          <span class="k">语言</span><span>{{ detail.user.language || '—' }}</span>
          <span class="k">注册</span><span>{{ formatTs(detail.user.created_at) }}</span>
          <span class="k">最近登录</span><span>{{ formatTs(detail.user.last_login_at) }}</span>
          <span class="k">注册 IP</span><span class="mono">{{ detail.user.register_ip || '—' }}</span>
          <span class="k">登录 IP</span><span class="mono">{{ detail.user.last_login_ip || '—' }}</span>
          <span class="k">家长密码</span><span>{{ pinText(detail.user) }}</span>
        </div>

        <div class="wallet">
          <el-tag type="success">积分 {{ detail.wallet?.points ?? 0 }}</el-tag>
          <el-tag type="warning">兑换券 {{ detail.wallet?.coupon ?? 0 }}</el-tag>
          <el-tag>钻石 {{ detail.wallet?.diamond ?? 0 }}</el-tag>
        </div>

        <h4 class="sub">账本流水（最近的在前）</h4>
        <el-table :data="ledger.rows" size="small" style="width: 100%">
          <el-table-column label="时间" width="160">
            <template #default="scope">{{ formatTs(scope.row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="币种" prop="currency" width="80" />
          <el-table-column label="变动" width="100">
            <template #default="scope">
              <span :class="scope.row.delta >= 0 ? 'plus' : 'minus'">
                {{ scope.row.delta >= 0 ? '+' : '' }}{{ scope.row.delta }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="余额" prop="balance_after" width="90" />
          <el-table-column label="来源" prop="source" width="130" />
          <el-table-column label="备注" prop="memo" min-width="160">
            <template #default="scope">
              <span class="dim">{{ scope.row.memo || '—' }}</span>
            </template>
          </el-table-column>
        </el-table>
        <div class="gva-pagination">
          <el-pagination
            v-model:current-page="ledger.page"
            :page-size="ledger.pageSize"
            :total="ledger.total"
            layout="total, prev, pager, next"
            @current-change="loadLedger"
          />
        </div>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 调余额 -->
    <el-dialog v-model="adjustVisible" title="调整余额" width="520px" @closed="resetAdjust">
      <el-alert type="warning" :closable="false" show-icon style="margin-bottom: 12px">
        <template #title>
          这会写一条 <code>manual_adjust</code> 账本流水，备注必填 ——
          将来对账时「这笔钱是谁为什么加的」只能靠它。
        </template>
      </el-alert>
      <el-form :model="adjustForm" label-width="90px">
        <el-form-item label="用户">
          <span>#{{ adjustForm.uid }} · {{ adjustForm.nickname || '—' }}</span>
        </el-form-item>
        <el-form-item label="币种">
          <el-select v-model="adjustForm.currency" style="width: 160px">
            <el-option label="积分" value="points" />
            <el-option label="兑换券" value="coupon" />
            <el-option label="钻石" value="diamond" />
          </el-select>
        </el-form-item>
        <el-form-item label="变动量">
          <el-input-number v-model="adjustForm.delta" :step="10" :precision="0" />
          <div class="hint">正数是加，负数是扣。填 0 服务端会拒。</div>
        </el-form-item>
        <el-form-item label="备注" required>
          <el-input
            v-model="adjustForm.memo"
            type="textarea"
            :rows="2"
            maxlength="128"
            show-word-limit
            placeholder="例：补发 10-30 打卡漏发的积分"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="adjustVisible = false">取消</el-button>
        <el-button type="warning" :loading="submitting" @click="submitAdjust">
          确认调整
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
  import { ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    adjustPlatformUserBalance,
    getPlatformUserDetail,
    getPlatformUserLedger,
    getPlatformUserList
  } from '@/api/tfUser'
  import { formatDate } from '@/utils/format'

  defineOptions({ name: 'MallPlatformUser' })

  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const searchInfo = ref({ mobile: '', nickname: '', status: '' })

  const detailVisible = ref(false)
  const detail = ref({ user: null, wallet: null })
  const ledger = ref({ rows: [], page: 1, pageSize: 10, total: 0 })

  const adjustVisible = ref(false)
  const submitting = ref(false)
  const adjustForm = ref({ uid: 0, nickname: '', currency: 'points', delta: 0, memo: '' })

  /** 0 禁用 / 1 正常 / 2 封禁（`entities.User` 里的常量） */
  const STATUS_TEXT = { 0: '禁用', 1: '正常', 2: '封禁' }
  const statusText = (v) => STATUS_TEXT[v] ?? `未知(${v})`
  const statusTagType = (v) => (v === 1 ? 'success' : v === 0 ? 'info' : 'danger')

  /**
   * 平台的时间戳是 **Unix 秒**，`formatDate` 吃的是毫秒。
   * 直接传秒会得到 1970 年，而且**不报错**。
   */
  const formatTs = (ts) => (ts ? formatDate(ts * 1000) : '—')

  /** 家长锁：unknown 和「没设」是两件事，别合并 */
  const pinText = (u) => {
    if (!u || u.pin_known === false) return '—（查不到）'
    return u.has_pin ? '已设置（家长区可进）' : '未设置（谁都能进家长区）'
  }

  const getTableData = async () => {
    const payload = {
      page: page.value,
      pageSize: pageSize.value,
      mobile: searchInfo.value.mobile?.trim() || '',
      nickname: searchInfo.value.nickname?.trim() || ''
    }
    /* ⚠️ 状态要**整个省掉**而不是传空串 / 0：
       服务端是 `*int`，传 0 等于「只看被禁用的」——
       下拉清空之后列表突然只剩禁用账号，看起来像「用户全没了」。 */
    if (searchInfo.value.status !== '' && searchInfo.value.status !== undefined) {
      payload.status = Number(searchInfo.value.status)
    }
    const res = await getPlatformUserList(payload)
    if (res.code === 0) {
      tableData.value = res.data.list ?? []
      total.value = res.data.total ?? 0
    }
  }

  const onSubmit = () => {
    page.value = 1
    getTableData()
  }
  const onReset = () => {
    searchInfo.value = { mobile: '', nickname: '', status: '' }
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

  const loadLedger = async () => {
    const id = detail.value.user?.id
    if (!id) return
    const res = await getPlatformUserLedger({
      id,
      page: ledger.value.page,
      pageSize: ledger.value.pageSize
    })
    if (res.code === 0) {
      ledger.value.rows = res.data.list ?? []
      ledger.value.total = res.data.total ?? 0
    }
  }

  const openDetail = async (row) => {
    detail.value = { user: row, wallet: null }
    ledger.value = { rows: [], page: 1, pageSize: 10, total: 0 }
    detailVisible.value = true
    // 列表里的余额是列表那一刻的；详情重新取一次（可能刚被别处改过）
    const res = await getPlatformUserDetail(row.id)
    if (res.code === 0) {
      /* ⚠️ 详情接口**不返回**家长锁状态（那是列表里批量算出来的），
         整份覆盖会让 `has_pin` 变 undefined —— 模板会把它画成「未设」，
         也就是「明明设了密码，详情里说没设」。保留列表那一份。 */
      detail.value = {
        ...res.data,
        user: { ...res.data.user, has_pin: row.has_pin, pin_known: row.pin_known }
      }
    }
    loadLedger()
  }
  const resetDetail = () => {
    detail.value = { user: null, wallet: null }
    ledger.value = { rows: [], page: 1, pageSize: 10, total: 0 }
  }

  const openAdjust = (row) => {
    adjustForm.value = {
      uid: row.id,
      nickname: row.nickname,
      currency: 'points',
      delta: 0,
      memo: ''
    }
    adjustVisible.value = true
  }
  const resetAdjust = () => {
    adjustForm.value = { uid: 0, nickname: '', currency: 'points', delta: 0, memo: '' }
  }

  const submitAdjust = async () => {
    const f = adjustForm.value
    // 服务端也会拒，但让运营等一个来回才知道「少了什么」是没必要的
    if (!f.delta) {
      ElMessage.warning('变动量不能是 0')
      return
    }
    if (!f.memo.trim()) {
      ElMessage.warning('备注必须填：将来对账只能靠它')
      return
    }
    const word = f.delta > 0 ? '增加' : '扣减'
    try {
      await ElMessageBox.confirm(
        `确认给 #${f.uid} ${word} ${Math.abs(f.delta)} ${currencyText(f.currency)}？`,
        '确认调整',
        { type: 'warning' }
      )
    } catch {
      return // 点了取消
    }
    submitting.value = true
    try {
      const res = await adjustPlatformUserBalance({
        uid: f.uid,
        currency: f.currency,
        delta: f.delta,
        memo: f.memo.trim()
      })
      if (res.code === 0) {
        ElMessage.success('已调整')
        adjustVisible.value = false
        getTableData()
      }
    } finally {
      // ⚠️ 放 finally：失败时按钮必须恢复，否则「提交中…」永远转下去
      submitting.value = false
    }
  }

  const currencyText = (v) =>
    ({ points: '积分', coupon: '兑换券', diamond: '钻石' })[v] ?? v

  getTableData()
</script>

<style scoped>
  .mono {
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  }
  .dim {
    color: var(--el-text-color-secondary);
  }
  .num {
    font-variant-numeric: tabular-nums;
  }
  .avatar {
    font-size: 20px;
  }
  .plus {
    color: var(--el-color-success);
    font-variant-numeric: tabular-nums;
  }
  .minus {
    color: var(--el-color-danger);
    font-variant-numeric: tabular-nums;
  }
  .detail .kv {
    display: grid;
    grid-template-columns: 80px 1fr 80px 1fr;
    gap: 8px 12px;
    margin-bottom: 12px;
  }
  .detail .kv .k {
    color: var(--el-text-color-secondary);
  }
  .detail .wallet {
    display: flex;
    gap: 8px;
    margin-bottom: 16px;
  }
  .detail .sub {
    margin: 0 0 8px;
    font-weight: 600;
  }
  .hint {
    margin-left: 8px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
</style>
