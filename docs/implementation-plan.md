---
goal: Web-based gamebook (Choose Your Own Adventure) with Go hexagonal backend and Next.js frontend
version: 1.0
date_created: 2026-04-17
last_updated: 2026-04-17
owner: ridopark@gmail.com
status: 'Planned'
tags: [feature, architecture, backend, frontend, ai, hexagonal]
---

# Introduction

![Status: Planned](https://img.shields.io/badge/status-Planned-blue)

Implementation plan for `solo-adeventure`: a single-player interactive gamebook where Anthropic Claude generates narrative pages with choices and a dual-provider image pipeline (Together AI primary, fal.ai fallback) renders illustrations. Go hexagonal backend on port 8084, Next.js 16 frontend on port 3004.

## 1. Requirements & Constraints

- **REQ-001**: Backend MUST expose REST endpoints `POST /stories`, `POST /stories/{id}/choose`, `GET /stories/{id}`, `POST /images`, `GET /health`.
- **REQ-002**: Claude MUST return structured Page JSON via tool-use with schema `{narrative, imagePrompt, choices[{label}], isEnding, endingType?, runningSummary}`.
- **REQ-003**: Image provider MUST be abstracted behind a single `ImageProvider` port; concrete adapters MUST be swappable via config.
- **REQ-004**: `FallbackImageProvider` MUST try Together AI FLUX schnell first; on 429/5xx/timeout/network error, MUST invoke fal.ai FLUX schnell; surface last error only if both fail.
- **REQ-005**: Story state MUST persist in an in-memory store keyed by story UUID.
- **REQ-006**: Frontend MUST persist active `storyId` in `localStorage` and rehydrate on refresh via `GET /stories/{id}`.
- **REQ-007**: Ending pages MUST set `isEnding=true`, return `choices: []`, and disable further choice input.
- **SEC-001**: API keys (`ANTHROPIC_API_KEY`, `TOGETHER_API_KEY`, `FAL_KEY`) MUST be read from environment only; never committed.
- **SEC-002**: CORS MUST restrict to `http://localhost:3004` in development; wildcard only inside dev builds.
- **CON-001**: Backend module path `github.com/ridopark/solo-adeventure/backend`.
- **CON-002**: Frontend uses Next.js 16 App Router, React 19, Tailwind 4.
- **CON-003**: No database; in-memory only for MVP.
- **GUD-001**: Hexagonal layout `backend/internal/{domain,ports,adapters,app,config,logger}` mirroring shooter.
- **GUD-002**: Adapters MUST NOT import each other; only via ports.
- **GUD-003**: stdlib `http.ServeMux` (Go 1.22+ pattern matching) -- no third-party router.
- **PAT-001**: Composite pattern for `FallbackImageProvider`.
- **PAT-002**: Dependency injection via constructor in `cmd/solo-adeventure-server/main.go`.
- **PAT-003**: Text generation serial, image generation offloaded via `errgroup` so image failure never blocks page return.

## 2. Implementation Steps

### Implementation Phase 1

- GOAL-001: Scaffold repository structure, Go module, Next.js app, tooling.

| Task     | Description                                                                                       | Completed | Date       |
| -------- | ------------------------------------------------------------------------------------------------- | --------- | ---------- |
| TASK-001 | Create directory tree (backend/internal/{domain,ports,app,config,logger,adapters/*}, apps/web/*) | ✅        | 2026-04-17 |
| TASK-002 | Run `go mod init github.com/ridopark/solo-adeventure/backend`; add deps: anthropic-sdk-go, google/uuid, rs/zerolog | | |
| TASK-003 | Initialize Next.js 16 in `apps/web/` with TypeScript, Tailwind 4, App Router                     |           |            |
| TASK-004 | Author `Makefile` targets `dev`, `dev-backend`, `dev-web`, `build`, `test`, `tidy`, `clean`      | ✅        | 2026-04-17 |
| TASK-005 | Author `scripts/start.sh` tmux launcher (backend 8084, web 3004)                                  | ✅        | 2026-04-17 |
| TASK-006 | Author `.env.example` with API keys + `PORT=8084`                                                 | ✅        | 2026-04-17 |

### Implementation Phase 2

- GOAL-002: Define pure domain types and port interfaces (no implementations).

| Task     | Description                                                                                     | Completed | Date |
| -------- | ----------------------------------------------------------------------------------------------- | --------- | ---- |
| TASK-007 | Define `Story`, `Page`, `Choice`, `StylePrefix`, `PageDraft` in `domain/story.go`              |           |      |
| TASK-008 | Define app DTOs (`StartStoryInput`, `ProgressInput`, outputs) in `domain/dto.go`               |           |      |
| TASK-009 | Define `StoryProvider`, `ImageProvider`, `StoryStore` interfaces in `ports/ports.go`           |           |      |
| TASK-010 | Domain errors: `ErrStoryNotFound`, `ErrInvalidChoice`                                          |           |      |

### Implementation Phase 3

- GOAL-003: Implement outbound adapters.

| Task     | Description                                                                                    | Completed | Date |
| -------- | ---------------------------------------------------------------------------------------------- | --------- | ---- |
| TASK-011 | `adapters/llm/anthropic.go` implements `StoryProvider` via Claude tool-use (`emit_page` tool) |           |      |
| TASK-012 | `adapters/image/together.go` implements `ImageProvider` via Together REST                     |           |      |
| TASK-013 | `adapters/image/fal.go` implements `ImageProvider` via fal.run REST                           |           |      |
| TASK-014 | `adapters/image/fallback.go` composite: try primary, fall back on transient errors            |           |      |
| TASK-015 | `adapters/image/classify.go` shared transient-error classifier                                 |           |      |
| TASK-016 | `adapters/store/memory.go` in-memory `StoryStore` with sync.RWMutex                            |           |      |

### Implementation Phase 4

- GOAL-004: App service orchestration and HTTP router.

| Task     | Description                                                                                    | Completed | Date |
| -------- | ---------------------------------------------------------------------------------------------- | --------- | ---- |
| TASK-017 | `app.Service` methods: `StartStory`, `ProgressStory`, `GetStory`, `GenerateImage`             |           |      |
| TASK-018 | Service orchestrates text (serial) + image (errgroup, best-effort)                             |           |      |
| TASK-019 | `adapters/http/router.go` stdlib ServeMux with 5 routes, CORS middleware, recovery, req log   |           |      |
| TASK-020 | `config/config.go` env loader; `logger/logger.go` zerolog setup                                |           |      |
| TASK-021 | `cmd/solo-adeventure-server/main.go` composes adapters -> fallback -> service -> http         |           |      |

### Implementation Phase 5

- GOAL-005: Frontend landing page and topic submission.

| Task     | Description                                                                                | Completed | Date |
| -------- | ------------------------------------------------------------------------------------------ | --------- | ---- |
| TASK-022 | `app/page.tsx` + `components/TopicInput.tsx` (topic only for MVP; defer genre picker)     |           |      |
| TASK-023 | `lib/api.ts` typed fetch client; `lib/types.ts` mirrors Go DTOs; `lib/env.ts` BACKEND_URL |           |      |
| TASK-024 | On submit: POST /stories -> stash initial page in sessionStorage -> route to /story/[id]  |           |      |

### Implementation Phase 6

- GOAL-006: Story play page with narrative, illustration, choices, loading states.

| Task     | Description                                                                             | Completed | Date |
| -------- | --------------------------------------------------------------------------------------- | --------- | ---- |
| TASK-025 | `app/story/[id]/page.tsx` Client Component mounting `<StoryView storyId={id} />`      |           |      |
| TASK-026 | `useStory` custom hook (useReducer state machine: idle/hydrating/page_ready/choosing/ended/error) |   |      |
| TASK-027 | `StoryView`, `NarrativeBlock`, `Illustration`, `ChoiceButtons`, `Skeleton` components  |           |      |
| TASK-028 | Skeleton UI during `choosing`; fade-in on image load                                    |           |      |

### Implementation Phase 7

- GOAL-007: localStorage cache and refresh-safe hydration.

| Task     | Description                                                                                | Completed | Date |
| -------- | ------------------------------------------------------------------------------------------ | --------- | ---- |
| TASK-029 | `useLocalStoryCache(id)` wrapper                                                            |           |      |
| TASK-030 | On mount: hydrate from localStorage first, then reconcile with `GET /stories/{id}`         |           |      |
| TASK-031 | On 404 from backend: show "adventure faded" banner, clear local cache                      |           |      |

### Implementation Phase 8

- GOAL-008: Ending screen and copy-URL share.

| Task     | Description                                                                   | Completed | Date |
| -------- | ----------------------------------------------------------------------------- | --------- | ---- |
| TASK-032 | `EndingCard` rendered when `isEnding=true`; tint by `endingType`             |           |      |
| TASK-033 | "Copy link" button using `navigator.clipboard.writeText(location.href)`      |           |      |
| TASK-034 | "Start new adventure" clears localStorage and routes to `/`                  |           |      |

### Implementation Phase 9

- GOAL-009: E2E checklist, README, deployment wiring.

| Task     | Description                                                          | Completed | Date |
| -------- | -------------------------------------------------------------------- | --------- | ---- |
| TASK-035 | Manual E2E script in `docs/e2e-checklist.md`                         |           |      |
| TASK-036 | Root `README.md` (setup, env, run, architecture notes)               | ✅        | 2026-04-17 |
| TASK-037 | `deployments/Dockerfile` (backend multi-stage), `Dockerfile.web`     |           |      |
| TASK-038 | `deployments/Caddyfile` reverse-proxy `/api/*` -> backend, `/` -> web |           |      |

## 3. Alternatives

- **ALT-001**: WebSocket streaming for narrative -- rejected; REST sufficient for turn-based play.
- **ALT-002**: Redis/Postgres store -- rejected; in-memory meets CON-003 for MVP.
- **ALT-003**: Single image provider (Together only) -- rejected; REQ-004 mandates fallback.
- **ALT-004**: chi/gin router -- rejected; stdlib ServeMux (Go 1.22+) sufficient and matches shooter (GUD-003).
- **ALT-005**: OpenRouter unified gateway -- viable; can be added as a third `ImageProvider` adapter without touching use cases (this is the hexagonal payoff).

## 4. Dependencies

- **DEP-001**: `github.com/anthropics/anthropic-sdk-go`
- **DEP-002**: `github.com/google/uuid`
- **DEP-003**: `github.com/rs/zerolog`
- **DEP-004**: `golang.org/x/sync/errgroup`
- **DEP-005**: `github.com/stretchr/testify` (dev)
- **DEP-006**: Next.js 16, React 19, Tailwind 4 (npm)
- **DEP-007**: Together AI account + API key
- **DEP-008**: fal.ai account + API key (fallback)
- **DEP-009**: Anthropic API key

## 5. Files

Enumerated per phase above. Directory tree created in Phase 1.

## 6. Testing

- **TEST-001**: `fallback.Provider` -- primary OK: secondary not invoked.
- **TEST-002**: `fallback.Provider` -- primary 429/5xx: secondary invoked, result returned.
- **TEST-003**: `fallback.Provider` -- primary non-transient error (4xx not 429): no fallback, error surfaced.
- **TEST-004**: `fallback.Provider` -- both fail: last error returned.
- **TEST-005**: `memstore` concurrency with `go test -race`.
- **TEST-006**: `app.Service.ProgressStory` -- image failure returns page with empty URL and no error.
- **TEST-007**: `app.Service.ProgressStory` -- text provider error returns wrapped error.
- **TEST-008**: HTTP handler -- `POST /stories` returns 201 with storyId.
- **TEST-009**: HTTP handler -- `GET /stories/{id}` returns 404 when missing.
- **TEST-010**: HTTP handler -- invalid `choiceIndex` returns 400.
- **TEST-011**: Anthropic adapter -- stub server returning tool-use response; parsed `PageDraft` matches schema.
- **TEST-012**: Frontend manual E2E per `docs/e2e-checklist.md`.

## 7. Risks & Assumptions

- **RISK-001**: Together AI free tier rate limits may trigger frequent fallback; mitigate with per-request logging and a `provider` field on returned images.
- **RISK-002**: Claude tool-use output may drift from schema; mitigate with strict JSON validation and one retry.
- **RISK-003**: In-memory store loses state on restart; acceptable per CON-003 but must surface to UI (REQ-006 banner).
- **RISK-004**: Image URLs from providers may expire; mitigate in a later iteration by caching/proxying to our own store.
- **ASSUMPTION-001**: Single backend instance (no horizontal scaling) is acceptable for prototype.
- **ASSUMPTION-002**: English-only narrative generation.
- **ASSUMPTION-003**: User operates in modern browser supporting `navigator.clipboard`.

## 8. Related Specifications / Further Reading

- Anthropic tool-use docs: https://docs.anthropic.com/en/docs/build-with-claude/tool-use
- Together AI images: https://docs.together.ai/docs/images
- fal.ai FLUX schnell: https://fal.ai/models/fal-ai/flux/schnell
- Shooter hexagonal reference: `/home/ridopark/src/shooter/backend/internal/`
