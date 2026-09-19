# Sharing

Sharing is the main organic acquisition channel for a Mini Program and the PRD positions portraits as "shareable". This document defines what is shared, where, how it is tracked, and how privacy is protected (D-13, D-22).

## 1. Goals

1. Every share lands a new user in a flow that ends in "生成同款", not on a static page.
2. The user's own photo or work is shared only when the user explicitly chooses "分享作品"; works stay private otherwise.
3. Every share is attributable, so PRD §2 questions ("which module spreads") can be answered.
4. Zero canvas code in the client: posters are rendered server-side so they look the same on every platform, including the future App.

## 2. Share surfaces

| Surface | Where | What is shared | Landing |
|---|---|---|---|
| **Template card** (WeChat chat share) | template detail, template list long-press, result page "分享模板" | title + template cover (5:4 crop) | `template-detail?id=&s=` |
| **Work card** (WeChat chat share) | result page / work detail "分享作品" | title + the user's work thumbnail with AI label (5:4) | `template-detail?id=&s=` (or `spec-library?spec=&s=` for ID photos) with a "好友用它生成了" hero |
| **Poster** (image to Moments / other apps) | result page / work detail "生成海报" | 750×1334 poster: work image, template name, slogan, mini program code | scan → `template-detail?id=&s=` |
| **Moments** (`onShareTimeline`) | template detail, module tabs | title + cover | single-page mode of the same path |
| **Module tab** | three content tabs (the 写真 tab covers 写真 and 头像) | tab banner card | tab path; 头像 uses `/pages/portrait/index?seg=avatar` (D-24) |
| App (later) | same entry points via `uni.share` | same payloads | universal link → same paths |

Not offered: sharing the raw work image file from the app (users can save to album and share manually), sharing another user's work, public galleries.

## 3. Flow

```
user taps 分享作品 / 分享模板 / 生成海报
   │
   ▼
POST /v1/shares { type: template|work|poster|tab, template_id?, spec_id?, work_id? }
   │   server: create share row (ULID), build path with s=<share_id>,
   │           for work/poster: copy work thumb to shares/{id}.jpg (private, 24 h signed URL, AI label burned in)
   │           for poster: render poster (worker or inline), return signed URL
   ▼
client: y-share-sheet
   ├─ 发送给朋友 → onShareAppMessage returns { title, path, imageUrl }   (must be user-triggered button open-type="share")
   ├─ 分享到朋友圈 → onShareTimeline (template/tab only)
   └─ 保存海报 → download poster → saveImageToPhotosAlbum
   │
   ▼
receiver opens path with s=
   ├─ App.vue onLaunch/onShow: read s (query or scene from mini program code) → POST /v1/shares/{s}/open
   ├─ landing page renders normally; when the share is a work share, shows the shared preview as hero
   └─ if the receiver later logs in for the first time → users.acquired_share_id = s (attribution)
```

Attribution is last-touch, kept for 7 days from open to first login, stored on the user.

## 4. Client implementation (uni-app)

- `composables/usePageShare(getPayload)` registers `onShareAppMessage` and `onShareTimeline` (from `@dcloudio/uni-app`) on a page and lazily creates the share row when the user taps the share button, so no share rows exist without user intent. WeChat requires share to be initiated by a `<button open-type="share">`; the y-share-sheet buttons are those buttons.
- `platform/share.mp.ts` implements `share(payload)`; `platform/share.app.ts` will call `uni.share` with `provider: 'weixin'`, `scene: 'WXSceneSession' | 'WXSceneTimeline'`.
- Share images: `imageUrl` must be a public HTTPS URL on a whitelisted domain; 5:4 aspect (500×400 recommended). Template covers get a pre-generated 5:4 variant in the admin asset pipeline; work shares use the server-made 5:4 copy.
- Poster preview: `y-poster-preview` shows the rendered poster and a "保存到相册" button; saving needs album permission (album service handles denial).
- Deep-link parsing: `s` from `options.query` (chat share) or `options.scene` (mini program code, `decodeURIComponent`, format `s=<id>`). Route to the page with the same params; keep the params out of `flow.ts`.

## 5. Server implementation (Go)

