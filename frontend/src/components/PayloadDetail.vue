<script setup lang="ts">
import type { DetailGroup } from '@/utils/formatField'

defineProps<{
  groups: DetailGroup[]
  libraryTitles?: Record<string, string>
}>()

const emit = defineEmits<{
  libraryClick: [id: string]
}>()
</script>

<template>
  <div class="payload-detail">
    <div v-for="g in groups" :key="g.title" class="group">
      <div class="group-title">{{ g.title }}</div>
      <div class="items">
        <div v-for="item in g.items" :key="item.label" class="row">
          <div class="label">{{ item.label }}</div>
          <div class="value">
            <template v-if="item.kind === 'libraries' && item.libraryIds?.length">
              <button
                v-for="libId in item.libraryIds"
                :key="libId"
                type="button"
                class="lib-chip"
                @click="emit('libraryClick', libId)"
              >
                <span class="lib-title">{{ libraryTitles?.[libId] || libId }}</span>
                <span v-if="libraryTitles?.[libId]" class="lib-id">{{ libId }}</span>
              </button>
            </template>
            <template v-else>{{ item.value }}</template>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.payload-detail { width: 100%; }
.group {
  background: #fff;
  border-radius: 12px;
  padding: 16px 18px;
  margin-bottom: 12px;
  box-shadow: 0 1px 6px rgba(0, 0, 0, 0.05);
  border-left: 4px solid #3b82f6;
}
.group-title {
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 14px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e2e8f0;
}
.items { display: flex; flex-direction: column; gap: 12px; }
.row {
  display: grid;
  grid-template-columns: minmax(100px, 32%) 1fr;
  gap: 12px 16px;
  align-items: start;
}
.label {
  font-size: 13px;
  color: #64748b;
  line-height: 1.5;
  padding-top: 2px;
}
.value {
  font-size: 15px;
  color: #334155;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}
.lib-chip {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  margin: 0 8px 8px 0;
  padding: 8px 12px;
  border: 1px solid #bfdbfe;
  border-radius: 8px;
  background: #eff6ff;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s, border-color 0.15s;
}
.lib-chip:hover {
  background: #dbeafe;
  border-color: #93c5fd;
}
.lib-title {
  font-size: 14px;
  font-weight: 500;
  color: #1d4ed8;
}
.lib-id {
  font-size: 11px;
  color: #64748b;
}
@media (max-width: 520px) {
  .row { grid-template-columns: 1fr; gap: 4px; }
}
</style>
