---
name: dev-start
description: Build, commit, and restart the dev server. Use when the user asks to run the app, rebuild, restart, deploy local changes, or says 'start', 'run it', 'ship it', 'rebuild', 'restart', 'RCR', 'dev-start'.
---

# Dev Start: Build, Commit & Restart

Full cycle: build backend, commit changes, shutdown and restart services in tmux.

## Workflow

Execute these steps **in order**. Stop on any failure.

### Step 1: Build backend

```bash
cd backend && go build -o bin/shooter-server ./cmd/shooter-server
```

Verify the build succeeds (exit code 0) before proceeding. If the build fails, **stop here** -- do not commit broken code.

### Step 2: Run tests

```bash
cd backend && go test ./...
```

If tests fail, **stop here**.

### Step 3: Commit changes

Stage all changed files and create a commit. Use a concise message describing what changed. Include the Co-Authored-By trailer.

If there are no changes to commit, skip this step.

### Step 4: Shutdown running services

```bash
./scripts/shutdown.sh
```

This stops `shooter-server` and `shooter-web` tmux sessions.

### Step 5: Start all services

```bash
./scripts/start.sh
```

This builds the Go binary, starts `shooter-server` (port 8083) and `shooter-web` (port 3003) in tmux sessions with logging.

### Step 6: Verify services are running

```bash
tmux has-session -t shooter-server 2>/dev/null && echo "shooter-server: running" || echo "shooter-server: NOT running"
tmux has-session -t shooter-web 2>/dev/null && echo "shooter-web: running" || echo "shooter-web: NOT running"
```

Both services must be running. If either failed, check logs:

```bash
tail -20 logs/shooter-server.log
tail -20 logs/shooter-web.log
```

## Quick Reference

| Step | Command | Abort on failure? |
|------|---------|-------------------|
| Build | `cd backend && go build -o bin/shooter-server ./cmd/shooter-server` | **Yes** |
| Test | `cd backend && go test ./...` | **Yes** |
| Commit | `git add + git commit` | Skip if no changes |
| Shutdown | `./scripts/shutdown.sh` | No |
| Start | `./scripts/start.sh` | **Yes** |
| Verify | `tmux has-session` | Report status |

## Important Notes

- All commands run from the project root: `/home/ridopark/src/shooter`
- Logs are written to `logs/shooter-server.log` and `logs/shooter-web.log`
- Frontend runs at http://localhost:3003, backend at http://localhost:8083
