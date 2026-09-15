export interface SafeArea { statusBar: number; navHeight: number; capsuleRight: number; top: number }

let cached: SafeArea | null = null
export function safeArea(): SafeArea {
  if (cached) return cached
  const info = uni.getSystemInfoSync()
  let statusBar = info.statusBarHeight || 0
  let navHeight = 44
  let capsuleRight = 0
  // #ifdef MP-WEIXIN
  try {
    const rect = uni.getMenuButtonBoundingClientRect()
    navHeight = rect.height + (rect.top - statusBar) * 2
    capsuleRight = info.windowWidth - rect.left
  } catch { /* fall through */ }
  // #endif
  // #ifdef H5
  statusBar = 0
  // #endif
  cached = { statusBar, navHeight, capsuleRight, top: statusBar + navHeight }
  return cached
}

const TAB_PATHS = ['/pages/idphoto/index', '/pages/pro/index', '/pages/portrait/index', '/pages/mine/index']

export function go(path: string) {
  // switchTab rejects query strings; tab pages take cross-page state from stores instead
  if (TAB_PATHS.includes(path.split('?')[0])) return uni.switchTab({ url: path.split('?')[0] })
  return uni.navigateTo({ url: path })
}
export function replace(path: string) { return uni.redirectTo({ url: path }) }
export function back(delta = 1) {
  const pages = getCurrentPages()
  if (pages.length > delta) return uni.navigateBack({ delta })
  return uni.switchTab({ url: '/pages/idphoto/index' })
}
export function openWebview(url: string, title = '') {
  return uni.navigateTo({ url: `/pages/webview/index?url=${encodeURIComponent(url)}&title=${encodeURIComponent(title)}` })
}
export function toast(title: string, icon: 'none' | 'success' = 'none') {
  uni.showToast({ title, icon, duration: 2000 })
}
