<script setup lang="ts">
import { onLoad, onShow } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api } from '@/api'
import type { ChecklistListItem, JournalRow } from '@/api/types'

const tab = ref<'journal' | 'checklist'>('journal')
const journals = ref<JournalRow[]>([])
const checklists = ref<ChecklistListItem[]>([])
const loading = ref(false)
const filterCode = ref('')
const filterName = ref('')

onLoad((query) => {
  filterCode.value = String(query?.code || '')
  filterName.value = String(query?.name || '')
})

async function load() {
  loading.value = true
  try {
    const base = { limit: 50 as const }
    const code = filterCode.value || undefined
    if (tab.value === 'journal') {
      const res = await api.getJournals({ ...base, code })
      journals.value = res.journals || []
    } else {
      const res = await api.getChecklists({ ...base, code })
      checklists.value = res.checklists || []
    }
  } catch (e: unknown) {
    uni.showToast({ title: (e as Error).message, icon: 'none' })
  } finally {
    loading.value = false
  }
}

function updateTitle() {
  const title = filterCode.value
    ? `${filterName.value || filterCode.value} · 记录`
    : '决策记录'
  uni.setNavigationBarTitle({ title })
}

function switchTab(t: 'journal' | 'checklist') {
  tab.value = t
  load()
}

function actionLabel(t: string) {
  const m: Record<string, string> = {
    buy: '建仓', add: '加仓', sell: '卖出', inspect: '巡检', watch: '观察', import: '导入',
  }
  return m[t] || t
}

function statusLabel(s: string) {
  const m: Record<string, string> = {
    approved: '已批准', submitted: '待批准', draft: '草稿', rejected: '已拒绝',
  }
  return m[s] || s
}

function formatTime(iso: string) {
  if (!iso) return ''
  return iso.replace('T', ' ').slice(0, 16)
}

function openJournalDetail(j: JournalRow) {
  uni.navigateTo({ url: `/pages/records/detail?kind=journal&id=${j.id}` })
}

function openChecklistDetail(c: ChecklistListItem) {
  uni.navigateTo({ url: `/pages/records/detail?kind=checklist&id=${c.id}` })
}

function journalMeta(j: JournalRow): string {
  const parts: string[] = []
  if (j.lot_id) parts.push(`lot ${j.lot_id}`)
  if (j.checklist_submission_id) parts.push(`决策 ${j.checklist_submission_id}`)
  return parts.join(' · ')
}

function checklistMeta(c: ChecklistListItem): string {
  const parts: string[] = []
  if (c.submitted_at) parts.push(`提交 ${formatTime(c.submitted_at)}`)
  if (c.approved_at) parts.push(`批准 ${formatTime(c.approved_at)}`)
  return parts.join(' · ')
}

function goBack() {
  uni.navigateBack()
}

onShow(() => {
  updateTitle()
  load()
})
</script>

