import type { SubscribePlatform } from './types'
export const subscribe: SubscribePlatform = {
  request: (templateId) => new Promise((resolve) => {
    if (!templateId) return resolve(false)
    ;(uni as any).requestSubscribeMessage({
      tmplIds: [templateId],
      success: (r: any) => resolve(r && r[templateId] === 'accept'),
      fail: () => resolve(false),
    })
  }),
}
