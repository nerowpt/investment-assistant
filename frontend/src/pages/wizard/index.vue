<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, nextTick, ref } from 'vue'
import { api } from '@/api'
import DynamicForm from '@/components/DynamicForm.vue'
import { useWizardStore } from '@/store/wizard'
import { flattenPayload, readFormNumber } from '@/utils/payload'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { charCount } from '@/utils/text'

const store = useWizardStore()
const busy = ref(false)
const submitError = ref('')
/** 卖出向导：当前标的可卖股数上限（open lot 之和，来自 portfolio） */
const maxSellShares = ref<number | null>(null)
const { run: runAction } = useAsyncAction(busy, submitError)
/** 标的代码是否由首页/续办带入（带入后只读展示，手填场景保持可编辑） */
const codeFromNav = ref(false)
const formRef = ref<InstanceType<typeof DynamicForm> | null>(null)
const sellPlan = ref<Record<string, unknown> | null>(null)

const stepTitle = computed(() => {
  const m: Record<string, string> = {
    form: '填写决策表单',
    risk: '风险检查',
    confirm: '确认执行',
    done: '完成',
  }
  return m[store.step] || ''
})

const riskWarnings = computed(() => {
  const r = store.riskResult as Record<string, unknown> | null
  if (!r) return []
  return (r.warnings as unknown[]) || []
})

const riskBlocks = computed(() => {
  const r = store.riskResult as Record<string, unknown> | null
  if (!r) return []
  return (r.hard_blocks as unknown[]) || []
})

const exceptionCharCount = computed(() => charCount(store.exceptionReason))

const emotionSelfCheckCount = computed(() => charCount(store.emotionSelfCheck))

const currentEmotionTag = computed(() =>
  String(store.emotionTag || store.values['emotion_tag'] || '').toLowerCase(),
)

const showEmotionCheck = computed(() => {
  if (store.emotionCheckNeeded) return true
  return ['fomo', 'greedy', 'anxious'].includes(currentEmotionTag.value)
})

const emotionTagLabel = computed(() => {
  const m: Record<string, string> = { fomo: 'FOMO', greedy: '贪婪', anxious: '焦虑', calm: '冷静' }
  return m[currentEmotionTag.value] || currentEmotionTag.value || '高风险情绪'
})

const relatedLibIds = computed(() => {
  const libs = store.values['related_library_ids']
  return Array.isArray(libs) ? (libs as string[]).filter(Boolean) : []
})

function formatRiskItem(item: unknown): string {
  if (!item || typeof item !== 'object') return String(item ?? '')
  const msg = (item as Record<string, unknown>).message
  return typeof msg === 'string' && msg ? msg : JSON.stringify(item)
}

