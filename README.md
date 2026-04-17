# solo-adeventure

A web-based gamebook in the Choose Your Own Adventure style. Enter a topic, pick a choice per page, and Claude generates the next narrative beat with an AI illustration.

## Stack

| Layer     | Tech                                                         |
|-----------|--------------------------------------------------------------|
| Frontend  | Next.js 16, React 19, TypeScript, Tailwind CSS 4             |
| Backend   | Go 1.24, hexagonal architecture (domain/ports/adapters/app)  |
| Story LLM | Anthropic Claude (Sonnet) via SDK, tool-use structured output |
| Images    | Together AI FLUX schnell (primary, free tier) + fal.ai FLUX schnell (fallback), behind a single `ImageProvider` port |
| Transport | REST (JSON)                                                  |
| Storage   | In-memory (MVP)                                              |

## Ports

- Backend: 8084
- Frontend: 3004

## Quick Start

Prerequisites: Go 1.24+, Node 22+, tmux.

```bash
cp .env.example .env     # fill in API keys
./scripts/start.sh       # launches backend + frontend in tmux

# or run separately
make dev-backend
make dev-web
make test
```

Open http://localhost:3004.

## Project Structure

```
solo-adeventure/
├── apps/web/                        Next.js 16 frontend
│   ├── app/                         pages (landing, /story/[id])
│   ├── components/                  TopicInput, StoryView, Illustration, ...
│   ├── hooks/                       useStory, useLocalStoryCache
│   └── lib/                         api client, types (mirror Go DTOs), env
├── backend/                         Go backend
│   ├── cmd/solo-adeventure-server/  entry point
│   └── internal/
│       ├── domain/                  Story, Page, Choice, DTOs -- pure
│       ├── ports/                   StoryProvider, ImageProvider, StoryStore
│       ├── app/                     Service -- orchestrates the ports
│       ├── adapters/
│       │   ├── http/                REST router + middleware
│       │   ├── llm/                 anthropic.go -- StoryProvider impl
│       │   ├── image/               together.go, fal.go, fallback.go, classify.go
│       │   └── store/               memory.go
│       ├── config/                  env-based config
│       └── logger/                  zerolog setup
├── docs/                            implementation-plan.md + future design docs
├── scripts/                         start.sh (tmux launcher)
├── deployments/                     (to be filled: Dockerfile, Caddyfile, compose)
├── CLAUDE.md                        repo rules for AI agents
├── Makefile
└── .env.example
```

## Architecture -- Hexagonal Payoff

The `ImageProvider` port has one method:

```go
type ImageProvider interface {
    Generate(ctx context.Context, req ImageRequest) (ImageResult, error)
}
```

Together, fal, and the `FallbackImageProvider` composite all implement it. Adding OpenRouter or Gemini means dropping in `adapters/image/openrouter.go` and changing one line in `cmd/solo-adeventure-server/main.go`. The service, HTTP handlers, and domain never know who served the image.

Same pattern for `StoryProvider` -- today Anthropic, tomorrow anything else.

## Backend API

| Method | Path                        | Purpose                                              |
|--------|-----------------------------|------------------------------------------------------|
| GET    | `/health`                   | liveness                                             |
| POST   | `/stories`                  | `{topic}` -> `{storyId, stylePrefix, page}`          |
| POST   | `/stories/{id}/choose`      | `{choiceIndex}` -> `{page}`                          |
| GET    | `/stories/{id}`             | full story (for refresh)                             |
| POST   | `/images`                   | `{prompt, stylePrefix}` -> `{url, provider}` (retry hook) |

## Deployment

Frontend goes to **Cloudflare Pages** (static export). Backend goes to **Oracle Cloud Infrastructure** (OCI free-tier ARM64 VM, systemd). Follows the same pattern as the `shooter` project.

### One-time setup

**OCI VM** (Ubuntu 22.04 ARM64, free tier):
- SSH access for `ubuntu` user.
- Port 8084 reachable (or fronted by Caddy/nginx on 80/443).
- Nothing else -- CI provisions `/opt/solo-adeventure/` and the systemd unit.

**Cloudflare Pages**:
- Create a Pages project named `solo-adeventure` (Direct Upload workflow -- no git link).
- Generate a scoped API token with `Cloudflare Pages:Edit`.

**GitHub Actions secrets** (repo -> Settings -> Secrets -> Actions):

| Secret                      | Value                                                      |
|-----------------------------|------------------------------------------------------------|
| `CLOUDFLARE_API_TOKEN`      | Cloudflare Pages API token                                 |
| `CLOUDFLARE_ACCOUNT_ID`     | Cloudflare account id                                      |
| `OCI_HOST`                  | public IP or DNS of the OCI VM                             |
| `OCI_USER`                  | `ubuntu`                                                   |
| `OCI_SSH_KEY`               | private SSH key (PEM)                                      |
| `ANTHROPIC_API_KEY`         | Claude API key                                             |
| `TOGETHER_API_KEY`          | Together AI key (primary image)                            |
| `FAL_KEY`                   | fal.ai key (fallback image)                                |
| `NEXT_PUBLIC_BACKEND_URL`   | e.g. `https://solo-adv-api.example.com`                   |
| `CORS_ALLOW_ORIGIN`         | e.g. `https://solo-adeventure.pages.dev`                  |

### Flow

Push to `main` -> `.github/workflows/ci.yml` runs:
1. **test**: `go test -race ./...`, ARM64 build verification.
2. **frontend-build**: `STATIC_EXPORT=1 npm run build` -> `apps/web/out/`.
3. **deploy-frontend**: `wrangler pages deploy out/ --project-name=solo-adeventure`.
4. **deploy-backend**: cross-compile ARM64 binary, `scp` to `/opt/solo-adeventure/`, install `.env` + systemd unit on first run, `systemctl restart solo-adeventure-server`, `curl /health` to verify.

### Manual deploy (shortcut)

```bash
# Backend
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o solo-adeventure-server ./cmd/solo-adeventure-server
scp solo-adeventure-server ubuntu@<oci-host>:/opt/solo-adeventure/
ssh ubuntu@<oci-host> "sudo systemctl restart solo-adeventure-server"

# Frontend
cd apps/web
STATIC_EXPORT=1 NEXT_PUBLIC_BACKEND_URL=https://solo-adv-api.example.com npm run build
npx wrangler pages deploy out/ --project-name=solo-adeventure
```

### Local Docker (alternative to Cloudflare + OCI)

```bash
cd deployments
# uses ANTHROPIC_API_KEY, TOGETHER_API_KEY, FAL_KEY from your shell
docker compose up --build
```

Runs backend on `:8084` and frontend on `:3004` behind the `Caddyfile` if you add Caddy to the compose file.

### Route shape (why `/story?id=...`)

Cloudflare Pages static export does not support Next.js dynamic segments (`/story/[id]`) without `generateStaticParams`. The route is a single static page at `/story` that reads the id from `useSearchParams`. Behavior is identical; only the URL shape changed.

## Implementation plan

See `docs/implementation-plan.md`. Status: **Planned**.
