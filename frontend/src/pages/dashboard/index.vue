<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { api } from '@/api'
import DoctorRepairPanel from '@/components/DoctorRepairPanel.vue'
import type { ChecklistListItem, DoctorIssue, PortfolioPosition, RepairAction } from '@/api/types'

const positions = ref<PortfolioPosition[]>([])
const pendingItems = ref<ChecklistListItem[]>([])
const watchCount = ref(0)
const doctorOk = ref(true)
const doctorIssues = ref<DoctorIssue[]>([])
const repairActions = ref<RepairAction[]>([])
const doctorExpanded = ref(false)
const repairSaving = ref(false)
const loading = ref(true)

const readonlyIssues = computed(() => {
  const covered = new Set(
    repairActions.value.map((a) => `${a.code}|${a.subject || ''}|${a.title}`),
  )
  return doctorIssues.value.filter(
    (iss) => !covered.has(`${iss.code || ''}|${iss.subject || ''}|${iss.title}`),
  )
})

const actions = [
  { type: 'add', title: '我要加仓', desc: '已有持仓，追加买入', icon: '➕', needsCode: true, pickHolding: true },
  { type: 'sell', title: '我要卖出', desc: '减仓或清仓，FIFO 自动分配', icon: '📉', needsCode: true, pickHolding: true },
  { type: 'inspect', title: '巡检持仓', desc: '检视持有逻辑是否仍成立', icon: '🔍', needsCode: true, pickHolding: true },
]

