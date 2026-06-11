<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { RepairAction } from '@/api/types'

const props = defineProps<{
  actions: RepairAction[]
  saving?: boolean
}>()

const emit = defineEmits<{
  save: [payload: { id: string; enabled: boolean; values: Record<string, string> }[]]
}>()

interface ActionState {
  enabled: boolean
  values: Record<string, string>
}

const localState = ref<Record<string, ActionState>>({})

function initState(actions: RepairAction[]) {
  const next: Record<string, ActionState> = {}
  for (const a of actions) {
    const values: Record<string, string> = {}
    let enabled = true
    for (const f of a.fields) {
      if (f.type === 'hidden' && f.value) values[f.key] = f.value
      if (f.key === 'enabled') enabled = f.default ?? true
      if (f.type === 'checkbox' && f.key !== 'enabled') {
        values[f.key] = f.default ? 'true' : 'false'
      } else if (f.value && f.type !== 'checkbox') {
        values[f.key] = f.value
      }
    }
    next[a.id] = { enabled, values }
  }
  localState.value = next
}

watch(() => props.actions, (a) => initState(a), { immediate: true })

const visibleFields = (a: RepairAction) => a.fields.filter((f) => f.type !== 'hidden' && f.key !== 'enabled')

const enabledCount = computed(() =>
  Object.values(localState.value).filter((s) => s.enabled).length,
)

function selectAll(on: boolean) {
  for (const id of Object.keys(localState.value)) {
    localState.value[id].enabled = on
  }
}

function toggleCheckbox(actionId: string, key: string, checked: boolean) {
  const st = localState.value[actionId]
  if (!st) return
  st.values[key] = checked ? 'true' : 'false'
}

function submit() {
  const payload = props.actions.map((a) => ({
    id: a.id,
    enabled: localState.value[a.id]?.enabled ?? false,
    values: { ...localState.value[a.id]?.values },
  }))
  emit('save', payload)
}
</script>

<template>
  <view class="repair-panel">
    <view class="repair-toolbar">
      <text class="repair-toolbar-text">已选 {{ enabledCount }} / {{ actions.length }} 项</text>
      <view class="repair-toolbar-btns">
        <text class="link" @click="selectAll(true)">全选</text>
        <text class="link" @click="selectAll(false)">全不选</text>
      </view>
    </view>

    <view v-for="a in actions" :key="a.id" class="repair-card">
      <view
        class="repair-head"
        @click="localState[a.id].enabled = !localState[a.id]?.enabled"
      >
        <view :class="['repair-check', localState[a.id]?.enabled && 'on']">
          <text class="check-box">{{ localState[a.id]?.enabled ? '☑' : '☐' }}</text>
          <text class="repair-title">
            <text v-if="a.code" class="issue-code">{{ a.code }}</text>
            {{ a.subject ? `${a.subject} · ` : '' }}{{ a.title }}
          </text>
        </view>
      </view>
      <text v-if="a.detail" class="repair-detail">发现：{{ a.detail }}</text>
      <text v-if="a.hint" class="repair-hint">建议：{{ a.hint }}</text>

      <view v-if="localState[a.id]?.enabled" class="repair-fields">
        <view v-for="f in visibleFields(a)" :key="f.key" class="field">
          <template v-if="f.type === 'checkbox'">
            <view
              class="field-check"
              @click="toggleCheckbox(a.id, f.key, localState[a.id]?.values[f.key] !== 'true')"
            >
              <text class="check-box">{{ localState[a.id]?.values[f.key] === 'true' ? '☑' : '☐' }}</text>
              <text>{{ f.label }}</text>
            </view>
            <text v-if="f.tip" class="field-tip">{{ f.tip }}</text>
          </template>
          <template v-else>
            <text class="field-label">{{ f.label }}</text>
            <input
              class="field-input"
              :type="f.type === 'number' ? 'number' : 'text'"
              :value="localState[a.id]?.values[f.key] || ''"
              @input="(e: Event) => { localState[a.id].values[f.key] = (e.target as HTMLInputElement).value }"
            />
            <text v-if="f.tip" class="field-tip">{{ f.tip }}</text>
          </template>
        </view>
      </view>
    </view>

    <button type="button" class="save-btn" :disabled="saving || enabledCount === 0" @click="submit">
      {{ saving ? '保存中…' : '应用并保存修复' }}
    </button>
  </view>
</template>

<style scoped>
.repair-panel { margin-top: 12rpx; }
.repair-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12rpx;
  font-size: 24rpx;
}
.repair-toolbar-text { color: #78350f; }
.repair-toolbar-btns { display: flex; gap: 16rpx; }
.link { color: #b45309; cursor: pointer; }
.repair-card {
  background: #fff;
  border-radius: 10rpx;
  padding: 16rpx 18rpx;
  margin-bottom: 10rpx;
  border: 1px solid #fde68a;
}
.repair-head { margin-bottom: 8rpx; }
.repair-check { display: flex; align-items: flex-start; gap: 10rpx; cursor: pointer; }
.repair-check.on .repair-title { color: #92400e; }
.check-box { font-size: 28rpx; line-height: 1.2; flex-shrink: 0; }
.repair-title { font-size: 26rpx; font-weight: 600; color: #78350f; line-height: 1.4; }
.issue-code {
  background: #fde68a;
  padding: 2rpx 8rpx;
  border-radius: 4rpx;
  margin-right: 8rpx;
  font-size: 22rpx;
}
.repair-detail, .repair-hint {
  font-size: 22rpx;
  color: #92400e;
  display: block;
  line-height: 1.45;
  margin-top: 6rpx;
}
.repair-hint { color: #b45309; }
.repair-fields {
  margin-top: 12rpx;
  padding-top: 12rpx;
  border-top: 1px dashed #fde68a;
}
.field { margin-bottom: 12rpx; }
.field-check { display: flex; align-items: flex-start; gap: 8rpx; font-size: 24rpx; color: #78350f; cursor: pointer; }
.field-label { font-size: 22rpx; color: #92400e; display: block; margin-bottom: 6rpx; }
.field-input {
  width: 100%;
  box-sizing: border-box;
  border: 1px solid #fcd34d;
  border-radius: 8rpx;
  padding: 10rpx 14rpx;
  font-size: 26rpx;
  background: #fffbeb;
}
.field-tip { font-size: 20rpx; color: #b45309; display: block; margin-top: 4rpx; }
.save-btn {
  margin-top: 16rpx;
  width: 100%;
  background: #d97706;
  color: #fff;
  border: none;
  border-radius: 12rpx;
  font-size: 28rpx;
  padding: 16rpx 0;
}
.save-btn:disabled { opacity: 0.6; }
</style>
