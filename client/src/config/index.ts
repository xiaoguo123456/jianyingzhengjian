/* Environment config. VITE_APP_ENV = dev | staging | prod (default dev).
   USE_MOCK: dev runs fully offline against the in-memory mock API so the UI
   can be reviewed before the Go backend exists. */

type AppEnv = 'dev' | 'staging' | 'prod'

const APP_ENV = ((import.meta as any).env?.VITE_APP_ENV as AppEnv) || 'dev'

const API_BASE: Record<AppEnv, string> = {
  dev: 'http://localhost:8080',
  staging: 'https://api-staging.example.com',
  prod: 'https://api.example.com',
}

/** VITE_API_BASE overrides the host for the current env (handy when 8080 is taken). */
const API_OVERRIDE = (import.meta as any).env?.VITE_API_BASE as string | undefined

export const env = {
  appEnv: APP_ENV,
  apiBase: API_OVERRIDE || API_BASE[APP_ENV],
  useMock: APP_ENV === 'dev' && ((import.meta as any).env?.VITE_USE_MOCK ?? 'true') !== 'false',
  /* Fill from the MP console; empty = rewarded video is simulated (dev only). */
  adUnitIds: { reward: '' },
  subscribeTemplateIds: { taskFinished: '' },
  privacyPolicyUrl: 'https://example.com/privacy',
  userAgreementUrl: 'https://example.com/agreement',
  clientVersion: '0.1.0',
}
