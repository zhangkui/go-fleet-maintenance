import axios, { type AxiosInstance, type AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { useAuthStore } from '@/stores/auth'

// 后端统一错误响应。
interface ApiErrorBody {
  code: string
  message: string
  details?: Record<string, unknown>
}

const client: AxiosInstance = axios.create({
  baseURL: '/',
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
})

// 请求拦截器：附加 Bearer token。
client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const auth = useAuthStore()
  if (auth.accessToken) {
    config.headers.Authorization = `Bearer ${auth.accessToken}`
  }
  return config
})

// 标记是否正在刷新令牌，避免并发刷新。
let refreshing = false
let pendingQueue: Array<{ resolve: (t: string) => void; reject: (e: unknown) => void }> = []

function processQueue(token: string | null, err: unknown | null) {
  pendingQueue.forEach((p) => {
    if (token) p.resolve(token)
    else p.reject(err)
  })
  pendingQueue = []
}

// 从错误响应中提取业务消息。
export function extractErrorMessage(err: unknown): string {
  const e = err as AxiosError<ApiErrorBody>
  if (e?.response?.data?.message) return e.response.data.message
  if (e?.message) return e.message
  return '发生未知错误'
}

export function extractErrorCode(err: unknown): string {
  const e = err as AxiosError<ApiErrorBody>
  return e?.response?.data?.code ?? 'unknown'
}

// 响应拦截器：401 自动尝试刷新一次，失败则跳转登录。
client.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiErrorBody>) => {
    const original = error.config as InternalAxiosRequestConfig & { _retried?: boolean; _refresh?: boolean }
    const status = error.response?.status

    // 仅对 401 且非刷新请求、且未重试过的请求尝试刷新。
    if (status === 401 && original && !original._retried && !original._refresh) {
      original._retried = true
      const auth = useAuthStore()

      if (!auth.refreshToken) {
        auth.clear()
        if (typeof window !== 'undefined') window.location.href = '/login'
        return Promise.reject(error)
      }

      if (refreshing) {
        // 排队等待刷新完成。
        return new Promise((resolve, reject) => {
          pendingQueue.push({
            resolve: (t: string) => {
              original.headers.Authorization = `Bearer ${t}`
              resolve(client(original))
            },
            reject: (e: unknown) => reject(e),
          })
        })
      }

      refreshing = true
      try {
        const newToken = await auth.refresh()
        processQueue(newToken, null)
        original.headers.Authorization = `Bearer ${newToken}`
        return client(original)
      } catch (e) {
        processQueue(null, e)
        auth.clear()
        if (typeof window !== 'undefined') window.location.href = '/login'
        return Promise.reject(error)
      } finally {
        refreshing = false
      }
    }

    return Promise.reject(error)
  },
)

export default client
