import { config } from '@/config'
import type { APIRequestError, Envelope } from './types'

export { APIRequestError } from './types'

export async function request<T>(options: {
  url: string
  method?: 'GET' | 'POST' | 'PUT'
  data?: unknown
}): Promise<T> {
  const url = `${config.apiBaseURL}${options.url}`
  return new Promise((resolve, reject) => {
    uni.request({
      url,
      method: options.method || 'GET',
      data: options.data,
      header: {
        'Content-Type': 'application/json',
        'X-Account-Id': config.accountId,
      },
      success: (res) => {
        const body = res.data as Envelope<T>
        const status = res.statusCode || 0
        if (!body || typeof body.success !== 'boolean') {
          reject(new Error(`响应格式异常: HTTP ${status}`))
          return
        }
        if (status >= 400 || !body.success) {
          reject(new APIRequestError(body.code, body.message || `请求失败 (HTTP ${status})`))
          return
        }
        resolve(body.data as T)
      },
      fail: (err) => reject(new Error(err.errMsg || '网络请求失败')),
    })
  })
}
