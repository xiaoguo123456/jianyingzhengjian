# Frontend Architecture — uni-app (Vue 3 + TypeScript)

## 1. Framework choice (D-19)

**uni-app, Vue 3, TypeScript, Vite CLI project.** First build target is `mp-weixin`; `app-plus` (Android/iOS) and `h5` are reachable later from the same codebase.

| Criterion | uni-app (chosen) | Native WeChat | Taro |
|---|---|---|---|
| Future native App | Same code, `build:app`; uni-AD for rewarded video on App | Rewrite | Taro RN is less mature than uni-app App runtime |
| WeChat-specific APIs (ads, subscribe messages, privacy) | `uni.*` wrappers for most; `wx.*` escape hatch inside `#ifdef MP-WEIXIN` | First-class | Similar to uni-app |
| Team stack | Vue 3 + Pinia, large Chinese ecosystem | WXML/WXSS, small surface | React |
| Toolchain | Vite, fast HMR, CI-friendly CLI | DevTools only | Webpack/Vite |
| Risk | Wrapper lag on brand-new WeChat APIs; mitigated by the escape hatch | None | Same as uni-app |

The cost of the choice is one extra layer between the code and WeChat. The mitigation is a **platform layer** (§5): every platform-specific capability lives behind one interface with per-platform implementations selected by conditional compilation. Business code never calls `wx.*` directly.

Minimum WeChat base library: `2.32.3` (privacy APIs). uni-app: latest 3.x (Vue 3). Node 20.

## 2. Toolchain

- Scaffold: `npx degit dcloudio/uni-preset-vue#vite-ts client` (no HBuilderX required for Mini Program builds; HBuilderX or the offline SDK is needed only when App packaging starts).
- Scripts: `dev:mp-weixin`, `build:mp-weixin`; later `build:app`, `build:h5`.
- State: Pinia.
- Styling: SCSS; design tokens as SCSS variables in `uni.scss` and CSS custom properties in `App.vue` (DESIGN_SYSTEM.md).
- Components: hand-written per the design system, auto-registered with `easycom`. `@dcloudio/uni-ui` is allowed for non-visual helpers only (`uni-popup`, `uni-transition`, `uni-load-more`).
- Lint/format: ESLint (`eslint-plugin-vue`, `typescript-eslint`) + Prettier. Type check with `vue-tsc --noEmit`.
- Tests: Vitest for `api/`, `store/`, `utils/`; `@dcloudio/uni-automator` for end-to-end on `mp-weixin`.
- CI: `build:mp-weixin` → `miniprogram-ci` preview/upload from `dist/build/mp-weixin` (DEPLOYMENT.md).

## 3. Project layout

