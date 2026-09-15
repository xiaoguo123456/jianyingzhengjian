# UI Review — 映己证件照写真馆 V1.0 Mockups

Reviewed against `docs/映己证件照写真馆_PRD_V1.0.md` (PRD V1.0).
Review date: 2026-09-10. Scope: the five mockups in this folder.

Severity legend:

- **P0** — conflicts with the PRD or blocks implementation; must be fixed before dev handoff.
- **P1** — consistency or usability problem; fix in the next design pass.
- **P2** — polish; fix when convenient.

## 1. Screen map

| File | Screen | PRD sections | Status |
|---|---|---|---|
| `5.png` | ID Photo tab (证件照) | 5–8 | Reviewed |
| `2.png` | Professional Photo tab (职业照) | 9–12 | Reviewed |
| `3.png` | Portrait tab (写真) | 13–16 | Reviewed |
| `4.png` | Avatar tab (头像) | 17–19 | Reviewed |
| `1.png` | Mine tab (我的) | 27–33 | Reviewed |

### Screens the PRD requires but the mockups do not cover

These pages are on the critical path and cannot be built from the current mockups. They need designs before development starts:

1. Spec library (证件照规格库) with category tabs (PRD 5.3).
2. Template list / "more" pages for each module, including the portrait waterfall layout (PRD 16).
3. Collection page (专题) for portrait (PRD 15).
4. Template detail page (PRD 20).
5. Upload page with shooting tips (PRD 21).
6. Photo check result / rejection state (PRD 22).
7. ID photo parameter page: background, clothing, beauty (PRD 8).
8. Rewarded-video prompt dialog and ad-failure state (PRD 23).
9. Generating page with staged status (PRD 24).
10. Result page, including ID-photo-only actions "change background / change clothing" (PRD 25).
11. Works library with category tabs and work detail (PRD 30–31).
12. Generation records, favorites, photo management, credits page, settings, profile edit.
13. Empty, loading, error and permission-denied states for every list and flow.
14. Share sheet (分享作品 / 仅分享模板 toggle, three actions) and poster preview (`docs/SHARING.md`).
15. Share landing hero on template detail when opened from a work share.

## 2. Cross-screen issues

### P0

**UI-01 · "AI" branding contradicts the PRD.**
PRD section 1 says the product "does not emphasise the AI concept". Every banner carries an AI pill (`AI照相馆`, `AI职业照`, `AI写真`, `AI头像生成`) and the profile card in `1.png` reads "用 AI 记录更美的自己". Either remove the pills and the slogan, or the PRD must change. The four pill labels are also inconsistent with each other.

**UI-02 · Filler descriptions the PRD explicitly bans.**
PRD 5.3 forbids copy with no information value ("专业形象", "提升印象"). The mockups are full of it: `2.png` "赢得心仪Offer / 专业职业形象 / 打造专业人设 / 展示企业形象 / 经典专业得体 / 干练职场风格 / 突出职业气质 / 真实自然高级"; `4.png` "社交必备 / 质感头像 / 自然治愈 / 个性艺术 / 甜蜜出圈"; `3.png` "自然真实有氛围 / 突出人物魅力 / 记录特别时刻". Replace with concrete facts (output size, aspect, what changes) or drop the second line.

**UI-03 · Promise and absolute-claim copy.**
`5.png` banner decoration says "3秒生成标准证件照" and "符合每一个标准". Generation runs through three staged states (PRD 24) and will take well over three seconds; "每一个标准" is an absolute claim under China's Advertising Law. Remove both. Same category: `2.png` "让机会主动找上你", `4.png` "遇见更好的自己" are acceptable but the hand-written script layer needs to go (see UI-05).

**UI-04 · Flow entry contradicts the PRD flow.**
All four content tabs put an "Upload photo" primary button at the bottom and make the banner tappable. PRD 4, 7 and 43 all define the order as "pick spec/template → upload". The mockups imply "upload → then what?". The decision taken in `docs/DECISIONS.md` (D-03) is: upload-first entry is allowed and lands on a picker step after upload. The design must add that picker step, or the primary button should navigate to the template/spec list instead.

### P1

**UI-05 · Hand-written decorative text on banners.**
Every banner has a script-font annotation with an arrow ("让好看的你 符合每一个标准", "专业形象加分", "记录更美的自己", "换个头像"). It must be shipped as an image slice, is not configurable from the admin backend (PRD 34 wants banners and hot content to be operable without a release), and is not part of the unified page structure in PRD 39. Recommend removing it and keeping banners as title + subtitle + configurable image.

**UI-06 · Chevron usage is inconsistent (PRD 40).**
`4.png` shows a chevron on every type card, every capability card and on the primary button; `2.png`, `3.png`, `5.png` show none in the same positions. Decide one rule and apply it everywhere. Recommended rule: chevron only on list rows and "more ›" links; never on grid cards or primary buttons.

