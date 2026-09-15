import { onLoad, onShow } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { useCatalogueStore } from '@/store/catalogue'
import { useFlowStore } from '@/store/flow'
import { useUserStore } from '@/store/user'
import type { HomePayload, Module, Spec, TemplateCard } from '@/types'
import { go } from '@/platform'
import { track } from './useAnalytics'

/** Shared loading + navigation logic for the four content tabs. */
export function useHome(module: Module, opts: { trackView?: boolean } = {}) {
  const user = useUserStore()
  const catalogue = useCatalogueStore()
  const flow = useFlowStore()
  const home = ref<HomePayload | null>(null)
  const loading = ref(true)
  const error = ref(false)

  async function load(force = false) {
    loading.value = !home.value
    error.value = false
    try {
      await user.ready()
      home.value = await catalogue.home(module, force)
    } catch {
      error.value = true
    } finally {
      loading.value = false
    }
  }
  onLoad(() => load())
  onShow(() => {
    if (opts.trackView !== false) track('tab_view', { module })
    if (home.value) load()
  })

  function startUpload(source: string) {
    track(source === 'banner' ? 'banner_click' : 'upload_start', { module, source })
    flow.start(module)
    go(`/pages/upload/index?module=${module}`)
  }
  function pickSpec(s: Spec) {
    track('spec_click', { spec_id: s.id })
    flow.start('idphoto')
    flow.setSpec(s)
    go(`/pages/upload/index?module=idphoto`)
  }
  function openTemplate(t: TemplateCard) {
    track('template_click', { template_id: t.id, module })
    go(`/pages/template-detail/index?id=${t.id}`)
  }
  function openCategory(id: string, name: string) {
    go(`/pages/template-list/index?module=${module}&category=${id}&title=${encodeURIComponent(name)}`)
  }
  function openCollection(id: string, name: string) {
    go(`/pages/template-list/index?module=${module}&collection=${id}&title=${encodeURIComponent(name)}`)
  }
  function openMore(title: string) {
    go(`/pages/template-list/index?module=${module}&title=${encodeURIComponent(title)}`)
  }

  return { home, loading, error, load, startUpload, pickSpec, openTemplate, openCategory, openCollection, openMore }
}
