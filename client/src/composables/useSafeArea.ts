import { computed } from 'vue'
import { safeArea } from '@/platform'

export function useSafeArea() {
  const sa = safeArea()
  return {
    statusBar: computed(() => sa.statusBar),
    navHeight: computed(() => sa.navHeight),
    top: computed(() => sa.top),
    capsuleRight: computed(() => sa.capsuleRight),
    topStyle: computed(() => `padding-top:${sa.top}px`),
  }
}