**UI-07 · Icon style is inconsistent (PRD 39).**
`4.png` avatar-type cards use coloured app-style icons (WeChat green, crown yellow, leaf green, palette, red hearts) on tinted squares, while every other screen uses single-colour blue line icons in a light-blue circle. Use the blue line-icon set everywhere. Also do not use the WeChat logo as an icon; WeChat's review rejects mini programs that reuse WeChat brand assets.

**UI-08 · Primary button label is inconsistent.**
`5.png` "上传照片开始制作"; `2.png` "上传照片生成职业照"; `3.png` "上传照片生成写真"; `4.png` "上传照片生成头像 ›". Use one pattern, recommended "上传照片生成{module}", with no chevron.

**UI-09 · Primary button is not sticky.**
On all four tabs the primary button sits at the end of the scroll content, below the fold on a standard device. Either make it a sticky bottom bar above the TabBar or drop it (the banner and cards already provide the same entry).

**UI-10 · Second row of cards looks tappable but has no destination.**
`5.png` "常用功能", `2.png` "职业风格", `4.png` "风格能力" are, per PRD 6, 11 and 19, explanatory content, not tools. They are drawn identically to tappable cards. Either restyle as a non-interactive feature strip (smaller, no card border) or give them a defined destination in the PRD. `3.png` "专题推荐" is a real entry point and should stay as cards.

**UI-11 · Every screen shows only the populated state.**
No empty states (new user with zero works, no favorites), no loading skeletons, no failure states, no permission-denied states (album, camera). These are needed for implementation.

### P2

**UI-12 · Banner headline pattern differs per tab.**
Three banners use "上传自拍，…" phrasing; `3.png` uses "解锁你的氛围感写真". Align to one pattern.

**UI-13 · Tab bar icon for "我的" is a smiley face.**
"头像" uses a person icon and "我的" a smiley. Users associate a person icon with "me". Recommend person icon for "我的" and a framed-portrait icon for "头像".

**UI-14 · Mockups are iOS-frame based only.**
Status bar 9:41, notch, home indicator. Confirm the design base width (375pt → 750rpx) and provide one Android reference so safe-area and capsule-avoidance rules are unambiguous.

## 3. Per-screen issues

### `5.png` — ID Photo tab

- **P0 UI-15 · Hot templates duplicate common specs.** "热门模板" shows "求职证件照 / 教师资格 / 公务员报名 / 简历标准照" with millimetre and pixel sizes. Three of them are the same 25×35 mm size as "一寸" and "简历照" in the row above. The PRD has no template concept for ID photos (templates are for the other three modules, PRD 20). Replace this block with a "Hot specs" list driven by the spec library, or remove it.
- **P0 UI-16 · Example sizes are not verified.** "公务员报名 35×45 mm / 413×531 px" does not match the national civil-service exam requirement, which is a 3:4 image (commonly 295×413 px). "教师资格 25×35 mm" varies by province. Sizes in the mockup will be copied into seed data; every value must be verified against the issuing authority before it ships.
- **P1 UI-17 · Two of four common specs have the same size.** "一寸" and "简历照" are both 25×35 mm, 295×413 px. Only four slots are allowed (PRD 5.3); use them for four distinct sizes (e.g. 一寸, 二寸, 小二寸, 签证 35×45).
- **P1 UI-18 · All four spec cards use the same icon.** No differentiation. Either show a proportional rectangle representing the aspect ratio, or drop the icon and let the size text carry the card.
- **P2 UI-19 · Banner subtitle differs from PRD 5.2.** PRD: "智能识别 · 自动裁切 · 多种规格"; mockup: "智能识别 · 自动裁剪 · 符合官方规格". Pick one; "符合官方规格" is again an absolute claim.

### `2.png` — Professional Photo tab

- **P1 UI-20 · Template cards carry a slogan line.** PRD 12 defines template cards as "image + name". The second line ("求职面试，赢得机会") is filler. Replace with a factual line or remove.
- **P1 UI-21 · Scene cards vs style cards look identical.** "热门场景" are tappable categories; "职业风格" is explanatory (see UI-10). They must not share the same visual treatment.
- **P2 UI-22 · Sample photos.** Both mockups reuse two faces. Final template covers need licensed, varied samples with a balanced gender mix.

### `3.png` — Portrait tab

- **P0 UI-23 · "氛围滤镜" is not in the PRD.** PRD 15 lists "氛围写真". A "filter" is an image-editing concept and contradicts PRD 42 ("no complex image editor"). Rename back to "氛围写真".
- **P1 UI-24 · Hot styles and featured templates overlap.** "新中式" appears in both rows, "生日写真" vs "清新生日照", "法式氛围" vs "法式街拍". The PRD does not explain how "热门风格" and "精选模板" differ. Either merge them or make "热门风格" a category row (icons or small thumbnails) and "精选模板" the only image row.
- **P1 UI-25 · Every sample is the same woman.** Eight images on one screen show one face. Male users will assume the module is not for them. Mix genders and ages in the samples.
- **P2 UI-26 · Featured template cards contradict PRD 16.** PRD wants a waterfall of image + name only. The horizontal card with description and chevron is acceptable on the tab home, but the "more" page must follow PRD 16.

