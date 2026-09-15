import { defineStore } from 'pinia'
import { api } from '@/api'
import type { HomePayload, Module, Template } from '@/types'

const TTL = 60_000

export const useCatalogueStore = defineStore('catalogue', {
  state: () => ({
    homes: {} as Partial<Record<Module, { data: HomePayload; at: number }>>,
    templates: {} as Record<string, Template>,
  }),
  actions: {
    async home(module: Module, force = false): Promise<HomePayload> {
      const hit = this.homes[module]
      if (!force && hit && Date.now() - hit.at < TTL) return hit.data
      const data = await api.home(module)
      this.homes[module] = { data, at: Date.now() }
      return data
    },
    async template(id: string, force = false): Promise<Template> {
      if (!force && this.templates[id]) return this.templates[id]
      const t = await api.template(id)
      this.templates[id] = t
      return t
    },
    async toggleFavorite(id: string): Promise<boolean> {
      const t = this.templates[id]
      const next = !(t?.is_favorited ?? false)
      if (next) await api.favorite(id); else await api.unfavorite(id)
      if (t) t.is_favorited = next
      for (const h of Object.values(this.homes)) {
        const c = h?.data.hot_templates?.find((x) => x.id === id)
        if (c) c.is_favorited = next
      }
      return next
    },
  },
})
