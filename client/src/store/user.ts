import { defineStore } from 'pinia'
import { api } from '@/api'
import { setReloginHandler, setToken } from '@/api/http'
import { auth, ads } from '@/platform'
import type { ClientConfig, Credits, User } from '@/types'

const TOKEN_KEY = 'yj_token'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: '' as string,
    user: null as User | null,
    credits: null as Credits | null,
    config: null as ClientConfig | null,
    booted: false,
    booting: null as Promise<void> | null,
    pendingShareId: '' as string,
  }),
  getters: {
    total: (s) => s.credits?.total ?? 0,
    adsEnabled: (s) => s.config?.ads_enabled ?? false,
    shareReward: (s) => s.config?.share_reward ?? { enabled: false, per_reward: 1, daily_cap: 3 },
  },
  actions: {
    bootstrap(): Promise<void> {
      if (this.booting) return this.booting
      this.booting = (async () => {
        setReloginHandler(async () => { await this.login(); return this.token })
        let saved = ''
        try { saved = uni.getStorageSync(TOKEN_KEY) || '' } catch { saved = '' }
        if (saved) { this.token = saved; setToken(saved) } else { await this.login() }
        await this.loadMe()
        this.booted = true
      })()
      return this.booting
    },
    /** Pages call this to wait for auth + config before their first request. */
    ready(): Promise<void> {
      return this.booted ? Promise.resolve() : this.bootstrap()
    },
    async login() {
      const cred = await auth.login()
      const r = await api.login({ ...cred, share_id: this.pendingShareId || undefined })
      this.token = r.token
      this.user = r.user
      setToken(r.token)
      try { uni.setStorageSync(TOKEN_KEY, r.token) } catch { /* ignore */ }
    },
    async loadMe() {
      const r = await api.me()
      this.user = r.user
      this.credits = r.credits
      this.config = r.config
      ads.configure(r.config.ad_unit_ids.reward)
    },
    async refreshCredits() {
      this.credits = await api.credits()
    },
    setCredits(c: Credits) { this.credits = c },
    async agreePrivacy() {
      await api.agreePrivacy('2026-09-01')
      if (this.user) this.user.privacy_agreed = true
    },
  },
})
