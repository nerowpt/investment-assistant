/** 从扁平或嵌套表单值读取数字（兼容 dot-path key 与嵌套 object） */
export function readFormNumber(values: Record<string, unknown>, key: string): number {
  const direct = values[key]
  if (direct !== undefined && direct !== null && direct !== '') {
    const n = typeof direct === 'number' ? direct : Number(direct)
    if (!Number.isNaN(n)) return n
  }
  const parts = key.split('.')
  let cur: unknown = values
  for (const p of parts) {
    if (!cur || typeof cur !== 'object' || Array.isArray(cur)) return NaN
    cur = (cur as Record<string, unknown>)[p]
  }
  return Number(cur)
}

/** 将嵌套 payload 展平为表单用的 dot-path key */
export function flattenPayload(
  obj: Record<string, unknown>,
  prefix = '',
): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      Object.assign(out, flattenPayload(v as Record<string, unknown>, key))
    } else {
      out[key] = v
    }
  }
  return out
}