### `4.png` — Avatar tab

- **P0 UI-27 · Five cards in one row, including couple avatars.** PRD 18 says exactly the opposite: four cards, couple avatar goes to "更多类型". The five cards are visibly cramped; the chevron collides with the label. Reduce to four.
- **P0 UI-28 · Couple avatar needs two faces.** The upload flow (PRD 21–22) is single-person with a face-count check. Couple avatar is deferred to V1.1 (see `docs/DECISIONS.md` D-09). Remove "情侣头像" from both the type row and the hot templates.
- **P1 UI-29 · "一键重生" wording.** PRD 19 says "重新生成". "重生" reads as "rebirth". Use "重新生成".
- **P1 UI-30 · Capability cards look like tools.** PRD 19 states these capabilities "are not independent editing tools". Drawn as tappable cards with chevrons they read as tools. Apply UI-10.
- **P1 UI-31 · Coloured icon set.** See UI-07.
- **P2 UI-32 · Illustration style.** "插画风 / 二次元风" produces stylised output that does not "keep the person's features" (PRD 8.3). Keep it if product wants it, but note it uses a different model path and should be flagged in the template config.

### `1.png` — Mine tab

- **P0 UI-33 · "我的作品" shows three categories, avatar is missing.** PRD 29 lists four. The three counts shown add up exactly to the "已生成 28 张作品" total, so avatar works are simply absent. If the row scrolls horizontally, show a peeking fourth card; otherwise use a 2×2 or 4-column layout.
- **P1 UI-34 · Features not in the PRD.** "编辑资料", "反馈建议", "清理缓存". Profile editing is actually required by WeChat's nickname/avatar APIs and is kept (D-11). "清理缓存" is meaningless for a mini program and overlaps with photo management; remove it. "反馈建议" duplicates "联系客服"; keep one.
- **P1 UI-35 · Three overlapping entries for generated images.** "我的作品", "生成记录", "照片管理 → 生成作品" all list generated images. See D-12 in `docs/DECISIONS.md` for the split: works = successful outputs, records = all tasks including failed, photo management = uploaded originals only. The mockup labels are fine once those definitions hold; the photo-management page must not show generated works.
- **P1 UI-36 · Duplicate entry for profile edit.** The chevron after the nickname and the "编辑资料 ›" button do the same thing. Keep the button, drop the chevron.
- **P2 UI-37 · Credits banner copy.** "今日可免费生成 3 次 / 看视频可继续获得免费生成次数" assumes a daily grant plus ad credits. That matches D-01, but the number 3 must come from config, and the second line should state the daily ad cap once known.
- **P2 UI-38 · "享受更多服务" subtitle.** Filler; replace with something factual or remove.

## 4. Designer checklist for the next pass

1. Remove AI pills, AI slogan and script-font banner decorations (UI-01, UI-05).
2. Rewrite or delete every second-line description that carries no information (UI-02).
3. Pick one chevron rule and one icon set; apply to all screens (UI-06, UI-07).
4. Fix the ID photo tab: four distinct specs, verified sizes, replace "hot templates" with spec-driven content (UI-15 to UI-18).
5. Fix the avatar tab: four type cards, no couple avatar, "重新生成" (UI-27 to UI-29).
6. Restyle explanatory rows so they do not look tappable (UI-10).
7. Add the missing screens listed in section 1, including all empty/loading/error states (UI-11).
8. Decide sticky vs non-sticky primary button and one label pattern (UI-08, UI-09).
9. Mine tab: four work categories, remove "清理缓存", single profile-edit entry (UI-33, UI-34, UI-36).
10. Provide the design-token sheet (colours, type scale, spacing, radii) matching `docs/DESIGN_SYSTEM.md`, and export icons as SVG.

## 5. Update after the first build (2026-09-15)

Reviewing the running Mini Program changed three earlier calls:

- **UI-09 sticky button: dropped.** With both a banner and a sticky bar, every tab had the same action twice and the bottom of the screen stacked a button bar on the TabBar. The hero card is now the single CTA with an explicit pill button.
- **UI-10 explanatory strips: removed entirely.** "制作包含 / 职业风格 / 生成包含" repeated the hero subtitle and looked like tools. The information stays in one line under the hero headline.
- **Template grids on tabs → horizontal rails.** 2×2 3:4 cards filled the screen with two templates; rails show ~2.5 cards per row and several rows. The "more" pages keep the 2-column grid.
- Category cards are photo tiles on every tab (icon circles only as fallback), TabBar active icons are filled, the page background is a neutral grey, and the credit number is baseline-aligned. See `docs/DESIGN_SYSTEM.md` §5.
- **Five tabs → four (D-24).** 头像 moved into the 写真 tab as a segment. The TabBar icons in the first rework rendered at about a quarter of their canvas because the SVG converter drew them at 24 px; they are now rendered at full size.
