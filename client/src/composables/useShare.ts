import { onShareAppMessage, onShareTimeline } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { api, type CreateShareInput } from '@/api'
import { platformName } from '@/api/http'
import { share as sharePlatform, toast } from '@/platform'
import { useUserStore } from '@/store/user'
import type { Share } from '@/types'
import { track } from '@/composables/useAnalytics'
import { moduleLinkPath } from '@/utils/routes'

/** Reads `s=` from launch options (query or mini-program-code scene) and reports the open. */
export function handleLaunchShare(options: any) {
  let s: string = options?.query?.s || ''
  if (!s && options?.scene && typeof options.scene === 'string') {
    const m = /(?:^|&)s=([^&]+)/.exec(decodeURIComponent(options.scene))
    if (m) s = m[1]
  }
  if (!s) return
  const user = useUserStore()
  user.pendingShareId = s
  api.openShare(s, platformName()).catch(() => {})
}

/**
 * Page-level share. `prepare` builds the share row lazily (only when the user taps a share action),
 * and WeChat's share hooks return the prepared payload synchronously.
 */
export function usePageShare(defaultInput: () => CreateShareInput | null) {
  const current = ref<Share | null>(null)
  const preparing = ref(false)

  async function prepare(input?: CreateShareInput): Promise<Share | null> {
    const i = input || defaultInput()
    if (!i) return null
    preparing.value = true
    try {
      current.value = await api.createShare(i)
      track('share_click', { surface: i.surface, type: i.type })
      return current.value
    } catch {
      toast('分享暂时不可用')
      return null
    } finally {
      preparing.value = false
    }
  }

  const payload = () => {
    const s = current.value
    if (!s) {
      const i = defaultInput()
      return { title: '映己证件照写真馆', path: moduleLinkPath(i?.module || 'idphoto') }
    }
    track('share_sent', { type: s.type })
    return { title: s.title, path: s.path, imageUrl: s.image_url }
  }
  onShareAppMessage(payload)
  onShareTimeline(() => { const p = payload(); return { title: p.title, query: p.path.split('?')[1] || '', imageUrl: (p as any).imageUrl } })

  async function shareDirect(scene: 'chat' | 'timeline') {
    const s = current.value || (await prepare())
    if (!s) return
    if (sharePlatform.nativeChatShare) return
    try {
      await sharePlatform.share({ title: s.title, path: s.path, imageUrl: s.image_url, scene })
    } catch {
      await sharePlatform.copyLink(s.path)
      toast('链接已复制')
    }
  }

  return { current, preparing, prepare, shareDirect }
}
