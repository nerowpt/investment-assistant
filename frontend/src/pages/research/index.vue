<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api } from '@/api'
import type { LibraryItem } from '@/api'
import type { ResearchDossier, ResearchPackResult } from '@/api/types'
import LibraryDetailModal from '@/components/LibraryDetailModal.vue'

const codeInput = ref('')
const activeCode = ref('')
const loading = ref(false)
const dossier = ref<ResearchDossier | null>(null)
const packLoadingMap = ref<Record<string, boolean>>({})
const packResults = ref<Record<string, ResearchPackResult>>({})
const saveLoading = ref<Record<string, boolean>>({})
const libraryModalVisible = ref(false)
const libraryModalLoading = ref(false)
const libraryModalItem = ref<LibraryItem | null>(null)

function previewSummary(text?: string, max = 56) {
  if (!text) return ''
  const t = text.replace(/\s+/g, ' ').trim()
  return t.length > max ? `${t.slice(0, max)}…` : t
}

async function showLibrary(id: string) {
  libraryModalItem.value = null
  libraryModalLoading.value = true
  libraryModalVisible.value = true
  try {
    const res = await api.getLibraryItem(id)
    libraryModalItem.value = res.item
  } catch (e: unknown) {
    libraryModalVisible.value = false
    uni.showToast({ title: (e as Error).message || '加载素材失败', icon: 'none' })
  } finally {
    libraryModalLoading.value = false
  }
}

function closeLibraryModal() {
  libraryModalVisible.value = false
  libraryModalItem.value = null
}

const zoneLabel: Record<string, string> = {
  watching: '观察区',
  researching: '研究区',
  holding: '已建仓',
  closed: '已卖出',
  swing: '波段',
}

