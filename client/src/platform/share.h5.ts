import type { SharePlatform } from './types'
export const share: SharePlatform = {
  nativeChatShare: false,
  timeline: false,
  async share() {},
  copyLink: (path) => new Promise((resolve) => {
    const url = `${location.origin}${location.pathname}#${path}`
    uni.setClipboardData({ data: url, success: () => resolve(), fail: () => resolve() })
  }),
}
