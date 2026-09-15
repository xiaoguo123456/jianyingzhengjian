import { defineStore } from 'pinia'
import type { PortraitSegment } from '@/utils/routes'

/** Cross-page UI state that cannot travel through switchTab URLs. */
export const useUiStore = defineStore('ui', {
  state: () => ({ portraitSegment: 'portrait' as PortraitSegment }),
  actions: {
    setPortraitSegment(s: PortraitSegment) { this.portraitSegment = s },
  },
})