onLoad(async (query) => {
  store.reset()
  const resumeId = String(query?.resume_id || '')

  try {
    if (resumeId) {
      const detail = await api.getChecklist(resumeId)
      const cs = detail.checklist
      store.checklistId = cs.id
      store.checklistType = cs.checklist_type
      store.code = cs.code
      store.name = cs.name
      codeFromNav.value = !!cs.code

      const schemaRes = await api.getSchema(cs.checklist_type)
      store.schema = schemaRes.schema
      const flat = flattenPayload(detail.payload || {})
      const cleaned = Object.fromEntries(
        Object.entries(flat).filter(([, val]) => val !== '' && val !== null),
      )
      store.values = { ...schemaRes.default_values, ...cleaned }

      if (cs.status === 'submitted') {
        store.riskResult = detail.risk_result || null
        store.approveBlocked = !!(detail.risk_result as Record<string, unknown>)?.approve_blocked
        store.step = 'confirm'
      } else {
        store.step = 'form'
      }
      return
    }

    const type = String(query?.type || 'buy')
    const code = String(query?.code || '')
    const name = String(query?.name || '')
    const journalId = String(query?.journal_id || '')
    const lotId = String(query?.lot_id || '')
    const from = String(query?.from || '')
    const watchId = String(query?.watch_id || '')

    store.checklistType = type
    store.code = code
    store.name = name
    store.linkedJournalId = journalId
    codeFromNav.value = !!code

    const res = await api.getSchema(type)
    store.schema = res.schema
    store.values = { ...res.default_values }
    if (journalId && (type === 'add' || type === 'inspect')) {
      store.values['linked_buy_journal_id'] = journalId
    }
    if (type === 'buy' && code && (from === 'watchlist' || from === 'research')) {
      const ctx = await api.getPoolBuyContext({
        code,
        from: from || 'research',
        watch_id: watchId || undefined,
      })
      if (ctx.name) store.name = ctx.name
      const p = ctx.prefill
      if (p.source_entry) store.values['source_entry'] = p.source_entry
      if (p.watchlist_origin_id) store.values['watchlist_origin_id'] = p.watchlist_origin_id
      if (p.related_library_ids?.length) {
        store.values['related_library_ids'] = [...p.related_library_ids]
      }
      if (p.buy_reason_summary) store.values['buy_reason_summary'] = p.buy_reason_summary
      if (p.investment_thesis) store.values['investment_thesis'] = p.investment_thesis
    }
    if (type === 'review' && code && lotId) {
      const ctx = await api.getLotReviewContext({ code, lot_id: lotId })
      if (ctx.name) store.name = ctx.name
      const p = ctx.prefill
      if (p.review_type) store.values['review_type'] = p.review_type
      if (p.target_lot_id) store.values['target_lot_id'] = p.target_lot_id
      if (p.target_code) store.values['target_code'] = p.target_code
      if (p.period_start) store.values['period_start'] = p.period_start
      if (p.period_end) store.values['period_end'] = p.period_end
      if (p['attribution.result_category']) {
        store.values['attribution.result_category'] = p['attribution.result_category']
      }
      if (p.confirmed_patterns?.length) {
        store.values['confirmed_patterns'] = [...p.confirmed_patterns]
      }
      if (p.notes) store.values['notes'] = p.notes
      store.lotReviewContext = ctx
    }
    if (type === 'sell' && code) {
      try {
        const pf = await api.getPortfolio(code)
        const pos = pf.positions?.find((p) => p.code === code)
        if (pos?.shares != null && pos.shares !== '') {
          const n = parseFloat(String(pos.shares))
          if (!Number.isNaN(n) && n > 0) maxSellShares.value = n
        }
      } catch {
        /* 可卖股数加载失败不阻断向导，preview 仍会校验 */
      }
    }
  } catch (e: unknown) {
    uni.showToast({ title: (e as Error).message, icon: 'none' })
  }
})

function isFilledArray(val: unknown): boolean {
  if (!Array.isArray(val) || val.length === 0) return false
  return val.some((s) => s != null && String(s).trim() !== '' && s !== '待填写')
}

function showFormError(msg: string) {
  uni.showModal({ title: '请完善表单', content: msg, showCancel: false })
}

