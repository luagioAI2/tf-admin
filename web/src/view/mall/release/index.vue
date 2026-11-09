<template>
  <div>
    <el-alert type="info" :closable="false" show-icon class="tip">
      <template #title>
        发的是<b>游戏包</b>：选一个 .mjs / .zip，填版本号，点发布。服务端会把文件交给平台落盘、
        自己算 SHA-256，再签进游戏目录 —— 客户端「设置 → 游戏 → 从服务器下载」里就能看到并安装。
        <br />
        包由平台托管（PackageBaseURL 指到哪就发到哪，将来换 CDN 只改那一处）。
        同一版本号<b>不能发第二次</b>：改了包要换个版本号，否则客户端缓存对不上。
      </template>
    </el-alert>

    <!-- ============================ 发布 ============================ -->
    <div class="gva-search-box">
      <el-form :inline="true" :model="form" label-width="84px">
        <el-form-item label="游戏">
          <el-select
            v-model="form.game_id"
            filterable
            placeholder="选一款游戏"
            style="width: 220px"
            @change="onGameChange"
          >
            <el-option
              v-for="g in gameOptions"
              :key="g.game_id"
              :label="`${g.emoji || '🎮'} ${g.name || g.game_id}`"
              :value="g.game_id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="版本号">
          <el-input v-model.trim="form.version" placeholder="1.0.0" style="width: 130px" />
        </el-form-item>
        <el-form-item label="通道">
          <el-select v-model="form.channel" style="width: 120px">
            <el-option label="正式 stable" value="stable" />
            <el-option label="测试 beta" value="beta" />
          </el-select>
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model.trim="form.release_notes" placeholder="这次改了什么（可空）" style="width: 240px" />
        </el-form-item>
        <el-form-item label="强制更新">
          <el-switch v-model="form.forced" />
        </el-form-item>
      </el-form>

      <div class="upload-row">
        <el-upload
          ref="uploadRef"
          :auto-upload="false"
          :limit="1"
          :on-change="onPick"
          :on-remove="onPickClear"
          :on-exceed="onExceed"
          accept=".mjs,.js,.zip"
          drag
          class="drop"
        >
          <div class="drop-title">把包拖进来，或点这里选</div>
          <div class="drop-note">.mjs / .js 单文件，或 .zip（里面要有 &lt;id&gt;.plugin.mjs + assets/）</div>
        </el-upload>

        <div class="upload-side">
          <div v-if="picked" class="picked">
            <div class="picked-name mono">{{ picked.name }}</div>
            <div class="note">{{ fmtSize(picked.size) }}</div>
          </div>
          <div v-else class="note">还没选文件</div>

          <el-button
            type="primary"
            :loading="uploading"
            :disabled="!canSubmit"
            @click="onSubmit"
          >
            {{ uploading ? '正在上传…' : '上传并发布' }}
          </el-button>
          <div v-if="uploading" class="note">大包要几十秒，别关页面</div>
        </div>
      </div>
    </div>

    <!-- ============================ 列表 ============================ -->
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-form :inline="true" :model="search">
          <el-form-item label="游戏 ID">
            <el-input
              v-model.trim="search.game_id"
              placeholder="按 game_id 筛"
              clearable
              style="width: 160px"
              @keyup.enter="onSearch"
            />
          </el-form-item>
          <el-form-item label="通道">
            <el-select v-model="search.channel" placeholder="全部" clearable style="width: 120px">
              <el-option label="stable" value="stable" />
              <el-option label="beta" value="beta" />
            </el-select>
          </el-form-item>
          <el-form-item label="只看在架">
            <el-switch v-model="search.only_active" @change="onSearch" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" icon="search" @click="onSearch">查询</el-button>
            <el-button icon="refresh" @click="onReset">重置</el-button>
          </el-form-item>
        </el-form>
        <span class="dim">共 {{ total }} 个版本</span>
      </div>

      <el-table v-loading="loading" :data="rows" style="width: 100%" row-key="id">
        <el-table-column align="left" label="游戏" min-width="170">
          <template #default="scope">
            <div class="name">{{ scope.row.game_id }}</div>
            <div class="note">{{ scope.row.channel || 'stable' }}</div>
          </template>
        </el-table-column>

        <el-table-column align="left" label="版本" width="120">
          <template #default="scope">
            <span class="mono">v{{ scope.row.version }}</span>
          </template>
        </el-table-column>

        <el-table-column align="left" label="状态" width="110">
          <template #default="scope">
            <el-tag v-if="scope.row.active" type="success" size="small">在架</el-tag>
            <el-tag v-else type="info" size="small">已下架</el-tag>
          </template>
        </el-table-column>

        <el-table-column align="left" label="包" min-width="220">
          <template #default="scope">
            <div class="note num">{{ fmtSize(scope.row.size) }}</div>
            <div class="note mono" :title="scope.row.sha256">{{ shortHash(scope.row.sha256) }}</div>
            <div class="note faint mono" :title="scope.row.url">{{ shortUrl(scope.row.url) }}</div>
          </template>
        </el-table-column>

        <el-table-column align="left" label="发布时间" width="160">
          <template #default="scope">
            <span class="note">{{ fmtTime(scope.row.published_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column align="left" label="说明" min-width="180">
          <template #default="scope">
            <span class="note">{{ scope.row.release_notes || '—' }}</span>
          </template>
        </el-table-column>

        <el-table-column align="left" label="操作" width="180" fixed="right">
          <template #default="scope">
            <el-button
              v-if="scope.row.active"
              type="warning"
              link
              icon="bottom"
              @click="onDeactivate(scope.row)"
            >
              下架
            </el-button>
            <!-- 下架是运营手滑的高频动作；没有这个反向开关，误点一次就只能换版本号重发 -->
            <el-button
              v-else
              type="success"
              link
              icon="top"
              @click="onActivate(scope.row)"
            >
              上架
            </el-button>
            <el-button type="danger" link icon="delete" @click="onDelete(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="gva-pagination">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="onPageChange"
          @size-change="onSizeChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { getGameOptions } from '@/api/mallGame'
  import {
    getReleaseList,
    uploadRelease,
    setReleaseActive,
    deleteRelease
  } from '@/api/mallRelease'

  defineOptions({ name: 'GameRelease' })

  const gameOptions = ref([])
  const form = ref({
    game_id: '',
    version: '',
    channel: 'stable',
    release_notes: '',
    forced: false
  })
  const picked = ref(null)
  const uploading = ref(false)
  const uploadRef = ref(null)

  const canSubmit = computed(
    () => !!form.value.game_id && !!form.value.version && !!picked.value && !uploading.value
  )

  const loadGames = async () => {
    try {
      const res = await getGameOptions()
      if (res.code === 0) gameOptions.value = res.data || []
    } catch {
      /* 下拉拉不到不影响看列表，别把整页拦下来 */
    }
  }

  /* 选游戏时把版本号预填成「当前在架版本 +1 的补丁位」——
     运营最常做的就是发个补丁，手敲版本号既慢又容易和旧版本撞。 */
  const onGameChange = () => {
    const same = rows.value.filter(
      (r) => r.game_id === form.value.game_id && (r.channel || 'stable') === form.value.channel
    )
    if (!form.value.version && same.length) {
      form.value.version = bumpPatch(same[0].version)
    }
  }

  const bumpPatch = (v) => {
    const m = /^(\d+)\.(\d+)\.(\d+)$/.exec(String(v || ''))
    if (!m) return ''
    return `${m[1]}.${m[2]}.${Number(m[3]) + 1}`
  }

  const onPick = (file) => {
    picked.value = file.raw || null
  }
  const onPickClear = () => {
    picked.value = null
  }
  /* limit=1 时再选第二个会被拦；这里直接换成新的，省得运营先删再选 */
  const onExceed = (files) => {
    uploadRef.value?.clearFiles()
    const f = files[0]
    if (f) {
      uploadRef.value?.handleStart(f)
      picked.value = f
    }
  }

  const onSubmit = async () => {
    if (!canSubmit.value) return
    const fd = new FormData()
    fd.append('game_id', form.value.game_id)
    fd.append('version', form.value.version)
    fd.append('channel', form.value.channel)
    fd.append('release_notes', form.value.release_notes)
    fd.append('forced', form.value.forced ? '1' : '')
    fd.append('file', picked.value)

    uploading.value = true
    try {
      const res = await uploadRelease(fd)
      /* ★ 判据是**业务码**，不是「没抛异常」。`request.js` 只在 code !== 0 时
         弹错误并 reject，所以走到这里就是成功了；但仍然把平台回的内容念一遍，
         运营才知道「发的是哪个版本」。 */
      if (res.code === 0) {
        ElMessage.success(`已发布 ${form.value.game_id} v${form.value.version}`)
        uploadRef.value?.clearFiles()
        picked.value = null
        form.value.version = ''
        form.value.release_notes = ''
        await load()
      }
    } finally {
      uploading.value = false
    }
  }

  /* ---------------------------- 列表 ---------------------------- */
  const rows = ref([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(20)
  const loading = ref(false)
  const search = ref({ game_id: '', channel: '', only_active: false })

  const load = async () => {
    loading.value = true
    try {
      const res = await getReleaseList({
        page: page.value,
        pageSize: pageSize.value,
        game_id: search.value.game_id,
        channel: search.value.channel,
        only_active: search.value.only_active
      })
      if (res.code === 0) {
        rows.value = res.data.list || []
        total.value = res.data.total || 0
      }
    } finally {
      loading.value = false
    }
  }

  const onSearch = () => {
    page.value = 1
    load()
  }
  const onReset = () => {
    search.value = { game_id: '', channel: '', only_active: false }
    page.value = 1
    load()
  }
  const onPageChange = (p) => {
    page.value = p
    load()
  }
  const onSizeChange = (s) => {
    pageSize.value = s
    page.value = 1
    load()
  }

  const onDeactivate = async (row) => {
    try {
      await ElMessageBox.confirm(
        `下架 v${row.version} 之后，客户端就拉不到它了（记录和文件都还在）。确定？`,
        '下架版本',
        { type: 'warning' }
      )
    } catch {
      return
    }
    const res = await setReleaseActive(row.id, false)
    if (res.code === 0) {
      ElMessage.success('已下架')
      load()
    }
  }

  const onActivate = async (row) => {
    try {
      await ElMessageBox.confirm(
        `把 v${row.version} 重新上架？客户端会重新在目录里看到它。`,
        '上架版本',
        { type: 'info' }
      )
    } catch {
      return
    }
    const res = await setReleaseActive(row.id, true)
    if (res.code === 0) {
      ElMessage.success('已上架')
      load()
    }
  }

  const onDelete = async (row) => {
    try {
      await ElMessageBox.confirm(
        `删掉 v${row.version} 这条记录。\n注意：这只删记录，包文件还在平台上（要清文件去平台那边）。`,
        '删除版本',
        { type: 'warning' }
      )
    } catch {
      return
    }
    const res = await deleteRelease(row.id)
    if (res.code === 0) {
      ElMessage.success('已删除')
      load()
    }
  }

  /* ---------------------------- 格式化 ---------------------------- */
  const fmtSize = (n) => {
    const v = Number(n) || 0
    if (v <= 0) return '—'
    if (v < 1024) return `${v} B`
    if (v < 1024 * 1024) return `${(v / 1024).toFixed(0)} KB`
    return `${(v / 1024 / 1024).toFixed(1)} MB`
  }
  const shortHash = (h) => (h ? `${h.slice(0, 12)}…${h.slice(-6)}` : '—')
  const shortUrl = (u) => {
    if (!u) return '—'
    return u.length > 46 ? `…${u.slice(-44)}` : u
  }
  /* 平台存的是 **Unix 秒**（不是毫秒）—— 直接用 new Date(秒) 会得到 1970 年 */
  const fmtTime = (t) => {
    const s = Number(t) || 0
    if (s <= 0) return '—'
    const d = new Date(s * 1000)
    const p = (x) => String(x).padStart(2, '0')
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
  }

  onMounted(async () => {
    await Promise.all([loadGames(), load()])
  })
</script>

<style scoped>
  .tip {
    margin-bottom: 12px;
  }
  .name {
    font-weight: 600;
  }
  .note {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }
  .note.faint {
    opacity: 0.75;
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

  /* 上传区：左边拖放、右边状态 + 按钮。
     用 flex 而不是 el-form 的 inline，是因为拖放框高度和输入框差太多，
     挤在一行里两边都难看。 */
  .upload-row {
    display: flex;
    gap: 16px;
    align-items: stretch;
    margin-top: 4px;
  }
  .drop {
    flex: 1 1 auto;
    min-width: 320px;
  }
  .drop :deep(.el-upload-dragger) {
    padding: 18px 12px;
  }
  .drop-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
  .drop-note {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .upload-side {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 8px;
    min-width: 200px;
  }
  .picked-name {
    font-size: 13px;
    word-break: break-all;
  }
</style>
