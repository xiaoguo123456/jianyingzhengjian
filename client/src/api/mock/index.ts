/* Offline mock of the Api contract. Keeps state in memory, simulates task
   progression, credits, ads and shares so every screen can be exercised
   without the Go backend. Dev toggles live on `mockState`. */
import type { Api, CreateTaskInput } from '../types'
import type { Credits, Module, Photo, Share, Task, TemplateCard, Work } from '@/types'
import { MODULE_NAME } from '@/types'
import { ApiError } from '@/utils/errors'
import { moduleLinkPath } from '@/utils/routes'
import { shortId, sleep } from '@/utils/uuid'
import {
  BANNERS, CATEGORIES, CLOTHING, COLLECTIONS, HOT_CATEGORY_IDS, HOT_TEMPLATE_IDS, INITIAL_WORKS,
  MOCK_USER, MORE_SPEC_IDS, RAILS, SAMPLE_OUTPUT, SPECS, SPEC_CATEGORIES, TEMPLATES,
} from './data'

interface AdSession { id: string; created_at: number; claimed: boolean }

export const mockState = {
  privacyAgreed: false,
  nickname: MOCK_USER.nickname,
  avatar: MOCK_USER.avatar_url,
  daily: 1,
  bonus: 0,
  adsToday: 0,
  adsEnabled: true,
  simulateFailure: false,
  rejectNextPhoto: false,
  photos: [] as Photo[],
  tasks: new Map<string, Task>(),
  works: [...INITIAL_WORKS] as Work[],
  favorites: new Set<string>(['t_korean']),
  adSessions: new Map<string, AdSession>(),
  shares: new Map<string, Share & { template_id?: string; spec_id?: string; work_id?: string }>(),
  sampleIdx: 0,
  rewardsTotal: 2,
}

const now = () => new Date().toISOString()
const delay = (ms = 250) => sleep(ms)

function credits(): Credits {
  return {
    daily_free_remaining: mockState.daily,
    bonus_credits: mockState.bonus,
    total: mockState.daily + mockState.bonus,
    ad_rewards_today: mockState.adsToday,
    ad_reward_daily_cap: 10,
    ads_enabled: mockState.adsEnabled,
  }
}

function card(t: typeof TEMPLATES[number]): TemplateCard {
  return { id: t.id, module: t.module, name: t.name, cover_url: t.cover_url, tags: t.tags, is_favorited: mockState.favorites.has(t.id), credit_cost: t.credit_cost }
}

function paged<T>(items: T[], page = 1, size = 20) {
  const start = (page - 1) * size
  return { items: items.slice(start, start + size), total: items.length, has_more: start + size < items.length }
}

/* Seed task records for the initial works so 生成记录 is not empty. */
for (const w of INITIAL_WORKS) {
  mockState.tasks.set(w.task_id, {
    id: w.task_id, status: 'success', stage: null, module: w.module, kind: w.spec ? 'idphoto' : 'template',
    uses_genmodel: w.ai_label, credits_consumed: w.ai_label ? 1 : 0, created_at: w.created_at, started_at: w.created_at,
    finished_at: w.created_at, work: w, error: null, refunded: false, template_id: w.template?.id, spec_id: w.spec?.id,
    photo_id: 'p_seed', params: w.meta, target_name: w.template?.name || w.spec?.name || '',
  })
}
mockState.tasks.set('tk_failed', {
  id: 'tk_failed', status: 'failed', stage: null, module: 'portrait', kind: 'template', uses_genmodel: true, credits_consumed: 1,
  created_at: new Date(Date.now() - 4 * 864e5).toISOString(), started_at: null, finished_at: new Date(Date.now() - 4 * 864e5).toISOString(),
  work: null, error: { code: 'PROVIDER_ERROR', message: '本次生成失败，生成次数已返还，请重新尝试。' }, refunded: true,
  template_id: 't_autumn', photo_id: 'p_seed', params: {}, target_name: '秋日写真',
})

function consume(n: number) {
  let left = n
  const fromDaily = Math.min(mockState.daily, left)
  mockState.daily -= fromDaily
  left -= fromDaily
  mockState.bonus -= left
  return { daily: fromDaily, bonus: left }
}

function nextSample(module: Module) {
  const list = SAMPLE_OUTPUT[module]
  return list[mockState.sampleIdx++ % list.length]
}

