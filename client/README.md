# 映己 client (uni-app)

Vue 3 + TypeScript + Vite, built with uni-app. Target `mp-weixin` first; `app-plus` and `h5` later. Architecture: `../docs/FRONTEND_ARCHITECTURE.md`.

## Run

```bash
pnpm install
pnpm dev:h5            # browser preview at http://localhost:5173 (mobile viewport recommended)
pnpm dev:mp-weixin     # writes dist/dev/mp-weixin; import that folder in WeChat DevTools
pnpm build:mp-weixin   # production bundle in dist/build/mp-weixin
pnpm type-check
```

WeChat DevTools: 导入项目 → 选择 `client/dist/dev/mp-weixin`（开发）或 `client/dist/build/mp-weixin`（发布），AppID 填测试号或正式 AppID。`src/manifest.json` → `mp-weixin.appid` 也要填。

## Mock mode

`VITE_APP_ENV=dev` (default) runs against the in-memory mock API (`src/api/mock`), so every screen works without the Go backend:

- credits, rewarded video (simulated modal when no ad unit is configured), tasks with staged progress, works, records, favorites, shares
- upload page shows "使用示例照片" so the flow can be exercised without picking a file
- 我的 → 设置 has a dev panel: simulate failure, reject next photo, toggle ads, reset/add credits

## Running against the Go backend

```bash
VITE_USE_MOCK=false VITE_API_BASE=http://localhost:8090 VITE_DEV_PORT=5175 pnpm dev:h5
```

`VITE_API_BASE` overrides the host for the current env; `VITE_DEV_PORT` sets the H5 dev port (the uni
CLI does not forward `--port`, so it is read in `vite.config.ts`). The backend's `CORS_ORIGINS` must
include the origin you use. For a staging host instead: `VITE_APP_ENV=staging pnpm dev:mp-weixin`.

## Layout

```
src/
├── pages/            tab pages + generation flow
├── pages-mine/       subpackage: works, records, favorites, photos, credits, profile, settings
├── components/       y-* components (easycom), see ../docs/DESIGN_SYSTEM.md
├── platform/         the only place that touches wx.* / platform APIs (#ifdef)
├── api/              typed client; mock/ and real.ts implement the same contract
├── store/            Pinia: user, catalogue, flow, tasks
├── composables/      useHome, useGenerate, usePolling, useShare, useAnalytics
├── config/           env + ad unit / subscribe template ids
├── utils/            icons (SVG data URIs), errors (all copy), format
└── static/           tabbar icons, mock sample images
```

## Rules

- Never call `wx.*` outside `src/platform/*.mp.ts`.
- Never name a component event `tap` or `click`: uni-app binds those as native events on the component root and the handler receives a DOM event instead of the payload. Use `select`, `press`, `change`, etc.
- Never set `Content-Type` on an upload: `uni.uploadFile` must generate its own multipart boundary. `api/http.ts` keeps `baseHeaders()` (auth and tracing only) separate from `headers()` (adds JSON) for this reason.
- All user-facing error copy lives in `src/utils/errors.ts`.
- Sizes render as `25×35 mm` / `295×413 px` via `src/utils/format.ts`.
