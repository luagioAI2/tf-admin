<template>
  <div>
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
            <el-button type="primary" icon="plus" @click="openCreate">新增商品</el-button>
            <span class="dim" style="margin-left: 12px">共 {{ itemTotal }} 个商品</span>
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
            <!-- 归属：`merchant_uid = 0` 是**平台自营**，不是「没填」。
                 商家后台只能看到 / 改到自己名下的那些（见 PortalService.MerchantItems），
                 所以这一列是运营判断「这条该不该我来动」的唯一依据。 -->
            <el-table-column align="left" label="归属" width="120">
              <template #default="scope">
                <span v-if="!scope.row.merchant_uid" class="dim">平台自营</span>
                <span v-else class="name">{{ merchantLabel(scope.row.merchant_uid) }}</span>
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
            <el-table-column align="left" label="操作" width="150" fixed="right">
              <template #default="scope">
                <el-button type="primary" link @click="openEdit(scope.row)">编辑</el-button>
                <el-button type="danger" link @click="onDelete(scope.row)">删除</el-button>
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

      <!-- ==================== ③ 激活码 ==================== -->
      <el-tab-pane label="激活码" name="code">
        <el-alert type="warning" :closable="false" show-icon class="tip">
          <template #title>
            一张卡 = 一个 6 位码。孩子把码敲进 App，那款游戏就进他的游戏库。
            <b>明文只在「生成」那一瞬间给你一次</b> —— 表里只存 sha256，
            关掉弹窗就再也拿不回来（只能作废重发）。生成完请<b>当场复制</b>去印卡片。
          </template>
        </el-alert>

        <div class="gva-search-box">
          <el-form :inline="true" :model="codeSearch">
            <el-form-item label="商品">
              <el-select
                v-model="codeSearch.item_id"
                placeholder="全部"
                clearable
                filterable
                style="width: 230px"
              >
                <el-option
                  v-for="it in codeItems"
                  :key="it.item_id"
                  :label="`${it.emoji || '🎁'} ${it.name}`"
                  :value="it.item_id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="批次">
              <el-select
                v-model="codeSearch.batch"
                placeholder="全部"
                clearable
                filterable
                style="width: 180px"
              >
                <el-option v-for="b in batches" :key="b" :label="b" :value="b" />
              </el-select>
            </el-form-item>
            <el-form-item label="状态">
              <el-select v-model="codeSearch.used" placeholder="全部" clearable style="width: 120px">
                <el-option label="未使用" :value="false" />
                <el-option label="已使用" :value="true" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" icon="search" @click="onCodeSearch">查询</el-button>
              <el-button icon="refresh" @click="onCodeReset">重置</el-button>
            </el-form-item>
          </el-form>
        </div>

        <div class="gva-table-box">
          <div class="gva-btn-list">
            <el-button type="primary" icon="plus" @click="openGenerate">生成激活码</el-button>
            <!-- 整批作废要先按批次筛选：不筛就等于「作废全部未使用的码」 -->
            <el-button type="danger" plain :disabled="!codeSearch.batch" @click="onVoidBatch">
              作废整批{{ codeSearch.batch ? `（${codeSearch.batch}）` : '' }}
            </el-button>
            <span class="dim" style="margin-left: 12px">共 {{ codeTotal }} 个码</span>
          </div>
          <el-table :data="codes" style="width: 100%" row-key="id">
            <el-table-column align="left" label="批次" width="150">
              <template #default="scope">
                <span class="mono">{{ scope.row.batch || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="商品" min-width="190">
              <template #default="scope">
                <span class="emoji">{{ itemById[scope.row.item_id]?.emoji || '🎁' }}</span>
                <span class="name">{{ itemById[scope.row.item_id]?.name || scope.row.item_id }}</span>
                <div v-if="itemById[scope.row.item_id]?.game_id" class="note faint mono">
                  → {{ itemById[scope.row.item_id].game_id }}
                </div>
              </template>
            </el-table-column>
            <el-table-column align="left" label="码尾" width="110">
              <template #default="scope">
                <!-- ★ 表里没有明文，只有后 4 位。家长拿卡来问「是不是这张」就看它 -->
                <span class="mono">····{{ scope.row.code_tail }}</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="状态" width="100">
              <template #default="scope">
                <el-tag :type="codeTagType(scope.row)" size="small">
                  {{ codeStateText(scope.row) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column align="left" label="谁用的" width="130">
              <template #default="scope">
                <span v-if="scope.row.used">{{ userName(scope.row.used_by_uid) }}</span>
                <span v-else class="dim">—</span>
              </template>
            </el-table-column>
            <el-table-column align="left" label="用掉时间" width="160">
              <template #default="scope">{{ formatTs(scope.row.used_at) }}</template>
            </el-table-column>
            <el-table-column align="left" label="生成时间" width="160">
              <template #default="scope">{{ formatTs(scope.row.created_at) }}</template>
            </el-table-column>
            <el-table-column align="left" label="操作" width="90" fixed="right">
              <template #default="scope">
                <el-button v-if="!scope.row.used" type="danger" link @click="onVoidOne(scope.row)">
                  作废
                </el-button>
                <span v-else class="dim">—</span>
              </template>
            </el-table-column>
          </el-table>
          <div class="gva-pagination">
            <el-pagination
              :current-page="codePage"
              :page-size="codePageSize"
              :page-sizes="[20, 50, 100, 200]"
              :total="codeTotal"
              layout="total, sizes, prev, pager, next, jumper"
              @current-change="onCodePage"
              @size-change="onCodeSize"
            />
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- ==================== 新建 / 编辑商品 ==================== -->
    <el-dialog
      v-model="editVisible"
      :title="isCreate ? '新增心愿商品' : '编辑心愿商品'"
      width="640px"
      destroy-on-close
    >
      <el-form :model="editForm" label-width="110px">
        <el-form-item label="分类" required>
          <el-select v-model="editForm.kind" style="width: 200px">
            <el-option v-for="(m, k) in KIND_META" :key="k" :label="m.label" :value="k" />
          </el-select>
          <div class="hint">
            「盲盒」和「单款游戏」是<b>数字商品</b>：兑换后直接进孩子的游戏库。
            其余是实物，走家长去外链买那条路。
          </div>
        </el-form-item>
        <el-form-item label="名字" required>
          <el-input v-model="editForm.name" placeholder="游戏盲盒 · 5 连开" clearable />
        </el-form-item>
        <el-form-item label="商品 id">
          <el-input
            v-model="editForm.item_id"
            :disabled="!isCreate"
            placeholder="留空自动生成"
            clearable
          />
          <div class="hint">
            C 端、家长定价、激活码都按它关联。
            <b>建好之后不要再改</b>（改了等于换了一件商品），所以编辑时是锁住的。
          </div>
        </el-form-item>
        <el-form-item label="图标">
          <el-input v-model="editForm.emoji" placeholder="🎁" clearable style="width: 90px" />
          <el-select v-model="editForm.tint" placeholder="底色" style="width: 130px; margin-left: 10px">
            <el-option v-for="t in TINTS" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="editForm.kind === 'gamebox'" label="哪一款游戏" required>
          <el-select v-model="editForm.game_id" filterable style="width: 320px">
            <el-option
              v-for="g in gameOptions"
              :key="g.game_id"
              :label="`${g.emoji || '🎮'} ${g.name || g.game_id}`"
              :value="g.game_id"
            />
          </el-select>
        </el-form-item>
        <el-form-item v-if="editForm.kind === 'blindbox'" label="盲盒池" required>
          <el-select
            v-model="editForm.pool"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            placeholder="选中所有可以开出来的游戏"
            style="width: 100%"
          >
            <el-option
              v-for="g in gameOptions"
              :key="g.game_id"
              :label="`${g.emoji || '🎮'} ${g.name || g.game_id}`"
              :value="g.game_id"
            />
          </el-select>
          <div class="hint">
            池子里写<b>全部</b>游戏是对的 —— 孩子已经有的那几款，
            开奖时会自动减掉，不会开出重复的。
          </div>
        </el-form-item>
        <el-form-item v-if="editForm.kind === 'blindbox'" label="一次开出几款">
          <el-input-number v-model="editForm.unlockCount" :min="1" :max="20" :step="1" />
          <span class="dim" style="margin-left: 8px">款</span>
          <div class="hint">
            1 = 单开盲盒；5 = 「5 连开」。<b>已拥有的不占名额</b> ——
            开奖时先减掉孩子已有的，再从剩下的里抽这么多款。
            剩余不足这个数时孩子那边换不了（不会「有几款给几款」）。
          </div>
        </el-form-item>
        <el-form-item label="合作方">
          <el-input v-model="editForm.partner_name" placeholder="泡泡玛特（实物才填）" clearable />
        </el-form-item>
        <el-form-item label="参考价">
          <el-input-number
            v-model="editForm.priceYuan"
            :min="0"
            :max="99999"
            :step="1"
            :precision="2"
          />
          <span class="dim" style="margin-left: 8px">元</span>
          <div class="hint">
            只是给家长看的<b>参考价</b>，不参与任何计算。
            真正扣多少券在家长自己那一屏定。
          </div>
        </el-form-item>
        <el-form-item label="适龄">
          <el-input-number v-model="editForm.ageMin" :min="0" :max="18" :step="1" />
          <span class="dim" style="margin: 0 8px">到</span>
          <el-input-number v-model="editForm.ageMax" :min="0" :max="18" :step="1" />
          <span class="dim" style="margin-left: 8px">岁（都不填 = 不显示）</span>
        </el-form-item>
        <el-form-item label="一句话说明">
          <el-input v-model="editForm.note" placeholder="随机开出一款你没玩过的游戏" clearable />
        </el-form-item>
        <el-form-item label="兑换须知">
          <el-input
            v-model="editForm.notice"
            type="textarea"
            :rows="2"
            placeholder="不指定款式 / 兑换后不退"
          />
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
          <div class="hint">数字越小越靠前。盲盒永远排在最前，不受这个影响。</div>
        </el-form-item>
        <!-- 归属：0 = 平台自营。改的是「谁维护它」，不是「谁收钱」（结算还没做）。
             商家后台只能看到 / 改到自己名下的商品，见 PortalService.MerchantItems。 -->
        <el-form-item label="归属">
          <el-select v-model="editForm.merchant_uid" filterable style="width: 320px">
            <el-option :value="0" label="平台自营（运营维护）" />
            <el-option
              v-for="o in ownerOptions"
              :key="o.value"
              :value="o.value"
              :label="o.label"
            />
          </el-select>
          <div class="hint">
            划给商家后，这条就只出现在他的后台里。<b>不会自动改价</b>，
            已经兑换过的订单也不受影响。
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
          <div class="hint">下架只是从孩子的心愿列表里藏起来，<b>已有订单不受影响</b>。</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitEdit">
          {{ isCreate ? '新建' : '保存' }}
        </el-button>
      </template>
    </el-dialog>
    <!-- ==================== 生成激活码 ==================== -->
    <el-dialog v-model="genVisible" title="生成激活码" width="540px" destroy-on-close>
      <el-form :model="genForm" label-width="100px">
        <el-form-item label="商品" required>
          <el-select
            v-model="genForm.item_id"
            filterable
            style="width: 100%"
            placeholder="只能选「游戏」分类的商品"
          >
            <el-option
              v-for="it in codeItems"
              :key="it.item_id"
              :label="`${it.emoji || '🎁'} ${it.name}`"
              :value="it.item_id"
            />
          </el-select>
          <div class="hint">
            码只解锁<b>一款具体游戏</b>，所以只能挂在「游戏」分类的商品上。
            挂在实物或盲盒上，核销会「成功」但游戏进不了库 —— 那张卡就白卖了。
            下拉里没有想卖的那款？先去「心愿商品」建一个分类为「游戏」的商品。
          </div>
        </el-form-item>
        <el-form-item label="数量" required>
          <el-input-number v-model="genForm.count" :min="1" :max="500" :step="10" />
          <span class="dim" style="margin-left: 8px">个（一次最多 500）</span>
        </el-form-item>
        <el-form-item label="批次号">
          <el-input v-model="genForm.batch" placeholder="留空自动（B20261103-1244）" clearable />
          <div class="hint">印卡片时按批次发；以后要「作废整批」就靠它。</div>
        </el-form-item>
        <el-form-item label="有效期">
          <el-input-number v-model="genForm.expireDays" :min="0" :max="3650" :step="30" />
          <span class="dim" style="margin-left: 8px">天（0 = 永不过期）</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="genVisible = false">取消</el-button>
        <el-button type="primary" :loading="genLoading" @click="submitGenerate">生成</el-button>
      </template>
    </el-dialog>

    <!-- ==================== 生成结果：明文只此一次 ==================== -->
    <el-dialog
      v-model="resultVisible"
      title="激活码已生成 —— 请现在复制走"
      width="560px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      :before-close="onResultClose"
    >
      <el-alert type="warning" :closable="false" show-icon>
        <template #title>
          表里<b>只存哈希</b>。这 {{ resultCodes.length }} 个码关掉这个窗口就
          <b>永远拿不回来</b>（只能作废重发）。请先「复制全部」。
        </template>
      </el-alert>
      <div class="codebox mono">
        <div v-for="(c, i) in resultCodes" :key="c" class="codeline">
          <span class="idx">{{ i + 1 }}</span>
          <span>{{ c }}</span>
        </div>
      </div>
      <template #footer>
        <el-button @click="copyCodes">{{ copied ? '已复制 ✓' : '复制全部' }}</el-button>
        <el-button type="primary" @click="closeResult">我已复制</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref, watch } from 'vue'
  import { useRoute } from 'vue-router'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    createWishItem,
    deleteWishItem,
    getWishItemList,
    getWishOrderList,
    updateWishItem
  } from '@/api/mallWish'
  import { getGameOptions } from '@/api/mallGame'
  import { getMerchantList } from '@/api/mallMerchant'
  import {
    generateWishCodes,
    getWishCodeBatches,
    getWishCodeList,
    voidWishCodes
  } from '@/api/mallWishCode'
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

  /* 同一个组件挂在三个菜单上：心愿商品 / 心愿单 / 游戏激活码。
     由路由 name 决定默认落在哪个页签。★ Vue Router 复用同一组件实例时
     **不会重新执行 setup**，所以必须 watch —— 只给 ref 一个初值会串页
     （从「心愿商品」点菜单进「心愿单」，还停在上一页那个页签）。 */
  const MODE_BY_NAME = { mallWish: 'item', wishOrder: 'order', gameCode: 'code' }
  const route = useRoute()
  const tab = ref(MODE_BY_NAME[route.name] ?? 'item')
  watch(
    () => route.name,
    (n) => { tab.value = MODE_BY_NAME[n] ?? 'item' }
  )

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

  /* ---------- 商家（「归属」下拉 + 列表里把 uid 换成店铺名） ---------- */
  const merchants = ref([])
  const merchantById = ref({})
  /** 优先用招商表里的 `name`（店铺名）：平台用户列表只拉前 200 条，
      游客能把首页顶满，商家常常根本不在里面，`userName` 会退化成 `#123`。 */
  const merchantLabel = (uid) => merchantById.value[uid]?.name || userName(uid)
  const loadMerchants = async () => {
    /* 只拉**已通过**的：待审核 / 被拒的还进不了后台，
       把商品划给他们 = 挂在一个不存在的入口上（商家自己也看不到）。 */
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

  /* ---------- 新建 / 编辑 / 删除商品 ---------- */
  const editVisible = ref(false)
  const submitting = ref(false)
  /** true = 这次是「新增」。同一个弹窗两用，靠它切标题、切 item_id 是否可改 */
  const isCreate = ref(false)
  const gameOptions = ref([])
  /** 图标底色档位，和 App 里 `WishTint` 的取值一一对应 */
  const TINTS = [
    { value: 'sun', label: '暖黄' },
    { value: 'grass', label: '草绿' },
    { value: 'sky', label: '天蓝' },
    { value: 'grape', label: '葡萄紫' },
    { value: 'tan', label: '沙色' },
    { value: 'berry', label: '莓红' }
  ]

  const blankForm = () => ({
    item_id: '',
    kind: 'blindbox',
    name: '',
    emoji: '🎁',
    tint: 'sun',
    partner_name: '',
    game_id: '',
    pool: [],
    /** 盲盒一次开几款。1 = 单开（老口径），5 = 5 连开 */
    unlockCount: 1,
    /** 元。提交时 ×100 成 `price_cents` —— 界面上写「69 元」比「6900 分」好核对 */
    priceYuan: 0,
    ageMin: 0,
    ageMax: 0,
    note: '',
    notice: '',
    buy_url: '',
    buy_label: '',
    sort_order: 100,
    /** 0 = 平台自营。商家后台看不到 `merchant_uid = 0` 的商品 */
    merchant_uid: 0,
    archived: false
  })
  const editForm = ref(blankForm())

  /** 归属下拉：值 0 = 平台自营，其余是已通过审核的商家。 */
  const ownerOptions = computed(() => {
    const opts = merchants.value.map((m) => ({
      value: m.uid,
      label: `${m.name || '（没填店名）'} · #${m.uid}`
    }))
    /* 兜底：商品挂在「已经不是已通过商家」的人名下（申请被撤 / 账号注销），
       下拉里必须留着这一项，否则 el-select 只显示一个光秃秃的数字，
       运营会以为归属没读到，随手一改就把数据改错了。 */
    const cur = editForm.value.merchant_uid
    if (cur && !opts.some((o) => o.value === cur)) {
      opts.unshift({ value: cur, label: `#${cur}（不在已通过名单里）` })
    }
    return opts
  })

  /** `[6,12]` 这种 JSON 文本 → 两个数字。坏数据当「没填」，别让编辑框炸掉 */
  const parseAge = (raw) => {
    try {
      const v = JSON.parse(raw || '[]')
      if (!Array.isArray(v)) return [0, 0]
      return [Number(v[0]) || 0, Number(v[1]) || 0]
    } catch {
      return [0, 0]
    }
  }
  const parsePool = (raw) => {
    try {
      const v = JSON.parse(raw || '[]')
      return Array.isArray(v) ? v.filter((x) => typeof x === 'string') : []
    } catch {
      return []
    }
  }

  const openCreate = () => {
    isCreate.value = true
    editForm.value = blankForm()
    editVisible.value = true
  }

  const openEdit = (row) => {
    isCreate.value = false
    const [ageMin, ageMax] = parseAge(row.age_range)
    editForm.value = {
      item_id: row.item_id,
      kind: row.kind || 'blindbox',
      name: row.name || '',
      emoji: row.emoji || '',
      tint: row.tint || 'sun',
      partner_name: row.partner_name || '',
      game_id: row.game_id || '',
      pool: parsePool(row.pool),
      unlockCount: row.unlock_count || 1,
      priceYuan: (row.price_cents || 0) / 100,
      ageMin,
      ageMax,
      note: row.note || '',
      notice: row.notice || '',
      buy_url: row.buy_url || '',
      buy_label: row.buy_label || '',
      sort_order: row.sort_order ?? 100,
      merchant_uid: row.merchant_uid || 0,
      archived: !!row.archived
    }
    editVisible.value = true
  }

  const submitEdit = async () => {
    const f = editForm.value
    const name = (f.name ?? '').trim()
    if (!name) {
      ElMessage.error('名字不能为空')
      return
    }
    /* 这两条后端也拦，但在这里先说 —— 用户填了半天才知道不行 */
    if (f.kind === 'gamebox' && !f.game_id) {
      ElMessage.error('「单款游戏」必须选一款')
      return
    }
    if (f.kind === 'blindbox' && f.pool.length === 0) {
      ElMessage.error('盲盒至少要选一款候选游戏')
      return
    }
    /* 前端先挡一下明显不对的链接：后端只做非空校验，
       写进 `javascript:...` 会原样出现在家长确认页的 href 上。 */
    const url = (f.buy_url ?? '').trim()
    if (url && !/^https?:\/\//i.test(url)) {
      ElMessage.error('外链必须以 http:// 或 https:// 开头')
      return
    }

    /* 适龄：两个都没填就不传 —— 传 `[0,0]` 会被当成「0~0 岁」显示出来 */
    const age = f.ageMin || f.ageMax ? [f.ageMin || 0, f.ageMax || 0] : []
    const common = {
      kind: f.kind,
      name,
      emoji: (f.emoji ?? '').trim(),
      tint: f.tint,
      partner_name: (f.partner_name ?? '').trim(),
      /* 分类换了就把用不上的那个字段清掉，否则「盲盒改成游戏」之后
         池子还留在表里，C 端按 kind 读不到、看着像数据错了 */
      game_id: f.kind === 'gamebox' ? f.game_id : '',
      pool: f.kind === 'blindbox' ? f.pool : [],
      /* 非盲盒一律回到 1：不然「盲盒改成游戏」之后 unlock_count 还留着 5，
         C 端不读它，但后台列表和历史数据会看着像配错了 */
      unlock_count: f.kind === 'blindbox' ? Math.max(1, f.unlockCount || 1) : 1,
      price_cents: Math.round((f.priceYuan || 0) * 100),
      age_range: age,
      note: (f.note ?? '').trim(),
      notice: (f.notice ?? '').trim(),
      buy_url: url,
      buy_label: (f.buy_label ?? '').trim(),
      sort_order: f.sort_order,
      merchant_uid: f.merchant_uid || 0,
      archived: f.archived
    }

    submitting.value = true
    try {
      const res = isCreate.value
        ? await createWishItem({ ...common, item_id: (f.item_id ?? '').trim() })
        : await updateWishItem({ ...common, item_id: f.item_id })
      if (res.code === 0) {
        ElMessage.success(isCreate.value ? '已新建' : '已保存')
        editVisible.value = false
        getItems()
      }
    } finally {
      /* ⚠️ 放 finally：失败时按钮必须恢复，否则「保存」永远转下去 */
      submitting.value = false
    }
  }

  /* ★ 删除只对「一次都没被换过」的商品有效。
     换过的会被后端挡住 —— 那些订单的历史要能解释（孩子当时确实付了券）。
     所以这里先问一句，并把「只想不卖」的替代方案写在提示里。 */
  const onDelete = async (row) => {
    try {
      await ElMessageBox.confirm(
        `确定删除「${row.name}」吗？\n\n只删商品本身，不退款、不影响已有订单。\n已经被人兑换过的商品删不掉 —— 那种请用「编辑 → 下架」。`,
        '删除商品',
        { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
      )
    } catch {
      return /* 用户点了取消 */
    }
    const res = await deleteWishItem({ item_id: row.item_id })
    if (res.code === 0) {
      ElMessage.success('已删除')
      getItems()
    }
  }

  /* ---------- 激活码 ---------- */
  const codes = ref([])
  const codeTotal = ref(0)
  const codePage = ref(1)
  const codePageSize = ref(20)
  const codeSearch = ref({ item_id: '', batch: '', used: '' })
  const batches = ref([])

  /* ★ 商品下拉必须用**全量**清单，不能用 `items` ——
     那个是当前页（默认 10 条）。用它做下拉会「明明有这件商品却选不到」，
     而且换一页下拉内容就变，像随机坏掉。 */
  const allItems = ref([])
  const itemById = computed(() =>
    Object.fromEntries(allItems.value.map((it) => [it.item_id, it]))
  )
  /* 能挂码的只有「游戏」分类且绑了 game_id 的 ——
     后端 GenerateCodes 也会拦，这里先过滤省得白填一遍。 */
  const codeItems = computed(() =>
    allItems.value.filter((it) => it.kind === 'gamebox' && !!it.game_id)
  )

  const loadAllItems = async () => {
    const res = await getWishItemList({ page: 1, pageSize: 200 })
    if (res.code === 0) allItems.value = res.data.list ?? []
  }

  const getCodes = async () => {
    const data = { page: codePage.value, pageSize: codePageSize.value }
    if (codeSearch.value.item_id) data.item_id = codeSearch.value.item_id
    if (codeSearch.value.batch) data.batch = codeSearch.value.batch
    /* `used` 是 bool 指针：`false` = 只看没用过的，`undefined` 才是不过滤。
       空值必须整个字段不传 —— 传 `''` 会被当成 false，看着像「筛了但没筛」。 */
    if (codeSearch.value.used === true || codeSearch.value.used === false) {
      data.used = codeSearch.value.used
    }
    const res = await getWishCodeList(data)
    if (res.code === 0) {
      codes.value = res.data.list ?? []
      codeTotal.value = res.data.total ?? 0
    }
  }
  const onCodeSearch = () => {
    codePage.value = 1
    getCodes()
  }
  const onCodeReset = () => {
    codeSearch.value = { item_id: '', batch: '', used: '' }
    codePage.value = 1
    getCodes()
  }
  const onCodePage = (v) => {
    codePage.value = v
    getCodes()
  }
  const onCodeSize = (v) => {
    codePageSize.value = v
    getCodes()
  }

  const getBatches = async () => {
    try {
      const r = await getWishCodeBatches()
      batches.value = r?.data ?? []
    } catch {
      batches.value = [] /* 拉不到就退化成空下拉，别拦整页 */
    }
  }

  /* 过期是**核销时真会拦**的（平台 `RedeemCode` 里比 `expire_at`），
     所以这里敢写「已过期」；不然就是在骗运营。 */
  const isExpired = (row) =>
    !row.used && row.expire_at > 0 && row.expire_at * 1000 < Date.now()
  const codeStateText = (row) => (row.used ? '已使用' : isExpired(row) ? '已过期' : '未使用')
  const codeTagType = (row) => (row.used ? 'info' : isExpired(row) ? 'warning' : 'success')

  /* ---------- 生成 ---------- */
  const genVisible = ref(false)
  const genLoading = ref(false)
  const genForm = ref({ item_id: '', count: 50, batch: '', expireDays: 0 })

  const openGenerate = () => {
    genForm.value = {
      item_id: codeItems.value[0]?.item_id ?? '',
      count: 50,
      batch: '',
      expireDays: 0
    }
    genVisible.value = true
  }

  const submitGenerate = async () => {
    const f = genForm.value
    if (!f.item_id) {
      ElMessage.error('先选一个「游戏」分类的商品')
      return
    }
    if (!(f.count > 0)) {
      ElMessage.error('数量要大于 0')
      return
    }
    genLoading.value = true
    try {
      const res = await generateWishCodes({
        item_id: f.item_id,
        count: f.count,
        batch: (f.batch ?? '').trim(),
        expire_days: f.expireDays || 0
      })
      if (res.code === 0) {
        resultCodes.value = res.data.codes ?? []
        copied.value = false
        genVisible.value = false
        resultVisible.value = true
        /* 批次号可能是后端自动补的，刷新下拉 + 列表才看得到 */
        getBatches()
        getCodes()
      }
    } finally {
      /* ⚠️ 放 finally：失败时按钮必须恢复，否则「生成」永远转下去 */
      genLoading.value = false
    }
  }

  /* ---------- 结果面板（明文只此一次） ---------- */
  const resultVisible = ref(false)
  const resultCodes = ref([])
  const copied = ref(false)

  /* 剪贴板：`navigator.clipboard` 只在 https / localhost 下可用，
     而后台常常是 http://内网IP ⇒ 必须有 execCommand 兜底，
     否则「复制」静默失败，运营以为复制到了，其实印出来是空的。 */
  const writeClipboard = async (text) => {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(text)
        return true
      }
    } catch {
      /* 落到下面的兜底 */
    }
    try {
      const ta = document.createElement('textarea')
      ta.value = text
      ta.style.position = 'fixed'
      ta.style.opacity = '0'
      document.body.appendChild(ta)
      ta.select()
      const ok = document.execCommand('copy')
      document.body.removeChild(ta)
      return ok
    } catch {
      return false
    }
  }

  const copyCodes = async () => {
    const ok = await writeClipboard(resultCodes.value.join('\n'))
    copied.value = ok
    if (ok) ElMessage.success('已复制，去印卡片吧')
    else ElMessage.error('复制失败，请手动全选复制')
  }

  const closeResult = () => {
    resultVisible.value = false
    resultCodes.value = []
  }

  /* ★ 关窗口前问一句：明文只有这一份，误点一下就得作废重发整批。 */
  const onResultClose = async (done) => {
    if (resultCodes.value.length && !copied.value) {
      try {
        await ElMessageBox.confirm(
          `这 ${resultCodes.value.length} 个码还没复制。关掉之后表里只剩哈希，找不回来了 —— 只能作废重发。`,
          '确定关掉？',
          { type: 'warning', confirmButtonText: '关掉（不要了）', cancelButtonText: '回去复制' }
        )
      } catch {
        return /* 用户点了「回去复制」 */
      }
    }
    resultCodes.value = []
    done()
  }

  /* ---------- 作废 ---------- */
  const onVoidOne = async (row) => {
    try {
      await ElMessageBox.confirm(
        `作废「····${row.code_tail}」这张码？（批次 ${row.batch || '—'}）\n\n作废后家长再输会提示「码无效」。已经用掉的码不能作废。`,
        '作废激活码',
        { type: 'warning', confirmButtonText: '作废', cancelButtonText: '取消' }
      )
    } catch {
      return /* 用户点了取消 */
    }
    const res = await voidWishCodes({ id: row.id })
    if (res.code === 0) {
      ElMessage.success('已作废')
      getCodes()
    }
  }

  const onVoidBatch = async () => {
    const b = codeSearch.value.batch
    if (!b) {
      ElMessage.warning('先在上面按批次筛选，再作废整批')
      return
    }
    try {
      await ElMessageBox.confirm(
        `作废批次「${b}」里所有没用过的码？\n\n已经用掉的不动（那个孩子的游戏已经到账）。这个操作不可撤销。`,
        '作废整批',
        { type: 'warning', confirmButtonText: '作废整批', cancelButtonText: '取消' }
      )
    } catch {
      return /* 用户点了取消 */
    }
    const res = await voidWishCodes({ batch: b })
    if (res.code === 0) {
      ElMessage.success(`已作废 ${res.data?.voided ?? 0} 个`)
      getCodes()
    }
  }

  onMounted(async () => {
    await loadUsers()
    /* 商家名单失败不拦页面：归属下拉退化成只有「平台自营」，
       总比整页白屏好（改价 / 上下架才是这一屏的主活）。 */
    await loadMerchants()
    /* 游戏下拉：盲盒池和「单款游戏」都要按 game_id 选。
       失败不拦页面 —— 拉不到就退化成空下拉，比整页白屏好。 */
    try {
      const r = await getGameOptions()
      gameOptions.value = r?.data ?? []
    } catch {
      gameOptions.value = []
    }
    await loadAllItems()
    await Promise.all([getItems(), getOrders(), getCodes(), getBatches()])
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
  /* 生成结果：明文要能一眼扫完、对得上号，所以等宽 + 序号 */
  .codebox {
    margin-top: 12px;
    max-height: 320px;
    overflow: auto;
    padding: 8px 12px;
    border: 1px solid var(--el-border-color);
    border-radius: 6px;
    background: var(--el-fill-color-light);
    font-size: 13px;
    line-height: 1.8;
  }
  .codeline {
    display: flex;
    gap: 12px;
  }
  .codeline .idx {
    min-width: 32px;
    color: var(--el-text-color-secondary);
    text-align: right;
  }
</style>