function runTask(task: Task, refund: { daily: number; bonus: number }) {
  const gen = task.uses_genmodel
  const t = gen ? [400, 1200, 3000, 4200] : [200, 600, 1200, 1600]
  const set = (patch: Partial<Task>) => Object.assign(task, patch)
  setTimeout(() => set({ status: 'processing', stage: 'queued', started_at: now() }), t[0])
  setTimeout(() => set({ stage: 'processing' }), t[1])
  if (mockState.simulateFailure && gen) {
    setTimeout(() => {
      mockState.daily += refund.daily
      mockState.bonus += refund.bonus
      set({ status: 'failed', stage: null, finished_at: now(), refunded: true, error: { code: 'PROVIDER_ERROR', message: '本次生成失败，生成次数已返还，请重新尝试。' } })
    }, t[2])
    return
  }
  setTimeout(() => set({ stage: 'finishing' }), t[2])
  setTimeout(() => {
    const spec = task.spec_id ? SPECS.find((s) => s.id === task.spec_id) : undefined
    const tpl = task.template_id ? TEMPLATES.find((x) => x.id === task.template_id) : undefined
    const url = nextSample(task.module)
    const work: Work = {
      id: shortId('w_'), module: task.module, url, thumb_url: url,
      width: spec ? spec.width_px : tpl?.output.width || 1200, height: spec ? spec.height_px : tpl?.output.height || 1600,
      created_at: now(), task_id: task.id, ai_label: gen,
      spec: spec ? { id: spec.id, name: spec.name, width_mm: spec.width_mm, height_mm: spec.height_mm, width_px: spec.width_px, height_px: spec.height_px } : undefined,
      template: tpl ? { id: tpl.id, name: tpl.name } : undefined,
      meta: { ...task.params },
    }
    mockState.works.unshift(work)
    set({ status: 'success', stage: null, finished_at: now(), work })
  }, t[3])
}

