import type {
  AppEvent, Category, ClientConfig, ClothingOption, Collection, Credits, HomePayload, Module, Paged,
  Photo, PhotoCheck, Share, ShareLanding, ShareRewards, Spec, Task, Template, TemplateCard, User, Work, WorksSummary,
} from '@/types'

export interface LoginInput { provider: string; code: string; share_id?: string }
export interface UploadInput { filePath: string; module: Module }
export interface CreateTaskInput {
  kind: 'idphoto' | 'template'
  spec_id?: string
  template_id?: string
  photo_id: string
  params: Record<string, any>
  notify: boolean
  parent_task_id?: string | null
  idempotency_key: string
}
export interface TemplateQuery {
  module: Module
  category_id?: string
  collection_id?: string
  hot?: boolean
  page?: number
  page_size?: number
}
export interface CreateShareInput {
  type: 'template' | 'work' | 'poster' | 'tab'
  template_id?: string
  work_id?: string
  module?: Module
  surface: string
}

/** Contract implemented by both the HTTP client (real.ts) and the offline mock (mock/index.ts). */
export interface Api {
  login(input: LoginInput): Promise<{ token: string; user: User; is_new: boolean }>
  me(): Promise<{ user: User; credits: Credits; config: ClientConfig }>
  updateProfile(input: { nickname?: string; avatar_path?: string }): Promise<User>
  agreePrivacy(version: string): Promise<void>

  credits(): Promise<Credits>
  createAdSession(): Promise<{ session_id: string; ad_unit_id: string }>
  claimAd(sessionId: string, isEnded: boolean): Promise<{ granted: number; credits: Credits }>

  home(module: Module): Promise<HomePayload>
  specCategories(): Promise<{ id: string; name: string }[]>
  specs(q: { category_id?: string; hot?: boolean }): Promise<Spec[]>
  spec(id: string): Promise<Spec>
  templateCategories(module: Module): Promise<Category[]>
  templates(q: TemplateQuery): Promise<Paged<TemplateCard>>
  template(id: string): Promise<Template>
  collections(module: Module): Promise<Collection[]>
  collection(id: string): Promise<{ collection: Collection; templates: TemplateCard[] }>
  clothingOptions(): Promise<ClothingOption[]>

  uploadPhoto(input: UploadInput): Promise<{ photo: Photo; check: PhotoCheck }>
  photos(page?: number): Promise<Paged<Photo>>
  deletePhoto(id: string): Promise<void>

  createTask(input: CreateTaskInput): Promise<{ task: Task; credits: Credits }>
  task(id: string): Promise<Task>
  tasks(q: { status?: string; page?: number }): Promise<Paged<Task>>
  /** bg redraws an ID photo on another background (a new task, one credit). */
  regenerate(taskId: string, idempotencyKey: string, bg?: string): Promise<{ task: Task; credits: Credits }>

  works(q: { module?: Module | 'all'; page?: number }): Promise<Paged<Work>>
  worksSummary(): Promise<WorksSummary>
  work(id: string): Promise<Work>
  downloadUrl(id: string): Promise<{ url: string }>
  deleteWork(id: string): Promise<void>

  favorites(page?: number): Promise<Paged<TemplateCard>>
  favorite(templateId: string): Promise<void>
  unfavorite(templateId: string): Promise<void>

  createShare(input: CreateShareInput): Promise<Share>
  openShare(id: string, platform: string): Promise<ShareLanding>
  shareRewards(): Promise<ShareRewards>

  events(events: AppEvent[]): Promise<void>
}
