import type { SharePlatform } from './types'
export const share: SharePlatform = {
  nativeChatShare: true,
  timeline: true,
  async share() { /* WeChat shares through <button open-type="share"> and page hooks */ },
  copyLink: (path) => new Promise((resolve) => uni.setClipboardData({ data: path, success: () => resolve(), fail: () => resolve() })),
}
