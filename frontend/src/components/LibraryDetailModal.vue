<script setup lang="ts">
import type { LibraryItem } from '@/api'

defineProps<{
  visible: boolean
  loading: boolean
  item: LibraryItem | null
  stockCode?: string
  /** 决策向导：弹窗内可切换是否关联 */
  selectable?: boolean
  selected?: boolean
}>()

const emit = defineEmits<{ close: []; toggleSelect: [] }>()

const tierLabel = (tier: string) => {
  const m: Record<string, string> = { S: 'S 级', A: 'A 级', B: 'B 级', C: 'C 级', D: 'D 级' }
  return m[tier] || tier
}

function formatTime(iso: string) {
  if (!iso) return ''
  return iso.replace('T', ' ').slice(0, 16)
}
</script>

<template>
  <view v-if="visible" class="modal-mask" @click="emit('close')">
    <view class="modal-card" @click.stop>
      <view class="modal-head">
        <text class="modal-title">L1 素材详情</text>
        <text class="modal-close" @click="emit('close')">✕</text>
      </view>

      <view v-if="loading" class="modal-loading">加载中…</view>

      <template v-else-if="item">
        <text class="lib-name">{{ item.title }}</text>
        <view class="lib-meta">
          <text class="lib-tag tier">{{ tierLabel(item.tier) }}</text>
          <text v-if="item.source" class="lib-tag">来源：{{ item.source }}</text>
          <text v-if="stockCode" class="lib-tag">标的：{{ stockCode }}</text>
        </view>
        <text class="lib-id-line">{{ item.id }}</text>
        <scroll-view scroll-y class="lib-body">
          <text v-if="item.summary" class="lib-summary">{{ item.summary }}</text>
          <text v-else class="lib-summary muted">暂无摘要</text>
        </scroll-view>
        <text v-if="item.created_at" class="lib-time">收录于 {{ formatTime(item.created_at) }}</text>
        <view v-if="selectable && item" class="modal-foot">
          <button type="button" class="btn-select" @click="emit('toggleSelect')">
            {{ selected ? '取消关联' : '关联此素材' }}
          </button>
        </view>
      </template>
    </view>
  </view>
</template>

<style scoped>
.modal-mask {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: rgba(15, 23, 42, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 16px;
  box-sizing: border-box;
}
.modal-card {
  width: 100%;
  max-width: 480px;
  max-height: 80vh;
  background: #fff;
  border-radius: 16px;
  padding: 20px 22px 18px;
  box-shadow: 0 20px 50px rgba(15, 23, 42, 0.2);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.modal-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
  flex-shrink: 0;
}
.modal-title { font-size: 16px; font-weight: 600; color: #1e40af; }
.modal-close {
  font-size: 18px;
  color: #94a3b8;
  padding: 4px 8px;
  cursor: pointer;
  line-height: 1;
}
.modal-close:hover { color: #64748b; }
.modal-loading {
  text-align: center;
  color: #64748b;
  font-size: 15px;
  padding: 32px 0;
}
.lib-name {
  font-size: 17px;
  font-weight: 600;
  color: #0f172a;
  display: block;
  margin-bottom: 10px;
  flex-shrink: 0;
}
.lib-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.lib-tag {
  font-size: 12px;
  color: #475569;
  background: #f1f5f9;
  padding: 3px 10px;
  border-radius: 6px;
}
.lib-tag.tier { background: #dbeafe; color: #1d4ed8; }
.lib-id-line {
  font-size: 12px;
  color: #94a3b8;
  display: block;
  margin-bottom: 12px;
  flex-shrink: 0;
}
.lib-body {
  flex: 1;
  min-height: 0;
  max-height: 40vh;
  margin-bottom: 10px;
}
.lib-summary {
  font-size: 15px;
  color: #334155;
  line-height: 1.65;
  white-space: pre-wrap;
  display: block;
}
.lib-summary.muted { color: #94a3b8; }
.lib-time {
  font-size: 12px;
  color: #94a3b8;
  flex-shrink: 0;
}
.modal-foot {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid #e2e8f0;
  flex-shrink: 0;
}
.btn-select {
  width: 100%;
  background: #2563eb;
  color: #fff;
  border: none;
  border-radius: 10px;
  padding: 12px;
  font-size: 15px;
}
.btn-select:active { opacity: 0.9; }
</style>
