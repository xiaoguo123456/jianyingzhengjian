export type Module = 'idphoto' | 'pro' | 'portrait' | 'avatar'
export type TaskKind = 'idphoto' | 'template'
export type TaskStatus = 'waiting' | 'processing' | 'success' | 'failed'
export type TaskStage = 'queued' | 'processing' | 'finishing'

export const MODULE_NAME: Record<Module, string> = {
  idphoto: '证件照',
  pro: '职业照',
  portrait: '写真',
  avatar: '头像',
}

export interface Spec {
  id: string
  name: string
  width_mm: number
  height_mm: number
  width_px: number
  height_px: number
  dpi: number
  bg_default: string
  bg_allowed: string[]
  category?: { id: string; name: string }
  note?: string
  is_hot?: boolean
}

export interface Category {
  id: string
  module: Module
  name: string
  icon?: string
  cover_url?: string
  desc?: string
}

export interface TemplateCard {
  id: string
  module: Module
  name: string
  cover_url: string
  tags: string[]
  is_favorited: boolean
  credit_cost: number
}

export interface Template extends TemplateCard {
  subtitle: string
  sample_urls: string[]
  output: { width: number; height: number; aspect: string }
  style: 'photo' | 'illustration'
  category?: { id: string; name: string }
}

export interface Collection {
  id: string
  module: Module
  name: string
  cover_url: string
  desc: string
  template_count: number
}

export interface Banner {
  title: string
  subtitle: string
  image_url: string
  link: { type: 'idphoto_flow' | 'upload' | 'template' | 'collection' | 'url'; id?: string }
}

/** A horizontally scrolling row of templates on a tab home. */
export interface TemplateRail {
  title: string
  category_id?: string
  collection_id?: string
  templates: TemplateCard[]
}

export interface HomePayload {
  banner: Banner
  hot_specs?: Spec[]
  more_specs?: Spec[]
  hot_categories?: Category[]
  collections?: Collection[]
  hot_templates?: TemplateCard[]
  rails?: TemplateRail[]
}

export interface Photo {
  id: string
  width: number
  height: number
  preview_url: string
  expires_at: string
  created_at: string
}

export interface PhotoCheck {
  passed: boolean
  faces: number
  reasons: string[]
}

export interface IdPhotoParams {
  bg: string
  clothing: string
  beauty: 'natural' | 'light'
}

export interface ClothingOption {
  id: string
  name: string
  group: 'keep' | 'male' | 'female'
}

export interface WorkSpec {
  id: string
  name: string
  width_mm: number
  height_mm: number
  width_px: number
  height_px: number
}

export interface Work {
  id: string
  module: Module
  url: string
  thumb_url: string
  width: number
  height: number
  created_at: string
  spec?: WorkSpec
  template?: { id: string; name: string }
  meta: Record<string, any>
  ai_label: boolean
  task_id: string
}

export interface Task {
  id: string
  status: TaskStatus
  stage: TaskStage | null
  module: Module
  kind: TaskKind
  uses_genmodel: boolean
  credits_consumed: number
  created_at: string
  started_at: string | null
  finished_at: string | null
  work: Work | null
  error: { code: string; message: string } | null
  refunded: boolean
  template_id?: string
  spec_id?: string
  photo_id: string
  params: Record<string, any>
  target_name: string
}

export interface Credits {
  daily_free_remaining: number
  bonus_credits: number
  total: number
  ad_rewards_today: number
  ad_reward_daily_cap: number
  ads_enabled: boolean
}

export interface User {
  id: string
  nickname: string
  avatar_url: string
  works_count: number
  privacy_agreed: boolean
}

export interface ClientConfig {
  ads_enabled: boolean
  ad_unit_ids: { reward: string }
  subscribe_template_ids: { task_finished: string }
  privacy_policy_url: string
  user_agreement_url: string
  retention_days: number
  share_reward: { enabled: boolean; per_reward: number; daily_cap: number }
}

export interface Share {
  id: string
  type: 'template' | 'work' | 'poster' | 'tab'
  path: string
  title: string
  image_url: string
  poster_url?: string
}

export interface ShareLanding {
  type: 'template' | 'work' | 'poster' | 'tab'
  template_id?: string
  spec_id?: string
  preview_url?: string
  title: string
}

export interface ShareRewards {
  enabled: boolean
  per_reward: number
  daily_cap: number
  earned_today: number
  earned_total: number
}

export interface WorksSummary {
  total: number
  by_module: Record<Module, number>
  recent: Work[]
}

export interface Paged<T> {
  items: T[]
  total: number
  has_more: boolean
}

export interface AppEvent {
  name: string
  props?: Record<string, any>
  ts: string
}

/** Current generation attempt (docs/FRONTEND_ARCHITECTURE.md §6). */
export interface FlowState {
  module: Module | null
  kind: 'idphoto' | 'template' | null
  templateId: string | null
  templateName: string | null
  specId: string | null
  spec: Spec | null
  photo: Photo | null
  params: IdPhotoParams
  idempotencyKey: string
  notifyRequested: boolean
  parentTaskId: string | null
}
