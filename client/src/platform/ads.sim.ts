import type { AdsPlatform } from './types'
import { sleep } from '@/utils/uuid'
/* Simulated rewarded video for dev/H5 and for mp-weixin when no ad unit is configured. */
export const ads: AdsPlatform = {
  simulated: true,
  configure() {},
  preload() {},
  async show() {
    const ended = await new Promise<boolean>((resolve) => {
      uni.showModal({
        title: '模拟激励视频',
        content: '开发环境未配置广告位。选择“看完”模拟完整观看，选择“退出”模拟中途退出。',
        confirmText: '看完',
        cancelText: '退出',
        success: (r) => resolve(!!r.confirm),
        fail: () => resolve(false),
      })
    })
    await sleep(1100)
    return { ended }
  },
}