function validateFormSync(values?: Record<string, unknown>): string | null {
  const v = values ?? store.values ?? {}
  if (store.checklistType === 'review') {
    if (!String(v['period_start'] || '').trim() || !String(v['period_end'] || '').trim()) {
      return '请填写复盘区间（开始与结束日期）'
    }
    if (!isFilledArray(v['confirmed_patterns'])) {
      return '请填写至少一条「已确认模式/教训」（不能使用默认占位「待填写」）'
    }
    if (String(v['review_type'] || '') === 'lot_attribution' && !String(v['target_lot_id'] || '').trim()) {
      return '单笔 lot 归因须指定目标 lot'
    }
  }
  if (store.checklistType === 'buy') {
    const libs = v['related_library_ids']
    const libIds = Array.isArray(libs) ? libs : []
    if (libIds.length === 0 && !String(v['no_library_reason'] || '').trim()) {
      return '未选择 L1 素材时，须填写「无 L1 素材说明」'
    }
    const drivers = v['expected_return_driver']
    if (!Array.isArray(drivers) || drivers.length === 0) {
      return '请至少选择一项「收益驱动」'
    }
    const pct = readFormNumber(v, 'position_size_plan.initial_pct')
    if (!pct || Number.isNaN(pct)) {
      return '请填写「初始仓位 %」'
    }
    const maxPct = readFormNumber(v, 'position_size_plan.max_pct')
    if (!maxPct || Number.isNaN(maxPct)) {
      return '请填写「最大仓位 %」'
    }
    const price = readFormNumber(v, 'execution_price')
    if (!price || Number.isNaN(price)) {
      return '请填写「实际成交价」'
    }
    const shares = readFormNumber(v, 'shares')
    if (!shares || Number.isNaN(shares)) {
      return '请填写「买入股数」'
    }
    if (!isFilledArray(v['reversal_conditions'])) {
      return '请填写至少一条「逻辑反转条件」（不能使用默认占位「待填写」）'
    }
    if (!isFilledArray(v['identified_risks'])) {
      return '请填写至少一条「已识别风险」'
    }
    const pctOver = formRef.value?.isPctOverLimit?.() || pct >= 10
    if (pctOver && !String(v['position_size_plan.override_reason'] || '').trim()) {
      return '初始仓位超过 M7 警告线，请填写「仓位超限说明」'
    }
  }
  if (store.checklistType === 'sell') {
    const sellShares = readFormNumber(v, 'sell_shares')
    if (!sellShares || Number.isNaN(sellShares) || sellShares <= 0) {
      return '请填写「卖出股数」'
    }
    if (maxSellShares.value != null && sellShares > maxSellShares.value) {
      return `卖出股数不能超过当前可卖 ${maxSellShares.value} 股`
    }
    const price = readFormNumber(v, 'execution_price')
    if (!price || Number.isNaN(price) || price <= 0) {
      return '请填写「实际成交价」'
    }
  }
  return null
}

function onFormUpdate(v: Record<string, unknown>) {
  store.$patch({ values: v })
}

function goBackToForm() {
  store.step = 'form'
}

function goBackToRisk() {
  store.step = 'risk'
}

function buildExceptionBody(): Record<string, unknown> | null {
  const tomorrow = new Date()
  tomorrow.setMonth(tomorrow.getMonth() + 1)
  return {
    triggered_rule_id: 'm7_manual',
    exception_reason: store.exceptionReason,
    expected_compensation: '若判断错误将在复盘时主动减仓并写 review',
    review_date: tomorrow.toISOString().slice(0, 10),
    confirm_text: '我已理解风险，自愿继续',
    library_item_ids: relatedLibIds.value,
  }
}

function validateExceptionBeforeSubmit(): string | null {
  if (!store.approveBlocked) return null
  if (exceptionCharCount.value < 80) {
    return `例外说明须不少于 80 字（当前 ${exceptionCharCount.value} 字）`
  }
  if (relatedLibIds.value.length === 0) {
    return '硬拦截例外须关联至少 1 条 L1 素材。请点击「返回修改」，在表单中选择关联素材后再提交。'
  }
  return null
}

function validateEmotionBeforeSubmit(): string | null {
  if (!showEmotionCheck.value) return null
  if (!store.emotionSelfCheck.trim()) {
    return `情绪标签为「${emotionTagLabel.value}」，须填写情绪自检说明后再提交`
  }
  if (emotionSelfCheckCount.value < 10) {
    return `情绪自检说明建议不少于 10 字（当前 ${emotionSelfCheckCount.value} 字）`
  }
  return null
}

function buildSubmitBody(): { exception?: Record<string, unknown>; emotion_self_check?: string } {
  const body: { exception?: Record<string, unknown>; emotion_self_check?: string } = {}
  if (showEmotionCheck.value && store.emotionSelfCheck.trim()) {
    body.emotion_self_check = store.emotionSelfCheck.trim()
  }
  if (store.approveBlocked) {
    body.exception = buildExceptionBody()!
  } else if (riskWarnings.value.length > 0) {
    body.exception = { acked: true, ack_note: '已在向导中确认风险提示' }
  }
  return body
}

