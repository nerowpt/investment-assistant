<script setup lang="ts">
import { onLoad, onShow } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { api } from '@/api'
import type { PoolItem, PoolZone } from '@/api/types'

const loading = ref(true)
const zones = ref<PoolZone[]>([])
const activeZone = ref('watching')

const currentZone = computed(() => zones.value.find((z) => z.id === activeZone.value))

async function load() {
  loading.value = true
  try {
    const res = await api.getPool()
    zones.value = res.zones || []
    if (!zones.value.some((z) => z.id === activeZone.value) && zones.value.length > 0) {
      activeZone.value = zones.value[0].id
    }
  } catch (e: unknown) {
    uni.showToast({ title: (e as Error).message || '加载失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

function switchZone(id: string) {
  activeZone.value = id
}

function startBuy(item: PoolItem) {
  const q = new URLSearchParams()
  q.set('type', 'buy')
  q.set('code', item.code)
  q.set('name', item.name || item.code)
  if (item.zone === 'watching') {
    q.set('from', 'watchlist')
    if (item.ref_id) q.set('watch_id', item.ref_id)
  } else if (item.zone === 'researching') {
    q.set('from', 'research')
  }
  uni.navigateTo({ url: `/pages/wizard/index?${q.toString()}` })
}

function startAction(item: PoolItem, action: string) {
  const q = new URLSearchParams()
  q.set('code', item.code)
  q.set('name', item.name || item.code)
  if (action === 'buy') {
    startBuy(item)
    return
  }
  if (action === 'research') {
    goResearch(item)
    return
  }
  if (action === 'add') {
    q.set('type', 'add')
    uni.navigateTo({ url: `/pages/wizard/index?${q.toString()}` })
    return
  }
  if (action === 'inspect') {
    q.set('type', 'inspect')
    uni.navigateTo({ url: `/pages/wizard/index?${q.toString()}` })
    return
  }
  if (action === 'sell') {
    q.set('type', 'sell')
    uni.navigateTo({ url: `/pages/wizard/index?${q.toString()}` })
    return
  }
  if (action === 'records') {
    q.set('code', item.code)
    q.set('name', item.name || item.code)
    uni.navigateTo({ url: `/pages/records/index?${q.toString()}` })
    return
  }
  if (action === 'review') {
    q.set('code', item.code)
    q.set('name', item.name || item.code)
    uni.navigateTo({ url: `/pages/review/index?${q.toString()}` })
  }
}

const actionLabel: Record<string, string> = {
  buy: '我要买入',
  research: '去研究',
  add: '加仓',
  inspect: '巡检',
  sell: '卖出',
  review: '复盘',
  records: '查看记录',
}

function goResearch(item: PoolItem) {
  uni.navigateTo({ url: `/pages/research/index?code=${item.code}` })
}

const emptyHint: Record<string, string> = {
  watching: '暂无观察标的。可通过 CLI watch checklist 或后续「研究页」加入观察池。',
  researching: '暂无研究区标的。为某只股票入库 L1 素材后，会自动出现在这里。',
  holding: '暂无持仓。从观察区或研究区发起「我要买入」开始建仓。',
  closed: '暂无已清仓标的。减仓（部分卖出）后仍留在「已建仓」，须全部卖完才会出现在这里。',
  swing: '暂无波段仓。建仓时选择「波段」类型会归入此区。',
}

onLoad((query) => {
  const z = String(query?.zone || '')
  if (['watching', 'researching', 'holding', 'closed', 'swing'].includes(z)) {
    activeZone.value = z
  }
})

onShow(() => load())
</script>

<template>
  <view class="page">
    <view class="header">
      <text class="title">选股看板</text>
      <text class="subtitle">研究 → 观察 → 建仓 → 卖出，按分区管理标的</text>
    </view>

    <scroll-view scroll-x class="tabs" :show-scrollbar="false">
      <view
        v-for="z in zones"
        :key="z.id"
        :class="['tab', activeZone === z.id && 'active']"
        @click="switchZone(z.id)"
      >
        <text class="tab-title">{{ z.title }}</text>
        <text class="tab-count">{{ z.count }}</text>
      </view>
    </scroll-view>

    <view v-if="currentZone" class="zone-desc">{{ currentZone.description }}</view>

    <view v-if="loading" class="empty">加载中…</view>
    <view v-else-if="!currentZone?.items?.length" class="empty card">
      {{ emptyHint[activeZone] || '本区暂无标的' }}
    </view>
    <view v-else class="list">
      <view v-for="item in currentZone.items" :key="`${item.zone}-${item.code}-${item.ref_id || ''}`" class="card">
        <view class="card-head">
          <view>
            <text class="name">{{ item.name && item.name !== item.code ? item.name : item.code }}</text>
            <text v-if="item.name && item.name !== item.code" class="code">{{ item.code }}</text>
          </view>
          <text v-if="item.position_type" class="tag">{{ item.position_type === 'swing' ? '波段' : '主仓' }}</text>
        </view>
        <text v-if="item.summary" class="summary">{{ item.summary }}</text>
        <view class="meta">
          <text v-if="item.library_count > 0">L1 素材 {{ item.library_count }}</text>
          <text v-if="item.review_date">复查 {{ item.review_date }}</text>
          <text v-if="item.position_pct">{{ item.position_pct }}%</text>
          <text v-if="item.sell_count && item.state === 'holding'" class="sell-hint">已卖出 {{ item.sell_count }} 次（仍持仓）</text>
          <text v-if="item.closed_at">清仓 {{ item.closed_at }}</text>
        </view>
        <view class="actions">
          <button
            v-for="act in item.actions"
            :key="act"
            type="button"
            :class="['act-btn', act === 'buy' && 'primary']"
            @click="startAction(item, act)"
          >
            {{ actionLabel[act] || act }}
          </button>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.page {
  min-height: 100vh;
  background: #eef2f6;
  padding: 24rpx 24rpx 48rpx;
  box-sizing: border-box;
}
.header { margin-bottom: 20rpx; }
.title { font-size: 40rpx; font-weight: 700; display: block; color: #0f172a; }
.subtitle { font-size: 24rpx; color: #64748b; margin-top: 8rpx; display: block; }
.tabs {
  white-space: nowrap;
  margin-bottom: 12rpx;
  padding-bottom: 4rpx;
}
.tab {
  display: inline-flex;
  align-items: center;
  gap: 8rpx;
  padding: 12rpx 20rpx;
  margin-right: 12rpx;
  background: #fff;
  border-radius: 999rpx;
  border: 1px solid #e2e8f0;
}
.tab.active {
  background: #2563eb;
  border-color: #2563eb;
}
.tab.active .tab-title,
.tab.active .tab-count { color: #fff; }
.tab-title { font-size: 26rpx; color: #334155; }
.tab-count {
  font-size: 22rpx;
  background: rgba(0, 0, 0, 0.06);
  padding: 2rpx 10rpx;
  border-radius: 20rpx;
  color: #64748b;
}
.tab.active .tab-count { background: rgba(255, 255, 255, 0.25); color: #fff; }
.zone-desc {
  font-size: 24rpx;
  color: #64748b;
  margin-bottom: 16rpx;
}
.empty {
  text-align: center;
  color: #94a3b8;
  font-size: 28rpx;
  padding: 40rpx 24rpx;
  line-height: 1.5;
}
.empty.card {
  background: #fff;
  border-radius: 16rpx;
}
.list { display: flex; flex-direction: column; gap: 16rpx; }
.card {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  box-shadow: 0 2rpx 12rpx rgba(15, 23, 42, 0.05);
  border-left: 6rpx solid #3b82f6;
}
.card-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}
.name { font-size: 32rpx; font-weight: 600; display: block; color: #0f172a; }
.code { font-size: 24rpx; color: #64748b; }
.tag {
  font-size: 22rpx;
  background: #eff6ff;
  color: #2563eb;
  padding: 4rpx 12rpx;
  border-radius: 8rpx;
}
.sell-hint { color: #b45309; }
.summary {
  margin-top: 12rpx;
  font-size: 26rpx;
  color: #475569;
  line-height: 1.45;
  display: block;
}
.meta {
  margin-top: 12rpx;
  display: flex;
  flex-wrap: wrap;
  gap: 16rpx;
  font-size: 22rpx;
  color: #94a3b8;
}
.actions {
  margin-top: 16rpx;
  display: flex;
  flex-wrap: wrap;
  gap: 12rpx;
}
.act-btn {
  font-size: 26rpx;
  padding: 10rpx 20rpx;
  border-radius: 10rpx;
  background: #f1f5f9;
  color: #334155;
  border: none;
  margin: 0;
  line-height: 1.4;
}
.act-btn.primary {
  background: #2563eb;
  color: #fff;
}
</style>
