import type { AuthPlatform } from './types'
export const auth: AuthPlatform = {
  login: () => new Promise((resolve, reject) => {
    uni.login({ provider: 'weixin', success: (r) => resolve({ provider: 'wechat_mp', code: r.code }), fail: reject })
  }),
  checkSession: () => new Promise((resolve) => uni.checkSession({ success: () => resolve(true), fail: () => resolve(false) })),
}
