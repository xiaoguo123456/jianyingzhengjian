# Design System

Derived from the V1.0 mockups in `ui/`, PRD sections 39–40, and the fixes listed in `ui/UI_REVIEW.md`. Values are given in design points (375 pt wide artboard) and in rpx (750 rpx = 375 pt, so 1 pt = 2 rpx).

## 1. Principles

1. **One blue system.** A single primary blue, one gradient for the primary action, tinted circles for icons. No second accent colour.
2. **Information over decoration.** Every text line states a fact (size, output, what changes) or is removed. No slogans on cards, no script-font annotations, no "AI" pills.
3. **Same skeleton on every tab.** Title → hero (the only call to action) → primary grid of photo tiles → template rails → TabBar (PRD 39). No feature strips, no sticky button: one CTA per screen.
4. **One chevron.** `›` only on list rows and "更多 ›" links. Never on grid cards or buttons.
5. **Template cards are image + name.** Tags allowed: `NEW`, `热门`.

## 2. Tokens

Implemented as SCSS variables in `uni.scss` and CSS custom properties in `App.vue`, consumed by every component.

### 2.1 Colour

| Token | Value | Use |
|---|---|---|
| `--color-primary` | `#2F7BF6` | active tab, links, icon strokes, chevrons on hot links |
| `--color-primary-strong` | `#1F66E0` | pressed state |
| `--color-primary-soft` | `#E8F1FF` | icon circle background, credit banner base |
| `--color-primary-tint` | `#F2F7FF` | selected card background |
| `--gradient-primary` | `linear-gradient(90deg, #3A8DFF 0%, #35C3E6 100%)` | primary button, "看视频领次数" button |
| `--gradient-banner` | `linear-gradient(135deg, #DCEBFF 0%, #EAF3FF 60%, #FFFFFF 100%)` | tab banners |
| `--color-bg` | `#F2F4F8` | page background (neutral cool grey; the earlier `#EEF3FA` read too blue next to the photo tiles) |
| `--color-surface` | `#FFFFFF` | cards, sheets, TabBar |
| `--color-border` | `#E6ECF5` | 1 px separators |
| `--color-text` | `#111827` | titles, primary text |
| `--color-text-2` | `#4B5563` | body |
| `--color-text-3` | `#8A94A6` | captions, sizes, placeholders |
| `--color-success` | `#22C55E` | check passed |
| `--color-warning` | `#F59E0B` | photo quality warnings |
| `--color-danger` | `#EF4444` | failed, delete |
| `--color-mask` | `rgba(17,24,39,0.45)` | sheet backdrop |
| `--tab-inactive` | `#6B7280` | TabBar inactive icon and label |

ID photo background swatches: white `#FFFFFF`, blue `#438EDB`, red `#FF0000`, grey `#808080`. These are content values from the spec library, not theme tokens.

### 2.2 Typography

System font stack (`-apple-system, PingFang SC, Helvetica Neue, Microsoft YaHei, sans-serif`). No web fonts.

| Token | Size | Weight | Line height | Use |
|---|---|---|---|---|
| `--font-display` | 20 pt / 40 rpx | 700 | 1.2 | tab page title and 写真/头像 segment labels (reduced from 28 pt on 2026-09-15 so every tab header matches) |
| `--font-h1` | 20 pt / 40 rpx | 700 | 1.3 | banner headline |
| `--font-h2` | 17 pt / 34 rpx | 600 | 1.3 | section titles, template detail name |
| `--font-body-strong` | 15 pt / 30 rpx | 600 | 1.4 | card titles |
| `--font-body` | 14 pt / 28 rpx | 400 | 1.5 | body, list rows |
| `--font-caption` | 12 pt / 24 rpx | 400 | 1.4 | sizes, subtitles, TabBar labels |
| `--font-mono-size` | 12 pt / 24 rpx | 400 | 1.4 | mm/px values (tabular figures via `font-variant-numeric: tabular-nums`) |

### 2.3 Spacing (4 pt grid)

| Token | pt | rpx | Use |
|---|---|---|---|
| `--space-1` | 4 | 8 | inline gaps |
| `--space-2` | 8 | 16 | between text lines |
| `--space-3` | 12 | 24 | card inner padding, grid gap |
| `--space-4` | 16 | 32 | page horizontal margin, between sections' content |
| `--space-5` | 24 | 48 | between sections |
| `--space-6` | 32 | 64 | above primary button |

### 2.4 Radius, elevation, borders

| Token | Value |
|---|---|
| `--radius-sm` | 8 pt (thumbnails, tags) |
| `--radius-md` | 12 pt (cards) |
| `--radius-lg` | 16 pt (banner, sheets) |
| `--radius-pill` | 999 pt (buttons, pills) |
| `--shadow-card` | `0 2px 8px rgba(47,123,246,0.06)` |
| `--shadow-sheet` | `0 -4px 16px rgba(17,24,39,0.08)` |
| border | 1 px `--color-border` on white cards over the tinted background is optional; shadow alone is the default |