async function nextFromForm() {
  if (!store.schema) {
    showFormError('表单未加载，请返回重试')
    return
  }
  if (store.checklistType !== 'review' && !store.code && ['buy', 'add', 'sell', 'inspect'].includes(store.checklistType)) {
    showFormError('请返回首页选择标的，或在表单中填写标的代码')
    return
  }
  const formValues = formRef.value?.getFormValues() ?? store.values
  store.$patch({ values: formValues })
  await nextTick()
  const formErr = validateFormSync(formValues)
  if (formErr) {
    showFormError(formErr)
    return
  }
  await runAction(async () => {
    let newDraftId = ''
    if (store.checklistId) {
      await api.updateDraft(store.checklistId, {
        code: store.code,
        name: store.name,
        values: store.values,
      })
    } else {
      const draft = await api.createDraft({
        checklist_type: store.checklistType,
        code: store.code,
        name: store.name,
        values: store.values,
      })
      newDraftId = draft.id
      store.checklistId = draft.id
    }
    try {
      const preview = await api.preview(store.checklistId)
      store.approveBlocked = preview.approve_blocked
      store.riskResult = preview.risk_result
      store.emotionCheckNeeded = !!preview.emotion_check_needed
      store.emotionTag = preview.emotion_tag || String(store.values['emotion_tag'] || '')
      store.step = 'risk'
    } catch (e: unknown) {
      // 本次新建的 draft 在 preview 失败时自动作废，避免首页残留「进行中」幽灵草稿
      if (newDraftId) {
        try {
          await api.reject(newDraftId, '预览校验未通过，自动作废')
        } catch {
          /* 作废失败不掩盖原始错误 */
        }
        store.checklistId = ''
      }
      throw e
    }
  })
}

async function doSubmit() {
  const emotionErr = validateEmotionBeforeSubmit()
  if (emotionErr) {
    submitError.value = emotionErr
    return
  }
  if (store.approveBlocked) {
    const err = validateExceptionBeforeSubmit()
    if (err) {
      submitError.value = err
      return
    }
  }
  const body = buildSubmitBody()
  await runAction(async () => {
    await api.submit(store.checklistId, body)
    if (store.checklistType === 'sell') {
      sellPlan.value = await api.plan(store.checklistId)
    }
    store.step = 'confirm'
  })
}

async function retrySubmitWithException() {
  await doSubmit()
}

function goConfirm() {
  void doSubmit()
}

async function doApprove() {
  await runAction(async () => {
    const res = await api.approve(store.checklistId)
    store.approveResult = res as unknown as Record<string, unknown>
    store.step = 'done'
  })
}

function goHome() {
  uni.reLaunch({ url: '/pages/dashboard/index' })
}
</script>

