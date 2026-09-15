import type { AdsPlatform } from './types'
import { ApiError } from '@/utils/errors'
import { ads as simulated } from './ads.sim'

let instance: any = null
let unitId = ''
let loaded = false

function ensure() {
  if (instance || !unitId) return instance
  instance = (uni as any).createRewardedVideoAd({ adUnitId: unitId })
  instance.onLoad(() => { loaded = true })
  instance.onError(() => { loaded = false })
  return instance
}

export const ads: AdsPlatform = {
  get simulated() { return !unitId },
  configure(id) { unitId = id || ''; instance = null; loaded = false; ensure() },
  preload() { if (!unitId) return; const ad = ensure(); ad.load().catch(() => {}) },
  show() {
    if (!unitId) return simulated.show()
    const ad = ensure()
    return new Promise((resolve, reject) => {
      const onClose = (res: any) => {
        ad.offClose(onClose)
        resolve({ ended: !!(res && res.isEnded) })
      }
      ad.onClose(onClose)
      const attempt = () => ad.show().catch(() => ad.load().then(() => ad.show()))
      attempt().catch(() => { ad.offClose(onClose); loaded = false; reject(new ApiError('AD_LOAD_FAILED', '', 0)) })
    })
  },
}
