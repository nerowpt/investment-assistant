<script setup lang="ts">

import { onLoad } from '@dcloudio/uni-app'

import { computed, ref } from 'vue'

import { api, type LibraryItem } from '@/api'

import LibraryDetailModal from '@/components/LibraryDetailModal.vue'
import PayloadDetail from '@/components/PayloadDetail.vue'

import type { ChecklistListItem, FormSchema, JournalRow } from '@/api/types'

import { buildDetailGroups, formatRiskMessages } from '@/utils/formatField'



const kind = ref<'journal' | 'checklist'>('journal')

const recordId = ref('')

const loading = ref(true)

const error = ref('')



const journal = ref<JournalRow | null>(null)

const checklist = ref<ChecklistListItem | null>(null)

const payload = ref<Record<string, unknown>>({})

const riskResult = ref<Record<string, unknown> | null>(null)

const exception = ref<Record<string, unknown> | null>(null)

const emotionSelfCheck = ref('')

const schema = ref<FormSchema | null>(null)

const libraryTitles = ref<Record<string, string>>({})

const selectedLibrary = ref<LibraryItem | null>(null)

const libraryLoading = ref(false)

const libraryModalVisible = ref(false)



const detailGroups = computed(() => {

  if (!schema.value) return []

  return buildDetailGroups(schema.value, payload.value)

})



const riskMessages = computed(() => formatRiskMessages(riskResult.value))



const checklistType = computed(() => {

  if (kind.value === 'checklist') return checklist.value?.checklist_type || 'buy'

  return journal.value?.action_type || 'buy'

})



const stockCode = computed(() => journal.value?.code || checklist.value?.code || '')



const actionLabel = (t: string) => {

  const m: Record<string, string> = {

    buy: '建仓', add: '加仓', sell: '卖出', inspect: '巡检', watch: '观察', import: '导入',

  }

  return m[t] || t

}



const statusLabel = (s: string) => {

  const m: Record<string, string> = {

    approved: '已批准', submitted: '待批准', draft: '草稿', rejected: '已拒绝',

  }

  return m[s] || s

}



function formatTime(iso: string) {

  if (!iso) return ''

  return iso.replace('T', ' ').slice(0, 16)

}



async function preloadLibraries(code: string) {

  try {

    const res = await api.getLibrary(code)

    const titles: Record<string, string> = {}

    for (const item of res.items) titles[item.id] = item.title || item.id

    libraryTitles.value = titles

  } catch {

    libraryTitles.value = {}

  }

}



async function showLibrary(id: string) {

  selectedLibrary.value = null

  libraryLoading.value = true

  libraryModalVisible.value = true

  try {

    const res = await api.getLibraryItem(id)

    selectedLibrary.value = res.item

    if (res.item.title) {

      libraryTitles.value = { ...libraryTitles.value, [id]: res.item.title }

    }

  } catch (e: unknown) {

    libraryModalVisible.value = false

    uni.showToast({ title: (e as Error).message || '加载素材失败', icon: 'none' })

  } finally {

    libraryLoading.value = false

  }

}



function closeLibrary() {

  libraryModalVisible.value = false

  selectedLibrary.value = null

  libraryLoading.value = false

}



async function load() {

  loading.value = true

  error.value = ''

  libraryModalVisible.value = false

  selectedLibrary.value = null

  try {

    if (kind.value === 'journal') {

      const res = await api.getJournal(recordId.value)

      journal.value = res.journal

      payload.value = (res.payload as Record<string, unknown>) || {}

    } else {

      const res = await api.getChecklist(recordId.value)

      checklist.value = res.checklist

      payload.value = res.payload || {}

      riskResult.value = res.risk_result || null

      exception.value = (res as { exception?: Record<string, unknown> }).exception || null

      emotionSelfCheck.value = (res as { emotion_self_check?: string }).emotion_self_check || ''

    }

    const schemaRes = await api.getSchema(checklistType.value)

    schema.value = schemaRes.schema

    const code = journal.value?.code || checklist.value?.code

    if (code) await preloadLibraries(code)

    const title =

      kind.value === 'journal'

        ? `${journal.value?.name || journal.value?.code || ''} · 交易详情`

        : `${checklist.value?.name || checklist.value?.code || ''} · 决策详情`

    uni.setNavigationBarTitle({ title })

  } catch (e: unknown) {

    error.value = (e as Error).message

  } finally {

    loading.value = false

  }

}



function openLinkedChecklist() {

  if (!journal.value?.checklist_submission_id) return

  uni.navigateTo({

    url: `/pages/records/detail?kind=checklist&id=${journal.value.checklist_submission_id}`,

  })

}



function resumeWizard() {

  if (!checklist.value) return

  if (checklist.value.status !== 'draft' && checklist.value.status !== 'submitted') return

  uni.navigateTo({ url: `/pages/wizard/index?resume_id=${checklist.value.id}` })

}



function goBack() {

  uni.navigateBack()

}



onLoad((query) => {

  kind.value = String(query?.kind || 'journal') === 'checklist' ? 'checklist' : 'journal'

  recordId.value = String(query?.id || '')

  if (!recordId.value) {

    error.value = '缺少记录 id'

    loading.value = false

    return

  }

  void load()

})

</script>



