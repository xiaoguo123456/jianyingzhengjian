import type { AuthPlatform } from './types'
/* Dev-only identity for the H5 preview; the real H5 build would use phone login. */
export const auth: AuthPlatform = {
  login: async () => ({ provider: 'h5_dev', code: 'dev' }),
  checkSession: async () => true,
}
