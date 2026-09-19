import type { PrivacyPlatform } from './types'
export const privacy: PrivacyPlatform = {
  ensureAuthorized: () => new Promise((resolve, reject) => {
    if (typeof wx === 'undefined' || !wx.requirePrivacyAuthorize) return resolve()
    wx.requirePrivacyAuthorize({ success: () => resolve(), fail: reject })
  }),
}