<template>

  <view class="page">

    <view class="page-inner">

      <view v-if="loading" class="empty">加载中…</view>

      <view v-else-if="error" class="empty error">{{ error }}</view>



      <template v-else>

        <view class="head-card">

          <view class="head-top">

            <text class="badge">{{ actionLabel(checklistType) }}</text>

            <text v-if="checklist" :class="['status', checklist.status]">{{ statusLabel(checklist.status) }}</text>

          </view>

          <text class="head-title">

            {{ (journal || checklist)?.name || (journal || checklist)?.code }}

            <text v-if="(journal || checklist)?.code" class="code">({{ (journal || checklist)?.code }})</text>

          </text>

          <text class="head-time">{{ formatTime((journal || checklist)?.created_at || '') }}</text>

          <view v-if="journal?.checklist_submission_id" class="link-row" @click="openLinkedChecklist">

            <text class="link-label">关联决策表单</text>

            <text class="link-id">{{ journal.checklist_submission_id }} ›</text>

          </view>

          <view v-if="journal?.lot_id" class="meta-row">

            <text class="meta-label">关联 lot</text>

            <text class="meta-val">{{ journal.lot_id }}</text>

          </view>

          <view v-if="checklist?.submitted_at" class="meta-row">

            <text class="meta-label">提交时间</text>

            <text class="meta-val">{{ formatTime(checklist.submitted_at) }}</text>

          </view>

          <view v-if="checklist?.approved_at" class="meta-row">

            <text class="meta-label">批准时间</text>

            <text class="meta-val">{{ formatTime(checklist.approved_at) }}</text>

          </view>

        </view>



        <view v-if="riskMessages.length" class="risk-box">

          <text class="section-title">风险检查结果</text>

          <text v-for="(m, i) in riskMessages" :key="i" class="risk-line">{{ m }}</text>

        </view>



        <view v-if="emotionSelfCheck" class="extra-box">

          <text class="section-title">情绪自检</text>

          <text class="extra-text">{{ emotionSelfCheck }}</text>

        </view>



        <view v-if="exception" class="extra-box">

          <text class="section-title">风险例外说明</text>

          <text class="extra-text">{{ (exception as Record<string, string>).exception_reason || JSON.stringify(exception) }}</text>

        </view>



        <text class="section-title main">决策表单内容</text>

        <PayloadDetail

          :groups="detailGroups"

          :library-titles="libraryTitles"

          @library-click="showLibrary"

        />



        <view v-if="checklist && (checklist.status === 'draft' || checklist.status === 'submitted')" class="action-row">

          <button type="button" class="btn primary" @click="resumeWizard">

            {{ checklist.status === 'draft' ? '继续填写' : '前往批准' }}

          </button>

        </view>



        <button type="button" class="btn back" @click="goBack">返回列表</button>

      </template>

    </view>

    <LibraryDetailModal

      :visible="libraryModalVisible"

      :loading="libraryLoading"

      :item="selectedLibrary"

      :stock-code="stockCode"

      @close="closeLibrary"

    />

  </view>

</template>



<style scoped>

.page {

  min-height: 100vh;

  background: #eef2f6;

  padding: 24rpx 0 48rpx;

}

.page-inner {

  max-width: 720px;

  margin: 0 auto;

  padding: 0 24rpx;

  box-sizing: border-box;

}

.empty { text-align: center; color: #94a3b8; padding: 48rpx; }

.empty.error { color: #b91c1c; }

.head-card {

  background: #fff;

  border-radius: 16rpx;

  padding: 24rpx;

  margin-bottom: 20rpx;

  box-shadow: 0 2rpx 12rpx rgba(15, 23, 42, 0.06);

  border-left: 6rpx solid #2563eb;

}

.head-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12rpx; }

.badge { background: #eff6ff; color: #2563eb; font-size: 22rpx; padding: 4rpx 12rpx; border-radius: 8rpx; }

.head-title { font-size: 34rpx; font-weight: 600; display: block; color: #0f172a; }

.code { font-size: 28rpx; color: #64748b; font-weight: 400; }

.head-time { font-size: 24rpx; color: #94a3b8; margin-top: 8rpx; display: block; }

.link-row {

  margin-top: 16rpx;

  padding-top: 16rpx;

  border-top: 1px dashed #e2e8f0;

  display: flex;

  justify-content: space-between;

  cursor: pointer;

}

.link-label { font-size: 26rpx; color: #64748b; }

.link-id { font-size: 26rpx; color: #2563eb; }

.meta-row { display: flex; justify-content: space-between; margin-top: 8rpx; }

.meta-label { font-size: 24rpx; color: #94a3b8; }

.meta-val { font-size: 24rpx; color: #475569; }

.section-title { font-size: 28rpx; font-weight: 600; color: #1e293b; display: block; margin-bottom: 12rpx; }

.section-title.main { margin-top: 4rpx; margin-bottom: 16rpx; }

.risk-box {

  background: #fffbeb;

  border: 1rpx solid #fde68a;

  border-radius: 12rpx;

  padding: 20rpx;

  margin-bottom: 16rpx;

}

.risk-line { display: block; font-size: 26rpx; color: #92400e; line-height: 1.5; margin-bottom: 8rpx; }

.extra-box {

  background: #fff;

  border-radius: 12rpx;

  padding: 20rpx;

  margin-bottom: 16rpx;

  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);

}

.extra-text { font-size: 26rpx; color: #334155; line-height: 1.6; white-space: pre-wrap; display: block; }

.status { font-size: 22rpx; }

.status.approved { color: #16a34a; }

.status.submitted { color: #d97706; }

.status.draft { color: #64748b; }

.status.rejected { color: #dc2626; }

.action-row { margin-top: 24rpx; }

.btn { margin-top: 24rpx; border-radius: 16rpx; font-size: 30rpx; width: 100%; }

.btn.primary { background: #2563eb; color: #fff; }

.btn.back { background: #fff; color: #334155; border: 1px solid #e2e8f0; }

</style>


