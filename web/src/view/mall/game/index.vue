<template>
  <div>
    <el-alert type="info" :closable="false" show-icon class="tip">
      <template #title>
        游戏目录是随 App 客户端包发布的 —— 名字、封面、入口地址都跟着包走，后台不改这些。
        这一页只调「积分 / 钻石定价、默认是否拥有、上下架、排序」，改完 App 下次拉取即生效。
      </template>
    </el-alert>

    <div class="gva-search-box">
      <el-form :inline="true" :model="search">
        <el-form-item label="游戏 ID">
          <el-input
            v-model="search.game_id"
            placeholder="按 game_id 搜"
            clearable
            style="width: 180px"
            @keyup.enter="onSearch"
          />
        </el-form-item>
        <el-form-item label="名称">
          <el-input
            v-model="search.name"
            placeholder="按名称搜"
            clearable
            style="width: 160px"
            @keyup.enter="onSearch"
          />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="search.kind" placeholder="全部" clearable style="width: 170px">
            <el-option v-for="(m, k) in KIND_META" :key="k" :label="m.label" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="search.status" placeholder="全部" clearable style="width: 130px">
            <el-option v-for="(m, k) in STATUS_META" :key="k" :label="m.label" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSearch">查询</el-button>
          <el-button icon="refresh" @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <div class="gva-btn-list">
        <span class="dim">共 {{ total }} 款游戏</span>
      </div>
      <el-table v-loading="loading" :data="rows" style="width: 100%" row-key="game_id">
        <el-table-column align="left" label="游戏" min-width="230">
          <template #default="scope">
            <div>
              <span class="emoji">{{ scope.row.emoji || '🎮' }}</span>
              <span class="name">{{ scope.row.name || '—' }}</span>
            </div>
            <div class="note mono">{{ scope.row.game_id }}</div>
            <div v-if="scope.row.tagline" class="note">{{ scope.row.tagline }}</div>
          </template>
        </el-table-column>
        <el-table-column align="left" label="分类" width="150">
          <template #default="scope">{{ kindText(scope.row.kind) }}</template>
        </el-table-column>
        <el-table-column align="left" label="定价" width="150">
          <template #default="scope">
            <span class="num">{{ scope.row.price_points ?? 0 }}</span>
            <span class="unit">积分</span>
            <template v-if="scope.row.price_diamond">
              <span class="dim"> / </span>
              <span class="num">{{ scope.row.price_diamond }}</span>
              <span class="unit">钻石</span>
            </template>
          </template>
        </el-table-column>
        <el-table-column align="left" label="默认拥有" width="100">
          <template #default="scope">
            <el-tag v-if="scope.row.default_owned" size="small" type="success">免费</el-tag>
            <span v-else class="dim">需兑换</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="状态" width="100">
          <template #default="scope">
            <el-tag size="small" :type="statusTagType(scope.row.status)">
              {{ statusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column align="left" label="排序" width="80">
          <template #default="scope"><span class="num">{{ scope.row.sort ?? 0 }}</span></template>
        </el-table-column>
        <el-table-column align="left" label="适龄 / 时长" width="150">
          <template #default="scope">
            <div class="note">{{ scope.row.age_hint || '—' }}</div>
            <div class="note faint">{{ scope.row.session_hint || '—' }}</div>
          </template>
        </el-table-column>
        <el-table-column align="left" label="操作" width="90" fixed="right">
          <template #default="scope">
            <el-button link type="primary" @click="openEdit(scope.row)">调整</el-button>
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
          @current-change="onPage"
          @size-change="onSize"
        />
      </div>
    </div>

    <el-dialog v-model="editVisible" title="调整游戏" width="480px">
      <el-form :model="editForm" label-width="110px">
        <el-form-item label="游戏">
          <span class="name">{{ editForm.name }}</span>
          <span class="note mono">{{ editForm.game_id }}</span>
        </el-form-item>
        <el-form-item label="积分定价">
          <el-input-number v-model="editForm.price_points" :min="0" :step="10" />
          <span class="unit">积分（0 = 不收积分）</span>
        </el-form-item>
        <el-form-item label="钻石定价">
          <el-input-number v-model="editForm.price_diamond" :min="0" :step="1" />
          <span class="unit">钻石（0 = 不收钻石）</span>
        </el-form-item>
        <el-form-item label="默认拥有">
          <el-switch v-model="editForm.default_owned" />
          <span class="unit">开 = 孩子不用兑换直接能玩</span>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="editForm.status" style="width: 200px">
            <el-option v-for="(m, k) in STATUS_META" :key="k" :label="m.label" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="editForm.sort" :min="0" :step="10" />
          <span class="unit">大的靠前</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { getGameList, updateGame } from '@/api/mallGame'

  defineOptions({ name: 'GameList' })

  /* 和 App `src/games/kind.ts` 的 GAME_KINDS 对齐 —— 后台自己编一套
     中文的话，运营对着孩子那边的分类名会认不出来。 */
  const KIND_META = {
    think: { label: '动脑 · 有产出' },
    play: { label: '纯玩 · 图个乐' },
    chill: { label: '纯玩 · 不产出' }
  }
  const kindText = (v) => KIND_META[v]?.label ?? v ?? '—'

  /* ★ 状态取值是服务端写死的白名单（server/service/mall/game.go 的 Update）：
       available / coming_soon / offline。写成 hidden / coming 会「保存失败：
       状态只能是 …」，界面却只表现为没保存成功 —— 客户端必须镜像同一份。 */
  const STATUS_META = {
    available: { label: '在架' },
    coming_soon: { label: '未上线' },
    offline: { label: '已下架' }
  }
  const statusText = (v) => STATUS_META[v]?.label ?? v ?? '—'
  const statusTagType = (v) => (v === 'available' ? 'success' : 'info')

  const search = ref({ game_id: '', name: '', kind: '', status: '' })
  const rows = ref([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(20)
  const loading = ref(false)

  const load = async () => {
    loading.value = true
    try {
      const res = await getGameList({
        page: page.value,
        pageSize: pageSize.value,
        game_id: search.value.game_id || undefined,
        name: search.value.name || undefined,
        kind: search.value.kind || undefined,
        status: search.value.status || undefined
      })
      if (res.code !== 0) {
        ElMessage.error(res.msg || '拉取游戏列表失败')
        return
      }
      rows.value = res.data?.list ?? []
      total.value = res.data?.total ?? 0
    } finally {
      loading.value = false
    }
  }

  const onSearch = () => {
    page.value = 1
    load()
  }
  const onReset = () => {
    search.value = { game_id: '', name: '', kind: '', status: '' }
    page.value = 1
    load()
  }
  const onPage = (p) => {
    page.value = p
    load()
  }
  const onSize = (s) => {
    pageSize.value = s
    page.value = 1
    load()
  }

  const editVisible = ref(false)
  const editForm = ref({})
  const openEdit = (row) => {
    editForm.value = {
      game_id: row.game_id,
      name: row.name,
      price_points: row.price_points ?? 0,
      price_diamond: row.price_diamond ?? 0,
      default_owned: !!row.default_owned,
      status: row.status || 'available',
      sort: row.sort ?? 0
    }
    editVisible.value = true
  }
  const saveEdit = async () => {
    const res = await updateGame({
      game_id: editForm.value.game_id,
      price_points: editForm.value.price_points,
      price_diamond: editForm.value.price_diamond,
      default_owned: editForm.value.default_owned,
      status: editForm.value.status,
      sort: editForm.value.sort
    })
    if (res.code !== 0) {
      ElMessage.error(res.msg || '保存失败')
      return
    }
    ElMessage.success('已保存')
    editVisible.value = false
    await load()
  }

  onMounted(load)
</script>

<style scoped>
  .tip {
    margin-bottom: 12px;
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
    margin-left: 6px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .dim {
    color: var(--el-text-color-secondary);
  }
  .num {
    font-variant-numeric: tabular-nums;
  }
  .mono {
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  }
</style>