<template>
  <view class="page">
    <view class="step-bar">
      <text class="step-type">{{ store.schema?.title || store.checklistType }}</text>
      <text v-if="codeFromNav && store.code" class="step-code">{{ store.name || store.code }} ({{ store.code }})</text>
      <text class="step-label">{{ stepTitle }}</text>
    </view>

    <!-- 步骤 1：表单 -->
    <view v-if="store.step === 'form' && store.schema" class="step-body">
      <view v-if="!codeFromNav && store.checklistType === 'buy'" class="code-row">
        <text class="label">标的代码</text>
        <textarea v-model="store.code" class="h5-code-input" rows="1" placeholder="如 600519" />
        <textarea v-model="store.name" class="h5-code-input" rows="1" placeholder="标的名称（可选）" />
      </view>
      <view v-if="store.lotReviewContext" class="review-context-box">
        <text class="ctx-title">对照信息（只读）</text>
        <text v-if="store.lotReviewContext.open_summary" class="ctx-line">
          建仓摘要：{{ store.lotReviewContext.open_summary }}
        </text>
        <template v-if="store.lotReviewContext.sell_context">
          <text v-if="store.lotReviewContext.sell_context.lesson" class="ctx-line">
            卖出教训：{{ store.lotReviewContext.sell_context.lesson }}
          </text>
          <text v-if="store.lotReviewContext.sell_context.thesis_result" class="ctx-line">
            逻辑验证：{{ store.lotReviewContext.sell_context.thesis_result }}
          </text>
        </template>
        <text class="ctx-meta">
          lot {{ store.lotReviewContext.lot.lot_id }} · 持有 {{ store.lotReviewContext.lot.holding_days }} 天
        </text>
      </view>
      <view v-if="store.checklistType === 'sell' && maxSellShares != null" class="sell-cap-hint">
        当前可卖股数：{{ maxSellShares }}（须 ≤ 此值）
      </view>
      <DynamicForm
        ref="formRef"
        :schema="store.schema"
        :model-value="store.values"
        :stock-code="store.code"
        @update:model-value="onFormUpdate"
      />
      <view v-if="submitError && store.step === 'form'" class="submit-error">{{ submitError }}</view>
      <button type="button" class="btn primary" :disabled="busy" @click.stop="nextFromForm">
        {{ busy ? '检查中…' : '下一步：提交并检查风险' }}
      </button>
    </view>

    <!-- 步骤 2：M7 -->
    <view v-else-if="store.step === 'risk'" class="step-body">
      <view v-if="riskWarnings.length" class="risk-box warn">
        <text class="risk-title">⚠️ 风险提示 ({{ riskWarnings.length }})</text>
        <text v-for="(w, i) in riskWarnings" :key="i" class="risk-item">{{ formatRiskItem(w) }}</text>
      </view>
      <view v-if="riskBlocks.length" class="risk-box block">
        <text class="risk-title">🛑 硬拦截 ({{ riskBlocks.length }})</text>
        <text v-for="(b, i) in riskBlocks" :key="i" class="risk-item">{{ formatRiskItem(b) }}</text>
      </view>
      <view v-if="!riskWarnings.length && !riskBlocks.length" class="risk-box ok">
        ✅ 风险检查通过，可以确认执行
      </view>

      <view v-if="store.approveBlocked" class="risk-hint-box">
        已触发硬拦截。你可以<strong>返回修改</strong>仓位等信息，或在下方填写例外说明后继续。
      </view>
      <view v-else-if="riskWarnings.length" class="risk-hint-box">
        存在风险提示，确认已知悉后可继续；也可返回修改表单。
      </view>

      <view v-if="sellPlan" class="plan-box">
        <text class="plan-title">FIFO 卖出计划预览</text>
        <text class="plan-meta">卖出 {{ sellPlan.sell_shares }} 股 @ {{ sellPlan.execution_price }}</text>
        <text v-for="(it, i) in (sellPlan.lot_allocation_plan as unknown[] || [])" :key="i" class="plan-item">
          lot {{ (it as any).lot_id }} → {{ (it as any).allocated_shares }} 股
        </text>
      </view>

      <view v-if="showEmotionCheck" class="exception-box emotion-box">
        <text class="label">情绪自检说明（必填）</text>
        <text class="field-tip">
          你选择了「{{ emotionTagLabel }}」情绪标签，须说明当前情绪状态与决策理由，确认非冲动交易。
        </text>
        <text class="char-count" :class="{ ok: emotionSelfCheckCount >= 10 }">
          已输入 {{ emotionSelfCheckCount }} 字（建议 ≥10 字）
        </text>
        <textarea
          v-model="store.emotionSelfCheck"
          class="exception-textarea"
          rows="4"
          maxlength="500"
          placeholder="例如：虽感到贪婪，但本次买入基于…，并已设置仓位上限与止损约束。"
        />
      </view>

      <view v-if="store.approveBlocked" class="exception-box">
        <text class="label">风险例外说明（必填，须 ≥80 字）</text>
        <text class="char-count" :class="{ ok: exceptionCharCount >= 80 }">
          已输入 {{ exceptionCharCount }} / 80 字
        </text>
        <textarea
          v-model="store.exceptionReason"
          class="exception-textarea"
          rows="8"
          maxlength="2000"
          placeholder="说明为何在硬拦截下仍要继续、补偿措施与后续约束…"
        />
        <text v-if="relatedLibIds.length === 0" class="inline-warn">
          还须关联 L1 素材：当前未选择，请点击下方「返回修改」在表单中勾选素材。
        </text>
        <text v-else class="inline-ok">已关联 {{ relatedLibIds.length }} 条 L1 素材</text>
      </view>

      <view v-if="submitError" class="submit-error">{{ submitError }}</view>

      <view class="action-row">
        <button type="button" class="btn secondary" :disabled="busy" @click="goBackToForm">返回修改</button>
        <button
          v-if="store.approveBlocked"
          type="button"
          class="btn warn"
          :disabled="busy"
          @click="retrySubmitWithException"
        >
          {{ busy ? '提交中…' : '提交例外并继续' }}
        </button>
        <button v-else type="button" class="btn primary" :disabled="busy" @click="goConfirm">
          {{ busy ? '提交中…' : '确认并继续' }}
        </button>
      </view>
    </view>

    <!-- 步骤 3：确认 -->
    <view v-else-if="store.step === 'confirm'" class="step-body">
      <view class="confirm-box">
        <text class="confirm-title">确认落库</text>
        <text class="confirm-desc">
          点击后将正式写入账本（journal / lot / portfolio），此操作不可通过「作废」撤销。
        </text>
      </view>
      <view v-if="submitError" class="submit-error">{{ submitError }}</view>
      <view class="action-row">
        <button type="button" class="btn secondary" :disabled="busy" @click="goBackToRisk">返回上一步</button>
        <button type="button" class="btn primary" :disabled="busy" @click="doApprove">
          {{ busy ? '执行中…' : '确认执行' }}
        </button>
      </view>
    </view>

    <!-- 步骤 4：完成 -->
    <view v-else-if="store.step === 'done'" class="step-body">
      <view class="done-box">
        <text class="done-icon">✅</text>
        <text class="done-title">决策已落库</text>
        <text class="done-desc">系统已自动处理所有 ID，你无需记录任何编号。</text>
        <text v-if="store.approveResult?.journal_id" class="done-meta">
          已创建交易记录（内部已关联）
        </text>
      </view>
      <button class="btn primary" @click="goHome">返回首页</button>
    </view>
  </view>