### 2.5 Iconography

- Single set: 24 pt line icons, 1.75 pt stroke, rounded caps, drawn in `--color-primary`, exported as SVG then PNG @2x/@3x.
- Placed on a 48 pt circle filled `--color-primary-soft` for category and feature cards.
- TabBar icons: 24 pt, outline inactive (`--tab-inactive`), filled primary active.
- Never use third-party brand marks (WeChat logo) as icons.

### 2.6 Motion

- Page transitions: platform default.
- Sheets: 240 ms ease-out slide up, backdrop fade.
- Generating page: looping stage animation (Lottie or CSS), stage label crossfade 200 ms.
- Button press: opacity 0.85, no scale.

## 3. Layout

- Safe areas: custom nav on tab pages respects `statusBarHeight` and the capsule rect; content bottom padding `env(safe-area-inset-bottom)` plus TabBar height.
- Grid: page margin 16 pt; 4-column card grid with 12 pt gutters (card width ≈ 76 pt); 2-column template grid with 12 pt gutters; portrait "more" page uses a 2-column masonry.
- Tabs have no sticky button. The hero card carries the CTA; flow pages (template detail, params, confirm) keep a sticky primary button.
- Horizontal rails bleed to the screen edges (negative page margin) with 16 pt edge padding and 10 pt gutters; card width 132 pt (3:4) or 120 pt (1:1) so ~2.5 cards are visible.

## 4. Components

| Component | Anatomy | States |
|---|---|---|
| `nav-bar` | large title 20 pt, capsule-safe right area, 10 pt bottom spacing; tabs use no subtitle. Segment variant: two 20 pt labels on one baseline, active 700 in `--color-text` with a 14×3 pt primary bar below, inactive 500 in `--color-text-3` | default, segment |
| `hero` (`y-banner`) | white→light-blue card 16 pt radius, headline h1, one factual subtitle, gradient pill button (立即制作 / 上传照片生成), right-side photo 104×138 pt with 12 pt radius and soft shadow; whole card tappable | loading skeleton |
| `section-header` | h2 left, optional `更多 ›` caption right in `--color-text-3` | — |
| `spec-card` | aspect-ratio glyph in the spec's background colour with a white person silhouette, name body-strong, `25×35 mm` (text-2), `295×413 px` (text-3), tabular nums | default, selected (tint bg + primary border) |
| `category-card` | photo tile 1:1.28 with the name on a bottom scrim (white 14 pt 600); icon-circle variant only when no cover exists | default, pressed |
| `template-rail` | horizontal scroll of `template-card`s, 132 pt wide (120 pt square), 10 pt gutters, edge-bleed | — |
| `template-card` | cover 3:4 (1:1 for avatar) 12 pt radius, no card chrome, name 14 pt 600 below; tag pill top-left (热门 = white pill dark text, NEW = primary) | default, favorited (heart) |
| `template-row` | 88×117 pt cover, name, one factual caption, chevron right | — |
| `primary-button` | full width 48 pt, gradient, white 16 pt 600 text, leading icon optional; disabled = 40 % opacity | default, loading (spinner), disabled |
| `credit-bar` | white card 16 pt radius: caption "生成次数", number 32 pt 700 with unit "次" on one baseline, one-line breakdown caption, right gradient pill "看视频 +1 次" (hidden when ads are off or the daily cap is reached; chevron instead when tappable). Optional bottom row with a divider: gift icon + invite reward copy + chevron | ads on / off, tappable, with invite row |
| `ad-sheet` | title "免费生成", body "观看一段视频即可获得 1 次生成机会", primary button "看视频免费生成", secondary "取消" | loading, error copy |
| `picker-sheet` | list of specs or template cards after upload-first entry (D-03) | — |
| `photo-check-result` | preview, pass/fail badge, reason list, buttons "重新上传 / 继续" | pass, fail |
| `stage-indicator` | three stages with the active one highlighted, no percentage | — |
| `list-row` | 24 pt icon, label body, chevron | — |
| `empty-state` | 96 pt illustration, one line, optional button | — |
| `skeleton` | grey blocks matching card shapes | — |
| `ai-label` | 10 pt caption chip "AI 生成" bottom-right of result images | — |
| `share-sheet` | title 分享, optional toggle 分享作品 / 仅分享模板 (default 仅分享模板), 5:4 card preview, three actions 发送给朋友 / 分享到朋友圈 / 保存海报, one caption line with the invite reward from config | loading (poster rendering), permission denied |
| `poster-preview` | full-screen 750×1334 poster, buttons 保存到相册 / 取消 | saving |
| `share-landing-hero` | 3:4 shared image with `ai-label`, caption 好友用「{template}」生成了这张，试试同款, primary 上传照片生成同款 | fallback to plain template detail when revoked |

