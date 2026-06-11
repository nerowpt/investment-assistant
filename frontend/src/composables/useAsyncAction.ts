import { nextTick, type Ref } from 'vue'

/** 从 API / 网络异常提取用户可读文案 */
export function extractErrorMessage(e: unknown): string {
  if (e && typeof e === 'object' && 'message' in e) {
    const msg = String((e as Error).message || '').trim()
    if (msg) return msg
  }
  return '操作失败'
}

type AsyncActionOptions = {
  /** 失败时是否 toast，默认 true */
  toast?: boolean
}

/**
 * 向导/表单通用异步包装：保证 busy 必定解除，错误写入 errorRef 并可 toast。
 */
export function useAsyncAction(busy: Ref<boolean>, errorRef: Ref<string>) {
  async function run<T>(fn: () => Promise<T>, opts?: AsyncActionOptions): Promise<T | undefined> {
    if (busy.value) return undefined
    busy.value = true
    errorRef.value = ''
    try {
      return await fn()
    } catch (e: unknown) {
      const msg = extractErrorMessage(e)
      errorRef.value = msg
      if (opts?.toast !== false) {
        const title = msg.length > 48 ? `${msg.slice(0, 48)}…` : msg
        uni.showToast({ title, icon: 'none', duration: 3500 })
      }
      return undefined
    } finally {
      busy.value = false
      await nextTick()
    }
  }

  return { run }
}