export const mockApi: Api = {
  async login() {
    await delay(200)
    return { token: 'mock-token', user: userObj(), is_new: false }
  },
  async me() {
    await delay(150)
    return {
      user: userObj(),
      credits: credits(),
      config: {
        ads_enabled: mockState.adsEnabled,
        ad_unit_ids: { reward: '' },
        subscribe_template_ids: { task_finished: '' },
        privacy_policy_url: 'https://example.com/privacy',
        user_agreement_url: 'https://example.com/agreement',
        retention_days: 30,
        share_reward: { enabled: true, per_reward: 1, daily_cap: 3 },
      },
    }
  },
  async updateProfile(input) {
    await delay(300)
    if (input.nickname !== undefined) mockState.nickname = input.nickname
    if (input.avatar_path) mockState.avatar = input.avatar_path
    return userObj()
  },
  async agreePrivacy() { mockState.privacyAgreed = true },

  async credits() { await delay(100); return credits() },
  async createAdSession() {
    await delay(100)
    if (!mockState.adsEnabled) throw new ApiError('CONFLICT', '', 409)
    if (mockState.adsToday >= 10) throw new ApiError('AD_SESSION_INVALID', '', 422, { reason: 'cap_reached' })
    const s: AdSession = { id: shortId('ad_'), created_at: Date.now(), claimed: false }
    mockState.adSessions.set(s.id, s)
    return { session_id: s.id, ad_unit_id: '' }
  },
  async claimAd(id, isEnded) {
    await delay(150)
    const s = mockState.adSessions.get(id)
    if (!s || s.claimed) throw new ApiError('AD_SESSION_INVALID', '', 422, { reason: 'duplicate' })
    if (!isEnded) { s.claimed = true; throw new ApiError('AD_SESSION_INVALID', '', 422, { reason: 'not_ended' }) }
    if (Date.now() - s.created_at < 1000) throw new ApiError('AD_SESSION_INVALID', '', 422, { reason: 'too_fast' })
    s.claimed = true
    mockState.bonus += 1
    mockState.adsToday += 1
    return { granted: 1, credits: credits() }
  },

  async home(module) {
    await delay(200)
    const base = { banner: BANNERS[module] }
    if (module === 'idphoto') {
      return { ...base, hot_specs: SPECS.filter((s) => s.is_hot), more_specs: MORE_SPEC_IDS.map((id) => SPECS.find((s) => s.id === id)!) }
    }
    const byId = (id: string) => card(TEMPLATES.find((t) => t.id === id)!)
    return {
      ...base,
      hot_categories: HOT_CATEGORY_IDS[module].map((id) => CATEGORIES.find((c) => c.id === id)!),
      collections: module === 'portrait' ? COLLECTIONS : undefined,
      hot_templates: HOT_TEMPLATE_IDS[module].map(byId),
      rails: (RAILS[module] || []).map((r) => ({ title: r.title, category_id: r.category_id, collection_id: r.collection_id, templates: r.ids.map(byId) })),
    }
  },
  async specCategories() { await delay(); return SPEC_CATEGORIES },
  async specs(o) {
    await delay()
    return SPECS.filter((s) => (!o.category_id || s.category?.id === o.category_id) && (!o.hot || s.is_hot))
  },
  async spec(id) { await delay(); const s = SPECS.find((x) => x.id === id); if (!s) throw new ApiError('NOT_FOUND', '', 404); return s },
  async templateCategories(module) { await delay(); return CATEGORIES.filter((c) => c.module === module) },
  async templates(o) {
    await delay(200)
    let list = TEMPLATES.filter((t) => t.module === o.module)
    if (o.category_id) list = list.filter((t) => t.category?.id === o.category_id)
    if (o.collection_id) list = list.slice(0, 4)
    if (o.hot) list = list.filter((t) => t.tags.includes('热门'))
    return paged(list.map(card), o.page, o.page_size)
  },
  async template(id) {
    await delay(150)
    const t = TEMPLATES.find((x) => x.id === id)
    if (!t) throw new ApiError('NOT_FOUND', '', 404)
    return { ...t, is_favorited: mockState.favorites.has(id) }
  },
  async collections(module) { await delay(); return COLLECTIONS.filter((c) => c.module === module) },
  async collection(id) {
    await delay()
    const c = COLLECTIONS.find((x) => x.id === id)
    if (!c) throw new ApiError('NOT_FOUND', '', 404)
    return { collection: c, templates: TEMPLATES.filter((t) => t.module === 'portrait').slice(0, 4).map(card) }
  },
  async clothingOptions() { return CLOTHING },

  async uploadPhoto(input) {
    await delay(900)
    if (mockState.rejectNextPhoto) {
      mockState.rejectNextPhoto = false
      throw new ApiError('PHOTO_REJECTED', '', 422, { reasons: ['blurry', 'too_dark'] })
    }
    const p: Photo = { id: shortId('p_'), width: 3024, height: 4032, preview_url: input.filePath, expires_at: new Date(Date.now() + 30 * 864e5).toISOString(), created_at: now() }
    mockState.photos.unshift(p)
    return { photo: p, check: { passed: true, faces: 1, reasons: [] } }
  },
  async photos(page = 1) { await delay(); return paged(mockState.photos, page) },
  async deletePhoto(id) { await delay(); mockState.photos = mockState.photos.filter((p) => p.id !== id) },

  async createTask(input) {
    await delay(300)
    for (const t of mockState.tasks.values()) if ((t as any)._key === input.idempotency_key) return { task: t, credits: credits() }
    return createTask(input)
  },
  async task(id) {
    await delay(120)
    const t = mockState.tasks.get(id)
    if (!t) throw new ApiError('NOT_FOUND', '', 404)
    return { ...t }
  },
  async tasks(o) {
    await delay()
    let list = [...mockState.tasks.values()].sort((a, b) => (a.created_at < b.created_at ? 1 : -1))
    if (o.status) list = list.filter((t) => t.status === o.status)
    return paged(list, o.page)
  },
  async regenerate(taskId, key) {
    await delay(200)
    const src = mockState.tasks.get(taskId)
    if (!src) throw new ApiError('NOT_FOUND', '', 404)
    return createTask({ kind: src.kind as any, spec_id: src.spec_id, template_id: src.template_id, photo_id: src.photo_id, params: src.params, notify: false, parent_task_id: taskId, idempotency_key: key })
  },

  async works(o) {
    await delay(200)
    const list = mockState.works.filter((w) => !o.module || o.module === 'all' || w.module === o.module)
    return paged(list, o.page)
  },
  async worksSummary() {
    await delay(150)
    const by: Record<Module, number> = { idphoto: 0, pro: 0, portrait: 0, avatar: 0 }
    for (const w of mockState.works) by[w.module]++
    return { total: mockState.works.length, by_module: by, recent: mockState.works.slice(0, 4) }
  },
  async work(id) { await delay(120); const w = mockState.works.find((x) => x.id === id); if (!w) throw new ApiError('NOT_FOUND', '', 404); return w },
  async downloadUrl(id) { const w = mockState.works.find((x) => x.id === id); if (!w) throw new ApiError('NOT_FOUND', '', 404); return { url: w.url } },
  async recolor(id, bg) {
    await delay(500)
    const src = mockState.works.find((x) => x.id === id)
    if (!src) throw new ApiError('NOT_FOUND', '', 404)
    const w: Work = { ...src, id: shortId('w_'), created_at: now(), meta: { ...src.meta, bg, recolored_from: src.id } }
    mockState.works.unshift(w)
    return w
  },
  async deleteWork(id) { await delay(); mockState.works = mockState.works.filter((w) => w.id !== id) },

  async favorites(page = 1) { await delay(); return paged(TEMPLATES.filter((t) => mockState.favorites.has(t.id)).map(card), page) },
  async favorite(id) { await delay(100); mockState.favorites.add(id) },
  async unfavorite(id) { await delay(100); mockState.favorites.delete(id) },

  async createShare(input) {
    await delay(300)
    const id = shortId('s_')
    let template_id = input.template_id
    let spec_id: string | undefined
    let image = ''
    let title = ''
    if (input.type === 'template' && template_id) {
      const t = TEMPLATES.find((x) => x.id === template_id)!
      image = t.cover_url
      title = `「${t.name}」上传照片就能生成同款`
    } else if ((input.type === 'work' || input.type === 'poster') && input.work_id) {
      const w = mockState.works.find((x) => x.id === input.work_id)!
      image = w.url
      template_id = w.template?.id
      spec_id = w.spec?.id
      title = `我用「${w.template?.name || w.spec?.name}」生成了这张，试试同款`
    } else {
      title = `上传自拍，生成${MODULE_NAME[input.module || 'idphoto']}`
      image = BANNERS[input.module || 'idphoto'].image_url
    }
    const path = template_id
      ? `/pages/template-detail/index?id=${template_id}&s=${id}`
      : spec_id ? `/pages/spec-library/index?spec=${spec_id}&s=${id}` : moduleLinkPath(input.module || 'idphoto', { s: id })
    const share = { id, type: input.type, path, title, image_url: image, poster_url: input.type === 'poster' ? image : undefined, template_id, spec_id, work_id: input.work_id }
    mockState.shares.set(id, share)
    return share
  },
  async openShare(id) {
    await delay(100)
    const s = mockState.shares.get(id)
    if (!s) throw new ApiError('NOT_FOUND', '', 404)
    return { type: s.type, template_id: s.template_id, spec_id: s.spec_id, preview_url: s.type === 'template' ? undefined : s.image_url, title: s.title }
  },
  async shareRewards() { await delay(); return { enabled: true, per_reward: 1, daily_cap: 3, earned_today: 0, earned_total: mockState.rewardsTotal } },

  async events() { /* no-op in mock */ },
}