async function load() {
  loading.value = true
  try {
    const [p, w, d, drafts, submitted] = await Promise.all([
      api.getPortfolio(),
      api.getWatchlist(),
      api.doctor('all').catch(() => ({ ok: false, issues: [{ title: 'doctor 不可用' }] })),
      api.getChecklists({ status: 'draft', limit: 20 }),
      api.getChecklists({ status: 'submitted', limit: 20 }),
    ])
    positions.value = (p.positions || []).filter((x) => x.state === 'holding')
    pendingItems.value = [...(drafts.checklists || []), ...(submitted.checklists || [])]
    watchCount.value = (w.items || []).length
    doctorOk.value = d.ok
    doctorIssues.value = d.issues || []
    repairActions.value = d.repair_actions || []
    doctorExpanded.value = !d.ok
  } catch (e: unknown) {
    uni.showToast({ title: (e as Error).message || '加载失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

function positionTypeLabel(t?: string) {
  const m: Record<string, string> = { core: '主仓', swing: '波段' }
  return m[t || ''] || t || '—'
}

function stateLabel(s: string) {
  const m: Record<string, string> = { holding: '持有中', closed: '已清仓' }
  return m[s] || s
}

function formatShares(shares?: string | null) {
  if (!shares) return '—'
  const n = Number(shares)
  return Number.isNaN(n) ? shares : `${n.toLocaleString()} 股`
}

function pendingStatusLabel(s: string) {
  const m: Record<string, string> = { draft: '草稿', submitted: '待批准' }
  return m[s] || s
}

function pendingActionLabel(t: string) {
  const m: Record<string, string> = {
    buy: '建仓', add: '加仓', sell: '卖出', inspect: '巡检', watch: '观察',
  }
  return m[t] || t
}

function resumePending(c: ChecklistListItem) {
  const q = new URLSearchParams()
  q.set('resume_id', c.id)
  uni.navigateTo({ url: `/pages/wizard/index?${q.toString()}` })
}

function dismissPending(c: ChecklistListItem) {
  uni.showModal({
    title: '作废此决策？',
    content: `${c.summary || c.id}\n作废后将从「进行中的决策」移除，不影响已批准记录。`,
    confirmText: '作废',
    confirmColor: '#dc2626',
    success: async (res) => {
      if (!res.confirm) return
      try {
        await api.reject(c.id, '用户在首页手动作废')
        pendingItems.value = pendingItems.value.filter((x) => x.id !== c.id)
        uni.showToast({ title: '已作废', icon: 'success' })
      } catch (e: unknown) {
        uni.showToast({ title: (e as Error).message || '作废失败', icon: 'none' })
      }
    },
  })
}

function goPositionRecords(p: PortfolioPosition) {
  const q = new URLSearchParams()
  q.set('code', p.code)
  q.set('name', p.name || p.code)
  uni.navigateTo({ url: `/pages/records/index?${q.toString()}` })
}

function startWizard(action: (typeof actions)[0], pos?: PortfolioPosition) {
  const q = new URLSearchParams()
  q.set('type', action.type)
  if (pos) {
    q.set('code', pos.code)
    q.set('name', pos.name || pos.code)
    const jid = pos.journal_ids?.[0]
    if (jid) q.set('journal_id', jid)
  }
  uni.navigateTo({ url: `/pages/wizard/index?${q.toString()}` })
}

function goPool(zone?: string) {
  const q = zone ? `?zone=${zone}` : ''
  uni.navigateTo({ url: `/pages/pool/index${q}` })
}

function goBuyFromPool() {
  uni.navigateTo({ url: '/pages/pool/index?zone=watching' })
}

function goResearch() {
  uni.navigateTo({ url: '/pages/research/index' })
}

function onAction(action: (typeof actions)[0]) {
  if (action.pickHolding && positions.value.length === 0) {
    uni.showModal({
      title: '暂无持仓',
      content: '请先通过「我要买入」建立持仓',
      showCancel: false,
    })
    return
  }
  if (action.pickHolding && positions.value.length === 1) {
    startWizard(action, positions.value[0])
    return
  }
  if (action.pickHolding) {
    uni.showActionSheet({
      itemList: positions.value.map((p) => `${p.name || p.code} (${p.code})`),
      success: (res) => startWizard(action, positions.value[res.tapIndex]),
    })
    return
  }
  startWizard(action)
}

async function saveRepairs(
  actions: { id: string; enabled: boolean; values: Record<string, string> }[],
) {
  repairSaving.value = true
  try {
    const res = await api.doctorRepair(actions)
    doctorOk.value = res.ok
    doctorIssues.value = res.issues || []
    repairActions.value = res.repair_actions || []
    uni.showToast({
      title: res.message || (res.ok ? '修复完成' : '部分问题仍待处理'),
      icon: res.ok ? 'success' : 'none',
    })
    if (res.ok) doctorExpanded.value = false
    else await load()
  } catch (e: unknown) {
    uni.showToast({ title: (e as Error).message || '保存失败', icon: 'none' })
  } finally {
    repairSaving.value = false
  }
}

onShow(() => load())
</script>

<template>
  <view class="page">
    <view class="header">
      <text class="title">投资助手</text>
      <text class="subtitle">决策闭环</text>
    </view>

    <view v-if="!doctorOk" class="banner warn">
      <view class="banner-head" @click="doctorExpanded = !doctorExpanded">
        <text>数据体检未通过（{{ doctorIssues.length }} 项）</text>
        <text class="banner-toggle">{{ doctorExpanded ? '收起' : '展开详情' }}</text>
      </view>
      <text class="banner-hint">
        portfolio.yaml 与 SQLite 账本不一致。可在下方勾选修复项后保存对齐，修复前建议勿 approve 新决策。
      </text>
      <view v-if="doctorExpanded">
        <DoctorRepairPanel
          v-if="repairActions.length > 0"
          :actions="repairActions"
          :saving="repairSaving"
          @save="saveRepairs"
        />
        <view v-if="readonlyIssues.length > 0" class="readonly-block">
          <text class="readonly-title">其他问题（须手动处理）</text>
          <view v-for="(iss, i) in readonlyIssues" :key="i" class="issue-item">
            <text class="issue-title">
              <text v-if="iss.code" class="issue-code">{{ iss.code }}</text>
              {{ iss.subject ? `${iss.subject} · ` : '' }}{{ iss.title }}
            </text>
            <text v-if="iss.detail" class="issue-detail">发现：{{ iss.detail }}</text>
            <text v-if="iss.hint" class="issue-hint">处理：{{ iss.hint }}</text>
          </view>
        </view>
        <view v-if="repairActions.length === 0 && readonlyIssues.length === 0" class="issue-list">
          <view v-for="(iss, i) in doctorIssues" :key="i" class="issue-item">
            <text class="issue-title">
              <text v-if="iss.code" class="issue-code">{{ iss.code }}</text>
              {{ iss.subject ? `${iss.subject} · ` : '' }}{{ iss.title }}
            </text>
            <text v-if="iss.detail" class="issue-detail">发现：{{ iss.detail }}</text>
            <text v-if="iss.hint" class="issue-hint">处理：{{ iss.hint }}</text>
          </view>
        </view>
      </view>
      <text class="banner-cli">也可在终端运行 inv doctor 查看完整报告</text>
    </view>

    <view v-if="pendingItems.length > 0" class="section">
      <view class="section-head">
        <text class="section-title">进行中的决策</text>
        <text class="hint-inline">{{ pendingItems.length }} 项未完成</text>
      </view>
      <view class="pending-list">
        <view
          v-for="c in pendingItems"
          :key="c.id"
          class="pending-card"
          @click="resumePending(c)"
        >
          <view class="pending-head">
            <text class="pending-name">{{ c.name || c.code }}</text>
            <text :class="['pending-status', c.status]">{{ pendingStatusLabel(c.status) }}</text>
          </view>
          <view class="pending-meta">
            <text class="tag">{{ pendingActionLabel(c.checklist_type) }}</text>
            <text>{{ c.code }}</text>
          </view>
          <text class="pending-summary">{{ c.summary || '点击继续完成决策流程' }}</text>
          <view class="pending-foot">
            <text class="pending-hint">
              {{ c.status === 'draft' ? '继续填写 ›' : '前往批准落库 ›' }}
            </text>
            <text class="pending-dismiss" @click.stop="dismissPending(c)">作废</text>
          </view>
        </view>
      </view>
    </view>

    <view class="section">
      <view class="section-head">
        <text class="section-title">当前持仓</text>
        <text class="hint-inline">点击卡片查看该标的记录</text>
      </view>
      <view v-if="loading" class="empty">加载中…</view>
      <view v-else-if="positions.length === 0" class="empty card">
        暂无持仓，点击下方「我要买入」开始
      </view>
      <view v-else class="pos-list">
        <view
          v-for="p in positions"
          :key="p.code"
          class="pos-card"
          @click="goPositionRecords(p)"
        >
          <view class="pos-head">
            <text class="pos-name">{{ p.name || p.code }}</text>
            <text class="pos-state">{{ stateLabel(p.state) }} ›</text>
          </view>
          <view class="pos-meta">
            <text class="code">{{ p.code }}</text>
            <text class="pct">{{ p.position_pct }}%</text>
            <text class="tag">{{ positionTypeLabel(p.position_type) }}</text>
          </view>
          <view class="pos-detail">
            <text>成本价 ¥{{ p.cost_basis || '—' }}</text>
            <text>持股 {{ formatShares(p.shares) }}</text>
          </view>
          <view v-if="p.entry_date" class="pos-foot">建仓日 {{ p.entry_date }}</view>
        </view>
      </view>
      <view v-if="watchCount > 0" class="hint">观察池 {{ watchCount }} 只标的</view>
    </view>

    <view class="section">
      <view class="section-head">
        <text class="section-title">生命周期</text>
        <text class="hint-inline" @click="goPool()">选股看板 ›</text>
      </view>
      <view class="lifecycle-row three">
        <view class="lifecycle-card" @click="goResearch">
          <text class="lc-icon">🔬</text>
          <text class="lc-title">研究股票</text>
          <text class="lc-desc">拉取估值/新闻并入 L1</text>
        </view>
        <view class="lifecycle-card" @click="goPool()">
          <text class="lc-icon">📋</text>
          <text class="lc-title">选股看板</text>
          <text class="lc-desc">观察 / 研究 / 持仓</text>
        </view>
        <view class="lifecycle-card primary" @click="goBuyFromPool()">
          <text class="lc-icon">📈</text>
          <text class="lc-title">我要买入</text>
          <text class="lc-desc">从观察区选股</text>
        </view>
      </view>
    </view>

    <view class="section">
      <text class="section-title">持仓操作</text>
      <view class="action-grid">
        <view
          v-for="a in actions"
          :key="a.type"
          class="action-card"
          @click="onAction(a)"
        >
          <text class="action-icon">{{ a.icon }}</text>
          <text class="action-title">{{ a.title }}</text>
          <text class="action-desc">{{ a.desc }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.page {
  padding: 32rpx;
  min-height: 100vh;
}
.header {
  margin-bottom: 32rpx;
}
.title {
  font-size: 44rpx;
  font-weight: 700;
  display: block;
}
.subtitle {
  font-size: 26rpx;
  color: #64748b;
  margin-top: 8rpx;
  display: block;
}
.banner {
  padding: 20rpx 24rpx;
  border-radius: 12rpx;
  font-size: 26rpx;
  margin-bottom: 24rpx;
}
.banner.warn {
  background: #fef3c7;
  color: #92400e;
}
.banner-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
}
.banner-toggle {
  font-size: 24rpx;
  color: #b45309;
}
.banner-hint {
  display: block;
  margin-top: 8rpx;
  font-size: 24rpx;
  line-height: 1.4;
}
.banner-cli {
  display: block;
  margin-top: 12rpx;
  font-size: 22rpx;
  color: #b45309;
}
.issue-list {
  margin-top: 16rpx;
  max-height: 480rpx;
  overflow-y: auto;
}
.issue-item {
  background: rgba(255, 255, 255, 0.6);
  border-radius: 8rpx;
  padding: 12rpx 16rpx;
  margin-bottom: 8rpx;
}
.issue-title {
  font-size: 24rpx;
  font-weight: 500;
  display: block;
}
.issue-code {
  background: #fde68a;
  padding: 2rpx 8rpx;
  border-radius: 4rpx;
  margin-right: 8rpx;
}
.issue-detail,
.issue-hint {
  font-size: 22rpx;
  display: block;
  margin-top: 6rpx;
  line-height: 1.4;
  color: #78350f;
}
.issue-hint {
  color: #92400e;
}
.readonly-block {
  margin-top: 16rpx;
  padding-top: 12rpx;
  border-top: 1px dashed #fcd34d;
}
.readonly-title {
  font-size: 24rpx;
  font-weight: 600;
  color: #92400e;
  display: block;
  margin-bottom: 8rpx;
}
.section {
  margin-bottom: 40rpx;
}
.section-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16rpx;
}
.section-title {
  font-size: 32rpx;
  font-weight: 600;
}
.hint-inline {
  font-size: 24rpx;
  color: #94a3b8;
}
.empty {
  color: #94a3b8;
  font-size: 28rpx;
  padding: 24rpx;
}
.card {
  background: #fff;
  border-radius: 16rpx;
}
.pending-list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}
.pending-card {
  background: #fffbeb;
  border: 1rpx solid #fde68a;
  border-radius: 16rpx;
  padding: 24rpx;
}
.pending-card:active { opacity: 0.88; }
.pending-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.pending-name { font-size: 30rpx; font-weight: 600; }
.pending-status { font-size: 22rpx; }
.pending-status.draft { color: #64748b; }
.pending-status.submitted { color: #d97706; font-weight: 600; }
.pending-meta {
  margin-top: 8rpx;
  font-size: 24rpx;
  color: #64748b;
  display: flex;
  gap: 12rpx;
  align-items: center;
}
.pending-summary {
  margin-top: 10rpx;
  font-size: 26rpx;
  color: #78350f;
  display: block;
  line-height: 1.4;
}
.pending-foot {
  margin-top: 8rpx;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.pending-hint {
  font-size: 24rpx;
  color: #b45309;
}
.pending-dismiss {
  font-size: 24rpx;
  color: #dc2626;
  padding: 8rpx 12rpx;
}
.pos-list {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}
.pos-card {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  box-shadow: 0 2rpx 12rpx rgba(0, 0, 0, 0.04);
}
.pos-card:active {
  opacity: 0.88;
  background: #f8fafc;
}
.pos-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.pos-name {
  font-size: 32rpx;
  font-weight: 600;
}
.pos-state {
  font-size: 24rpx;
  color: #2563eb;
}
.pos-meta {
  margin-top: 8rpx;
  font-size: 26rpx;
  color: #64748b;
  display: flex;
  gap: 16rpx;
  align-items: center;
}
.pos-detail {
  margin-top: 12rpx;
  font-size: 26rpx;
  color: #475569;
  display: flex;
  gap: 24rpx;
}
.pos-foot {
  margin-top: 8rpx;
  font-size: 22rpx;
  color: #94a3b8;
}
.pct {
  color: #2563eb;
  font-weight: 600;
}
.tag {
  background: #f1f5f9;
  padding: 4rpx 12rpx;
  border-radius: 8rpx;
  font-size: 22rpx;
}
.hint {
  margin-top: 12rpx;
  font-size: 24rpx;
  color: #94a3b8;
}
.lifecycle-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16rpx;
}
.lifecycle-row.three {
  grid-template-columns: 1fr 1fr 1fr;
}
@media (max-width: 520px) {
  .lifecycle-row.three { grid-template-columns: 1fr; }
}
.lifecycle-card {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  box-shadow: 0 2rpx 12rpx rgba(0, 0, 0, 0.04);
}
.lifecycle-card.primary {
  background: #eff6ff;
  border: 1rpx solid #bfdbfe;
}
.lifecycle-card:active { opacity: 0.88; }
.lc-icon { font-size: 36rpx; display: block; }
.lc-title { font-size: 28rpx; font-weight: 600; margin-top: 8rpx; display: block; }
.lc-desc { font-size: 22rpx; color: #64748b; margin-top: 6rpx; line-height: 1.35; display: block; }
.action-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20rpx;
  margin-top: 16rpx;
}
.action-card {
  background: #fff;
  border-radius: 16rpx;
  padding: 28rpx 24rpx;
  box-shadow: 0 2rpx 12rpx rgba(0, 0, 0, 0.04);
}
.action-card:active {
  opacity: 0.85;
}
.action-icon {
  font-size: 40rpx;
  display: block;
}
.action-title {
  font-size: 30rpx;
  font-weight: 600;
  margin-top: 12rpx;
  display: block;
}
.action-desc {
  font-size: 22rpx;
  color: #94a3b8;
  margin-top: 8rpx;
  line-height: 1.4;
  display: block;
}
</style>
