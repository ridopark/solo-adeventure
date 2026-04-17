#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p "$ROOT/logs"

echo "=== solo-adeventure: starting services ==="

if [ -f "$ROOT/.env" ]; then
  set -a
  source "$ROOT/.env"
  set +a
  echo "Loaded .env"
fi

for port in 8084 3004; do
  pid=$(lsof -ti ":$port" 2>/dev/null || true)
  if [ -n "$pid" ]; then
    echo "Killing process on port $port (PID $pid)"
    kill "$pid" 2>/dev/null || true
    sleep 1
  fi
done

echo "Building backend..."
cd "$ROOT/backend" && go build -o bin/solo-adeventure-server ./cmd/solo-adeventure-server
cd "$ROOT"

if tmux has-session -t solo-adv-server 2>/dev/null; then
  tmux kill-session -t solo-adv-server
fi
tmux new-session -d -s solo-adv-server \
  "cd $ROOT/backend && set -a && source $ROOT/.env && set +a && ./bin/solo-adeventure-server 2>&1 | tee $ROOT/logs/solo-adv-server.log"
echo "solo-adv-server started in tmux"

if tmux has-session -t solo-adv-web 2>/dev/null; then
  tmux kill-session -t solo-adv-web
fi
tmux new-session -d -s solo-adv-web \
  "cd $ROOT/apps/web && PORT=3004 NEXT_PUBLIC_BACKEND_URL=${NEXT_PUBLIC_BACKEND_URL:-http://localhost:8084} npm run dev 2>&1 | tee $ROOT/logs/solo-adv-web.log"
echo "solo-adv-web started in tmux"

sleep 2
echo ""
echo "=== services ==="
tmux has-session -t solo-adv-server 2>/dev/null && echo "  solo-adv-server: running" || echo "  solo-adv-server: NOT running"
tmux has-session -t solo-adv-web    2>/dev/null && echo "  solo-adv-web:    running" || echo "  solo-adv-web:    NOT running"
echo ""
echo "Backend:  http://localhost:${PORT:-8084}"
echo "Frontend: http://localhost:3004"
echo ""
echo "Logs: tail -f logs/solo-adv-server.log"
echo "      tail -f logs/solo-adv-web.log"
