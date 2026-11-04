<template>
  <div>
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-select v-model="archived" placeholder="全部" clearable class="sel" @change="search">
          <el-option label="在架" :value="false" />
          <el-option label="已下架" :value="true" />
        </el-select>
        <el-input
          v-model="keyword"
          placeholder="搜商品名"
          clearable
          class="kw"
          @keyup.enter="search"
          @clear="search"
        />
        <el-button type="primary" @click="search">查询</el-button>
        <el-button type="success" @click="openCreate">上架新商品</el-button>
      </div>

      <el-table :data="rows" v-loading="loading" row-key="item_id" class="tbl">
        <el-table-column label="商品" min-width="220">
          <template #default="{ row }">
            <span class="emoji">{{ row.emoji || '🎁' }}</span>
            <span class="name">{{ row.name }}</span>
            <div class="muted">{{ row.partner_name || '—' }}</div>
          </template>
        </el-table-column>

        <el-table-column label="参考价" width="110" align="right">
          <template #default="{ row }">
            <span v-if="row.price_cents">{{ yuan(row.price_cents) }}</span>
            <span v-else class="muted">未填</span>
          </template>
        </el-table-column>

        <el-table-column label="购买链接" min-width="200">
          <template #default="{ row }">
            <el-link
              v-if="row.buy_url"
              type="primary"
              :href="row.buy_url"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ row.buy_label || '打开' }} ↗
            </el-link>
            <span v-else class="muted">未填</span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.archived ? 'info' : 'success'" size="small">
              {{ row.archived ? '已下架' : '在架' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openEdit(row)">编辑</el-button>
            <el-button
              :type="row.archived ? 'success' : 'warning'"
              link
              @click="toggleArchived(row)"
            >
              {{ row.archived ? '重新上架' : '下架' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty
        v-if="!loading && !rows.length"
        description="还没有商品。"
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

    <el-dialog v-model="dlg" :title="form.itemId ? '编辑商品' : '上架新商品'" width="560px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="96px">
        <el-form-item label="商品名" prop="name">
          <el-input v-model="form.name" maxlength="40" show-word-limit />
        </el-form-item>
        <el-form-item label="表情">
          <el-input v-model="form.emoji" maxlength="4" class="short" placeholder="🎁" />
          <span class="muted ml">商品图标</span>
        </el-form-item>
        <el-form-item label="店铺名">
          <el-input v-model="form.partnerName" maxlength="30" placeholder="显示在商品旁边" />
        </el-form-item>
        <el-form-item label="参考价">
          <el-input v-model="form.priceYuan" class="short" placeholder="199.00" />
          <span class="muted ml">元，仅供参考</span>
        </el-form-item>
        <el-form-item label="购买链接">
          <el-input v-model="form.buyUrl" placeholder="https://…" />
          <div class="muted">
            必须是 http(s) 开头的地址。
          </div>
        </el-form-item>
        <el-form-item label="链接文字">
          <el-input v-model="form.buyLabel" class="short" maxlength="10" placeholder="去下单" />
        </el-form-item>
        <el-form-item label="一句话说明">
          <el-input v-model="form.note" maxlength="60" show-word-limit />
        </el-form-item>
        <el-form-item label="注意事项">
          <el-input v-model="form.notice" type="textarea" :rows="2" maxlength="120" show-word-limit />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sortOrder" :min="0" :max="999" />
          <span class="muted ml">数值小的排前面</span>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
  import { onMounted, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { getMerchantItems, saveMerchantItem } from '@/api/portal'

  defineOptions({ name: 'MerchantItems' })

  const rows = ref([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(10)
  const keyword = ref('')
  const archived = ref(null)
  const loading = ref(false)

  const dlg = ref(false)
  const saving = ref(false)
  const formRef = ref(null)

  const emptyForm = () => ({
    itemId: '',
    name: '',
    emoji: '🎁',
    partnerName: '',
    priceYuan: '',
    note: '',
    notice: '',
    buyUrl: '',
    buyLabel: '去下单',
    sortOrder: 0
  })
  const form = reactive(emptyForm())

  const rules = {
    name: [{ required: true, message: '商品名不能为空', trigger: 'blur' }]
  }

  /* 价格在库里是**分**（int64），界面上是元。
     用字符串收、字符串回，别用 float 乘除 —— 0.1 + 0.2 那种误差会变成
     「199.00 存进去变成 198.99」。 */
  const yuan = (cents) => (cents / 100).toFixed(2)

  const load = async () => {
    loading.value = true
    try {
      const res = await getMerchantItems({
        page: page.value,
        pageSize: pageSize.value,
        keyword: keyword.value,
        archived: archived.value === null ? undefined : archived.value
      })
      if (res.code === 0) {
        rows.value = res.data?.list ?? []
        total.value = res.data?.total ?? 0
      }
    } catch (e) {
      ElMessage.error('商品没拉到：' + (e?.message ?? e))
    } finally {
      loading.value = false
    }
  }

  const search = () => {
    page.value = 1
    load()
  }

  const openCreate = () => {
    Object.assign(form, emptyForm())
    dlg.value = true
  }

  const openEdit = (row) => {
    Object.assign(form, {
      itemId: row.item_id,
      name: row.name ?? '',
      emoji: row.emoji ?? '🎁',
      partnerName: row.partner_name ?? '',
      priceYuan: row.price_cents ? yuan(row.price_cents) : '',
      note: row.note ?? '',
      notice: row.notice ?? '',
      buyUrl: row.buy_url ?? '',
      buyLabel: row.buy_label || '去下单',
      sortOrder: row.sort_order ?? 0
    })
    dlg.value = true
  }

  const toCents = (s) => {
    const t = String(s ?? '').trim()
    if (!t) return 0
    const n = Number(t)
    if (!Number.isFinite(n) || n < 0) return NaN
    return Math.round(n * 100)
  }

  const submit = () => {
    formRef.value.validate(async (ok) => {
      if (!ok) return
      const cents = toCents(form.priceYuan)
      if (Number.isNaN(cents)) {
        ElMessage.error('参考价填个数字，或者留空')
        return
      }
      saving.value = true
      try {
        const res = await saveMerchantItem({
          itemId: form.itemId,
          name: form.name.trim(),
          emoji: form.emoji,
          partnerName: form.partnerName,
          priceCents: cents,
          note: form.note,
          notice: form.notice,
          buyUrl: form.buyUrl.trim(),
          buyLabel: form.buyLabel,
          sortOrder: form.sortOrder
        })
        if (res.code === 0) {
          ElMessage.success('已保存')
          dlg.value = false
          load()
        }
      } catch (e) {
        ElMessage.error('保存失败：' + (e?.message ?? e))
      } finally {
        saving.value = false
      }
    })
  }

  /* 上下架走的是同一个保存接口 —— 服务端只有那一个写入口，
     也就只有一处需要校验「这件是不是你的」。 */
  const toggleArchived = async (row) => {
    try {
      const res = await saveMerchantItem({
        itemId: row.item_id,
        name: row.name,
        emoji: row.emoji,
        partnerName: row.partner_name,
        priceCents: row.price_cents ?? 0,
        note: row.note,
        notice: row.notice,
        buyUrl: row.buy_url,
        buyLabel: row.buy_label,
        sortOrder: row.sort_order ?? 0,
        archived: !row.archived
      })
      if (res.code === 0) {
        ElMessage.success(row.archived ? '已重新上架' : '已下架')
        load()
      }
    } catch (e) {
      ElMessage.error('操作失败：' + (e?.message ?? e))
    }
  }

  onMounted(load)
</script>

<style scoped>
  .sel {
    width: 120px;
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
  .ml {
    margin-left: 8px;
  }
  .short {
    width: 140px;
  }
  .tbl {
    width: 100%;
  }
</style>
