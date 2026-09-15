import type { AuthPlatform } from './types'
export const auth: AuthPlatform = {
  login: () => new Promise((resolve, reject) => {
    uni.login({ provider: 'weixin', success: (r: any) => resolve({ provider: 'wechat_app', code: r.code || r.authResult?.code }), fail: reject })
  }),
  checkSession: async () => true,
}
