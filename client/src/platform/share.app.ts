import type { SharePlatform } from './types'
export const share: SharePlatform = {
  nativeChatShare: false,
  timeline: true,
  share: (p) => new Promise((resolve, reject) => {
    ;(uni as any).share({
      provider: 'weixin', scene: p.scene === 'timeline' ? 'WXSceneTimeline' : 'WXSceneSession', type: 5,
      title: p.title, imageUrl: p.imageUrl, miniProgram: { path: p.path },
      success: () => resolve(), fail: reject,
    })
  }),
  copyLink: (path) => new Promise((resolve) => uni.setClipboardData({ data: path, success: () => resolve(), fail: () => resolve() })),
}
