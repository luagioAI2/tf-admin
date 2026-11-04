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
        <!-- 账号类型：游客是 App 一启动就自动建的临时账号，id 最大 ⇒
             永远排在最前面。不筛的话第一页全是一屏「小客人」（0 分 0 流水），
             真正要看的用户被挤到后面几页去。 -->
        <el-form-item label="账号类型">
          <el-select
            v-model="searchInfo.isGuest"
            placeholder="全部"
            clearable
            style="width: 130px"
          >
            <el-option label="正式账号" value="0" />
            <el-option label="游客（临时）" value="1" />
          </el-select>
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
        <!-- 列宽是**量出来的**，不是拍的：容器 clientW=1140，
             11 列固定宽合计必须 ≤1022，剩下给「昵称」这个 flex 列，
             否则总宽超过容器 ⇒ 固定右列压住「注册时间」的右边。
             实测头像 emoji 20px、手机号 11 位等宽数字 ≈86px，都能再收。 -->
        <el-table-column align="left" label="头像" width="64">
          <template #default="scope">
            <span class="avatar">{{ scope.row.avatar || '👤' }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="昵称" min-width="118">
          <template #default="scope">
            <span>{{ scope.row.nickname || '—' }}</span>
            <el-tag v-if="scope.row.is_guest" size="small" type="info" style="margin-left: 6px">
              游客
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column align="left" label="手机号" width="128">
          <template #default="scope">
            <!-- 这里是**明文**手机号。C 端只显示 138****8000，
                 明文只给有权限的运营看，别复制到工单 / 群里。 -->
            <span class="mono">{{ scope.row.mobile || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="状态" width="80">
          <template #default="scope">
            <el-tag :type="statusTagType(scope.row.status)">
              {{ statusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column align="left" label="积分" width="80">
          <template #default="scope">
            <span class="num">{{ scope.row.points ?? 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="兑换券" width="80">
          <template #default="scope">
            <span class="num">{{ scope.row.coupon ?? 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="钻石" width="80">
          <template #default="scope">
            <span class="num">{{ scope.row.diamond ?? 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="家长密码" width="90">
          <template #default="scope">
            <!-- 家长不是另一种账号：同一个用户多设一个 4 位密码就能进家长区。
                 pin_known=false 表示 `user_settings` 表还没建（老库），
                 这时画「—」而不是「未设」—— 把「不知道」画成「没有」是在撒谎。 -->
            <el-tag v-if="scope.row.pin_known === false" size="small" type="info">—</el-tag>
            <el-tag v-else-if="scope.row.has_pin" size="small" type="success">已设置</el-tag>
            <el-tag v-else size="small" type="info" effect="plain">未设</el-tag>
          </template>
        </el-table-column>
        <!-- 19 个字符（2026-10-29 23:05:06）在 155px 会折行，172px + nowrap 才一行 -->
        <el-table-column align="left" label="注册时间" width="172" class-name="nowrap-col">
          <template #default="scope">
            {{ formatTs(scope.row.created_at) }}
          </template>
        </el-table-column>
        <!-- ⚠️ 太窄会把「详情 / 流水」和「调余额」折成两行，看着像坏了；
             175px + nowrap 刚好一行 -->
        <el-table-column align="left" label="操作" width="175" fixed="right" class-name="op-col">
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
          <!-- 账号类型：列表里那个「游客」小标签太容易被忽略，而这一屏是
               「调余额 / 重置密码」之前必须看的地方 —— 游客没有手机号，
               给他重置密码是无效操作（他下次启动还是按设备号自动进）。
               `is_guest` / `mobile_verified` 服务端本来就在返回，以前只是没显示。 -->
          <span class="k">账号类型</span>
          <span>
            <el-tag v-if="detail.user.is_guest" size="small" type="info">游客（临时账号）</el-tag>
            <el-tag v-else size="small" type="success">正式账号</el-tag>
            <span v-if="detail.user.is_guest" class="dim" style="margin-left: 8px">
              没绑手机号，进度只在他那台设备上
            </span>
          </span>
          <span class="k">手机号</span>
          <span class="mono">
            {{ detail.user.mobile || '—' }}
            <el-tag
              v-if="detail.user.mobile"
              size="small"
              :type="detail.user.mobile_verified ? 'success' : 'warning'"
              style="margin-left: 6px"
            >
              {{ detail.user.mobile_verified ? '已验证' : '未验证' }}
            </el-tag>
          </span>
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
        <!-- 放详情里而不是列表的操作列：那一列已经 175px 刚好一行，
             再塞一个按钮就会折成两行（看着像布局坏了）。
             而且重置密码要先看清楚是谁，本来就得先开详情。 -->
        <el-button type="danger" plain data-reset-open style="float: left" @click="openReset">
          重置登录密码
        </el-button>
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

    <!-- 重置登录密码 -->
    <el-dialog v-model="resetVisible" title="重置登录密码" width="520px" @closed="resetReset">
      <el-alert type="warning" :closable="false" show-icon style="margin-bottom: 12px">
        <template #title>
          服务端生成一个新密码，<b>只显示这一次</b>（库里只存哈希，关掉就查不到了）。
          重置后这个账号<b>所有设备的登录都会失效</b>，用户得用新密码重新登录。
        </template>
      </el-alert>

      <el-form v-if="!resetResult" :model="resetForm" label-width="90px">
        <el-form-item label="用户">
          <span>#{{ resetForm.uid }} · {{ resetForm.nickname || '—' }}</span>
        </el-form-item>
        <el-form-item label="备注" required>
          <el-input
            v-model="resetForm.memo"
            data-reset-memo
            type="textarea"
            :rows="2"
            maxlength="128"
            show-word-limit
            placeholder="例：工单 #1234 家长忘了密码"
          />
        </el-form-item>
      </el-form>

      <div v-else class="pwd-box">
        <div class="pwd-hint">新密码（抄下来 / 念给用户，关掉就看不到了）</div>
        <div class="pwd" data-reset-password>{{ resetResult }}</div>
        <el-button size="small" data-reset-copy @click="copyPwd">复制</el-button>
      </div>

      <template #footer>
        <template v-if="!resetResult">
          <el-button @click="resetVisible = false">取消</el-button>
          <el-button type="danger" :loading="resetting" data-reset-submit @click="submitReset">
            确认重置
          </el-button>
        </template>
        <el-button v-else type="primary" data-reset-done @click="resetVisible = false">
          我记下了
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
    getPlatformUserList,
    resetPlatformUserPassword
  } from '@/api/tfUser'
  import { formatDate } from '@/utils/format'

  defineOptions({ name: 'MallPlatformUser' })

  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const searchInfo = ref({ mobile: '', nickname: '', status: '', isGuest: '' })

  const detailVisible = ref(false)
  const detail = ref({ user: null, wallet: null })
  const ledger = ref({ rows: [], page: 1, pageSize: 10, total: 0 })

  const adjustVisible = ref(false)
  const submitting = ref(false)
  const adjustForm = ref({ uid: 0, nickname: '', currency: 'points', delta: 0, memo: '' })

  /* 重置登录密码。
     `resetResult` 有值 = 已经重置成功，弹窗切成「显示新密码」那一面。
     ⚠️ 它只在内存里：关掉弹窗（`resetReset`）就清空，再也不显示。 */
  const resetVisible = ref(false)
  const resetting = ref(false)
  const resetResult = ref('')
  const resetForm = ref({ uid: 0, nickname: '', memo: '' })

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
    /* 同样要「整个省掉」：服务端是 `*int`，传 0 等于「只看正式账号」——
       那正是「正式账号」这个选项本身，但**不选**的时候必须是「不筛」。 */
    if (searchInfo.value.isGuest !== '' && searchInfo.value.isGuest !== undefined) {
      payload.is_guest = Number(searchInfo.value.isGuest)
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
    searchInfo.value = { mobile: '', nickname: '', status: '', isGuest: '' }
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

  const openReset = () => {
    const u = detail.value.user
    if (!u) return
    resetForm.value = { uid: u.id, nickname: u.nickname, memo: '' }
    resetResult.value = ''
    resetVisible.value = true
  }
  const resetReset = () => {
    resetForm.value = { uid: 0, nickname: '', memo: '' }
    resetResult.value = ''
    resetting.value = false
  }

  const submitReset = async () => {
    const f = resetForm.value
    // 服务端也拒，但让人等一个来回才知道「少了什么」是没必要的
    if (!f.memo.trim()) {
      ElMessage.warning('备注必须填：将来查「密码怎么被改过」只能靠它')
      return
    }
    try {
      await ElMessageBox.confirm(
        `确认重置 #${f.uid} 的登录密码？这会让他在所有设备上掉线。`,
        '确认重置',
        { type: 'warning' }
      )
    } catch {
      return // 点了取消
    }
    resetting.value = true
    try {
      const res = await resetPlatformUserPassword({ uid: f.uid, memo: f.memo.trim() })
      if (res.code === 0) {
        /* ⚠️ 空密码要**说出来**：界面会停在空白框上，看着像「什么都没发生」，
           而实际上密码已经被改掉了 —— 那时用户手里还是旧密码。 */
        resetResult.value = res.data?.password ?? ''
        if (!resetResult.value) ElMessage.warning('接口没返回新密码，请再重置一次')
      }
      // 失败（游客 / 没手机号 / 无权限）由 request 拦截器弹 msg，这里不用重复
    } finally {
      resetting.value = false
    }
  }

  const copyPwd = async () => {
    try {
      await navigator.clipboard.writeText(resetResult.value)
      ElMessage.success('已复制')
    } catch {
      ElMessage.warning('复制失败，请手动抄写')
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
  /* 操作列 / 时间列都不折行：折了之后一格两行，看着像布局坏了 */
  :deep(.op-col .cell),
  :deep(.nowrap-col .cell) {
    white-space: nowrap;
  }
  .hint {
    margin-left: 8px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .pwd-box {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  .pwd-hint {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  /* 要念得清、抄得对：等宽 + 大字号 + 字间距 */
  .pwd {
    padding: 10px 14px;
    border-radius: 6px;
    background: var(--el-fill-color-light);
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 22px;
    letter-spacing: 2px;
    user-select: all;
  }
</style>
