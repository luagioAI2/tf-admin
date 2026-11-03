<template>
  <div>
    <el-alert type="info" :closable="false" show-icon class="tip">
      <template #title>
        心愿商品是<b>全站一份</b>（不像兑换商品那样每个孩子一份），
        所以这里没有「所属」，看到的就是所有人共享的那份清单。
        后台只能改 <b>外链 / 外链文字 / 上下架 / 排序</b>；
        新建商品和「改订单状态」是孩子和家长在 App 里做的事，这里没有入口
        —— 能改状态就等于能凭空扣券、退券。
      </template>
    </el-alert>

    <el-tabs v-model="tab" class="tabs">
      <!-- ==================== ① 商品 ==================== -->
      <el-tab-pane label="心愿商品" name="item">
        <div class="gva-search-box">
          <el-form :inline="true" :model="itemSearch">
            <el-form-item label="分类">
              <el-select v-model="itemSearch.kind" placeholder="全部" clearable style="width: 140px">
                <el-option v-for="(m, k) in KIND_META" :key="k" :label="m.label" :value="k" />
              </el-select>
            </el-form-item>
            <el-form-item label="状态">
              <el-select
                v-model="itemSearch.archived"
                placeholder="全部"
                clearable
                style="width: 120px"
              >
                <el-option label="在架" :value="false" />
                <el-option label="已下架" :value="true" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" icon="search" @click="onItemSearch">查询</el-button>
              <el-button icon="refresh" @click="onItemReset">重置</el-button>
            </el-form-item>
          </el-form>
        </div>

        <div class="gva-table-box">
          <div class="gva-btn-list">
            <span class="dim">共 {{ itemTotal }} 个商品</span>
          </div>
          <el-table :data="items" style="width: 100%" row-key="item_id">
            <el-table-column align="left" label="商品" min-width="220">
              <template #default="scope">
                <span class="emoji">{{ scope.row.emoji || '🎁' }}</span>
                <span class="name">{{ scope.row.name }}</span>
                <div v-if="scope.row.note" class="note">{{ scope.row.note }}</div>
                <div v-if="scope.row.notice" class="note faint">须知：{{ scope.row.notice }}</div>
              </template>
            </el-table-column>
            <el-table-column align="left" label="分类" width="90">
              <template #default="scope">{{ kindText(scope.row.kind) }}</template>
            </el-table-column>
            <el-table-column align="left" label="参考价" width="100">
              <template #default="scope">
                <!-- ⚠️ 这个字段是**人民币分**，不是券。0 表示「没标价」（游戏类都是 0） -->
                <span v-if="scope.row.price_cents > 0" class="num">
                  ¥{{ (scope.row.price_cents / 100).toFixed(2) }}
                </span>
                <span v-else class="dim">—</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="合作方" width="110">
              <template #default="scope">
                <span v-if="scope.row.partner_name">{{ scope.row.partner_name }}</span>
                <span v-else class="dim">—</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="外链" min-width="180">
              <template #default="scope">
                <a
                  v-if="scope.row.buy_url"
                  :href="scope.row.buy_url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="link"
                >
                  {{ scope.row.buy_label || '（没写文字）' }}
                </a>
                <span v-else class="dim">—</span>
                <div v-if="scope.row.buy_url" class="note faint mono">
                  {{ shortUrl(scope.row.buy_url) }}
                </div>
              </template>
            </el-table-column>
            <el-table-column align="left" label="排序" width="70">
              <template #default="scope">
                <span class="num">{{ scope.row.sort_order }}</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="状态" width="90">
              <template #default="scope">
                <el-tag :type="scope.row.archived ? 'info' : 'success'" size="small">
                  {{ scope.row.archived ? '已下架' : '在架' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column align="left" label="更新于" width="160">
              <template #default="scope">{{ formatTs(scope.row.updated_at) }}</template>
            </el-table-column>
            <el-table-column align="left" label="操作" width="110" fixed="right">
              <template #default="scope">
                <el-button type="primary" link @click="openEdit(scope.row)">改外链</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="gva-pagination">
            <el-pagination
              :current-page="itemPage"
              :page-size="itemPageSize"
              :page-sizes="[10, 30, 50, 100]"
              :total="itemTotal"
              layout="total, sizes, prev, pager, next, jumper"
              @current-change="onItemPage"
              @size-change="onItemSize"
            />
          </div>
        </div>
      </el-tab-pane>

      <!-- ==================== ② 心愿单 ==================== -->
      <el-tab-pane label="心愿单" name="order">
        <div class="gva-search-box">
          <el-form :inline="true" :model="ordSearch">
            <el-form-item label="状态">
              <el-select v-model="ordSearch.status" placeholder="全部" clearable style="width: 150px">
                <el-option
                  v-for="(t, v) in STATUS_TEXT"
                  :key="v"
                  :label="t"
                  :value="v"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="所属">
              <el-select
                v-model="ordSearch.uid"
                placeholder="全部用户"
                clearable
                filterable
                style="width: 200px"
              >
                <el-option v-for="u in users" :key="u.id" :label="userLabel(u)" :value="u.id" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" icon="search" @click="onOrdSearch">查询</el-button>
              <el-button icon="refresh" @click="onOrdReset">重置</el-button>
            </el-form-item>
          </el-form>
        </div>

        <div class="gva-table-box">
          <div class="gva-btn-list">
            <span class="dim">共 {{ ordTotal }} 条</span>
          </div>
          <el-table :data="orders" style="width: 100%" row-key="order_id">
            <el-table-column align="left" label="时间" width="160">
              <template #default="scope">{{ formatTs(scope.row.created_at) }}</template>
            </el-table-column>
            <el-table-column align="left" label="孩子" width="140">
              <template #default="scope">{{ userName(scope.row.uid) }}</template>
            </el-table-column>
            <el-table-column align="left" label="商品" min-width="180">
              <template #default="scope">
                <span class="emoji">{{ scope.row.item_emoji || '🎁' }}</span>
                <span class="name">{{ scope.row.item_name }}</span>
                <div class="note faint">{{ kindText(scope.row.kind) }}</div>
              </template>
            </el-table-column>
            <el-table-column align="left" label="券" width="80">
              <template #default="scope">
                <span class="num">{{ scope.row.voucher_price }}</span>
                <span class="unit">券</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="状态" width="110">
              <template #default="scope">
                <el-tag :type="statusTagType(scope.row.status)" size="small">
                  {{ statusText(scope.row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column align="left" label="券在谁手里" width="130">
              <template #default="scope">
                <!-- ★ 这一列比「状态」更值得看：committed/confirmed 时券押在平台，
                     其余状态券已经回孩子钱包。运营对账只看状态会算错。 -->
                <span v-if="holdsVoucher(scope.row)" class="warn">平台押着</span>
                <span v-else class="dim">孩子钱包</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="备注" min-width="150">
              <template #default="scope">
                <span class="note">{{ orderNote(scope.row) || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="单号" width="180">
              <template #default="scope">
                <span class="mono dim">{{ scope.row.order_id }}</span>
              </template>
            </el-table-column>
          </el-table>
          <div class="gva-pagination">
            <el-pagination
              :current-page="ordPage"
              :page-size="ordPageSize"
              :page-sizes="[10, 30, 50, 100]"
              :total="ordTotal"
              layout="total, sizes, prev, pager, next, jumper"
              @current-change="onOrdPage"
              @size-change="onOrdSize"
            />
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- ==================== 改外链 / 上下架 ==================== -->
    <el-dialog v-model="editVisible" title="调整心愿商品" width="520px" destroy-on-close>
      <el-form :model="editForm" label-width="100px">
        <el-form-item label="商品">
          <span>{{ editForm.emoji }} {{ editForm.name }}</span>
        </el-form-item>
        <el-form-item label="外链">
          <el-input
            v-model="editForm.buy_url"
            placeholder="https://…（留空 = 不给链接）"
            clearable
          />
          <div class="hint">
            只有<b>家长的确认页</b>会看到这个链接，孩子那边看不到。
            留空时确认页只显示「去店里买」这类不带链接的说明。
          </div>
        </el-form-item>
        <el-form-item label="外链文字">
          <el-input v-model="editForm.buy_label" placeholder="去淘宝购买" clearable />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="editForm.sort_order" :min="0" :max="9999" :step="1" />
          <div class="hint">数字越小越靠前。</div>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch
            v-model="editForm.archived"
            :active-value="false"
            :inactive-value="true"
            active-text="在架"
            inactive-text="下架"
          />
          <div class="hint">下架只是从孩子的心愿列表里藏起来，<b>已有订单不受影响</b>。</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { getWishItemList, getWishOrderList, updateWishItem } from '@/api/mallWish'
  import { getPlatformUserList } from '@/api/tfUser'
  import { formatDate } from '@/utils/format'

  defineOptions({ name: 'MallWish' })

  /* 和 App 里 `domain/wish.ts` 的 WISH_KIND_META 保持一致 ——
     改了这里也要改那边，否则后台显示「其他」、孩子看到「✨ 其他」。 */
  const KIND_META = {
    blindbox: { label: '盲盒' },
    gamebox: { label: '游戏' },
    toy: { label: '玩具' },
    study: { label: '学习' },
    food: { label: '美食' },
    outing: { label: '出行' },
    collectible: { label: '其他' }
  }
  const kindText = (v) => KIND_META[v]?.label ?? v ?? '—'

  /* 状态机（`entities/wish.go` 的 WishStatus）：
     wished（孩子想要，券不动）→ committed（确定兑换，**扣券**）
       → confirmed（家长已去外链买）→ done（收到货）
     cancelled 从任何非终态都能进，**退券**。 */
  const STATUS_TEXT = {
    wished: '孩子想要',
    committed: '已兑换待兑现',
    confirmed: '家长已下单',
    done: '已完成',
    cancelled: '已取消'
  }
  const statusText = (v) => STATUS_TEXT[v] ?? v ?? '—'
  const statusTagType = (v) =>
    v === 'done' ? 'success' : v === 'cancelled' ? 'info' : v === 'wished' ? 'warning' : 'primary'

  /* ★ 券押在平台手里的状态 —— 和 `WishStatus.HasHeldVouchers()` 一致。
     只看「状态」不看这一列的话，对账时会把「已扣券但还没兑现」的券
     当成孩子钱包里的余额。 */
  const holdsVoucher = (row) => row.status === 'committed' || row.status === 'confirmed'

  const orderNote = (row) => {
    if (row.status === 'cancelled') return row.cancel_reason
    if (row.tracking_no) return `快递 ${row.tracking_no}`
    if (row.opened_game_id) return `开出 ${row.opened_game_id}`
    return ''
  }

  /* 平台的时间戳是 **Unix 秒**，`formatDate` 吃毫秒。传秒会得到 1970 年且不报错。 */
  const formatTs = (ts) => (ts ? formatDate(ts * 1000) : '—')
  const shortUrl = (u) => (u.length > 42 ? u.slice(0, 40) + '…' : u)

  const tab = ref('item')

  /* ---------- 用户（把 uid 换成昵称） ---------- */
  const users = ref([])
  const userById = ref({})
  const userName = (uid) => userById.value[uid]?.nickname || `#${uid}`
  const userLabel = (u) => `${u.nickname || '（没起名）'} · ${u.mobile || '#' + u.id}`
  const loadUsers = async () => {
    const res = await getPlatformUserList({ page: 1, pageSize: 200 })
    if (res.code !== 0) return
    users.value = res.data.list ?? []
    userById.value = Object.fromEntries(users.value.map((u) => [u.id, u]))
  }

  /* ---------- 商品 ---------- */
  const items = ref([])
  const itemTotal = ref(0)
  const itemPage = ref(1)
  const itemPageSize = ref(10)
  const itemSearch = ref({ kind: '', archived: '' })

  const getItems = async () => {
    const data = { page: itemPage.value, pageSize: itemPageSize.value }
    if (itemSearch.value.kind) data.kind = itemSearch.value.kind
    /* `archived` 是 bool 指针：传 `''` 会被当成 false（= 只看在架），
       传 `undefined` 才是「不过滤」⇒ 空值必须整个字段不传。 */
    if (itemSearch.value.archived === true || itemSearch.value.archived === false) {
      data.archived = itemSearch.value.archived
    }
    const res = await getWishItemList(data)
    if (res.code === 0) {
      items.value = res.data.list ?? []
      itemTotal.value = res.data.total ?? 0
    }
  }
  const onItemSearch = () => {
    itemPage.value = 1
    getItems()
  }
  const onItemReset = () => {
    itemSearch.value = { kind: '', archived: '' }
    itemPage.value = 1
    getItems()
  }
  const onItemPage = (v) => {
    itemPage.value = v
    getItems()
  }
  const onItemSize = (v) => {
    itemPageSize.value = v
    getItems()
  }

  /* ---------- 心愿单 ---------- */
  const orders = ref([])
  const ordTotal = ref(0)
  const ordPage = ref(1)
  const ordPageSize = ref(10)
  const ordSearch = ref({ status: '', uid: '' })

  const getOrders = async () => {
    const data = { page: ordPage.value, pageSize: ordPageSize.value }
    if (ordSearch.value.status) data.status = ordSearch.value.status
    if (ordSearch.value.uid) data.uid = ordSearch.value.uid
    const res = await getWishOrderList(data)
    if (res.code === 0) {
      orders.value = res.data.list ?? []
      ordTotal.value = res.data.total ?? 0
    }
  }
  const onOrdSearch = () => {
    ordPage.value = 1
    getOrders()
  }
  const onOrdReset = () => {
    ordSearch.value = { status: '', uid: '' }
    ordPage.value = 1
    getOrders()
  }
  const onOrdPage = (v) => {
    ordPage.value = v
    getOrders()
  }
  const onOrdSize = (v) => {
    ordPageSize.value = v
    getOrders()
  }

  /* ---------- 改外链 / 上下架 ---------- */
  const editVisible = ref(false)
  const submitting = ref(false)
  const editForm = ref({
    item_id: '',
    name: '',
    emoji: '',
    buy_url: '',
    buy_label: '',
    sort_order: 0,
    archived: false
  })

  const openEdit = (row) => {
    editForm.value = {
      item_id: row.item_id,
      name: row.name,
      emoji: row.emoji,
      buy_url: row.buy_url ?? '',
      buy_label: row.buy_label ?? '',
      sort_order: row.sort_order ?? 0,
      archived: !!row.archived
    }
    editVisible.value = true
  }

  const submitEdit = async () => {
    const f = editForm.value
    /* 前端先挡一下明显不对的链接：后端只做非空校验，
       写进 `javascript:...` 会原样出现在家长确认页的 href 上。 */
    const url = (f.buy_url ?? '').trim()
    if (url && !/^https?:\/\//i.test(url)) {
      ElMessage.error('外链必须以 http:// 或 https:// 开头')
      return
    }
    submitting.value = true
    try {
      const res = await updateWishItem({
        item_id: f.item_id,
        buy_url: url,
        buy_label: (f.buy_label ?? '').trim(),
        sort_order: f.sort_order,
        archived: f.archived
      })
      if (res.code === 0) {
        ElMessage.success('已保存')
        editVisible.value = false
        getItems()
      }
    } finally {
      /* ⚠️ 放 finally：失败时按钮必须恢复，否则「保存」永远转下去 */
      submitting.value = false
    }
  }

  onMounted(async () => {
    await loadUsers()
    await Promise.all([getItems(), getOrders()])
  })
</script>

<style scoped>
  .tip {
    margin-bottom: 12px;
  }
  .tabs {
    margin-top: 4px;
  }
  .emoji {
    margin-right: 6px;
    font-size: 16px;
  }
  .name {
    font-weight: 600;
  }
  .note {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.4;
  }
  .note.faint {
    opacity: 0.75;
  }
  .unit {
    margin-left: 2px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .dim {
    color: var(--el-text-color-secondary);
  }
  /* 券押在平台 = 有人要为此负责，别用灰字 */
  .warn {
    color: var(--el-color-warning);
    font-weight: 600;
  }
  .num {
    font-variant-numeric: tabular-nums;
  }
  .link {
    color: var(--el-color-primary);
    text-decoration: none;
  }
  .link:hover {
    text-decoration: underline;
  }
  .mono {
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  }
  .hint {
    margin-top: 2px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }
</style>
