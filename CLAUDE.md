## Rules
- Read before writing. Don't re-read unchanged files.
- Edit, don't rewrite. Minimal diffs.
- No sycophantic openers ("Sure!", "Great question!") or closing fluff.
- No restating the question. No unsolicited suggestions.
- ASCII only: no em dashes, smart quotes, or Unicode decoration.
- Plain text over tables/headers unless structure aids clarity.
- No comments in code unless the WHY is non-obvious.
- No speculative abstractions, error handling for impossible cases, or backwards-compat shims.
- Parallelize independent tool calls in one message.
- User instructions override this file.

## Architecture
- Go backend: hexagonal (domain -> ports -> adapters). Handles story generation orchestration, image generation with fallback, in-memory story store.
- Next.js 16 frontend: Client-Component play screen, custom useReducer state machine, Tailwind 4.
- Communication: REST only. No SSE, no WebSocket.
- No database for MVP: in-memory story store.
- Hexagonal payoff: ImageProvider port lets us swap Together/fal/OpenRouter/Gemini without touching use cases.

## Ports
- Backend: 8084
- Frontend: 3004

## Build & Run
- Backend: `cd backend && go build -o bin/solo-adeventure-server ./cmd/solo-adeventure-server`
- Frontend: `cd apps/web && npm run dev`
- Tests: `cd backend && go test ./...`
- Full cycle: `./scripts/start.sh`

## Module paths
- Go: `github.com/ridopark/solo-adeventure/backend`

## Conventions
- Error wrap: `fmt.Errorf("component: action: %w", err)`
- Logger: zerolog, `.With().Str("component", name).Logger()`
- Tests: table-driven, `t.Run()` + `stretchr/testify`
- HTTP: stdlib `http.ServeMux` (Go 1.22+ pattern matching). No chi.