```
client/
├── src/
│   ├── main.ts                     # createSSRApp, Pinia, global error handler
│   ├── App.vue                     # launch: auth bootstrap, config fetch, ad preload
│   ├── pages.json                  # routes, tabBar, subPackages, easycom, custom nav per page
│   ├── manifest.json               # appid per platform, mp-weixin settings, app-plus modules (later)
│   ├── uni.scss                    # design tokens (SCSS)
│   ├── env/                        # env.dev.ts / env.staging.ts / env.prod.ts, picked by VITE_APP_ENV
│   ├── pages/
│   │   ├── idphoto/index.vue       # Tab 1
│   │   ├── pro/index.vue           # Tab 2
│   │   ├── portrait/index.vue      # Tab 3: segments 写真 / 头像 (D-24)
│   │   ├── mine/index.vue          # Tab 4
│   │   ├── spec-library/index.vue
│   │   ├── template-list/index.vue # module / category / collection lists; masonry for portrait
│   │   ├── template-detail/index.vue
│   │   ├── upload/index.vue        # shared upload + photo check + picker sheet (D-03)
│   │   ├── idphoto-params/index.vue
│   │   ├── confirm/index.vue
│   │   ├── generating/index.vue
│   │   ├── result/index.vue
│   │   └── webview/index.vue       # privacy policy, agreements
│   ├── pages-mine/                 # subPackage
│   │   ├── works/index.vue  work-detail/index.vue  records/index.vue
│   │   ├── favorites/index.vue  photos/index.vue  credits/index.vue
│   │   ├── profile-edit/index.vue  settings/index.vue
│   ├── components/                 # easycom: `y-` prefix
│   │   ├── y-nav-bar/ y-banner/ y-section-header/ y-spec-card/ y-category-card/
│   │   ├── y-feature-strip/ y-template-card/ y-template-row/ y-primary-button/
│   │   ├── y-credit-bar/ y-ad-sheet/ y-picker-sheet/ y-photo-check-result/
│   │   ├── y-stage-indicator/ y-list-row/ y-empty-state/ y-skeleton/ y-error-state/
│   │   ├── y-ai-label/ y-share-sheet/ y-poster-preview/
│   ├── platform/                   # the only place with #ifdef and wx.* calls (see §5)
│   │   ├── index.ts                # exports typed singletons: auth, ads, share, privacy, subscribe, media, album, nav
│   │   ├── auth.mp.ts  auth.app.ts
│   │   ├── ads.mp.ts   ads.app.ts
│   │   ├── share.mp.ts share.app.ts
│   │   ├── privacy.mp.ts privacy.app.ts
│   │   ├── subscribe.mp.ts subscribe.noop.ts
│   │   ├── media.ts  album.ts  nav.ts   # cross-platform uni.* with small #ifdef branches
│   ├── api/                        # typed HTTP client, one file per API area (API.md)
│   │   ├── http.ts  auth.ts  catalogue.ts  photos.ts  tasks.ts  credits.ts  works.ts
│   │   ├── favorites.ts  records.ts  shares.ts  events.ts
│   ├── store/                      # Pinia
│   │   ├── user.ts      # profile, token, credits, client config
│   │   ├── catalogue.ts # per-module home cache with TTL
│   │   ├── flow.ts      # current generation attempt (single source of truth)
│   │   ├── tasks.ts     # active task ids
│   │   └── ui.ts        # cross-page UI state (写真 tab segment)
│   ├── composables/                # usePolling, useShare, usePageShare, useSafeArea, useAsyncState
│   ├── utils/                      # rpx, format, spec (mm/px), errors (all user-facing copy), uuid
│   └── static/                     # tabbar icons, images (small only; large assets come from OSS)
├── tests/                          # vitest unit, automator e2e
├── package.json  vite.config.ts  tsconfig.json  .eslintrc.cjs
```

## 4. Navigation model

- `pages.json`: four tab pages with `navigationStyle: "custom"` and the `y-nav-bar` component (large title, capsule-safe on Mini Program, status-bar-safe on App). All other pages use the default navigation bar.
- Native `tabBar` (4 items: 证件照 / 职业照 / 写真 / 我的; 头像 is a segment inside 写真, D-24). No custom tab bar component. Icons are 162×162 PNGs rendered from the line-icon set, filled when selected.
- `uni.switchTab` rejects query strings, so `platform/nav.ts` strips them from tab paths and segment selection goes through the `ui` store. `utils/routes.ts` maps a module to its tab path and builds share links.
- `subPackages`: everything under `pages-mine/` to keep the main package well under WeChat's 2 MB limit; `preloadRule` preloads it when the Mine tab is shown.
- Page stack depth stays ≤ 10: `generating` → `result` uses `uni.redirectTo`; "换一个模板" uses `uni.navigateBack`.
- Deep links: `template-detail?id=&s=` (share), `result?task_id=` (subscribe message), `spec-library?category=`, `portrait?seg=avatar&s=` (tab share into the 头像 segment). Scene parameters from mini program codes are decoded in `App.vue` `onLaunch`/`onShow` and routed through `platform/nav.ts`.

## 5. Platform layer

Each capability exports one interface and picks an implementation at compile time:

```ts
// platform/index.ts
// #ifdef MP-WEIXIN
export { auth } from './auth.mp'
export { ads } from './ads.mp'
export { share } from './share.mp'
export { privacy } from './privacy.mp'
export { subscribe } from './subscribe.mp'
// #endif
// #ifdef APP-PLUS
export { auth } from './auth.app'
export { ads } from './ads.app'
export { share } from './share.app'
export { privacy } from './privacy.app'
export { subscribe } from './subscribe.noop'
// #endif
export { media } from './media'
export { album } from './album'
export { nav } from './nav'
```

