<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api } from '@/api'
import type { FieldSchema, FormSchema, LibraryItem } from '@/api/types'
import LibraryDetailModal from '@/components/LibraryDetailModal.vue'

const props = defineProps<{
  schema: FormSchema
  modelValue: Record<string, unknown>
  stockCode?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [Record<string, unknown>]
}>()

const libraryItems = ref<LibraryItem[]>([])
const libraryLoading = ref(false)
const showAddLib = ref(false)
const newLibTitle = ref('')
const newLibText = ref('')
const libraryModalVisible = ref(false)
const libraryModalLoading = ref(false)
const libraryModalItem = ref<LibraryItem | null>(null)
const libraryModalId = ref('')
const riskLimits = ref<{ warning: number; hard: number } | null>(null)
const pctOverLimit = ref(false)
const lastPctChecked = ref<number | null>(null)
/** 组件内表单状态（与 store 同步，避免受控输入丢值） */
const localValues = ref<Record<string, unknown>>({})
/** 数字字段显示用草稿（保留用户输入过程，防止提交前被清空） */
const numberDrafts = ref<Record<string, string>>({})

const selectedLibIds = computed(() => {
  const v = localValues.value['related_library_ids']
  return Array.isArray(v) ? (v as string[]) : []
})

const showNoLibraryReason = computed(() => selectedLibIds.value.length === 0)

const groupedFields = computed(() => {
  const map = new Map<string, FieldSchema[]>()
  for (const g of props.schema.groups) {
    map.set(g, [])
  }
  for (const f of props.schema.fields) {
    if (f.key === 'no_library_reason' || f.key === 'position_size_plan.override_reason') continue
    const list = map.get(f.group) || []
    list.push(f)
    map.set(f.group, list)
  }
  return [...map.entries()].filter(([, fields]) => fields.length > 0)
})

function getVal(key: string): unknown {
  return localValues.value[key]
}

function setVal(key: string, val: unknown) {
  localValues.value = { ...localValues.value, [key]: val }
  emit('update:modelValue', { ...localValues.value })
}

function applySchemaDefaults(base: Record<string, unknown>): Record<string, unknown> {
  const next = { ...base }
  for (const f of props.schema?.fields ?? []) {
    if (next[f.key] === undefined && f.default !== undefined && f.default !== null) {
      if (f.type === 'number' && f.default === 0) continue
      next[f.key] = f.default
    }
  }
  if (next['related_library_ids'] === undefined) {
    next['related_library_ids'] = []
  }
  return next
}

// uni-app H5 部分场景用 detail.value，标准 DOM 用 target/currentTarget.value
function eventValue(e: Event): string {
  const ev = e as Event & { detail?: { value?: string } }
  if (ev.detail?.value !== undefined) return String(ev.detail.value)
  const t = (e.currentTarget ?? e.target) as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | null
  return t?.value ?? ''
}

function onEnumSelect(field: FieldSchema, e: Event) {
  setVal(field.key, eventValue(e))
}

function arrayVal(field: FieldSchema): string[] {
  const v = getVal(field.key)
  return Array.isArray(v) ? (v as string[]) : []
}

function toggleMultiEnum(field: FieldSchema, value: string) {
  const cur = [...arrayVal(field)]
  const i = cur.indexOf(value)
  if (i >= 0) cur.splice(i, 1)
  else cur.push(value)
  setVal(field.key, cur)
}

function isMultiSelected(field: FieldSchema, value: string): boolean {
  return arrayVal(field).includes(value)
}

function toggleLibrary(id: string) {
  const cur = [...selectedLibIds.value]
  const i = cur.indexOf(id)
  if (i >= 0) cur.splice(i, 1)
  else cur.push(id)
  setVal('related_library_ids', cur)
}

function previewSummary(text?: string, max = 48) {
  if (!text) return ''
  const t = text.replace(/\s+/g, ' ').trim()
  return t.length > max ? `${t.slice(0, max)}…` : t
}

async function openLibraryDetail(id: string) {
  libraryModalId.value = id
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
  libraryModalId.value = ''
}

function toggleLibraryFromModal() {
  if (libraryModalId.value) toggleLibrary(libraryModalId.value)
}