async function loadDossier(code: string) {
  const c = code.trim()
  if (!c || c.length !== 6 || !/^\d+$/.test(c)) {
    uni.showToast({ title: '请输入 6 位股票代码', icon: 'none' })
    return
  }
  loading.value = true
  activeCode.value = c
  packResults.value = {}
  packLoadingMap.value = {}
  try {
    dossier.value = await api.getResearch(c)
    const titleName = dossier.value.name && dossier.value.name !== c ? dossier.value.name : c
    uni.setNavigationBarTitle({ title: `${titleName} · 研究` })
  } catch (e: unknown) {
    dossier.value = null
    uni.showToast({ title: (e as Error).message || '加载失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

function onSearch() {
  void loadDossier(codeInput.value)
}

async function fetchPack(packId: string) {
  if (!activeCode.value || packLoadingMap.value[packId]) return
  packLoadingMap.value = { ...packLoadingMap.value, [packId]: true }
  try {
    const res = await api.fetchResearchPack(activeCode.value, packId)
    packResults.value = { ...packResults.value, [packId]: res }
  } catch (e: unknown) {
    uni.showToast({ title: (e as Error).message || '拉取失败', icon: 'none' })
  } finally {
    const next = { ...packLoadingMap.value }
    delete next[packId]
    packLoadingMap.value = next
  }
}

async function saveToLibrary(packId: string) {
  const res = packResults.value[packId]
  if (!res || !activeCode.value) return
  saveLoading.value = { ...saveLoading.value, [packId]: true }
  try {
    const out = await api.saveResearchToLibrary(activeCode.value, {
      title: res.title,
      text: res.body,
      tier: res.suggest_tier || res.tier || 'B',
    })
    uni.showToast({ title: `已入库 ${out.library_id}`, icon: 'success' })
    await loadDossier(activeCode.value)
  } catch (e: unknown) {
    uni.showToast({ title: (e as Error).message || '入库失败', icon: 'none' })
  } finally {
    saveLoading.value = { ...saveLoading.value, [packId]: false }
  }
}

function startBuy() {
  if (!activeCode.value || !dossier.value) return
  const q = new URLSearchParams()
  q.set('type', 'buy')
  q.set('code', activeCode.value)
  q.set('name', dossier.value.name || activeCode.value)
  q.set('from', 'research')
  uni.navigateTo({ url: `/pages/wizard/index?${q.toString()}` })
}

function goPool() {
  uni.navigateTo({ url: '/pages/pool/index?zone=researching' })
}

onLoad((query) => {
  const c = String(query?.code || '').trim()
  if (c) {
    codeInput.value = c
    void loadDossier(c)
  }
})
</script>

<template>
  <view class="page">
    <view class="search-card">
      <text class="search-label">研究一只股票</text>
      <view class="search-row">
        <input
          v-model="codeInput"
          class="code-input"
          type="text"
          maxlength="6"
          placeholder="6 位代码，如 000858"
        />
        <button type="button" class="search-btn" :disabled="loading" @click="onSearch">
          {{ loading ? '…' : '开始' }}
        </button>
      </view>
      <text class="search-hint">拉取外部事实数据；入库后可在买入时关联 L1 素材</text>
    </view>

    <view v-if="loading" class="empty">加载研究档案…</view>

    <template v-else-if="dossier">
      <view class="head-card">
        <view class="head-top">
          <view>
            <text class="name">{{ dossier.name && dossier.name !== dossier.code ? dossier.name : dossier.code }}</text>
            <text v-if="dossier.name && dossier.name !== dossier.code" class="code">{{ dossier.code }}</text>
          </view>
          <button type="button" class="buy-btn" @click="startBuy">我要买入</button>
        </view>
        <view v-if="dossier.zones?.length" class="zones">
          <text v-for="z in dossier.zones" :key="z" class="zone-tag">{{ zoneLabel[z] || z }}</text>
        </view>
        <view v-if="!dossier.worker_ok" class="worker-warn">
          {{ dossier.worker_message || 'data-worker 不可用' }}
        </view>
      </view>

      <view v-if="dossier.library_items?.length" class="section">
        <text class="section-title">已有 L1 素材（{{ dossier.library_items.length }}）</text>
        <view v-for="lib in dossier.library_items" :key="lib.id" class="lib-row">
          <view class="lib-row-main">
            <text class="lib-title">{{ lib.title }}</text>
            <text class="lib-meta">{{ lib.tier }} · {{ lib.id }}</text>
            <text v-if="lib.summary" class="lib-preview">{{ previewSummary(lib.summary) }}</text>
          </view>
          <button type="button" class="lib-view-btn" @click="showLibrary(lib.id)">查看</button>
        </view>
      </view>

      <view class="section">
        <text class="section-title">按需拉取数据包</text>
        <text class="section-hint">仅呈现事实，不下买卖结论；确认后可纳入 L1</text>
        <view v-for="pack in dossier.packs" :key="pack.id" class="pack-card">
          <view class="pack-head">
            <text class="pack-title">{{ pack.title }}</text>
            <button
              type="button"
              class="fetch-btn"
              :disabled="!dossier.worker_ok || !!packLoadingMap[pack.id]"
              @click="fetchPack(pack.id)"
            >
              {{ packLoadingMap[pack.id] ? '拉取中…' : '拉取' }}
            </button>
          </view>
          <text class="pack-desc">{{ pack.description }}</text>
          <view v-if="packResults[pack.id]" class="pack-result">
            <text class="result-summary">{{ packResults[pack.id].summary }}</text>
            <text class="result-body">{{ packResults[pack.id].body }}</text>
            <view class="result-actions">
              <text class="result-meta">
                {{ packResults[pack.id].source }} · tier {{ packResults[pack.id].suggest_tier }}
              </text>
              <button
                type="button"
                class="save-btn"
                :disabled="saveLoading[pack.id]"
                @click="saveToLibrary(pack.id)"
              >
                {{ saveLoading[pack.id] ? '入库中…' : '纳入 L1' }}
              </button>
            </view>
          </view>
        </view>
      </view>

      <button type="button" class="link-btn" @click="goPool">在选股看板查看 ›</button>
    </template>

    <LibraryDetailModal
      :visible="libraryModalVisible"
      :loading="libraryModalLoading"
      :item="libraryModalItem"
      :stock-code="activeCode"
      @close="closeLibraryModal"
    />
  </view>
</template>

<style scoped>
.page {
  min-height: 100vh;
  background: #eef2f6;
  padding: 24rpx 24rpx 48rpx;
  box-sizing: border-box;
}
.search-card {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  margin-bottom: 16rpx;
}
.search-label { font-size: 30rpx; font-weight: 600; display: block; margin-bottom: 12rpx; }
.search-row { display: flex; gap: 12rpx; }
.code-input {
  flex: 1;
  border: 1px solid #e2e8f0;
  border-radius: 10rpx;
  padding: 14rpx 16rpx;
  font-size: 30rpx;
}
.search-btn {
  background: #2563eb;
  color: #fff;
  border: none;
  border-radius: 10rpx;
  padding: 0 28rpx;
  font-size: 28rpx;
}
.search-hint { font-size: 22rpx; color: #94a3b8; margin-top: 10rpx; display: block; }
.empty { text-align: center; color: #94a3b8; padding: 48rpx; }
.head-card {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  margin-bottom: 16rpx;
  border-left: 6rpx solid #8b5cf6;
}
.head-top { display: flex; justify-content: space-between; align-items: flex-start; }
.name { font-size: 34rpx; font-weight: 600; display: block; }
.code { font-size: 24rpx; color: #64748b; }
.buy-btn {
  background: #2563eb;
  color: #fff;
  font-size: 26rpx;
  padding: 10rpx 20rpx;
  border-radius: 10rpx;
  border: none;
}
.zones { margin-top: 12rpx; display: flex; flex-wrap: wrap; gap: 8rpx; }
.zone-tag {
  font-size: 22rpx;
  background: #f3e8ff;
  color: #6d28d9;
  padding: 4rpx 12rpx;
  border-radius: 8rpx;
}
.worker-warn {
  margin-top: 12rpx;
  font-size: 24rpx;
  color: #b45309;
  background: #fffbeb;
  padding: 12rpx;
  border-radius: 8rpx;
}
.section { margin-bottom: 16rpx; }
.section-title { font-size: 28rpx; font-weight: 600; display: block; margin-bottom: 6rpx; }
.section-hint { font-size: 22rpx; color: #94a3b8; display: block; margin-bottom: 12rpx; }
.lib-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
  background: #fff;
  border-radius: 10rpx;
  padding: 14rpx 16rpx;
  margin-bottom: 8rpx;
}
.lib-row-main { flex: 1; min-width: 0; }
.lib-title { font-size: 26rpx; display: block; }
.lib-meta { font-size: 22rpx; color: #94a3b8; }
.lib-preview { font-size: 22rpx; color: #64748b; margin-top: 6rpx; display: block; line-height: 1.4; }
.lib-view-btn {
  flex-shrink: 0;
  font-size: 24rpx;
  color: #2563eb;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 8rpx;
  padding: 8rpx 18rpx;
  line-height: 1.4;
}
.pack-card {
  background: #fff;
  border-radius: 14rpx;
  padding: 20rpx;
  margin-bottom: 12rpx;
}
.pack-head { display: flex; justify-content: space-between; align-items: center; }
.pack-title { font-size: 28rpx; font-weight: 600; }
.fetch-btn {
  font-size: 24rpx;
  background: #f1f5f9;
  color: #334155;
  border: none;
  padding: 8rpx 18rpx;
  border-radius: 8rpx;
}
.fetch-btn:disabled { opacity: 0.5; }
.pack-desc { font-size: 22rpx; color: #64748b; margin-top: 8rpx; display: block; }
.pack-result {
  margin-top: 14rpx;
  padding-top: 14rpx;
  border-top: 1px dashed #e2e8f0;
}
.result-summary { font-size: 26rpx; color: #1e40af; font-weight: 500; display: block; }
.result-body {
  font-size: 24rpx;
  color: #475569;
  white-space: pre-wrap;
  margin-top: 10rpx;
  display: block;
  line-height: 1.5;
}
.result-actions {
  margin-top: 12rpx;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.result-meta { font-size: 22rpx; color: #94a3b8; }
.save-btn {
  font-size: 24rpx;
  background: #8b5cf6;
  color: #fff;
  border: none;
  padding: 8rpx 18rpx;
  border-radius: 8rpx;
}
.link-btn {
  width: 100%;
  background: #fff;
  color: #64748b;
  border: 1px solid #e2e8f0;
  border-radius: 12rpx;
  font-size: 28rpx;
  margin-top: 8rpx;
}
</style>