| Capability | Interface | Mini Program impl | App impl (later) |
|---|---|---|---|
| Auth | `login(): Promise<{provider, code}>` | `uni.login({provider:'weixin'})` → `provider: 'wechat_mp'` | WeChat OAuth via `uni.login` (oauth module) → `wechat_app`, or phone login |
| Ads | `preload()`, `show(): Promise<{ended: boolean}>`, `onError` | `uni.createRewardedVideoAd({adUnitId})` (wraps `wx`) | same API via uni-AD with App ad unit |
| Share | `usePageShare(payload)`, `poster(shareId)` | `onShareAppMessage` / `onShareTimeline` from `@dcloudio/uni-app` | `uni.share` with WeChat SDK; poster saved to album |
| Privacy | `ensureAuthorized(): Promise<boolean>` | `wx.requirePrivacyAuthorize`, `wx.onNeedPrivacyAuthorization` → privacy sheet | in-app consent dialog stored server-side |
| Subscribe | `request(templateId): Promise<boolean>` | `uni.requestSubscribeMessage` | no-op (push notifications later) |
| Media | `pickImage(): Promise<File>` | `uni.chooseImage({count:1, sourceType:['album','camera'], sizeType:['original']})`, compress > 4 MB via `uni.compressImage` | same |
| Album | `save(url)` | `uni.downloadFile` → `uni.saveImageToPhotosAlbum`; on denial open `uni.openSetting` | same |
| Nav | safe areas, capsule rect, `openWebview` | `uni.getMenuButtonBoundingClientRect` in `#ifdef MP-WEIXIN` | `statusBarHeight` |

The backend login endpoint already accepts `provider`, and identities are stored per provider (DATA_MODEL.md `user_identities`), so adding the App does not touch the user model.

## 6. State management

Pinia stores hold server state and flow state; presentational state stays in components.

- `flow.ts`: `{ module, kind, templateId | specId, photoId, params, idempotencyKey, notifyRequested }`. All flow pages read from it; URL params carry only IDs for deep links.
- `user.ts`: token (persisted via `uni.setStorageSync`), profile, credits, client config (`ads_enabled`, ad unit ids, template ids, share config). Credits refresh after task creation, ad claim, and Mine `onShow`.
- `catalogue.ts`: home payload per module, 60 s TTL; template detail cache.
- `tasks.ts`: active task ids for `generating` and `result`.

## 7. Networking (`api/http.ts`)

- Base URL from `env/`; the Mini Program release build also checks `uni.getAccountInfoSync().miniProgram.envVersion` to prevent a trial build from hitting prod.
- Headers: `Authorization: Bearer`, `X-Request-Id`, `X-Client-Version`, `X-Platform` (`mp-weixin` / `app-android` / `app-ios`).
- Envelope mapping to `ApiError`; toast unless `silent`. 401 → `auth.relogin()` once, retry once. 402 → return to caller to open `y-ad-sheet`.
- Timeouts: 15 s JSON, 60 s upload. Uploads via `uni.uploadFile` to `POST /v1/photos`.
- All hosts whitelisted in the MP console (`request`, `uploadFile`, `downloadFile`).

## 8. Generation flow on the client

```
confirm.onLoad
  ├─ subscribe.request(taskFinishedTemplateId)      # optional, non-blocking, result stored in flow
  ├─ credits.get()
  └─ ads.preload() if ads_enabled and total == 0

confirm.onGenerate
  ├─ tasks.create(flow)  ─ 201 ─▶ generating?task_id
  ├─                     ─ 402 ─▶ y-ad-sheet.open()
  │        onWatch → ads.show()
  │          ├─ ended  → credits.claim(sessionId) → tasks.create(flow) (same idempotency key)
  │          └─ !ended → toast "未完整观看，未获得次数"
  └─                     ─ 503 ─▶ toast "生成服务暂不可用"

generating  (usePolling: 2 s, 5 s after 30 s, stop at 5 min)
  ├─ stage label from task.stage
  ├─ success → uni.redirectTo result?work_id
  └─ failed  → error state with reason + "次数已返还" + retry

result
  ├─ 保存图片 / 再生成一张 / 换一个模板 / (ID photo) 换背景 (free, instant) / 换服装 (1 credit)
  └─ 分享 → y-share-sheet (SHARING.md)
```

