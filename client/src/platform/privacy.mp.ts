import type { PrivacyPlatform } from './types'
export const privacy: PrivacyPlatform = {
  ensureAuthorized: () => new Promise((resolve) => {
    if (typeof wx === 'undefined' || !wx.requirePrivacyAuthorize) return resolve(true)
    wx.requirePrivacyAuthorize({ success: () => resolve(true), fail: () => resolve(false) })
  }),
}