<template>
  <view class="page">
    <view v-if="filterCode" class="pos-banner">
      <text class="pos-banner-name">{{ filterName || filterCode }}</text>
      <text class="pos-banner-code">{{ filterCode }}</text>
      <text class="pos-banner-hint">以下为该标的的交易与决策记录，点击卡片查看全量详情</text>
    </view>

    <view class="tabs">
      <text :class="['tab', tab === 'journal' && 'active']" @click="switchTab('journal')">交易记录</text>
      <text :class="['tab', tab === 'checklist' && 'active']" @click="switchTab('checklist')">决策表单</text>
    </view>

    <view v-if="loading" class="empty">加载中…</view>

    <view v-else-if="tab === 'journal'">
      <view v-if="journals.length === 0" class="empty card">
        <text>该标的暂无交易记录</text>
        <text v-if="filterCode" class="empty-hint">
          若首页有该持仓但此处为空，可能是 portfolio 模板残留、账本未同步（见数据体检提示）
        </text>
      </view>
      <view v-for="j in journals" :key="j.id" class="card clickable" @click="openJournalDetail(j)">
        <view class="card-head">
          <text class="badge">{{ actionLabel(j.action_type) }}</text>
          <text class="time">{{ formatTime(j.created_at) }}</text>
        </view>
        <text class="summary">{{ j.summary || '—' }}</text>
        <text v-if="journalMeta(j)" class="meta">{{ journalMeta(j) }}</text>
        <text v-if="j.name && !filterCode" class="sub">{{ j.code }} {{ j.name }}</text>
        <text class="tap-hint">查看全量详情 ›</text>
      </view>
    </view>

    <view v-else>
      <view v-if="checklists.length === 0" class="empty card">
        <text>该标的暂无决策表单</text>
        <text v-if="filterCode" class="empty-hint">
          若首页有该持仓但此处为空，可能是 portfolio 模板残留、账本未同步（见数据体检提示）
        </text>
      </view>
      <view v-for="c in checklists" :key="c.id" class="card clickable" @click="openChecklistDetail(c)">
        <view class="card-head">
          <text class="badge">{{ actionLabel(c.checklist_type) }}</text>
          <text :class="['status', c.status]">{{ statusLabel(c.status) }}</text>
        </view>
        <text class="summary">{{ c.summary || '—' }}</text>
        <text v-if="checklistMeta(c)" class="meta">{{ checklistMeta(c) }}</text>
        <view class="card-foot">
          <text class="time">{{ formatTime(c.created_at) }}</text>
          <text v-if="c.status === 'draft'" class="resume-hint">草稿 · 详情内可继续填写</text>
          <text v-else-if="c.status === 'submitted'" class="resume-hint">待批准 · 详情内可前往批准</text>
          <text v-else class="tap-hint-inline">查看全量详情 ›</text>
        </view>
      </view>
    </view>

    <button type="button" class="btn" @click="goBack">返回</button>
  </view>
</template>

<style scoped>
.page { padding: 24rpx 32rpx; min-height: 100vh; }
.pos-banner {
  background: #fff; border-radius: 16rpx; padding: 24rpx; margin-bottom: 24rpx;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.04);
}
.pos-banner-name { font-size: 34rpx; font-weight: 600; display: block; }
.pos-banner-code { font-size: 26rpx; color: #64748b; margin-top: 4rpx; display: block; }
.pos-banner-hint { font-size: 24rpx; color: #94a3b8; margin-top: 8rpx; display: block; }
.tabs { display: flex; gap: 24rpx; margin-bottom: 24rpx; }
.tab { font-size: 30rpx; color: #94a3b8; padding-bottom: 8rpx; }
.tab.active { color: #2563eb; font-weight: 600; border-bottom: 4rpx solid #2563eb; }
.empty { text-align: center; color: #94a3b8; padding: 48rpx; font-size: 28rpx; }
.empty-hint {
  display: block; margin-top: 16rpx; font-size: 24rpx; line-height: 1.5; color: #b45309;
}
.card {
  background: #fff; border-radius: 16rpx; padding: 24rpx; margin-bottom: 16rpx;
  box-shadow: 0 2rpx 8rpx rgba(0, 0, 0, 0.04);
}
.card.clickable:active { opacity: 0.88; background: #f8fafc; }
.card-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12rpx; }
.badge {
  background: #eff6ff; color: #2563eb; font-size: 22rpx; padding: 4rpx 12rpx; border-radius: 8rpx;
}
.summary { font-size: 28rpx; color: #334155; display: block; line-height: 1.5; font-weight: 500; }
.meta { font-size: 24rpx; color: #64748b; margin-top: 8rpx; display: block; }
.sub { font-size: 24rpx; color: #94a3b8; margin-top: 8rpx; display: block; }
.card-foot {
  display: flex; justify-content: space-between; align-items: center; margin-top: 12rpx; flex-wrap: wrap; gap: 8rpx;
}
.time { font-size: 22rpx; color: #cbd5e1; }
.resume-hint { font-size: 22rpx; color: #2563eb; }
.tap-hint { font-size: 22rpx; color: #2563eb; margin-top: 12rpx; display: block; }
.tap-hint-inline { font-size: 22rpx; color: #2563eb; }
.status { font-size: 22rpx; }
.status.approved { color: #16a34a; }
.status.submitted { color: #d97706; }
.status.draft { color: #64748b; }
.status.rejected { color: #dc2626; }
.btn { margin-top: 32rpx; background: #f1f5f9; color: #334155; border-radius: 16rpx; }
</style>
