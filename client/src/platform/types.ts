export interface AuthPlatform {
  /** Returns the identity provider name and a one-time code for POST /v1/auth/login. */
  login(): Promise<{ provider: string; code: string }>
  checkSession(): Promise<boolean>
}
export interface AdsPlatform {
  readonly simulated: boolean
  configure(adUnitId: string): void
  preload(): void
  /** Resolves when the ad closes. `ended` is true only for a complete view. Rejects with AD_LOAD_FAILED. */
  show(): Promise<{ ended: boolean }>
}
export interface SharePlatform {
  /** true when the platform renders share via <button open-type="share"> (WeChat). */
  readonly nativeChatShare: boolean
  readonly timeline: boolean
  /** Direct share for platforms with an imperative API (App). */
  share(payload: { title: string; path: string; imageUrl: string; scene: 'chat' | 'timeline' }): Promise<void>
  copyLink(path: string): Promise<void>
}
export interface PrivacyPlatform {
  /** Ensures platform-level privacy authorisation (WeChat privacy popup); rejects with the platform error otherwise. */
  ensureAuthorized(): Promise<void>
}
export interface SubscribePlatform {
  request(templateId: string): Promise<boolean>
}
