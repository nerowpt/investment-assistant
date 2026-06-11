import { request } from './request'
import type {
  ApproveResult,
  ChecklistListItem,
  DraftResult,
  FormSchema,
  JournalRow,
  PortfolioPosition,
  SchemaResponse,
  SubmitResult,
} from './types'

export const api = {
  health: () => request<{ status: string }>({ url: '/api/health' }),

  getPortfolio: (code?: string) =>
    request<{ positions: PortfolioPosition[] }>({
      url: code ? `/api/portfolio?code=${code}` : '/api/portfolio',
    }),

  getWatchlist: () => request<{ items: unknown[] }>({ url: '/api/watchlist' }),

  getPool: (zone?: string) =>
    request<import('./types').PoolResponse>({
      url: zone ? `/api/pool?zone=${encodeURIComponent(zone)}` : '/api/pool',
    }),

  getResearch: (code: string) =>
    request<import('./types').ResearchDossier>({
      url: `/api/research/${encodeURIComponent(code)}`,
    }),

  fetchResearchPack: (code: string, pack: string) =>
    request<import('./types').ResearchPackResult>({
      url: `/api/research/${encodeURIComponent(code)}/fetch`,
      method: 'POST',
      data: { pack },
    }),

  saveResearchToLibrary: (code: string, body: { title: string; text: string; tier?: string }) =>
    request<{ library_id: string; title: string; code: string }>({
      url: `/api/research/${encodeURIComponent(code)}/library`,
      method: 'POST',
      data: body,
    }),

  getReviewWorkbench: (code: string) =>
    request<import('./types').ReviewWorkbenchResponse>({
      url: `/api/review/workbench?code=${encodeURIComponent(code)}`,
    }),

  getLotReviewContext: (params: { code: string; lot_id: string }) =>
    request<import('./types').LotReviewContextResponse>({
      url: `/api/review/lot-context?code=${encodeURIComponent(params.code)}&lot_id=${encodeURIComponent(params.lot_id)}`,
    }),

  getPoolBuyContext: (params: { code: string; from?: string; watch_id?: string }) => {
    const q = new URLSearchParams()
    q.set('code', params.code)
    if (params.from) q.set('from', params.from)
    if (params.watch_id) q.set('watch_id', params.watch_id)
    return request<import('./types').BuyContextResponse>({
      url: `/api/pool/buy-context?${q.toString()}`,
    })
  },

  getSchema: (type: string) =>
    request<SchemaResponse>({ url: `/api/checklist/schema?type=${type}` }),

  createDraft: (body: {
    checklist_type: string
    code?: string
    name?: string
    values?: Record<string, unknown>
  }) => request<DraftResult>({ url: '/api/checklist', method: 'POST', data: body }),

  updateDraft: (
    id: string,
    body: { code?: string; name?: string; values?: Record<string, unknown> },
  ) => request<DraftResult>({ url: `/api/checklist/${id}`, method: 'PUT', data: body }),

  getChecklist: (id: string) =>
    request<{
      checklist: ChecklistListItem
      payload: Record<string, unknown>
      risk_result?: Record<string, unknown>
      exception?: Record<string, unknown>
      emotion_self_check?: string
    }>({ url: `/api/checklists/${id}` }),

  preview: (id: string) =>
    request<{
      approve_blocked: boolean
      hard_block_count: number
      warning_count: number
      risk_result: Record<string, unknown>
      exception_required: boolean
      emotion_check_needed?: boolean
      emotion_tag?: string
    }>({ url: `/api/checklist/${id}/preview`, method: 'POST' }),

  submit: (id: string, body?: { emotion_self_check?: string; exception?: Record<string, unknown> }) =>
    request<{ submit: SubmitResult; risk_result?: Record<string, unknown> }>({
      url: `/api/checklist/${id}/submit`,
      method: 'POST',
      data: body || {},
    }),

  plan: (id: string) =>
    request<Record<string, unknown>>({ url: `/api/checklist/${id}/plan`, method: 'POST' }),

  approve: (id: string) =>
    request<ApproveResult>({ url: `/api/checklist/${id}/approve`, method: 'POST' }),

  reject: (id: string, reason: string) =>
    request({ url: `/api/checklist/${id}/reject`, method: 'POST', data: { reason } }),

  getJournal: (id: string) =>
    request<{
      journal: JournalRow
      payload?: Record<string, unknown>
      snapshot_summary?: Record<string, unknown>
    }>({ url: `/api/journals/${id}` }),

  getJournals: (params?: { code?: string; action_type?: string; limit?: number }) => {
    const q = new URLSearchParams()
    if (params?.code) q.set('code', params.code)
    if (params?.action_type) q.set('action_type', params.action_type)
    if (params?.limit) q.set('limit', String(params.limit))
    const qs = q.toString()
    return request<{ journals: JournalRow[] }>({ url: `/api/journals${qs ? `?${qs}` : ''}` })
  },

  getChecklists: (params?: { status?: string; type?: string; code?: string; limit?: number }) => {
    const q = new URLSearchParams()
    if (params?.status) q.set('status', params.status)
    if (params?.type) q.set('type', params.type)
    if (params?.code) q.set('code', params.code)
    if (params?.limit) q.set('limit', String(params.limit))
    const qs = q.toString()
    return request<{ checklists: ChecklistListItem[] }>({ url: `/api/checklists${qs ? `?${qs}` : ''}` })
  },

  doctor: (scope = 'all') =>
    request<{
      ok: boolean
      issues: import('./types').DoctorIssue[]
      repair_actions?: import('./types').RepairAction[]
    }>({
      url: `/api/doctor?scope=${scope}`,
    }),

  doctorRepair: (actions: { id: string; enabled: boolean; values?: Record<string, string> }[]) =>
    request<{
      ok: boolean
      message?: string
      issues: import('./types').DoctorIssue[]
      repair_actions?: import('./types').RepairAction[]
    }>({ url: '/api/doctor/repair', method: 'POST', data: { actions } }),

  getLibrary: (stock: string) =>
    request<{ items: LibraryItem[] }>({ url: `/api/library?stock=${encodeURIComponent(stock)}` }),

  getLibraryItem: (id: string) =>
    request<{ item: LibraryItem }>({ url: `/api/library/${encodeURIComponent(id)}` }),

  quickAddLibrary: (body: { title: string; text: string; stock: string; tier?: string }) =>
    request<{ library_id: string; title: string }>({
      url: '/api/library/quick-add',
      method: 'POST',
      data: body,
    }),

  getRiskRules: () =>
    request<{
      position_limits: {
        single_stock: { warning_pct: number; hard_block_pct: number }
        single_sector: { warning_pct: number; hard_block_pct: number }
        total_equity: { warning_pct: number; hard_block_pct: number }
        single_thesis: { warning_pct: number; hard_block_pct: number }
      }
    }>({ url: '/api/risk/rules' }),

  riskCheck: (body: {
    scenario: string
    code: string
    planned_position_pct_after: number
  }) =>
    request<{
      warnings: unknown[]
      hard_blocks: unknown[]
      approve_blocked: boolean
    }>({ url: '/api/risk/check', method: 'POST', data: body }),
}

export interface LibraryItem {
  id: string
  title: string
  tier: string
  source: string
  summary?: string
  created_at: string
}

export type { FormSchema, FieldSchema } from './types'