async function loadLibrary() {
  if (!props.stockCode) return
  libraryLoading.value = true
  try {
    const res = await api.getLibrary(props.stockCode)
    libraryItems.value = res.items || []
  } catch {
    libraryItems.value = []
  } finally {
    libraryLoading.value = false
  }
}

async function submitQuickAdd() {
  if (!props.stockCode || !newLibTitle.value.trim() || !newLibText.value.trim()) {
    uni.showToast({ title: '请填写标题和内容', icon: 'none' })
    return
  }
  try {
    const res = await api.quickAddLibrary({
      title: newLibTitle.value.trim(),
      text: newLibText.value.trim(),
      stock: props.stockCode,
    })
    await loadLibrary()
    setVal('related_library_ids', [...selectedLibIds.value, res.library_id])
    showAddLib.value = false
    newLibTitle.value = ''
    newLibText.value = ''
    uni.showToast({ title: '素材已录入', icon: 'success' })
  } catch (e: unknown) {
    uni.showToast({ title: (e as Error).message, icon: 'none' })
  }
}

async function loadRiskRules() {
  try {
    const res = await api.getRiskRules()
    const lim = res.position_limits?.single_stock
    if (lim) {
      riskLimits.value = { warning: lim.warning_pct, hard: lim.hard_block_pct }
    }
  } catch {
    riskLimits.value = null
  }
}

function isPctOverLimit(pct: number): boolean {
  if (!riskLimits.value || !pct || Number.isNaN(pct)) return false
  return pct >= riskLimits.value.warning
}

async function checkInitialPct(pct: number, showAlert = false) {
  if (!pct || Number.isNaN(pct)) {
    pctOverLimit.value = false
    return
  }
  if (lastPctChecked.value === pct && !showAlert) return
  lastPctChecked.value = pct

  let over = isPctOverLimit(pct)

  if (props.stockCode) {
    try {
      const res = await api.riskCheck({
        scenario: props.schema.checklist_type === 'add' ? 'add' : 'buy',
        code: props.stockCode,
        planned_position_pct_after: pct,
      })
      over = (res.warnings?.length || 0) > 0 || (res.hard_blocks?.length || 0) > 0 || over
    } catch {
      /* 本地阈值已判断 */
    }
  }

  pctOverLimit.value = over
  if (over && riskLimits.value && showAlert) {
    uni.showModal({
      title: '仓位超过 M7 风控线',
      content: `你填写的初始仓位 ${pct}% 超过单票警告线 ${riskLimits.value.warning}%（硬拦截 ${riskLimits.value.hard}%）。如仍要继续，请在下方填写超限说明。`,
      showCancel: false,
    })
  }
}

function showRiskRules() {
  if (!riskLimits.value) {
    uni.showToast({ title: '风控规则加载失败', icon: 'none' })
    return
  }
  uni.showModal({
    title: 'M7 单票仓位规则',
    content: `警告线：${riskLimits.value.warning}%\n硬拦截：${riskLimits.value.hard}%\n\n修改方式：编辑 data/accounts/{账户}/state/risk_rules.yaml`,
    showCancel: false,
  })
}

function arrayText(field: FieldSchema): string {
  return arrayVal(field).join('\n')
}

function onArrayInput(field: FieldSchema, e: Event) {
  const lines = eventValue(e)
    .split('\n')
    .map((s) => String(s ?? '').trim())
    .filter(Boolean)
  setVal(field.key, lines)
}

function textValue(field: FieldSchema): string {
  const v = getVal(field.key)
  if (v === undefined || v === null) return ''
  return String(v)
}

function numberText(field: FieldSchema): string {
  const k = field.key
  if (k in numberDrafts.value) return numberDrafts.value[k]
  const v = getVal(k)
  if (v === undefined || v === null || v === '') return ''
  if (v === 0) return ''
  return String(v)
}

function parseNumberField(raw: unknown): number | '' {
  const s = String(raw ?? '').trim()
  if (s === '' || s === '-') return ''
  const n = parseFloat(s)
  return Number.isNaN(n) ? '' : n
}

