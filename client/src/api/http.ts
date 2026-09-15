/* uni.request wrapper: envelope decoding, auth header, silent re-login on 401 (docs/FRONTEND_ARCHITECTURE.md §7). */
import { env } from '@/config'
import { ApiError } from '@/utils/errors'
import { uuid } from '@/utils/uuid'

let token = ''
let reloginHandler: (() => Promise<string>) | null = null

export function setToken(t: string) { token = t }
export function getToken() { return token }
export function setReloginHandler(fn: () => Promise<string>) { reloginHandler = fn }

export function platformName(): string {
  let p = 'h5'
  // #ifdef MP-WEIXIN
  p = 'mp-weixin'
  // #endif
  // #ifdef APP-PLUS
  p = 'app'
  // #endif
  return p
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  data?: any
  headers?: Record<string, string>
  timeout?: number
  _retried?: boolean
}

/** Auth and tracing headers. Never sets Content-Type: uploads must keep the
 * multipart boundary the platform generates. */
function baseHeaders(extra?: Record<string, string>) {
  return {
    Authorization: token ? `Bearer ${token}` : '',
    'X-Request-Id': uuid(),
    'X-Client-Version': env.clientVersion,
    'X-Platform': platformName(),
    ...(extra || {}),
  }
}

function headers(extra?: Record<string, string>) {
  return { 'Content-Type': 'application/json', ...baseHeaders(extra) }
}

async function decode<T>(status: number, body: any, retry: () => Promise<T>, retried?: boolean): Promise<T> {
  if (status >= 200 && status < 300 && body && body.code === 'OK') return body.data as T
  if (status === 401 && !retried && reloginHandler) {
    token = await reloginHandler()
    return retry()
  }
  throw new ApiError(body?.code || (status >= 500 ? 'INTERNAL' : 'BAD_REQUEST'), body?.message || '', status, body?.data)
}

export function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    uni.request({
      url: env.apiBase + path,
      method: opts.method || 'GET',
      data: opts.data,
      timeout: opts.timeout ?? 15000,
      header: headers(opts.headers),
      success: (res) => {
        decode<T>(res.statusCode, res.data, () => request<T>(path, { ...opts, _retried: true }), opts._retried)
          .then(resolve, reject)
      },
      fail: () => reject(new ApiError('NETWORK', '', 0)),
    })
  })
}

export function upload<T>(path: string, filePath: string, formData: Record<string, any> = {}, retried = false): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    uni.uploadFile({
      url: env.apiBase + path,
      filePath,
      name: 'file',
      formData,
      timeout: 60000,
      header: baseHeaders(),
      success: (res) => {
        let body: any = null
        try { body = JSON.parse(res.data) } catch { body = null }
        decode<T>(res.statusCode, body, () => upload<T>(path, filePath, formData, true), retried).then(resolve, reject)
      },
      fail: () => reject(new ApiError('NETWORK', '', 0)),
    })
  })
}
