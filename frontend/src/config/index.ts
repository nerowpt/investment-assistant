/** API 基础地址：H5 开发走 vite proxy；小程序/APP 改为实际服务器地址 */
const isH5 = typeof window !== 'undefined'

export const config = {
  /** 空字符串表示同源 / vite proxy */
  apiBaseURL: isH5 ? '' : 'http://127.0.0.1:8787',
  accountId: 'default',
}