function commitNumberField(field: FieldSchema, raw: unknown) {
  const val = parseNumberField(raw)
  if (val === '') {
    const next = { ...localValues.value }
    delete next[field.key]
    localValues.value = next
    emit('update:modelValue', { ...next })
    if (field.key === 'position_size_plan.initial_pct') {
      pctOverLimit.value = false
      lastPctChecked.value = null
    }
    return
  }
  setVal(field.key, val)
  if (field.key === 'position_size_plan.initial_pct') {
    void checkInitialPct(val, false)
  }
}

function onNumberInput(field: FieldSchema, e: Event) {
  const raw = eventValue(e)
  numberDrafts.value = { ...numberDrafts.value, [field.key]: raw }
  commitNumberField(field, raw)
}

function onNumberBlur(field: FieldSchema, e: Event) {
  const raw = eventValue(e)
  numberDrafts.value = { ...numberDrafts.value, [field.key]: raw }
  commitNumberField(field, raw)
  if (field.key === 'position_size_plan.initial_pct') {
    const val = parseNumberField(raw)
    if (typeof val === 'number') {
      lastPctChecked.value = null
      void checkInitialPct(val, true)
    }
  }
}

function seedNumberDrafts(values: Record<string, unknown>) {
  for (const f of props.schema?.fields ?? []) {
    if (f.type !== 'number') continue
    const n = values[f.key]
    if (n !== undefined && n !== null && n !== '') {
      numberDrafts.value[f.key] = String(n)
    }
  }
}

// 外部载入（续办/默认值）时合并到 local，不覆盖用户已填项
watch(
  () => props.modelValue,
  (incoming) => {
    if (!incoming || Object.keys(incoming).length === 0) return

    if (Object.keys(localValues.value).length === 0) {
      localValues.value = { ...incoming }
      seedNumberDrafts(incoming)
      return
    }

    const merged = { ...localValues.value }
    let changed = false
    for (const [k, v] of Object.entries(incoming)) {
      if (v === undefined || v === null || v === '') continue
      const cur = merged[k]
      if (cur === undefined || cur === null || cur === '') {
        merged[k] = v
        changed = true
      }
    }
    if (changed) {
      localValues.value = merged
      seedNumberDrafts(merged)
    }
  },
  { immediate: true, deep: true },
)

watch(
  () => props.schema,
  () => {
    if (!props.schema) return
    const next = applySchemaDefaults(localValues.value)
    const changed = JSON.stringify(next) !== JSON.stringify(localValues.value)
    if (changed) {
      localValues.value = next
      emit('update:modelValue', { ...next })
    }
    loadRiskRules()
  },
  { immediate: true },
)

watch(
  () => props.stockCode,
  () => loadLibrary(),
  { immediate: true },
)

watch(
  () => localValues.value['position_size_plan.initial_pct'],
  (v) => {
    const pct = Number(v)
    if (pct > 0 && !Number.isNaN(pct)) void checkInitialPct(pct, false)
  },
)

/** 合并 DOM/数字草稿到 values，空 DOM 不覆盖已有值 */
function mergeFormValues(base: Record<string, unknown>): Record<string, unknown> {
  const next = { ...base }

  for (const [key, raw] of Object.entries(numberDrafts.value)) {
    const val = parseNumberField(raw)
    if (val !== '') next[key] = val
  }

  if (typeof document === 'undefined') return next
  const root = document.querySelector('.dynamic-form')
  if (!root) return next

  const els = root.querySelectorAll<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>('[data-field-key]')
  els.forEach((el) => {
    const key = el.dataset.fieldKey
    if (!key) return
    const field = props.schema.fields.find((f) => f.key === key)
    if (!field) return
    const raw = String(el.value ?? '')
    const existing = next[key]
    const existingFilled =
      existing !== undefined && existing !== null && existing !== '' &&
      !(Array.isArray(existing) && existing.length === 0)

    if (field.type === 'number') {
      const val = parseNumberField(raw)
      if (val !== '') next[key] = val
      else if (raw.trim() === '' && existingFilled) return
      else if (val === '') return
    } else if (field.type === 'array') {
      if (raw.trim() === '' && existingFilled) return
      next[key] = raw.split('\n').map((s) => s.trim()).filter(Boolean)
    } else {
      if (raw.trim() === '' && existingFilled) return
      next[key] = raw
    }
  })
  return next
}