</template>

<style scoped>
.page {
  padding: 24rpx 32rpx 48rpx;
  min-height: 100vh;
}
.sell-cap-hint {
  font-size: 24rpx;
  color: #b45309;
  background: #fffbeb;
  border: 1rpx solid #fde68a;
  border-radius: 12rpx;
  padding: 16rpx 20rpx;
  margin-bottom: 16rpx;
}
.review-context-box {
  background: #f8fafc;
  border: 1rpx solid #e2e8f0;
  border-radius: 12rpx;
  padding: 20rpx 24rpx;
  margin-bottom: 24rpx;
}
.ctx-title { font-size: 26rpx; font-weight: 600; color: #334155; display: block; margin-bottom: 8rpx; }
.ctx-line { font-size: 24rpx; color: #475569; display: block; margin-bottom: 6rpx; line-height: 1.5; }
.ctx-meta { font-size: 22rpx; color: #94a3b8; display: block; margin-top: 8rpx; }
.step-bar {
  margin-bottom: 24rpx;
}
.step-type {
  font-size: 36rpx;
  font-weight: 700;
  display: block;
}
.step-code {
  font-size: 28rpx;
  color: #2563eb;
  margin-top: 8rpx;
  display: block;
}
.step-label {
  font-size: 26rpx;
  color: #64748b;
  margin-top: 4rpx;
  display: block;
}
.code-row {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  margin-bottom: 24rpx;
}
.label {
  font-size: 28rpx;
  font-weight: 500;
  display: block;
  margin-bottom: 12rpx;
}
.h5-code-input {
  width: 100%;
  box-sizing: border-box;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 12px;
  margin-bottom: 12rpx;
  font-size: 15px;
  color: #334155;
  background: #f8fafc;
  min-height: 42px;
  max-height: 42px;
  resize: none;
  overflow: hidden;
  pointer-events: auto;
}
.btn {
  margin-top: 32rpx;
  border-radius: 16rpx;
  font-size: 30rpx;
}
.btn.primary {
  background: #2563eb;
  color: #fff;
}
.btn.warn {
  background: #f59e0b;
  color: #fff;
}
.risk-box {
  border-radius: 16rpx;
  padding: 24rpx;
  margin-bottom: 20rpx;
  font-size: 26rpx;
}
.risk-box.warn {
  background: #fffbeb;
  border: 1rpx solid #fde68a;
}
.risk-box.block {
  background: #fef2f2;
  border: 1rpx solid #fecaca;
}
.risk-box.ok {
  background: #ecfdf5;
  border: 1rpx solid #a7f3d0;
}
.risk-title {
  font-weight: 600;
  display: block;
  margin-bottom: 12rpx;
}
.risk-item {
  display: block;
  margin-bottom: 8rpx;
  font-size: 26rpx;
  line-height: 1.5;
  word-break: break-word;
}
.risk-hint-box {
  background: #f8fafc;
  border: 1rpx solid #e2e8f0;
  border-radius: 12rpx;
  padding: 20rpx;
  margin-bottom: 20rpx;
  font-size: 26rpx;
  color: #475569;
  line-height: 1.5;
}
.plan-box {
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
  margin-bottom: 20rpx;
}
.plan-title {
  font-weight: 600;
  display: block;
}
.plan-meta,
.plan-item {
  font-size: 26rpx;
  color: #64748b;
  display: block;
  margin-top: 8rpx;
}
.exception-box {
  margin-top: 24rpx;
  background: #fff;
  border-radius: 16rpx;
  padding: 24rpx;
}
.emotion-box {
  border: 1rpx solid #fde68a;
  background: #fffbeb;
}
.field-tip {
  display: block;
  font-size: 24rpx;
  color: #64748b;
  margin-bottom: 12rpx;
  line-height: 1.4;
}
.char-count {
  display: block;
  font-size: 24rpx;
  color: #b45309;
  margin-bottom: 12rpx;
}
.char-count.ok {
  color: #16a34a;
}
.exception-textarea {
  width: 100%;
  min-height: 200px;
  border: 1px solid #e2e8f0;
  border-radius: 12rpx;
  padding: 12px;
  font-size: 15px;
  line-height: 1.5;
  box-sizing: border-box;
  resize: vertical;
  overflow: auto;
  color: #334155;
  background: #f8fafc;
}
.inline-warn {
  display: block;
  margin-top: 12rpx;
  font-size: 24rpx;
  color: #b45309;
  line-height: 1.4;
}
.inline-ok {
  display: block;
  margin-top: 12rpx;
  font-size: 24rpx;
  color: #16a34a;
}
.submit-error {
  margin-top: 16rpx;
  padding: 16rpx;
  background: #fef2f2;
  border: 1rpx solid #fecaca;
  border-radius: 12rpx;
  font-size: 24rpx;
  color: #b91c1c;
  line-height: 1.4;
  white-space: pre-wrap;
}
.action-row {
  display: flex;
  gap: 16rpx;
  margin-top: 32rpx;
  flex-wrap: wrap;
}
.action-row .btn {
  flex: 1;
  min-width: 140rpx;
  margin-top: 0;
}
.btn.secondary {
  background: #f1f5f9;
  color: #334155;
  border: 1rpx solid #e2e8f0;
}
.btn:disabled {
  opacity: 0.6;
}
.confirm-box,
.done-box {
  background: #fff;
  border-radius: 16rpx;
  padding: 40rpx 32rpx;
  text-align: center;
}
.confirm-title,
.done-title {
  font-size: 34rpx;
  font-weight: 600;
  display: block;
}
.confirm-desc,
.done-desc,
.done-meta {
  font-size: 26rpx;
  color: #64748b;
  margin-top: 16rpx;
  line-height: 1.5;
  display: block;
}
.done-icon {
  font-size: 64rpx;
  display: block;
}
</style>
