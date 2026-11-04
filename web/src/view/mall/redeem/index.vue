<template>
  <div>
    <el-tabs v-model="tab" class="tabs">
      <!-- ==================== ① 商品 ==================== -->
      <el-tab-pane label="兑换商品" name="item">
        <div class="gva-search-box">
          <el-form :inline="true" :model="itemSearch">
            <el-form-item label="所属">
              <el-select
                v-model="itemSearch.uid"
                placeholder="全部用户"
                clearable
                filterable
                style="width: 200px"
              >
                <el-option
                  v-for="u in users"
                  :key="u.id"
                  :label="userLabel(u)"
                  :value="u.id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="分类">
              <el-select v-model="itemSearch.category" placeholder="全部" clearable style="width: 140px">
                <el-option v-for="(t, v) in CATEGORY_TEXT" :key="v" :label="t" :value="v" />
              </el-select>
            </el-form-item>
            <el-form-item label="状态">
              <el-select v-model="itemSearch.archived" placeholder="全部" clearable style="width: 120px">
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
            <el-table-column align="left" label="商品" min-width="200">
              <template #default="scope">
                <span class="emoji">{{ scope.row.emoji || '🎁' }}</span>
                <span class="name">{{ scope.row.name }}</span>
                <div v-if="scope.row.note" class="note">{{ scope.row.note }}</div>
              </template>
            </el-table-column>
            <el-table-column align="left" label="所属" width="150">
              <template #default="scope">
                <span>{{ userName(scope.row.uid) }}</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="分类" width="90">
              <template #default="scope">
                {{ categoryText(scope.row.category) }}
              </template>
            </el-table-column>
            <el-table-column align="left" label="价格" width="90">
              <template #default="scope">
                <span class="num">{{ scope.row.price }}</span>
                <span class="unit">券</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="库存" width="80">
              <template #default="scope">
                <!-- -1 是「无限」，直接显示 -1 会让人以为卖光了 -->
                <span v-if="scope.row.stock < 0" class="dim">无限</span>
                <span v-else class="num" :class="{ zero: scope.row.stock === 0 }">
                  {{ scope.row.stock }}
                </span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="每天限" width="80">
              <template #default="scope">
                <span v-if="scope.row.limit_per_day > 0" class="num">{{ scope.row.limit_per_day }}</span>
                <span v-else class="dim">不限</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="一次性" width="80">
              <template #default="scope">
                <el-tag v-if="scope.row.once" size="small" type="info">一次性</el-tag>
                <span v-else class="dim">—</span>
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
            <el-table-column align="left" label="操作" width="130" fixed="right">
              <template #default="scope">
                <el-button type="primary" link @click="openEdit(scope.row)">改价 / 库存</el-button>
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

      <!-- ==================== ② 记录 ==================== -->
      <el-tab-pane label="兑换记录" name="record">
        <div class="gva-search-box">
          <el-form :inline="true" :model="recSearch">
            <el-form-item label="所属">
              <el-select
                v-model="recSearch.uid"
                placeholder="全部用户"
                clearable
                filterable
                style="width: 200px"
              >
                <el-option
                  v-for="u in users"
                  :key="u.id"
                  :label="userLabel(u)"
                  :value="u.id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="状态">
              <el-select v-model="recSearch.state" placeholder="全部" clearable style="width: 140px">
                <el-option label="待兑现" value="pending" />
                <el-option label="已兑现" value="fulfilled" />
                <el-option label="已取消" value="cancelled" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" icon="search" @click="onRecSearch">查询</el-button>
              <el-button icon="refresh" @click="onRecReset">重置</el-button>
            </el-form-item>
          </el-form>
        </div>

        <div class="gva-table-box">
          <div class="gva-btn-list">
            <span class="dim">共 {{ recTotal }} 条</span>
          </div>
          <el-table :data="records" style="width: 100%" row-key="record_id">
            <el-table-column align="left" label="时间" width="160">
              <template #default="scope">{{ formatTs(scope.row.created_at) }}</template>
            </el-table-column>
            <el-table-column align="left" label="孩子" width="150">
              <template #default="scope">{{ userName(scope.row.uid) }}</template>
            </el-table-column>
            <el-table-column align="left" label="商品" min-width="180">
              <template #default="scope">
                <span class="emoji">{{ scope.row.emoji || '🎁' }}</span>
                <span class="name">{{ scope.row.name }}</span>
                <span v-if="scope.row.quantity > 1" class="qty">×{{ scope.row.quantity }}</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="花费" width="140">
              <template #default="scope">
                <!-- 两种币分开记：只显示 cost 的话，家长问「扣的是积分还是券」答不上来 -->
                <span v-if="scope.row.coupon_paid" class="num">{{ scope.row.coupon_paid }} 券</span>
                <span v-if="scope.row.coupon_paid && scope.row.points_paid" class="dim"> + </span>
                <span v-if="scope.row.points_paid" class="num">{{ scope.row.points_paid }} 分</span>
                <span v-if="!scope.row.coupon_paid && !scope.row.points_paid" class="dim">—</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="状态" width="100">
              <template #default="scope">
                <el-tag :type="recTagType(scope.row)" size="small">
                  {{ recStatusText(scope.row) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column align="left" label="备注" min-width="160">
              <template #default="scope">
                <span class="note">{{ recNote(scope.row) || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="单号" width="180">
              <template #default="scope">
                <span class="mono dim">{{ scope.row.record_id }}</span>
              </template>
            </el-table-column>
          </el-table>
          <div class="gva-pagination">
            <el-pagination
              :current-page="recPage"
              :page-size="recPageSize"
              :page-sizes="[10, 30, 50, 100]"
              :total="recTotal"
              layout="total, sizes, prev, pager, next, jumper"
              @current-change="onRecPage"
              @size-change="onRecSize"
            />
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- ==================== 改价 / 库存 ==================== -->
    <el-dialog v-model="editVisible" title="调整商品" width="460px" destroy-on-close>
      <el-form :model="editForm" label-width="96px">
        <el-form-item label="商品">
          <span>{{ editForm.emoji }} {{ editForm.name }}</span>
          <span class="dim"> · {{ userName(editForm.uid) }}</span>
        </el-form-item>
        <el-form-item label="价格（券）">
          <el-input-number v-model="editForm.price" :min="1" :max="100000" :step="1" />
          <div class="hint">1 兑换券 = 1 分。</div>
        </el-form-item>
        <el-form-item label="库存">
          <el-input-number v-model="editForm.stock" :min="-1" :max="100000" :step="1" />
          <div class="hint">
            <b>-1 = 无限</b>；<b>0 = 卖光了</b>（孩子那边会显示成换不了，不是无限）。
          </div>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch
            v-model="editForm.archived"
            :active-value="false"
            :inactive-value="true"
            active-text="在架"
            inactive-text="下架"
          />
          <div class="hint">下架只是从孩子的清单里藏起来，<b>已有记录不受影响</b>。</div>
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
  import {
    getRedeemItemList,
    getRedeemRecordList,
    updateRedeemItem
  } from '@/api/mallRedeem'
  /* 用户接口只在 tfUser.js 有一份，别在 mallRedeem.js 再抄一个 ——
     两份实现迟早会漂（改了一处，另一处还是老样子）。 */
  import { getPlatformUserList } from '@/api/tfUser'
  import { formatDate } from '@/utils/format'

  defineOptions({ name: 'MallRedeem' })

  const CATEGORY_TEXT = {
    snack: '零食',
    gift: '礼物',
    screen: '屏幕时间',
    outing: '出门玩',
    privilege: '特权',
    custom: '自定义'
  }
  const categoryText = (v) => CATEGORY_TEXT[v] ?? v ?? '—'

  /* 平台的时间戳是 **Unix 秒**，而 `formatDate` 吃毫秒。
     直接传秒会得到 1970 年，而且不报错。 */
  const formatTs = (ts) => (ts ? formatDate(ts * 1000) : '—')

  const tab = ref('item')

  /* ---------- 用户（只为了把 uid 换成昵称） ---------- */
  const users = ref([])
  const userById = ref({})
  const userName = (uid) => userById.value[uid]?.nickname || `#${uid}`
  const userLabel = (u) => `${u.nickname || '（没起名）'} · ${u.mobile || '#' + u.id}`
  const loadUsers = async () => {
    /* pageSize 给大一点，一次拿完。用户多了要改成按需搜索 —— 那时候
       「记录里的 uid 换昵称」就得由服务端 join，别再拉全表。 */
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
  const itemSearch = ref({ uid: '', category: '', archived: '' })

  const getItems = async () => {
    const data = { page: itemPage.value, pageSize: itemPageSize.value }
    if (itemSearch.value.uid) data.uid = itemSearch.value.uid
    if (itemSearch.value.category) data.category = itemSearch.value.category
    /* ⚠️ `archived` 是 **bool 指针**：传 `''` 会被当成 false（= 只看在架），
       传 `undefined` 才是「不过滤」。所以空值必须**整个字段不传**。 */
    if (itemSearch.value.archived === true || itemSearch.value.archived === false) {
      data.archived = itemSearch.value.archived
    }
    const res = await getRedeemItemList(data)
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
    itemSearch.value = { uid: '', category: '', archived: '' }
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

  /* ---------- 记录 ---------- */
  const records = ref([])
  const recTotal = ref(0)
  const recPage = ref(1)
  const recPageSize = ref(10)
  const recSearch = ref({ uid: '', state: '' })

  const recStatusText = (r) => (r.cancelled ? '已取消' : r.fulfilled ? '已兑现' : '待兑现')
  const recTagType = (r) => (r.cancelled ? 'info' : r.fulfilled ? 'success' : 'warning')
  const recNote = (r) => (r.cancelled ? r.cancel_reason : r.fulfilled_note)

  const getRecords = async () => {
    const data = { page: recPage.value, pageSize: recPageSize.value }
    if (recSearch.value.uid) data.uid = recSearch.value.uid
    /* 同理：`cancelled` / `fulfilled` 都是 bool 指针，只传明确的 true/false */
    if (recSearch.value.state === 'cancelled') data.cancelled = true
    if (recSearch.value.state === 'fulfilled') {
      data.fulfilled = true
      data.cancelled = false
    }
    if (recSearch.value.state === 'pending') {
      data.fulfilled = false
      data.cancelled = false
    }
    const res = await getRedeemRecordList(data)
    if (res.code === 0) {
      records.value = res.data.list ?? []
      recTotal.value = res.data.total ?? 0
    }
  }
  const onRecSearch = () => {
    recPage.value = 1
    getRecords()
  }
  const onRecReset = () => {
    recSearch.value = { uid: '', state: '' }
    recPage.value = 1
    getRecords()
  }
  const onRecPage = (v) => {
    recPage.value = v
    getRecords()
  }
  const onRecSize = (v) => {
    recPageSize.value = v
    getRecords()
  }

  /* ---------- 改价 / 库存 ---------- */
  const editVisible = ref(false)
  const submitting = ref(false)
  const editForm = ref({
    item_id: '',
    name: '',
    emoji: '',
    uid: 0,
    price: 1,
    stock: -1,
    archived: false
  })

  const openEdit = (row) => {
    editForm.value = {
      item_id: row.item_id,
      name: row.name,
      emoji: row.emoji,
      uid: row.uid,
      price: row.price,
      stock: row.stock,
      archived: !!row.archived
    }
    editVisible.value = true
  }

  const submitEdit = async () => {
    const f = editForm.value
    submitting.value = true
    try {
      const res = await updateRedeemItem({
        item_id: f.item_id,
        price: f.price,
        stock: f.stock,
        archived: f.archived
      })
      if (res.code === 0) {
        ElMessage.success('已保存')
        editVisible.value = false
        /* 两个列表都要刷：记录页的「花费」不会变，但商品页的价/库存变了 */
        getItems()
      }
    } finally {
      /* ⚠️ 放 finally：失败时按钮必须恢复，否则「保存」永远转下去 */
      submitting.value = false
    }
  }

  onMounted(async () => {
    await loadUsers()
    await Promise.all([getItems(), getRecords()])
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
  .qty {
    margin-left: 6px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .unit {
    margin-left: 2px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .dim {
    color: var(--el-text-color-secondary);
  }
  .num {
    font-variant-numeric: tabular-nums;
  }
  /* 库存 0 要显眼：那是「换不了了」，不是一个普通数字 */
  .num.zero {
    color: var(--el-color-danger);
    font-weight: 600;
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