/** 提交前收集完整表单值 */
function getFormValues(): Record<string, unknown> {
  const merged = mergeFormValues(localValues.value)
  localValues.value = merged
  emit('update:modelValue', { ...merged })
  const pct = Number(merged['position_size_plan.initial_pct'])
  if (pct > 0 && !Number.isNaN(pct)) {
    pctOverLimit.value = isPctOverLimit(pct)
  }
  return merged
}

defineExpose({
  getFormValues,
  isPctOverLimit: () => pctOverLimit.value,
})

onMounted(() => {
  loadRiskRules()
  loadLibrary()
})
</script>

<template>
  <div class="dynamic-form">
    <div v-if="schema.description" class="form-desc">{{ schema.description }}</div>
    <div v-for="[group, fields] in groupedFields" :key="group" class="group">
      <div class="group-title">{{ group }}</div>
      <div v-for="field in fields" :key="field.key" class="field">
        <div class="field-label">
          {{ field.label }}<span v-if="field.required" class="required">*</span>
        </div>
        <div v-if="field.tip" class="field-tip">{{ field.tip }}</div>

        <div
          v-if="field.key === 'position_size_plan.initial_pct' && riskLimits"
          class="risk-hint"
        >
          当前单票规则：警告 {{ riskLimits.warning }}% · 硬拦截 {{ riskLimits.hard }}%
          <span class="risk-link" @click.stop="showRiskRules">查看规则</span>
        </div>

        <!-- 收益驱动：可点击选项块（H5 原生 checkbox 不可点） -->
        <div v-if="field.type === 'multi_enum'" class="chip-group">
          <div
            v-for="opt in field.options || []"
            :key="opt.value"
            :class="['chip', isMultiSelected(field, opt.value) && 'chip-active']"
            @click.stop="toggleMultiEnum(field, opt.value)"
          >
            {{ opt.label }}
          </div>
        </div>

        <!-- L1 素材多选 -->
        <div v-else-if="field.type === 'library_multi'" class="library-box">
          <div v-if="!stockCode" class="lib-hint">请先填写标的代码，再选择关联素材</div>
          <div v-else-if="libraryLoading" class="lib-hint">加载素材中…</div>
          <div v-else-if="libraryItems.length === 0" class="lib-hint">
            暂无该标的的 L1 素材，可点击下方新增
          </div>
          <div v-else class="lib-list">
            <div
              v-for="item in libraryItems"
              :key="item.id"
              :class="['lib-card', selectedLibIds.includes(item.id) && 'lib-card-active']"
            >
              <div class="lib-card-main" @click.stop="toggleLibrary(item.id)">
                <span class="lib-title">{{ item.title }}</span>
                <span class="lib-meta">{{ item.tier }} · {{ item.source }}</span>
                <span v-if="item.summary" class="lib-preview">{{ previewSummary(item.summary) }}</span>
                <span v-if="selectedLibIds.includes(item.id)" class="lib-selected-tag">已关联</span>
              </div>
              <div type="button" class="lib-view-btn" @click.stop="openLibraryDetail(item.id)">查看</div>
            </div>
          </div>
          <div
            type="button"
            class="btn-add-lib"
            @click.stop="showAddLib = !showAddLib"
          >
            {{ showAddLib ? '取消' : '+ 新增研究素材' }}
          </div>
          <div v-if="showAddLib" class="add-lib-form">
            <textarea v-model="newLibTitle" class="ctrl ctrl-line" rows="1" placeholder="素材标题" />
            <textarea v-model="newLibText" class="ctrl ctrl-area" rows="3" placeholder="正文摘要或关键观点" />
            <div class="btn-save-lib" @click.stop="submitQuickAdd">保存并选中</div>
          </div>
          <!-- 无素材说明：紧跟素材区，避免被分组逻辑隐藏 -->
          <div v-if="showNoLibraryReason" class="no-lib-reason">
            <div class="field-label">无 L1 素材说明 <span class="required">*</span></div>
            <div class="field-tip">未选择任何素材时必填，说明本次买入依据来源</div>
            <textarea
              class="ctrl ctrl-area"
              data-field-key="no_library_reason"
              :value="textValue({ key: 'no_library_reason' } as FieldSchema)"
              placeholder="请说明买入依据（个人判断须写清逻辑）"
              rows="2"
              @input="(e) => setVal('no_library_reason', eventValue(e as Event))"
            />
          </div>
        </div>

        <textarea
          v-else-if="field.type === 'textarea'"
          class="ctrl ctrl-area"
          :data-field-key="field.key"
          :value="textValue(field)"
          :placeholder="`请填写${field.label}`"
          :rows="field.rows || 3"
          @input="(e) => setVal(field.key, eventValue(e as Event))"
        />

        <textarea
          v-else-if="field.type === 'text' || field.type === 'date'"
          class="ctrl ctrl-line"
          :data-field-key="field.key"
          :value="textValue(field)"
          :placeholder="`请填写${field.label}`"
          rows="1"
          @input="(e) => setVal(field.key, eventValue(e as Event))"
        />

        <template v-else-if="field.type === 'number'">
          <input
            type="text"
            inputmode="decimal"
            class="ctrl ctrl-line"
            :data-field-key="field.key"
            :value="numberText(field)"
            :placeholder="`请填写${field.label}`"
            @input="(e) => onNumberInput(field, e as Event)"
            @blur="(e) => onNumberBlur(field, e as Event)"
          />
          <div
            v-if="field.key === 'position_size_plan.initial_pct' && pctOverLimit && riskLimits"
            class="field-warn"
          >
            已超过 M7 警告线 {{ riskLimits.warning }}%（硬拦截 {{ riskLimits.hard }}%），请填写下方「仓位超限说明」
          </div>
        </template>

        <select
          v-else-if="field.type === 'enum'"
          class="ctrl ctrl-select"
          :data-field-key="field.key"
          :value="String(getVal(field.key) ?? field.default ?? '')"
          @change="(e) => onEnumSelect(field, e as Event)"
        >
          <option v-for="opt in field.options || []" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>

        <div
          v-else-if="field.type === 'bool'"
          :class="['chip', !!getVal(field.key) && 'chip-active']"
          @click.stop="setVal(field.key, !getVal(field.key))"
        >
          {{ !!getVal(field.key) ? '是' : '否' }}
        </div>

        <textarea
          v-else-if="field.type === 'array'"
          class="ctrl ctrl-area"
          :data-field-key="field.key"
          :value="arrayText(field)"
          placeholder="每行一项"
          rows="3"
          @input="(e) => onArrayInput(field, e as Event)"
        />
      </div>

      <!-- 仓位超限说明：放在仓位计划分组末尾 -->
      <div v-if="group === '仓位计划' && pctOverLimit" class="field override-field">
        <div class="field-label">仓位超限说明 <span class="required">*</span></div>
        <div class="field-tip">初始仓位超过 M7 警告线，请说明为何仍可接受</div>
        <textarea
          class="ctrl ctrl-area"
          data-field-key="position_size_plan.override_reason"
          :value="textValue({ key: 'position_size_plan.override_reason' } as FieldSchema)"
          placeholder="说明超限理由与后续约束"
          rows="2"
          @input="(e) => setVal('position_size_plan.override_reason', eventValue(e as Event))"
        />
      </div>
    </div>
  </div>

  <LibraryDetailModal
    :visible="libraryModalVisible"
    :loading="libraryModalLoading"
    :item="libraryModalItem"
    :stock-code="stockCode"
    :selectable="true"
    :selected="libraryModalId ? selectedLibIds.includes(libraryModalId) : false"
    @close="closeLibraryModal"
    @toggle-select="toggleLibraryFromModal"
  />
