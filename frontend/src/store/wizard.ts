import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { FormSchema, LotReviewContextResponse } from '@/api/types'

export type WizardStep = 'form' | 'risk' | 'confirm' | 'done'

export const useWizardStore = defineStore('wizard', () => {
  const checklistType = ref('')
  const code = ref('')
  const name = ref('')
  const schema = ref<FormSchema | null>(null)
  const values = ref<Record<string, unknown>>({})
  const checklistId = ref('')
  const step = ref<WizardStep>('form')
  const riskResult = ref<Record<string, unknown> | null>(null)
  const approveBlocked = ref(false)
  const approveResult = ref<Record<string, unknown> | null>(null)
  const exceptionReason = ref('')
  const emotionCheckNeeded = ref(false)
  const emotionTag = ref('')
  const emotionSelfCheck = ref('')
  const linkedJournalId = ref('')
  /** 单笔 lot 复盘上下文（只读对照，H8.3） */
  const lotReviewContext = ref<LotReviewContextResponse | null>(null)

  function reset() {
    checklistType.value = ''
    code.value = ''
    name.value = ''
    schema.value = null
    values.value = {}
    checklistId.value = ''
    step.value = 'form'
    riskResult.value = null
    approveBlocked.value = false
    approveResult.value = null
    exceptionReason.value = ''
    emotionCheckNeeded.value = false
    emotionTag.value = ''
    emotionSelfCheck.value = ''
    linkedJournalId.value = ''
    lotReviewContext.value = null
  }

  return {
    checklistType,
    code,
    name,
    schema,
    values,
    checklistId,
    step,
    riskResult,
    approveBlocked,
    approveResult,
    exceptionReason,
    emotionCheckNeeded,
    emotionTag,
    emotionSelfCheck,
    linkedJournalId,
    lotReviewContext,
    reset,
  }
})