## 5. Page templates

### Content tab (证件照 / 职业照 / 写真)

The 写真 tab replaces the large title with two segment titles, 写真 and 头像 (D-24). Each segment renders the skeleton below with its own module data; switching scrolls to top.

```
nav-bar (large title only)
hero: headline, factual subtitle, pill CTA, photo          ← the only call to action
section: 常用规格 | 热门场景 | 热门风格 | 头像类型          (4 spec cards / photo tiles, "更多 ›")
idphoto: 常见用途 (spec rows) → 拍摄建议 (compact info card)
others:  推荐模板 | 精选模板 | 热门模板 (template-rail, "查看更多 ›")
         portrait: 专题 (2×2 wide photo cards with name + count)
         0–3 secondary rails from `home.rails` (e.g. 求职面试, 本周新增)
TabBar
```

Rationale (2026-09-15 review of the first build): the banner and the sticky button were the same action twice; the feature strips repeated the hero subtitle and looked tappable; 2×2 template cards were too large for an overview tab. Rails show more templates in less height and read as scrollable.

### Mine tab

```
header on a soft blue wash (no nav title, no card): avatar 64 pt with 3 pt white ring, nickname 20 pt 700, "已生成 N 张作品", translucent white pill "编辑资料"
credit-bar (tappable → credits page), overlapping the wash by 16 pt, with the invite row when share rewards are on
section 我的作品: 4 photo tiles 1:1.28 with name + count on a scrim; empty module = white tile with image icon and "0 张"; header link "全部 N 张 ›"
service card: 4-column icon grid 生成记录 / 收藏模板 / 照片管理 / 联系客服
list card: 设置 (账号与隐私)
footer links: 隐私政策 · 用户协议 (required by COMPLIANCE.md)
TabBar
```

Rationale (2026-09-15): the page title repeated the tab label; a six-row list of equal weight hid the frequent entries; the invite entry belongs with the credits it earns. Icons in list rows now sit in a fixed flex box so they centre on the text line.

### Flow pages

`template-detail`: cover full-width 3:4, name h2, factual line, sample strip, favorite icon top-right, sticky "上传照片生成同款".
`upload`: two large tiles (从相册选择 / 拍照), shooting tips list, privacy note.
`idphoto-params`: preview, background swatches, clothing grid grouped 男士/女士, beauty segmented control (自然 / 轻度).
`confirm`: summary card (target, photo thumb, params), credit-bar state, primary "开始生成".
`generating`: template/spec name, stage-indicator, "可离开页面，完成后通知你".
`result`: image with `ai-label`, actions row (保存图片 / 再生成一张 / 换一个模板), ID photo extra row (换背景, free and instant / 换服装, 1 credit), size caption for ID photo, share row (分享 → `share-sheet`).
`idphoto-params`: options that add a gen-model step (any clothing other than 保持原服装, beauty 轻度) show a caption "消耗 1 次生成机会"; the default combination shows "免费" (D-18, D-21).

### Share assets

- Chat share card image: 5:4, exported at 1000×800; template covers get this variant in the admin asset pipeline; work shares use the server-made copy with the AI label.
- Poster: 750×1334; image area top 750×1000 (aspect-fit on `--color-primary-soft`), template name h2, one slogan line caption, mini program code 160 pt bottom-right, brand mark bottom-left, 32 pt margins. Poster copy comes from config and follows the content rules in §6.

## 6. Content rules

- Sizes always as `25×35 mm` and `295×413 px` with a thin space before the unit, tabular figures.
- Card captions: only facts (size, output aspect, what changes). Banned: 专业形象, 提升印象, 社交必备, 甜蜜出圈, 赢得心仪Offer and similar.
- No absolute or promise words: 每一个标准, 3秒, 官方, 最.
- Button verbs: 上传照片 / 开始生成 / 保存图片 / 再生成一张 / 换一个模板 / 重新上传 / 看视频免费生成.
- Error copy is centralised in `utils/errors.ts` and mirrored in API.md.
- Sample photos: licensed, balanced gender and age mix; ID photo samples use the spec's default background.

## 7. Accessibility and device rules

- Minimum tap target 44 pt; list rows 56 pt.
- Text contrast ≥ 4.5:1 on cards; captions in `--color-text-3` on white pass (5.1:1).
- Support font scaling by using rpx for layout and pt-equivalent rpx for type; no fixed-height text containers.
- Test on 375×667 (small), 390×844, and a 360-wide Android device.
- Dark mode: not supported in V1.0; `darkmode: false` in `app.json`.
