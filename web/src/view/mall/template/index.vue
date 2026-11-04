<template>
  <div>
    <el-alert type="info" :closable="false" show-icon class="tip">
      <template #title>
        模板是<b>新用户首次打开 App 时铺一次</b>的内容 —— 改了不会影响已经在用的孩子，
        他们只会看到新增的条目。任务模板和兑换模板各自有一份「默认」，新用户按默认那份铺。
      </template>
    </el-alert>

    <el-tabs v-model="kind" class="tabs" @tab-change="onKindChange">
      <el-tab-pane label="任务模板" name="task" />
      <el-tab-pane label="兑换模板" name="redeem" />
    </el-tabs>

    <!-- 商家的推广码：印在物料上，注册时带给 App，决定这个用户铺哪份模板。
         超管没有「自己的」商家码（接口返回 uid=0），所以整块不显示。 -->
    <div v-if="invite.uid > 0" class="gva-table-box invite">
      <div class="invite-main">
        <div class="invite-label">我的推广码</div>
        <div class="invite-code mono" data-probe="invite-code">{{ invite.code || '—' }}</div>
      </div>
      <div class="invite-side">
        扫码 / 输码注册的用户，会铺<b>你设为默认的那份</b>模板；没设就用官方默认。
      </div>
      <el-button v-if="invite.code" text type="primary" @click="copyInvite">复制</el-button>
    </div>

    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="primary" icon="plus" @click="openCreate">新建模板</el-button>
        <span class="dim" style="margin-left: 12px">
          共 {{ rows.length }} 份{{ kind === 'task' ? '任务' : '兑换' }}模板
        </span>
      </div>

      <el-table v-loading="loading" :data="rows" style="width: 100%" row-key="id">
        <el-table-column align="left" label="模板" min-width="260">
          <template #default="scope">
            <span class="name" data-probe="tpl-name">{{ scope.row.name }}</span>
            <el-tag v-if="scope.row.source === 'seed'" size="small" type="info" class="ml6">
              初始
            </el-tag>
            <div v-if="scope.row.note" class="note">{{ scope.row.note }}</div>
          </template>
        </el-table-column>
        <el-table-column align="left" label="归属" width="180">
          <template #default="scope">
            <el-tag v-if="scope.row.scope === 'official'" size="small" type="success">官方</el-tag>
            <span v-else class="name">{{ merchantLabel(scope.row.owner_uid) }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="条目" width="90">
          <template #default="scope">
            <span class="num" data-probe="tpl-count">{{ scope.row.item_count }}</span>
            <span class="unit">条</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="默认" width="100">
          <template #default="scope">
            <el-tag v-if="scope.row.is_default" size="small" type="warning">默认</el-tag>
            <span v-else class="dim">—</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="排序" width="80">
          <template #default="scope"><span class="num">{{ scope.row.sort }}</span></template>
        </el-table-column>
        <el-table-column align="left" label="更新于" width="160">
          <template #default="scope">{{ formatTs(scope.row.updated_at) }}</template>
        </el-table-column>
        <el-table-column align="left" label="操作" width="230" fixed="right">
          <template #default="scope">
            <el-button type="primary" link @click="openItems(scope.row)">编辑内容</el-button>
            <el-button
              v-if="!scope.row.is_default"
              type="primary"
              link
              @click="onSetDefault(scope.row)"
            >
              设为默认
            </el-button>
            <el-button type="primary" link @click="openRename(scope.row)">改名</el-button>
            <el-button type="danger" link @click="onDeleteTemplate(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ==================== 模板条目 ==================== -->
    <el-drawer v-model="itemsVisible" :title="itemsTitle" size="880px" destroy-on-close>
      <div class="drawer-head">
        <el-select v-model="cycleFilter" placeholder="全部周期" clearable style="width: 150px">
          <el-option v-for="(m, k) in CYCLE_META" :key="k" :label="m" :value="k" />
        </el-select>
        <el-button type="primary" icon="plus" @click="openItemCreate">新增条目</el-button>
        <span class="dim">共 {{ visibleItems.length }} 条</span>
      </div>

      <!-- 任务模板：周期 / 时长 / 积分 / 签到 -->
      <el-table
        v-if="kind === 'task'"
        v-loading="detailLoading"
        :data="visibleItems"
        style="width: 100%"
        row-key="id"
      >
        <el-table-column align="left" label="任务" min-width="240">
          <template #default="scope">
            <span class="emoji">{{ scope.row.emoji || categoryEmoji(scope.row.category) }}</span>
            <span class="name" data-probe="item-name">{{ scope.row.name }}</span>
            <div v-if="scope.row.note" class="note">{{ scope.row.note }}</div>
          </template>
        </el-table-column>
        <el-table-column align="left" label="分类" width="90">
          <template #default="scope">{{ categoryText(scope.row.category) }}</template>
        </el-table-column>
        <el-table-column align="left" label="周期" width="100">
          <template #default="scope">
            <el-tag size="small" :type="scope.row.cycle === 'once' ? 'info' : 'primary'">
              {{ cycleText(scope.row.cycle) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column align="left" label="时长" width="90">
          <template #default="scope">
            <span class="num">{{ scope.row.planned_minutes }}</span>
            <span class="unit">分</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="积分" width="80">
          <template #default="scope">
            <span class="num">{{ scope.row.base_points }}</span>
            <span class="unit">分</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="签到" width="90">
          <template #default="scope">
            <el-tag v-if="extraFlag(scope.row, 'checkInEnabled')" size="small" type="success">
              打卡 {{ extraNum(scope.row, 'checkInTargetCount') || '—' }} 次
            </el-tag>
            <span v-else class="dim">—</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="操作" width="120" fixed="right">
          <template #default="scope">
            <el-button type="primary" link @click="openItemEdit(scope.row)">编辑</el-button>
            <el-button type="danger" link @click="onDeleteItem(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 兑换模板：价格 / 限购 / 库存 -->
      <el-table
        v-else
        v-loading="detailLoading"
        :data="visibleItems"
        style="width: 100%"
        row-key="id"
      >
        <el-table-column align="left" label="兑换品" min-width="240">
          <template #default="scope">
            <span class="emoji">{{ scope.row.emoji || '🎁' }}</span>
            <span class="name" data-probe="item-name">{{ scope.row.name }}</span>
            <div v-if="scope.row.note" class="note">{{ scope.row.note }}</div>
          </template>
        </el-table-column>
        <el-table-column align="left" label="分类" width="100">
          <template #default="scope">{{ redeemCategoryText(scope.row.category) }}</template>
        </el-table-column>
        <el-table-column align="left" label="价格" width="90">
          <template #default="scope">
            <span class="num">{{ scope.row.price }}</span>
            <span class="unit">券</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="每日限" width="90">
          <template #default="scope">
            <span v-if="scope.row.limit_per_day > 0" class="num">
              {{ scope.row.limit_per_day }}
            </span>
            <span v-else class="dim">不限</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="库存" width="90">
          <template #default="scope">
            <span v-if="scope.row.stock < 0" class="dim">不限</span>
            <span v-else class="num">{{ scope.row.stock }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="一次性" width="90">
          <template #default="scope">
            <el-tag v-if="scope.row.once" size="small" type="warning">换过就没</el-tag>
            <span v-else class="dim">可重复</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="操作" width="120" fixed="right">
          <template #default="scope">
            <el-button type="primary" link @click="openItemEdit(scope.row)">编辑</el-button>
            <el-button type="danger" link @click="onDeleteItem(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>

    <!-- ==================== 新建 / 改名 ==================== -->
    <el-dialog
      v-model="headVisible"
      :title="headIsCreate ? '新建模板' : '改模板名'"
      width="520px"
      destroy-on-close
    >
      <el-form :model="headForm" label-width="90px">
        <el-form-item label="名称" required>
          <el-input v-model="headForm.name" placeholder="例如：寒假作业模板" clearable />
        </el-form-item>
        <el-form-item label="备注">
          <el-input
            v-model="headForm.note"
            type="textarea"
            :rows="2"
            placeholder="写给同事看的说明，用户看不到"
          />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="headForm.sort" :min="0" :max="9999" :step="10" />
          <span class="unit">小的靠前</span>
        </el-form-item>
        <!-- 归属只有超管能选：商家账号后端会强制改成 merchant + 自己，
             这里给下拉反而会让人以为「能挂到别人名下」。 -->
        <template v-if="headIsCreate && isAdmin">
          <el-form-item label="归属">
            <el-select v-model="headForm.scope" style="width: 240px">
              <el-option label="官方（全站兜底）" value="official" />
              <el-option label="某个商家" value="merchant" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="headForm.scope === 'merchant'" label="商家" required>
            <el-select v-model="headForm.owner_uid" filterable style="width: 300px">
              <el-option
                v-for="m in merchantOptions"
                :key="m.uid"
                :label="`${m.name || '（没填店名）'} · #${m.uid}`"
                :value="m.uid"
              />
            </el-select>
            <div class="hint">只有<b>已通过审核</b>的商家能挂，否则内容会挂在一个进不来的入口上。</div>
          </el-form-item>
        </template>
        <el-form-item v-if="headIsCreate" label="设为默认">
          <el-switch v-model="headForm.is_default" />
          <div class="hint">
            同一分组只能有一份默认。官方默认是<b>全站兜底</b>；商家的默认只影响他推广的用户。
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="headVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitHead">
          {{ headIsCreate ? '新建' : '保存' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- ==================== 条目编辑 ==================== -->
    <el-dialog
      v-model="itemVisible"
      :title="itemIsCreate ? '新增条目' : '编辑条目'"
      width="640px"
      destroy-on-close
    >
      <el-form :model="itemForm" label-width="110px">
        <el-form-item :label="kind === 'task' ? '任务名' : '兑换品名'" required>
          <el-input v-model="itemForm.name" placeholder="例如：完成数学作业" clearable />
        </el-form-item>
        <el-form-item :label="kind === 'task' ? '任务 id' : '条目 id'">
          <el-input
            v-model="itemForm.item_id"
            :disabled="!itemIsCreate"
            placeholder="留空自动生成"
            clearable
          />
          <div class="hint">建好之后不要再改（等于换了一条），所以编辑时是锁住的。</div>
        </el-form-item>
        <el-form-item label="图标">
          <el-input v-model="itemForm.emoji" placeholder="📚" clearable style="width: 90px" />
          <span class="hint" style="margin-left: 10px">留空按分类取默认图标</span>
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="itemForm.category" style="width: 200px">
            <el-option
              v-for="(m, k) in kind === 'task' ? TASK_CATEGORY : REDEEM_CATEGORY"
              :key="k"
              :label="m"
              :value="k"
            />
          </el-select>
        </el-form-item>

        <template v-if="kind === 'task'">
          <el-form-item label="周期">
            <el-select v-model="itemForm.cycle" style="width: 200px">
              <el-option v-for="(m, k) in CYCLE_META" :key="k" :label="m" :value="k" />
            </el-select>
            <div class="hint">
              「每日」是每天一份；「长期」用每周 / 每月 / 每年 —— 到期结算，没做完会过期。
            </div>
          </el-form-item>
          <el-form-item label="预计时长">
            <el-input-number v-model="itemForm.planned_minutes" :min="0" :max="600" :step="5" />
            <span class="unit">分钟</span>
          </el-form-item>
          <el-form-item label="基础积分">
            <el-input-number v-model="itemForm.base_points" :min="0" :max="9999" :step="1" />
            <span class="unit">分</span>
            <div class="hint">孩子按时按质完成拿这么多；超时按倍率折算，写 0 = 只打卡不给分。</div>
          </el-form-item>
          <el-form-item label="签到打卡">
            <el-switch v-model="itemForm.checkInEnabled" />
            <template v-if="itemForm.checkInEnabled">
              <span class="unit" style="margin-left: 12px">目标</span>
              <el-input-number
                v-model="itemForm.checkInTargetCount"
                :min="1"
                :max="60"
                :step="1"
              />
              <span class="unit">次</span>
            </template>
            <div class="hint">
              开 = 这条任务走「打卡 N 次」的玩法（练字签到、晨读都是这类），
              不做完一次就算完成。
            </div>
          </el-form-item>
          <el-form-item label="超时 / 评分">
            <el-checkbox v-model="itemForm.allowOvertime">允许超时完成</el-checkbox>
            <el-checkbox v-model="itemForm.allowLateNoPenalty">逾期不扣分</el-checkbox>
            <el-checkbox v-model="itemForm.qualityRated">家长可评质量</el-checkbox>
          </el-form-item>
          <el-form-item label="提醒时间">
            <el-input
              v-model="itemForm.remindAt"
              placeholder="18:30（留空 = 不提醒）"
              clearable
              style="width: 160px"
            />
          </el-form-item>
          <el-form-item label="奖励道具">
            <el-input
              v-model="itemForm.rewardItemIds"
              placeholder="sticker,medal（留空 = 不给道具）"
              clearable
            />
            <div class="hint">英文逗号分隔。道具 id 要跟 App 里那套一致，写错不报错、只是发不出来。</div>
          </el-form-item>
        </template>

        <template v-else>
          <el-form-item label="价格">
            <el-input-number v-model="itemForm.price" :min="0" :max="99999" :step="5" />
            <span class="unit">券</span>
          </el-form-item>
          <el-form-item label="每日限兑">
            <el-input-number v-model="itemForm.limit_per_day" :min="0" :max="99" :step="1" />
            <span class="unit">次（0 = 不限）</span>
          </el-form-item>
          <el-form-item label="库存">
            <el-input-number v-model="itemForm.stock" :min="-1" :max="99999" :step="1" />
            <div class="hint">-1 = 不限量。填正数时兑换完就下架。</div>
          </el-form-item>
          <el-form-item label="一次性">
            <el-switch v-model="itemForm.once" />
            <div class="hint">开 = 每个孩子只能换一次（「一次免做家务」这类）。</div>
          </el-form-item>
        </template>

        <el-form-item label="说明">
          <el-input
            v-model="itemForm.note"
            type="textarea"
            :rows="2"
            placeholder="给孩子看的一句话"
          />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="itemForm.sort" :min="0" :max="9999" :step="1" />
          <div class="hint">小的靠前。新条目默认排在最后，不用手改。</div>
        </el-form-item>
        <!-- ★ 不认识的玩法参数必须原样带回去：`item/save` 是整条覆盖，
             这里只提交表单里那几个键，其余（createdByParent / rewardItemIds …）
             会**静默消失**，而且保存还报成功。 -->
        <el-form-item v-if="itemRestKeys.length" label="其余参数">
          <span class="dim mono">{{ itemRestKeys.join('、') }}</span>
          <div class="hint">这些玩法参数不在表单里，保存时会原样保留。</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="itemVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitItem">
          {{ itemIsCreate ? '新增' : '保存' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref, watch } from 'vue'
  import { useRoute } from 'vue-router'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    createTemplate,
    deleteTemplate,
    deleteTemplateItem,
    getInviteCode,
    getTemplateDetail,
    getTemplateList,
    saveTemplateItem,
    setDefaultTemplate,
    updateTemplate
  } from '@/api/mallTemplate'
  import { getMerchantList } from '@/api/mallMerchant'
  import { useUserStore } from '@/pinia'
  import { formatDate } from '@/utils/format'

  defineOptions({ name: 'MallTemplate' })

  /* 分类名和 App 里 `features/tasks/ui.tsx` / `features/redeem/RedeemSheet.tsx`
     的 CATEGORY 一一对应 —— 后台自己编一套中文，运营对着孩子那边的
     分类名会认不出来。 */
  const TASK_CATEGORY = {
    study: '学习',
    chore: '家务',
    sport: '运动',
    art: '艺术',
    reading: '阅读',
    habit: '习惯'
  }
  const TASK_EMOJI = {
    study: '📚',
    chore: '🧹',
    sport: '⚽',
    art: '🎨',
    reading: '📖',
    habit: '🌟'
  }
  const REDEEM_CATEGORY = {
    snack: '零食饮料',
    screen: '屏幕时间',
    outing: '出去玩',
    gift: '礼物',
    privilege: '特权',
    custom: '其他'
  }
  /* 周期取值和 App `domain/types.ts` 的 TaskCycle 一致 */
  const CYCLE_META = {
    once: '一次性',
    daily: '每日',
    weekly: '每周',
    monthly: '每月',
    yearly: '每年'
  }
  const CYCLE_ORDER = ['daily', 'weekly', 'monthly', 'yearly', 'once']

  const categoryText = (v) => TASK_CATEGORY[v] ?? v ?? '—'
  const categoryEmoji = (v) => TASK_EMOJI[v] ?? '📝'
  const redeemCategoryText = (v) => REDEEM_CATEGORY[v] ?? v ?? '—'
  const cycleText = (v) => CYCLE_META[v] ?? v ?? '—'

  /* 平台时间戳是 **Unix 秒**，`formatDate` 吃毫秒。传秒会得到 1970 年且不报错。 */
  const formatTs = (ts) => (ts ? formatDate(ts * 1000) : '—')

  const userStore = useUserStore()
  const authorityId = computed(() =>
    Number(userStore.userInfo?.authority?.authorityId ?? 0)
  )
  const isAdmin = computed(() => authorityId.value === 888)

  /* 同一个组件挂在三个菜单上：任务模板 / 兑换模板 / 商家的「我的模板」。
     ★ Vue Router 复用同一实例时**不重跑 setup**，所以必须 watch。 */
  const MODE_BY_NAME = { templateTask: 'task', templateRedeem: 'redeem', merchantTemplates: 'task' }
  const route = useRoute()
  const kind = ref(MODE_BY_NAME[route.name] ?? 'task')
  watch(
    () => route.name,
    (n) => {
      kind.value = MODE_BY_NAME[n] ?? 'task'
      load()
    }
  )

  /* ---------- 商家名单（归属列 + 超管新建时选商家） ---------- */
  const merchants = ref([])
  const merchantById = ref({})
  /** 优先用招商表里的店铺名：平台用户列表只拉前 200 条，商家常常不在里面 */
  const merchantLabel = (uid) => merchantById.value[uid]?.name || `#${uid}`
  const merchantOptions = computed(() => merchants.value)
  const loadMerchants = async () => {
    if (!isAdmin.value) return
    const res = await getMerchantList({
      role: 'merchant',
      status: 'approved',
      page: 1,
      pageSize: 200
    })
    if (res.code !== 0) return
    merchants.value = res.data.list ?? []
    merchantById.value = Object.fromEntries(merchants.value.map((m) => [m.uid, m]))
  }

  /* ---------- 推广码 ---------- */
  const invite = ref({ uid: 0, code: '' })
  const loadInvite = async () => {
    try {
      const res = await getInviteCode()
      if (res.code === 0) {
        invite.value = { uid: res.data?.uid ?? 0, code: res.data?.invite_code ?? '' }
      }
    } catch {
      /* 拿不到就不显示那块，不拦页面 */
    }
  }
  const copyInvite = async () => {
    try {
      await navigator.clipboard.writeText(invite.value.code)
      ElMessage.success('已复制')
    } catch {
      ElMessage.warning('复制失败，请手动选中复制')
    }
  }

  /* ---------- 模板列表 ---------- */
  const rows = ref([])
  const loading = ref(false)

  const load = async () => {
    loading.value = true
    try {
      const res = await getTemplateList({ page: 1, pageSize: 100, kind: kind.value })
      if (res.code !== 0) {
        ElMessage.error(res.msg || '拉取模板列表失败')
        return
      }
      rows.value = res.data?.list ?? []
    } finally {
      /* ⚠️ 放 finally：失败时表格不能一直转圈 */
      loading.value = false
    }
  }
  const onKindChange = () => load()

  /* ---------- 模板头（新建 / 改名） ---------- */
  const headVisible = ref(false)
  const headIsCreate = ref(true)
  const submitting = ref(false)
  const headForm = ref({})
  const editingId = ref(0)

  const openCreate = () => {
    headIsCreate.value = true
    editingId.value = 0
    headForm.value = { name: '', note: '', sort: 100, scope: 'official', owner_uid: 0, is_default: false }
    headVisible.value = true
  }
  const openRename = (row) => {
    headIsCreate.value = false
    editingId.value = row.id
    headForm.value = { name: row.name, note: row.note || '', sort: row.sort ?? 0 }
    headVisible.value = true
  }

  const submitHead = async () => {
    const f = headForm.value
    if (!(f.name ?? '').trim()) {
      ElMessage.error('模板名不能为空')
      return
    }
    submitting.value = true
    try {
      const res = headIsCreate.value
        ? await createTemplate({
            name: f.name.trim(),
            kind: kind.value,
            /* 商家账号这两个字段会被后端强制收敛成 merchant + 自己，
               传上去只是为了让超管那条路能选 */
            scope: isAdmin.value ? f.scope : undefined,
            owner_uid: isAdmin.value ? f.owner_uid || 0 : undefined,
            note: (f.note ?? '').trim(),
            sort: f.sort,
            is_default: f.is_default
          })
        : await updateTemplate({
            id: editingId.value,
            name: f.name.trim(),
            note: (f.note ?? '').trim(),
            sort: f.sort
          })
      if (res.code === 0) {
        ElMessage.success(headIsCreate.value ? '已新建' : '已保存')
        headVisible.value = false
        load()
      }
    } finally {
      submitting.value = false
    }
  }

  const onSetDefault = async (row) => {
    const res = await setDefaultTemplate({ id: row.id })
    if (res.code === 0) {
      ElMessage.success(`「${row.name}」已设为默认`)
      load()
    }
  }

  const onDeleteTemplate = async (row) => {
    try {
      await ElMessageBox.confirm(
        `删除模板「${row.name}」？\n\n它下面的 ${row.item_count} 条内容会一起删掉。\n已经用过这份模板的用户不受影响（他们本地那份早就铺好了）。`,
        '删除模板',
        { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
      )
    } catch {
      return /* 用户点了取消 */
    }
    const res = await deleteTemplate({ id: row.id })
    if (res.code === 0) {
      ElMessage.success('已删除')
      load()
    }
  }

  /* ---------- 条目 ---------- */
  const itemsVisible = ref(false)
  const detailLoading = ref(false)
  const detail = ref(null)
  const items = ref([])
  const cycleFilter = ref('')
  const itemsTitle = computed(() =>
    detail.value ? `${detail.value.name} · ${items.value.length} 条` : '模板内容'
  )

  const visibleItems = computed(() => {
    const list = cycleFilter.value
      ? items.value.filter((it) => it.cycle === cycleFilter.value)
      : items.value
    /* 按周期分组排（每日 → 每周 → 每月 → 每年 → 一次性），
       同组里按运营自己的排序值。运营找「每日任务」时不用满表翻。 */
    return [...list].sort((a, b) => {
      const ia = CYCLE_ORDER.indexOf(a.cycle)
      const ib = CYCLE_ORDER.indexOf(b.cycle)
      if (ia !== ib) return (ia < 0 ? 99 : ia) - (ib < 0 ? 99 : ib)
      return (a.sort ?? 0) - (b.sort ?? 0)
    })
  })

  const openItems = async (row) => {
    itemsVisible.value = true
    cycleFilter.value = ''
    detail.value = { ...row }
    items.value = []
    await loadDetail(row.id)
  }

  const loadDetail = async (id) => {
    detailLoading.value = true
    try {
      const res = await getTemplateDetail({ id })
      if (res.code !== 0) {
        ElMessage.error(res.msg || '拉取模板内容失败')
        return
      }
      detail.value = res.data
      items.value = res.data?.items ?? []
    } finally {
      detailLoading.value = false
    }
  }

  /* Extra 是「服务端不解释、客户端要保留」的玩法参数。表单里只放
     几个常用的，其余必须原样带回去 —— 见提交处。 */
  const parseExtra = (raw) => {
    try {
      const v = JSON.parse(raw || '{}')
      return v && typeof v === 'object' && !Array.isArray(v) ? v : {}
    } catch {
      return {}
    }
  }
  const extraFlag = (row, k) => parseExtra(row.extra)[k] === true
  const extraNum = (row, k) => Number(parseExtra(row.extra)[k]) || 0

  const FORM_EXTRA_KEYS = [
    'allowOvertime',
    'allowLateNoPenalty',
    'qualityRated',
    'checkInEnabled',
    'checkInTargetCount',
    'remindAt',
    'rewardItemIds'
  ]
  const itemRestKeys = ref([])

  const itemVisible = ref(false)
  const itemIsCreate = ref(false)
  const itemForm = ref({})

  const blankItem = () => ({
    id: 0,
    item_id: '',
    name: '',
    emoji: '',
    category: kind.value === 'task' ? 'study' : 'privilege',
    cycle: 'daily',
    planned_minutes: 30,
    base_points: 10,
    price: 30,
    limit_per_day: 0,
    stock: -1,
    once: false,
    note: '',
    sort: items.value.length,
    checkInEnabled: false,
    checkInTargetCount: 5,
    allowOvertime: true,
    allowLateNoPenalty: false,
    qualityRated: true,
    remindAt: '',
    rewardItemIds: ''
  })

  const openItemCreate = () => {
    itemIsCreate.value = true
    itemRestKeys.value = []
    itemForm.value = blankItem()
    itemVisible.value = true
  }

  const openItemEdit = (row) => {
    itemIsCreate.value = false
    const ex = parseExtra(row.extra)
    itemRestKeys.value = Object.keys(ex).filter((k) => !FORM_EXTRA_KEYS.includes(k))
    itemForm.value = {
      id: row.id,
      item_id: row.item_id,
      name: row.name || '',
      emoji: row.emoji || '',
      category: row.category || (kind.value === 'task' ? 'study' : 'privilege'),
      cycle: row.cycle || 'daily',
      planned_minutes: row.planned_minutes ?? 30,
      base_points: row.base_points ?? 10,
      price: row.price ?? 0,
      limit_per_day: row.limit_per_day ?? 0,
      stock: row.stock ?? -1,
      once: !!row.once,
      note: row.note || '',
      sort: row.sort ?? 0,
      checkInEnabled: ex.checkInEnabled === true,
      checkInTargetCount: Number(ex.checkInTargetCount) || 5,
      allowOvertime: ex.allowOvertime === true,
      allowLateNoPenalty: ex.allowLateNoPenalty === true,
      qualityRated: ex.qualityRated === true,
      remindAt: typeof ex.remindAt === 'string' ? ex.remindAt : '',
      rewardItemIds: Array.isArray(ex.rewardItemIds) ? ex.rewardItemIds.join(',') : ''
    }
    itemVisible.value = true
  }

  /** 把表单里那几个玩法参数拼回 Extra，并**保留**不认识的键 */
  const buildExtra = (f, restKeys, oldRaw) => {
    const old = parseExtra(oldRaw)
    const out = {}
    for (const k of restKeys) if (k in old) out[k] = old[k]
    if (kind.value === 'task') {
      out.allowOvertime = !!f.allowOvertime
      out.allowLateNoPenalty = !!f.allowLateNoPenalty
      out.qualityRated = !!f.qualityRated
      out.checkInEnabled = !!f.checkInEnabled
      /* 关掉签到**不等于**把目标次数抹掉：官方种子里「背诵」就是
         `checkInEnabled:false` + `checkInTargetCount:2`（以后想开就是 2 次）。
         无条件清空会让「只是改了个积分」顺手改掉玩法参数。 */
      if (f.checkInEnabled) out.checkInTargetCount = Math.max(1, f.checkInTargetCount || 1)
      else if ('checkInTargetCount' in old) out.checkInTargetCount = old.checkInTargetCount
      const at = (f.remindAt ?? '').trim()
      if (at) out.remindAt = at
      const items = (f.rewardItemIds ?? '')
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
      if (items.length) out.rewardItemIds = items
    }
    return JSON.stringify(out)
  }

  const submitItem = async () => {
    const f = itemForm.value
    if (!(f.name ?? '').trim()) {
      ElMessage.error('名字不能为空')
      return
    }
    const oldRaw = itemIsCreate.value
      ? '{}'
      : items.value.find((it) => it.id === f.id)?.extra ?? '{}'
    submitting.value = true
    try {
      const res = await saveTemplateItem({
        id: f.id || 0,
        template_id: detail.value.id,
        item_id: itemIsCreate.value ? (f.item_id ?? '').trim() : f.item_id,
        name: f.name.trim(),
        emoji: (f.emoji ?? '').trim(),
        category: f.category,
        /* 两种 kind 共用一张表：用不上的那半列一律写零值，
           否则「任务改成兑换品」之后旧列还留着数，看着像配错了 */
        cycle: kind.value === 'task' ? f.cycle : '',
        planned_minutes: kind.value === 'task' ? f.planned_minutes || 0 : 0,
        base_points: kind.value === 'task' ? f.base_points || 0 : 0,
        price: kind.value === 'redeem' ? f.price || 0 : 0,
        limit_per_day: kind.value === 'redeem' ? f.limit_per_day || 0 : 0,
        stock: kind.value === 'redeem' ? (f.stock ?? -1) : -1,
        once: kind.value === 'redeem' ? !!f.once : false,
        note: (f.note ?? '').trim(),
        sort: f.sort,
        extra: buildExtra(f, itemRestKeys.value, oldRaw)
      })
      if (res.code === 0) {
        ElMessage.success(itemIsCreate.value ? '已新增' : '已保存')
        itemVisible.value = false
        await loadDetail(detail.value.id)
        load() /* 条目数变了，列表那一列要跟着变 */
      }
    } finally {
      submitting.value = false
    }
  }

  const onDeleteItem = async (row) => {
    try {
      await ElMessageBox.confirm(
        `删除「${row.name}」？\n\n只删模板里的这一条，已经按这份模板铺出去的用户不受影响。`,
        '删除条目',
        { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
      )
    } catch {
      return /* 用户点了取消 */
    }
    const res = await deleteTemplateItem({ id: row.id, template_id: detail.value.id })
    if (res.code === 0) {
      ElMessage.success('已删除')
      await loadDetail(detail.value.id)
      load()
    }
  }

  onMounted(async () => {
    await loadInvite()
    /* 商家名单失败不拦页面：归属列退化成 `#123`，改名 / 编辑内容才是主活 */
    await loadMerchants()
    await load()
  })
</script>

<style scoped>
  .tip {
    margin-bottom: 12px;
  }
  .tabs {
    margin-top: 4px;
  }
  .ml6 {
    margin-left: 6px;
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
  .unit {
    margin-left: 4px;
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
  .hint {
    margin-top: 2px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }
  .invite {
    display: flex;
    align-items: center;
    gap: 18px;
  }
  .invite-main {
    display: flex;
    align-items: baseline;
    gap: 10px;
  }
  .invite-label {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .invite-code {
    font-size: 20px;
    font-weight: 700;
    letter-spacing: 0.04em;
  }
  .invite-side {
    flex: 1;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
  .drawer-head {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 12px;
  }
</style>