## 9. Performance rules

- Home payload cached; tab switch instant after first load.
- Lists paginate at 20 with `onReachBottom`; portrait masonry uses two columns with lazy images.
- Images are signed OSS URLs (works 10 min, catalogue and share images 24 h), so a page that stays open longer re-fetches its data instead of caching URLs. Lists use the server-made thumbnails (`thumb_url`) with `lazy-load` and `mode="aspectFill"` for covers.
- Keep `static/` tiny; every catalogue image is remote.
- Main package target < 1.5 MB after build; check with the DevTools code-size report on every release.

## 10. Error and empty states

Every list page renders one of skeleton / empty / error (with retry) / content. Every flow page handles: no network, 401 re-login, permission denied (camera, album), photo rejected, no credits, ad load failure, task failed, task timeout. All copy lives in `utils/errors.ts` and mirrors API.md.

## 11. Testing

- Unit: `api/`, `store/`, `utils/` with Vitest (uni APIs mocked).
- E2E: `uni-automator` script on `mp-weixin`: login, open each tab, run an ID photo flow against staging with `ads_enabled=false`.
- Manual matrix per release: iOS + Android WeChat, low-end Android, ads on/off, album permission denied, network loss mid-generation.

## 12. Implementation notes (learned while building)

- **Component event names.** uni-app treats `tap`/`click` (and other native event names) on a custom component as native listeners on its root element, so `@tap` on `<y-spec-card>` receives a DOM event instead of the emitted payload, and the handler can fire twice. All `y-*` components emit `select`, `press`, `watch`, `retry` and similar names, never `tap` or `click`.
- **`src/config/` not `src/env/`.** The scaffold ships `src/env.d.ts`; a folder named `env` shadows it in TypeScript module resolution.
- **Mock mode.** `VITE_APP_ENV=dev` uses `src/api/mock`, an in-memory implementation of the full API contract with simulated task progression, credits, ads and shares. The upload page offers a "使用示例照片" shortcut and the settings page has a dev panel (simulate failure, reject next photo, toggle ads, reset credits). None of this is compiled when `useMock` is false.
- **Icons.** In-page icons are SVG data URIs generated from `src/utils/icons.ts` (no binary assets, tintable); TabBar icons must be PNG and are generated from the same paths.
- **Sticky button.** `bottom: var(--window-bottom)` keeps the primary button above the TabBar on H5 and Mini Program alike.
- **H5 preview.** `pnpm dev:h5` renders every screen with the mock API; the launch flow, ad sheet, generating and result pages were verified there before the first Mini Program build. `VITE_USE_MOCK=false VITE_API_BASE=…` runs the same screens against the Go backend, which is how the API contract was checked end to end.
- **Upload headers.** `uni.uploadFile` must set its own multipart `Content-Type`; sending the JSON content type alongside it makes the server see a non-multipart body. `api/http.ts` keeps auth/tracing headers (`baseHeaders`) separate from the JSON header for this reason.
- **Dev port.** The uni CLI does not forward `--port`, so the H5 dev port is read from `VITE_DEV_PORT` in `vite.config.ts`.

## 13. Preparing for the App build (not in V1.0)

- Keep all `wx.*` usage inside `platform/*.mp.ts`; CI fails the build if `wx.` appears elsewhere (simple grep step).
- Use `uni.chooseImage`, `uni.saveImageToPhotosAlbum`, `uni.request` rather than `wx` equivalents.
- Design components with `rpx` and safe-area insets; no assumption of the capsule.
- Backend: identities per provider, `X-Platform` header, ad claims keyed by ad unit so App ad units can be added.
- Compliance for App (app store listings, SDK privacy declarations) is a separate track.
