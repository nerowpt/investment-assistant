/** 统一响应信封（对齐 04 §19.5：{code, data, success}） */
export interface Envelope<T = unknown> {
  code: number
  success: boolean
  message?: string
  data?: T
}

export class APIRequestError extends Error {
  code: number
  constructor(code: number, message: string) {
    super(message)
    this.code = code
  }
}

export interface FieldOption {
  value: string
  label: string
}

export interface FieldSchema {
  key: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'enum' | 'multi_enum' | 'library_multi' | 'bool' | 'array' | 'date'
  required?: boolean
  default?: unknown
  tip?: string
  group: string
  options?: FieldOption[]
  rows?: number
}

export interface FormSchema {
  checklist_type: string
  title: string
  description: string
  groups: string[]
  fields: FieldSchema[]
}

export interface SchemaResponse {
  schema: FormSchema
  default_values: Record<string, unknown>
  payload_template: Record<string, unknown>
}

export interface DraftResult {
  id: string
  status: string
}

export interface SubmitResult {
  id: string
  status: string
  approve_blocked: boolean
  hard_block_count: number
  warning_count: number
}

export interface ApproveResult {
  checklist_id: string
  journal_id?: string
  lot_id?: string
  snapshot_id?: string
  inspection_id?: string
  yaml_synced?: boolean
  sync_repair_id?: string
}

export interface DoctorIssue {
  code?: string
  subject?: string
  title: string
  detail?: string
  hint?: string
}

export interface RepairField {
  key: string
  label: string
  type: 'checkbox' | 'text' | 'number' | 'hidden'
  value?: string
  default?: boolean
  tip?: string
}

export interface PoolItem {
  code: string
  name: string
  zone: string
  ref_id?: string
  summary?: string
  library_count: number
  review_date?: string
  position_pct?: string
  position_type?: string
  state?: string
  entry_date?: string
  closed_at?: string
  sell_count?: number
  pool_tags?: string[]
  actions: string[]
}

export interface PoolZone {
  id: string
  title: string
  description: string
  count: number
  items: PoolItem[]
}

export interface PoolResponse {
  zones: PoolZone[]
  updated_at: string
}

export interface ResearchPackDef {
  id: string
  title: string
  description: string
}

export interface ResearchDossier {
  code: string
  name: string
  zones: string[]
  library_items: { id: string; title: string; tier: string; summary?: string }[]
  packs: ResearchPackDef[]
  worker_ok: boolean
  worker_message?: string
}

export interface ResearchPackResult {
  pack_id: string
  code: string
  title: string
  summary: string
  body: string
  source: string
  tier: string
  captured_at: string
  suggest_tier: string
}

export interface ClosedLotSummary {
  lot_id: string
  action_type: string
  position_type: string
  open_at: string
  close_at: string
  initial_pct: string
  cost_basis: string
  realized_return_pct?: string
  realized_pnl_amount?: string
  holding_days: number
  sell_journal_id?: string
  open_journal_id: string
  reviewed: boolean
  review_report_id?: string
}

export interface SellReviewContext {
  journal_id: string
  lesson?: string
  thesis_result?: string
  thesis_result_explanation?: string
  emotion_tag?: string
  sell_reason?: string
  sell_reason_detail?: string
}

export interface LotReviewPrefill {
  review_type: string
  target_lot_id: string
  target_code: string
  period_start: string
  period_end: string
  'attribution.result_category'?: string
  confirmed_patterns: string[]
  notes?: string
}

export interface ReviewWorkbenchResponse {
  code: string
  name: string
  closed_lots: ClosedLotSummary[]
}

export interface LotReviewContextResponse {
  code: string
  name: string
  lot: ClosedLotSummary
  open_summary?: string
  sell_context?: SellReviewContext
  prefill: LotReviewPrefill
}

export interface BuyContextResponse {
  code: string
  name: string
  from: string
  watch_id?: string
  prefill: {
    source_entry: string
    watchlist_origin_id?: string
    related_library_ids: string[]
    buy_reason_summary?: string
    investment_thesis?: string
  }
  library_items: { id: string; title: string; tier: string; summary?: string }[]
}

export interface RepairAction {
  id: string
  code: string
  subject?: string
  title: string
  detail?: string
  hint?: string
  action_type: string
  fields: RepairField[]
}

export interface PortfolioPosition {
  code: string
  name: string
  state: string
  position_pct: number | string
  position_type?: string
  cost_basis?: string
  shares?: string | null
  entry_date?: string
  stop_loss?: string
  journal_ids?: string[]
}

export interface JournalRow {
  id: string
  action_type: string
  code: string
  name?: string
  checklist_submission_id?: string
  lot_id?: string
  summary: string
  created_at: string
}

export interface ChecklistListItem {
  id: string
  checklist_type: string
  code: string
  name: string
  status: string
  summary?: string
  created_at: string
  approved_at?: string
  submitted_at?: string
}