- `service/share`: create, open, poster; ownership check for work shares; `shares.preview_key` copy made at creation with the AI label burned in.
- Poster rendering (`pipeline/poster`): 750×1334 canvas via `imaging` + `golang/freetype` for text; layout: work image (top, 3:4 for pro/portrait, 1:1 for avatar, spec ratio for ID photo), template name, one-line slogan from config, mini program code (`wxacode.getUnlimited`, `scene = "s=<share_id>"`, `page = pages/template-detail/index`), brand mark. Cached at `shares/{id}-poster.jpg`; regenerated only if the work changes.
- `wxacode.getUnlimited` is called with the app access token through `provider/wechat`; scene ≤ 32 chars, so the share id is a ULID (26 chars) prefixed by `s=`.
- Revocation: `DELETE /v1/shares/{id}` marks the share revoked and deletes the preview copy and poster; the landing page falls back to the plain template page.
- Deleting a work revokes every share that references it.

## 6. Tracking and metrics

Events (client): `share_click {surface, type}`, `share_sent {type}` (mp-weixin cannot confirm send; record the tap), `poster_saved`, `share_open {share_id}` (server-side via `/open`), `share_attributed` (server-side at first login).

Admin dashboard: shares by type and module, opens per share, new users per share, conversion (open → task created → work saved), top shared templates. Data lives in `shares`, `share_opens` and `events`.

## 7. Incentive (D-22, enabled at launch)

`config.share_reward_enabled = true` from V1.0 (product decision 2026-09-11). The sharer receives **+1 bonus credit** when a user acquired through their share completes a **first successful gen task**.

Rules:

- One reward per acquired user: ledger kind `share_reward`, `ref_type = user`, `ref_id = acquired_user_id`, unique.
- `share_reward_daily_cap` (default 3) per sharer per day; rewards beyond the cap are silently not granted and logged.
- No self-referral: the acquired user must not share a `unionid` or `device_id` with the sharer; the sharer's own device ids are recorded from `share_opens`.
- Only a **gen task** (`uses_genmodel = true`) counts. Since D-26 every task is one, so the reward always costs the acquired user one ad view or daily credit first.
- No reward for opens or installs, which are easy to fake and worthless.
- The switch stays in config so ops can pause it if abuse shows up; pausing does not revoke credits already granted.

Where the reward is granted: in the worker's `notify:task-finished` handler after a task reaches `success`, in one transaction that checks `users.acquired_share_id`, the uniqueness key and the cap, then writes the ledger row and enqueues a subscribe message "邀请奖励到账" to the sharer (if granted).

The PRD asks to avoid complex point systems; this is one rule and one ledger row.

## 8. Privacy and compliance

- Work and poster shares copy the image only after the user taps "分享作品" or "生成海报"; the copy is the only public object and carries the visible AI label (COMPLIANCE.md §3).
- The share preview is deleted on revoke, on work deletion, and 90 days after the last open (config).
- Share landing never reveals the sharer's identity beyond an optional nickname line "好友分享" (nickname off by default: `config.share_show_nickname = false`).
- Poster copy and slogans are covered by the banned-word list (COMPLIANCE.md §6).

## 9. UI

- `y-share-sheet`: title "分享", three actions (发送给朋友 / 分享到朋友圈 / 保存海报) plus a preview of the card image; on the result page a first-row toggle "分享作品 / 仅分享模板" defaulting to "仅分享模板". One caption line states the reward truthfully: "好友通过你的分享完成首次生成，你获得 1 次生成机会（每日最多 3 次）", numbers from config.
- Credits page: a row "邀请好友" with the same caption, the count of rewards earned, and a button that opens the share sheet for the current module's hot template.
- `y-poster-preview`: full-screen poster with "保存到相册" and "取消".
- Landing hero for work shares: 3:4 image with the AI label, caption "好友用「{template}」生成了这张，试试同款", primary button "上传照片生成同款".
- Design specs in DESIGN_SYSTEM.md §4.

## 10. Rollout

V1.0 ships template share, work share, poster, tracking and the incentive, all enabled at launch. The funnel dashboard (shares → opens → acquired users → first gen task → rewards) is watched daily during launch week; the reward cap or the switch is adjusted from config without a release if abuse or cost problems appear.
