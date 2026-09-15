# 映己证件照写真馆 — Documentation Index

A WeChat Mini Program that turns a selfie into ID photos, professional headshots, portraits and avatars. Users pay with attention: one rewarded video ad grants one generation credit.

| Document | Purpose |
|---|---|
| [映己证件照写真馆_PRD_V1.0.md](./映己证件照写真馆_PRD_V1.0.md) | Product requirements (Chinese, source of truth for scope) |
| [DECISIONS.md](./DECISIONS.md) | Decisions taken where the PRD is silent or contradictory; read this first |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | System overview, components, trust boundaries, key flows |
| [FRONTEND_ARCHITECTURE.md](./FRONTEND_ARCHITECTURE.md) | uni-app (Vue 3 + TS) client: layout, platform layer, state, navigation, App readiness |
| [BACKEND_ARCHITECTURE.md](./BACKEND_ARCHITECTURE.md) | Go service layout, packages, workers, providers, cross-cutting concerns |
| [DATA_MODEL.md](./DATA_MODEL.md) | MySQL schema and invariants |
| [API.md](./API.md) | HTTP API contract for the Mini Program and the admin console |
| [GENERATION_PIPELINE.md](./GENERATION_PIPELINE.md) | Engine routing (gen model vs cheap engines), credits, ads, task state machine, pipelines, failure handling |
| [SHARING.md](./SHARING.md) | Share surfaces, posters, attribution, incentive, privacy |
| [DESIGN_SYSTEM.md](./DESIGN_SYSTEM.md) | Tokens, components, page templates, content rules |
| [COMPLIANCE.md](./COMPLIANCE.md) | Privacy, AI-content labelling, WeChat review, advertising rules |
| [DEPLOYMENT.md](./DEPLOYMENT.md) | Environments, infrastructure, CI/CD, release checklist |
| [../ui/UI_REVIEW.md](../ui/UI_REVIEW.md) | Review of the V1.0 mockups with a designer checklist |

## Repository layout (target)

```
.
├── docs/                 # this folder
├── ui/                   # mockups and UI review
├── client/               # uni-app (Vue 3 + TypeScript): builds mp-weixin now, app-plus / h5 later
├── backend/              # Go: cmd/{api,worker,migrate}, internal/*, seed assets, smoke script
├── backend/              # Go services: api, worker, migrate
├── admin/                # Admin console (React + Ant Design), optional in V1.0
└── deploy/               # docker-compose, Dockerfiles, CI workflows
```

## Delivery plan (suggested)

| Milestone | Content | Exit criteria |
|---|---|---|
| M0 · Foundations (week 1–2) | Repo scaffolding, auth, credits ledger, spec/template catalogue, admin CRUD, COS storage | Mini Program logs in, tabs render catalogue from the API |
| M1 · ID photo (week 3–4) | Upload + photo check, spec crop, matting, background fill, result page, save to album | End-to-end ID photo without ads |
| M2 · Ads and credits (week 5) | Rewarded video flow, ad sessions, daily grant, refund on failure, subscribe messages | Credit flow passes the test matrix in GENERATION_PIPELINE.md |
| M3 · Template modules and sharing (week 6–7) | Professional / portrait / avatar via the gen model, template detail, favorites, works library, share cards, posters, invite reward | All four modules generate end to end; a share lands on a template with attribution and the reward test rows 21–24 pass |
| M4 · Compliance and launch (week 8) | Privacy authorisation, AI labels, content moderation, analytics events, review submission | Passes the COMPLIANCE.md checklist; submitted to WeChat review |

## Conventions

- Documents are written in English; product copy in the app is Chinese.
- Every decision that changes scope goes into DECISIONS.md with an ID (D-xx); other documents reference those IDs.
- API and schema changes update API.md and DATA_MODEL.md in the same change.