function userObj() {
  return { id: MOCK_USER.id, nickname: mockState.nickname, avatar_url: mockState.avatar, works_count: mockState.works.length, privacy_agreed: mockState.privacyAgreed }
}

function createTask(input: CreateTaskInput) {
  const tpl = input.template_id ? TEMPLATES.find((t) => t.id === input.template_id) : undefined
  const spec = input.spec_id ? SPECS.find((s) => s.id === input.spec_id) : undefined
  const usesGen = input.kind === 'template' || (input.params.clothing && input.params.clothing !== 'keep') || input.params.beauty === 'light'
  const cost = usesGen ? tpl?.credit_cost ?? 1 : 0
  if (usesGen && mockState.daily + mockState.bonus < cost) throw new ApiError('NO_CREDITS', '', 402, { credits: credits() })
  const refund = usesGen ? consume(cost) : { daily: 0, bonus: 0 }
  const module: Module = tpl ? tpl.module : 'idphoto'
  const task: Task = {
    id: shortId('tk_'), status: 'waiting', stage: 'queued', module, kind: input.kind, uses_genmodel: !!usesGen, credits_consumed: cost,
    created_at: now(), started_at: null, finished_at: null, work: null, error: null, refunded: false,
    template_id: input.template_id, spec_id: input.spec_id, photo_id: input.photo_id, params: input.params,
    target_name: tpl?.name || spec?.name || '',
  }
  ;(task as any)._key = input.idempotency_key
  mockState.tasks.set(task.id, task)
  runTask(task, refund)
  return { task: { ...task }, credits: credits() }
}
