import type { FieldSchema, FormSchema } from '@/api/types'
import { flattenPayload } from '@/utils/payload'

const ENUM_LABELS: Record<string, Record<string, string>> = {
  position_type: { core: '主仓', swing: '波段' },
  confidence: { high: '高', medium: '中', low: '低' },
  emotion_tag: { calm: '冷静', fomo: 'FOMO', greedy: '贪婪', anxious: '焦虑' },
  classification: {
    thesis_intact: '逻辑仍成立',
    thesis_weakened: '逻辑减弱',
    thesis_broken: '逻辑破裂',
    wait_for_style_switch: '等待风格切换',
  },
  planned_action: { hold: '持有', add: '加仓', reduce: '减仓', sell: '卖出', watch: '观察' },
  expected_return_driver: {
    earnings_growth: '业绩增长',
    valuation_repair: '估值修复',
    roe_improvement: 'ROE 中枢改善',
    narrative_repricing: '叙事重估',
    cycle_reversal: '周期反转',
    capital_flow: '资金面改善',
    event_catalyst: '事件催化',
    other: '其他',
  },
  source_entry: {
    manual: '手动录入',
    from_watchlist: '观察池',
    passive_discovery: '被动发现',
    from_inspection: '巡检转入',
    from_candidate: '候选转正',
    crawl: '爬取',
    import_batch: '批量导入',
  },
  opportunity_cost_benchmark: {
    HS300: '沪深300',
    CSI_TECH: '科创50',
    sector_index: '行业指数',
    custom: '自定义',
  },
}

function labelForEnum(field: FieldSchema | undefined, value: string): string {
  if (field?.options) {
    const opt = field.options.find((o) => o.value === value)
    if (opt) return opt.label
  }
  const map = ENUM_LABELS[field?.key.split('.').pop() || field?.key || ''] || {}
  return map[value] || value
}

function formatScalar(field: FieldSchema | undefined, value: unknown): string {
  if (value === undefined || value === null || value === '') return '—'
  if (typeof value === 'boolean') return value ? '是' : '否'
  if (Array.isArray(value)) {
    if (value.length === 0) return '—'
    if (field?.type === 'multi_enum') {
      return value.map((v) => labelForEnum(field, String(v))).join('、')
    }
    return value.map((v) => String(v)).join('；')
  }
  if (field?.type === 'enum') return labelForEnum(field, String(value))
  if (typeof value === 'number') return String(value)
  return String(value)
}

export interface DetailItem {
  label: string
  value: string
  kind?: 'text' | 'libraries'
  libraryIds?: string[]
}

export interface DetailGroup {
  title: string
  items: DetailItem[]
}

function pushItem(items: DetailItem[], field: FieldSchema | undefined, key: string, val: unknown) {
  if (field?.type === 'library_multi' || key === 'related_library_ids') {
    const ids = Array.isArray(val) ? val.map(String).filter(Boolean) : []
    if (ids.length === 0) return
    items.push({ label: field?.label || '关联 L1 素材', value: '', kind: 'libraries', libraryIds: ids })
    return
  }
  items.push({ label: field?.label || key, value: formatScalar(field, val) })
}

/** 按 schema 分组格式化 payload 为详情展示项 */
export function buildDetailGroups(schema: FormSchema, payload: Record<string, unknown>): DetailGroup[] {
  const flat = flattenPayload(payload)
  const fieldMap = new Map(schema.fields.map((f) => [f.key, f]))
  const used = new Set<string>()

  const groups: DetailGroup[] = []
  for (const groupName of schema.groups) {
    const items: DetailItem[] = []
    for (const f of schema.fields) {
      if (f.group !== groupName) continue
      if (f.key === 'no_library_reason' && flat['related_library_ids']) {
        const libs = flat['related_library_ids']
        if (Array.isArray(libs) && libs.length > 0) continue
      }
      const val = flat[f.key]
      if (val === undefined || val === null || val === '') continue
      if (f.type === 'bool' && val === false) {
        // 布尔 false 仍有业务含义（如低可信度素材确认=否）
        items.push({ label: f.label, value: '否' })
        used.add(f.key)
        continue
      }
      pushItem(items, f, f.key, val)
      used.add(f.key)
    }
    if (items.length > 0) groups.push({ title: groupName, items })
  }

  const extra: DetailItem[] = []
  for (const [key, val] of Object.entries(flat)) {
    if (used.has(key) || val === undefined || val === null || val === '') continue
    const f = fieldMap.get(key)
    pushItem(extra, f, key, val)
  }
  if (extra.length > 0) groups.push({ title: '其他', items: extra })
  return groups
}

export function formatRiskMessages(risk: Record<string, unknown> | null | undefined): string[] {
  if (!risk) return []
  const msgs: string[] = []
  for (const key of ['hard_blocks', 'warnings'] as const) {
    const arr = risk[key]
    if (!Array.isArray(arr)) continue
    for (const item of arr) {
      if (item && typeof item === 'object' && 'message' in item) {
        msgs.push(String((item as Record<string, unknown>).message))
      }
    }
  }
  return msgs
}
