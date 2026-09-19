# Compliance — Privacy, AI Content, WeChat Review, Advertising

This product processes facial images and produces AI-generated portraits. Both are regulated in China and both are checked in WeChat's review. This document lists the obligations and how the system meets them. Items marked **Legal review** need confirmation by counsel before launch; the technical default is stated so engineering is not blocked.

## 1. Applicable rules (as of 2026-09)

| Rule | Relevance |
|---|---|
| 个人信息保护法 (PIPL) | Facial images are **sensitive personal information**: separate consent, purpose limitation, minimum retention, deletion rights |
| 微信小程序用户隐私保护指引 / 隐私授权 API | Mini Programs must declare collected data in the MP console and obtain in-app authorisation before using camera, album, or uploading user images |
| 生成式人工智能服务管理暂行办法 | Generative AI services must use filed (备案) models or be filed themselves, moderate content, and label output |
| 人工智能生成合成内容标识办法 + GB 45438-2025 (effective 2025-09-01) | Explicit (visible) and implicit (metadata) labels on AI-generated images |
| 互联网信息服务深度合成管理规定 | Face generation/editing is "deep synthesis"; explicit labelling and consent |
| 广告法 | No absolute claims ("最", "每一个标准"), no misleading promises ("3秒生成") |
| 微信小程序平台运营规范 | Category, content security, ad usage, brand asset restrictions |

## 2. Privacy

### 2.1 Declarations (MP console)

- Fill the 用户隐私保护指引 listing: 选中的照片或视频信息 (album), 摄像头, 相册（仅写入）权限 (saving results), 用户上传的照片 (purpose: 生成证件照/写真), 昵称与头像 (user-entered). A missing entry makes `chooseImage` fail with errno 112.
- The photo is processed by third-party models on the NewAPI gateway: a multimodal model checks it at upload and an image model generates the result (D-26). The privacy policy must name this entrusted processing of face images, and the separate consent for sensitive personal information must cover it. **Legal review**.
- Privacy policy and user agreement hosted at a stable URL, opened in `pages/webview`. Both link from Mine tab footer and from the upload page.

### 2.2 In-app authorisation

- `app.json` → `"__usePrivacyCheck__": true`.
- `wx.onNeedPrivacyAuthorization` opens the privacy sheet; `wx.requirePrivacyAuthorize` is called before the first `chooseMedia`.
- The server records `privacy_agreed_at` with the policy version (`POST /v1/me/privacy-agree`); uploads are refused (403) if it is missing.
- The upload page shows a one-line notice: "照片仅用于本次生成，30 天后自动删除，可随时在「照片管理」中删除。"

### 2.3 Data minimisation and retention (D-16)

| Data | Retention | Mechanism |
|---|---|---|
| Uploaded originals | 30 days or user deletion | `photos.expires_at` + daily cleanup job; an OSS lifecycle rule on `yingji/<env>/originals/` as backstop |
| Intermediate layers | not persisted | kept in worker memory. Legacy ID-photo mattes from before D-26 are deleted with the last work that references them |
| Works | until user deletion | user action → soft delete + object delete within 24 h |
| Photo check results (face count, gender, issues) | with the photo | stored in `photos.check_result`, deleted with the photo |
| Provider-side copies | provider-dependent | both the check model and the image model receive the photo through the NewAPI gateway; confirm the gateway's and the upstream models' retention terms before launch |

EXIF (including GPS) is stripped from originals before storage. Originals and works are private objects served only through short-lived signed URLs.

### 2.4 User rights

- Delete a photo, delete a work, delete the account (`DELETE /v1/me`): anonymises the user row, deletes all objects, keeps ledger and task rows without personal data for accounting.
- Export is not offered in V1.0; requests are handled manually via customer service.

### 2.5 Minors

The product does not target minors. The privacy policy states this; no age gate in V1.0. **Legal review**: whether an age statement at first use is required for a face-processing service.

## 3. AI-generated content labelling

Applies to every output, ID photos included: since D-26 all of them are drawn by the gen model.

### 3.1 Implicit label (always)

Every output written by the worker carries metadata per GB 45438-2025: XMP/EXIF fields with `AIGC` marker, producer name, service identifier (mini program AppID), and content ID (work id). Implemented in `pipeline/label`.

### 3.2 Explicit label

- In-app: every result image renders the `ai-label` chip "AI 生成" and the result page states "本图片由 AI 生成".
- In the saved file: a small text label "AI生成" in the bottom-right corner (`local.DrawBadge`, about 4.5 % of the short side), on every output.
- Share previews and posters (SHARING.md) are copies of works and always carry the visible label burned in, since they leave the app.
- ID photos: the label is currently drawn inside the photo, bottom-right. A visible mark can make an ID photo unacceptable for official submission, so product and legal must decide between keeping it, moving it to a margin strip outside the spec area, or relying on the implicit label for ID photos. **Open decision**.

### 3.3 Model filing (备案)

Generation runs on third-party models. Choose providers whose models are filed under 生成式人工智能服务 and record the filing number in DEPLOYMENT.md. The current default is `gpt-image-2.5` through the NewAPI gateway (GENERATION_PIPELINE.md §8); confirm its filing status before submission, and switch the default to a filed model if it has none. WeChat's review currently asks Mini Programs offering generative AI features to declare the model source and filing information in the MP console; complete this before submission.

## 4. Content moderation (D-15)

- All uploaded originals and all outputs go through WeChat `security.mediaCheckAsync` (scene: profile / social).
- Originals flagged `risky` are rejected; the client shows the generic photo-rejected message.
- Outputs flagged `risky` are hidden; the task fails with `CONTENT_REJECTED` and is refunded.
- Prompts in `templates.gen_config` are reviewed by ops; a nightly report lists templates with a rejection rate above 5 %.

## 5. WeChat review checklist

- Category: 工具 › 图片处理 (confirm current category name in the console). Add 摄影 if required by the reviewer.
- Test account with credits and `ads_enabled=false` so reviewers can generate without ads; put instructions in the review notes.
- Server domains whitelisted: API (`request`, `uploadFile`), OSS public endpoint (`downloadFile`).
- No WeChat brand assets in icons (UI-07).
- Privacy authorisation must appear before any photo access; reviewers test this.
- Subscribe message template approved before use.
- Rewarded video ad unit created only after 流量主 approval; the app must behave correctly with `ads_enabled=false`.
- Customer service button (`open-type="contact"`) enabled and answered.
- Copy passes the Advertising Law check in §6.

## 6. Advertising and copy rules

Banned words in product copy and templates: 最, 第一, 官方, 每一个标准, 100%, 3秒生成, 保证通过. Copy review is part of the design-system content rules (DESIGN_SYSTEM.md §6) and admin template creation shows a warning when a banned word appears.

Rewarded video usage follows WeChat's policy: the reward is a feature credit, the ad is never auto-played, the user always taps to start, and the "watch to unlock" prompt describes the reward truthfully.

## 7. Launch checklist

- [ ] Privacy policy and user agreement published; versions recorded.
- [ ] MP console privacy declarations complete; `__usePrivacyCheck__` verified on a real device.
- [ ] Retention jobs and the OSS lifecycle rule on this project's `originals/` prefix enabled in production (never a bucket-wide rule: the bucket is shared).
- [ ] Implicit labels verified with an XMP reader; explicit labels visible on sample outputs, share previews and posters.
- [ ] Provider filing numbers recorded; MP console AI declaration filled.
- [ ] mediaCheckAsync callback tested end to end.
- [ ] Copy audit done against the banned-word list.
- [ ] Review test account and notes prepared.
