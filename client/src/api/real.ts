/* HTTP implementation of the Api contract against docs/API.md. */
import type { Api } from './types'
import { request, upload } from './http'

const q = (o: Record<string, any>) => {
  const parts = Object.keys(o).filter((k) => o[k] !== undefined && o[k] !== null && o[k] !== '').map((k) => `${k}=${encodeURIComponent(o[k])}`)
  return parts.length ? `?${parts.join('&')}` : ''
}

export const realApi: Api = {
  login: (input) => request('/v1/auth/login', { method: 'POST', data: input }),
  me: () => request('/v1/me'),
  updateProfile: async (input) => {
    if (input.avatar_path) await upload('/v1/me/avatar', input.avatar_path)
    if (input.nickname !== undefined) await request('/v1/me', { method: 'PUT', data: { nickname: input.nickname } })
    return (await request<{ user: any }>('/v1/me')).user
  },
  agreePrivacy: (version) => request('/v1/me/privacy-agree', { method: 'POST', data: { version } }),

  credits: () => request('/v1/credits'),
  createAdSession: () => request('/v1/ads/sessions', { method: 'POST' }),
  claimAd: (id, isEnded) => request(`/v1/ads/sessions/${id}/claim`, { method: 'POST', data: { is_ended: isEnded } }),

  home: (module) => request(`/v1/home/${module}`),
  specCategories: () => request('/v1/spec-categories'),
  specs: (o) => request<{ items: any[] }>(`/v1/specs${q({ category_id: o.category_id, hot: o.hot ? 1 : undefined, page_size: 100 })}`).then((r) => r.items),
  spec: (id) => request(`/v1/specs/${id}`),
  templateCategories: (module) => request(`/v1/template-categories${q({ module })}`),
  templates: (o) => request(`/v1/templates${q({ ...o, hot: o.hot ? 1 : undefined })}`),
  template: (id) => request(`/v1/templates/${id}`),
  collections: (module) => request(`/v1/collections${q({ module })}`),
  collection: (id) => request(`/v1/collections/${id}`),
  clothingOptions: () => request('/v1/clothing-options'),

  uploadPhoto: (input) => upload('/v1/photos', input.filePath, { module: input.module }),
  photos: (page = 1) => request(`/v1/photos${q({ page })}`),
  deletePhoto: (id) => request(`/v1/photos/${id}`, { method: 'DELETE' }),

  createTask: (input) => request('/v1/tasks', { method: 'POST', data: input, headers: { 'Idempotency-Key': input.idempotency_key } }),
  task: (id) => request(`/v1/tasks/${id}`),
  tasks: (o) => request(`/v1/tasks${q(o)}`),
  regenerate: (id, key, bg) => request(`/v1/tasks/${id}/regenerate`, { method: 'POST', headers: { 'Idempotency-Key': key }, data: bg ? { bg } : {} }),

  works: (o) => request(`/v1/works${q({ module: o.module === 'all' ? undefined : o.module, page: o.page })}`),
  worksSummary: () => request('/v1/works/summary'),
  work: (id) => request(`/v1/works/${id}`),
  downloadUrl: (id) => request(`/v1/works/${id}/download`),
  deleteWork: (id) => request(`/v1/works/${id}`, { method: 'DELETE' }),

  favorites: (page = 1) => request(`/v1/favorites${q({ page })}`),
  favorite: (id) => request(`/v1/favorites/${id}`, { method: 'PUT' }),
  unfavorite: (id) => request(`/v1/favorites/${id}`, { method: 'DELETE' }),

  createShare: (input) => request<{ share: any }>('/v1/shares', { method: 'POST', data: input }).then((r) => r.share),
  openShare: (id, platform) => request<{ share: any }>(`/v1/shares/${id}/open`, { method: 'POST', data: { platform } }).then((r) => r.share),
  shareRewards: () => request('/v1/shares/rewards'),

  events: (events) => request('/v1/events', { method: 'POST', data: { events } }),
}