</template>

<style scoped>
.dynamic-form { width: 100%; box-sizing: border-box; }
.form-desc { font-size: 14px; color: #64748b; margin-bottom: 12px; line-height: 1.5; }
.group {
  margin-bottom: 16px; background: #fff; border-radius: 8px;
  padding: 12px 16px; box-shadow: 0 1px 6px rgba(0, 0, 0, 0.04);
}
.group-title {
  font-size: 15px; font-weight: 600; color: #1e293b;
  margin-bottom: 10px; padding-bottom: 6px; border-bottom: 1px solid #e2e8f0;
}
.field { width: 100%; margin-bottom: 14px; }
.field-label { font-size: 14px; font-weight: 500; margin-bottom: 4px; }
.required { color: #ef4444; margin-left: 2px; }
.field-tip { font-size: 12px; color: #94a3b8; margin-bottom: 6px; line-height: 1.4; }
.field-warn {
  font-size: 12px; color: #b45309; background: #fff7ed;
  border: 1px solid #fdba74; border-radius: 6px; padding: 8px 10px; margin-top: 6px;
}
.risk-hint {
  font-size: 12px; color: #b45309; background: #fffbeb;
  border: 1px solid #fde68a; border-radius: 6px; padding: 8px 10px; margin-bottom: 8px;
}
.risk-link { color: #2563eb; margin-left: 8px; cursor: pointer; }
.chip-group { display: flex; flex-wrap: wrap; gap: 8px; }
.chip {
  padding: 8px 14px; border-radius: 20px; font-size: 14px;
  border: 1px solid #e2e8f0; background: #f8fafc; color: #334155;
  cursor: pointer; user-select: none; -webkit-tap-highlight-color: transparent;
}
.chip:active { opacity: 0.8; }
.chip-active {
  background: #eff6ff; border-color: #2563eb; color: #1d4ed8; font-weight: 500;
}
.lib-title { font-weight: 500; }
.lib-meta { font-size: 11px; color: #94a3b8; margin-top: 2px; }
.lib-preview { font-size: 12px; color: #64748b; margin-top: 4px; line-height: 1.4; }
.lib-list { display: flex; flex-direction: column; gap: 8px; }
.lib-card {
  display: flex;
  align-items: stretch;
  gap: 8px;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  background: #f8fafc;
  overflow: hidden;
}
.lib-card-active { border-color: #93c5fd; background: #eff6ff; }
.lib-card-main {
  flex: 1;
  padding: 10px 12px;
  cursor: pointer;
  min-width: 0;
}
.lib-selected-tag {
  display: inline-block;
  margin-top: 6px;
  font-size: 11px;
  color: #1d4ed8;
  background: #dbeafe;
  padding: 2px 8px;
  border-radius: 6px;
}
.lib-view-btn {
  flex-shrink: 0;
  align-self: center;
  margin-right: 10px;
  font-size: 13px;
  color: #2563eb;
  background: #fff;
  border: 1px solid #bfdbfe;
  border-radius: 8px;
  padding: 8px 14px;
  cursor: pointer;
  user-select: none;
}
.library-box { display: flex; flex-direction: column; gap: 10px; }
.lib-hint { font-size: 13px; color: #94a3b8; }
.no-lib-reason {
  margin-top: 12px; padding-top: 12px; border-top: 1px dashed #e2e8f0;
}
.override-field {
  margin-top: 8px; padding-top: 12px; border-top: 1px dashed #fde68a;
}
.btn-add-lib, .btn-save-lib {
  font-size: 13px; color: #2563eb; background: #eff6ff;
  border: 1px solid #bfdbfe; border-radius: 6px; padding: 10px 12px;
  cursor: pointer; text-align: center; user-select: none;
}
.btn-save-lib { background: #2563eb; color: #fff; border-color: #2563eb; }
.add-lib-form { display: flex; flex-direction: column; gap: 8px; }
.ctrl {
  width: 100%; display: block; box-sizing: border-box;
  border: 1px solid #e2e8f0; border-radius: 8px; padding: 10px 12px;
  font-size: 15px; line-height: 1.4; background: #f8fafc; color: #334155;
  font-family: inherit; outline: none;
}
.ctrl:focus { border-color: #93c5fd; background: #fff; }
.ctrl-line { min-height: 42px; max-height: 42px; resize: none; overflow: hidden; }
.ctrl-area { min-height: 72px; resize: vertical; }
.ctrl-select { height: 42px; appearance: auto; cursor: pointer; }
</style>
