<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api } from '@/api'
import type { ClosedLotSummary, ReviewWorkbenchResponse } from '@/api/types'

const loading = ref(false)
const workbench = ref<ReviewWorkbenchResponse | null>(null)

const actionTypeLabel: Record<string, string> = {
  buy: '建仓',
  add: '加仓',
  import: '导入',
}

const positionTypeLabel: Record<string, string> = {
  core: '主仓',
  swing: '波段',
}

async function load(code: string) {
  const c = code.trim()
  if (!c) return
  loading.value = true
  try {
    workbench.value = await api.getReviewWorkbench(c)
    const titleName =
      workbench.value.name && workbench.value.name !== c ? workbench.value.name : c
    uni.setNavigationBarTitle({ title: `${titleName} · 复盘` })
  } catch (e: unknown) {
    workbench.value = null
    uni.showToast({ title: (e as Error).message || '加载失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

function formatPct(v?: string) {
  if (!v) return '—'
  const n = parseFloat(v)
  if (Number.isNaN(n)) return v
  return `${n >= 0 ? '+' : ''}${n.toFixed(2)}%`
}

function formatPnl(v?: string) {
  if (!v) return '—'
  const n = parseFloat(v)
  if (Number.isNaN(n)) return v
  return `${n >= 0 ? '+' : ''}${n.toFixed(0)} 元`
}

function startReview(lot: ClosedLotSummary) {
  if (!workbench.value) return
  const q = new URLSearchParams()
  q.set('type', 'review')
  q.set('code', workbench.value.code)
  q.set('name', workbench.value.name || workbench.value.code)
  q.set('lot_id', lot.lot_id)
  uni.navigateTo({ url: `/pages/wizard/index?${q.toString()}` })
}

function goRecords() {
  if (!workbench.value) return
  const q = new URLSearchParams()
  q.set('code', workbench.value.code)
  q.set('name', workbench.value.name || workbench.value.code)
  uni.navigateTo({ url: `/pages/records/index?${q.toString()}` })
}

onLoad((query) => {
  const code = String(query?.code || '')
  if (code) void load(code)
})
</script>

<template>
  <view class="page">
    <view v-if="loading" class="loading">加载中…</view>

    <view v-else-if="workbench" class="content">
      <view class="header">
        <text class="title">
          {{ workbench.name && workbench.name !== workbench.code ? workbench.name : workbench.code }}
          <text class="code">{{ workbench.code }}</text>
        </text>
        <text class="subtitle">对已关闭 lot 做单笔归因复盘，沉淀可复用经验</text>
      </view>

      <view v-if="workbench.closed_lots.length === 0" class="empty">
        <text class="empty-title">暂无已关闭 lot</text>
        <text class="empty-hint">卖出 approve 并完全关闭 lot 后，会出现在这里。</text>
        <button class="btn-secondary" @click="goRecords">查看决策记录</button>
      </view>

      <view v-for="lot in workbench.closed_lots" :key="lot.lot_id" class="card">
        <view class="card-head">
          <text class="lot-id">{{ lot.lot_id }}</text>
          <text v-if="lot.reviewed" class="badge reviewed">已复盘</text>
          <text v-else class="badge pending">待复盘</text>
        </view>

        <view class="meta-row">
          <text>{{ actionTypeLabel[lot.action_type] || lot.action_type }}</text>
          <text>·</text>
          <text>{{ positionTypeLabel[lot.position_type] || lot.position_type }}</text>
          <text>·</text>
          <text>持有 {{ lot.holding_days }} 天</text>
        </view>

        <view class="stats">
          <view class="stat">
            <text class="stat-label">区间</text>
            <text class="stat-value">{{ lot.open_at?.slice(0, 10) }} → {{ lot.close_at?.slice(0, 10) }}</text>
          </view>
          <view class="stat">
            <text class="stat-label">实现收益</text>
            <text class="stat-value" :class="{ gain: parseFloat(lot.realized_return_pct || '0') >= 0, loss: parseFloat(lot.realized_return_pct || '0') < 0 }">
              {{ formatPct(lot.realized_return_pct) }}
            </text>
          </view>
          <view class="stat">
            <text class="stat-label">盈亏金额</text>
            <text class="stat-value">{{ formatPnl(lot.realized_pnl_amount) }}</text>
          </view>
        </view>

        <view class="actions">
          <button
            class="btn-primary"
            @click="startReview(lot)"
          >
            {{ lot.reviewed ? '再次复盘' : '写复盘' }}
          </button>
          <button v-if="lot.reviewed && lot.review_report_id" class="btn-link">
            {{ lot.review_report_id }}
          </button>
        </view>
      </view>

      <view class="footer-actions">
        <button class="btn-secondary" @click="goRecords">查看全部记录</button>
      </view>
    </view>

    <view v-else class="empty">
      <text class="empty-title">请从选股看板进入</text>
      <text class="empty-hint">已卖出区点击「复盘」可直达本页。</text>
    </view>
  </view>
</template>

<style scoped>
.page { min-height: 100vh; background: #f5f6f8; padding: 24rpx; box-sizing: border-box; }
.loading { text-align: center; color: #64748b; padding: 80rpx 0; }
.header { margin-bottom: 24rpx; }
.title { font-size: 36rpx; font-weight: 600; color: #0f172a; display: block; }
.code { font-size: 28rpx; color: #64748b; margin-left: 12rpx; font-weight: 400; }
.subtitle { font-size: 24rpx; color: #64748b; margin-top: 8rpx; display: block; }
.card {
  background: #fff; border-radius: 16rpx; padding: 28rpx; margin-bottom: 20rpx;
  box-shadow: 0 2rpx 8rpx rgba(15, 23, 42, 0.06);
}
.card-head { display: flex; align-items: center; gap: 12rpx; margin-bottom: 12rpx; }
.lot-id { font-size: 26rpx; font-weight: 600; color: #334155; flex: 1; word-break: break-all; }
.badge { font-size: 22rpx; padding: 4rpx 12rpx; border-radius: 8rpx; }
.badge.reviewed { background: #dcfce7; color: #166534; }
.badge.pending { background: #fef3c7; color: #92400e; }
.meta-row { font-size: 24rpx; color: #64748b; margin-bottom: 16rpx; display: flex; gap: 8rpx; flex-wrap: wrap; }
.stats { display: flex; flex-direction: column; gap: 12rpx; margin-bottom: 20rpx; }
.stat { display: flex; justify-content: space-between; align-items: center; }
.stat-label { font-size: 24rpx; color: #94a3b8; }
.stat-value { font-size: 26rpx; color: #1e293b; }
.stat-value.gain { color: #15803d; }
.stat-value.loss { color: #b91c1c; }
.actions { display: flex; align-items: center; gap: 16rpx; }
.btn-primary {
  flex: 1; background: #2563eb; color: #fff; font-size: 28rpx;
  border-radius: 12rpx; border: none; padding: 16rpx 0;
}
.btn-secondary {
  background: #fff; color: #334155; font-size: 28rpx;
  border-radius: 12rpx; border: 1rpx solid #e2e8f0; padding: 16rpx 32rpx;
}
.btn-link { font-size: 22rpx; color: #64748b; background: none; border: none; padding: 0; }
.footer-actions { margin-top: 12rpx; text-align: center; }
.empty { text-align: center; padding: 80rpx 32rpx; }
.empty-title { font-size: 30rpx; color: #334155; display: block; margin-bottom: 12rpx; }
.empty-hint { font-size: 24rpx; color: #94a3b8; display: block; margin-bottom: 32rpx; }
</style>
