import { env } from '@/config'
import type { Api } from './types'
import { mockApi } from './mock'
import { realApi } from './real'

export const api: Api = env.useMock ? mockApi : realApi
export type { Api, CreateTaskInput, CreateShareInput, TemplateQuery, UploadInput } from './types'
