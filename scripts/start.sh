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

for port in 8084 8085 8086 3004; do
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

if [ ! -d "$ROOT/services/tts/venv" ]; then
  echo "Creating TTS venv..."
  python3 -m venv "$ROOT/services/tts/venv"
  "$ROOT/services/tts/venv/bin/pip" install --quiet --upgrade pip
  "$ROOT/services/tts/venv/bin/pip" install --quiet -r "$ROOT/services/tts/requirements.txt"
fi

if tmux has-session -t solo-adv-tts 2>/dev/null; then
  tmux kill-session -t solo-adv-tts
fi
tmux new-session -d -s solo-adv-tts \
  "cd $ROOT/services/tts && ./venv/bin/uvicorn main:app --host 127.0.0.1 --port 8085 2>&1 | tee $ROOT/logs/solo-adv-tts.log"
echo "solo-adv-tts started in tmux"

if [ ! -d "$ROOT/services/depth/venv" ]; then
  echo "Creating depth venv (downloads ONNX + model on first boot)..."
  python3 -m venv "$ROOT/services/depth/venv"
  "$ROOT/services/depth/venv/bin/pip" install --quiet --upgrade pip
  "$ROOT/services/depth/venv/bin/pip" install --quiet -r "$ROOT/services/depth/requirements.txt"
fi

if tmux has-session -t solo-adv-depth 2>/dev/null; then
  tmux kill-session -t solo-adv-depth
fi
tmux new-session -d -s solo-adv-depth \
  "cd $ROOT/services/depth && HF_HOME=$ROOT/services/depth/hf-cache DEPTH_THREADS=4 ./venv/bin/uvicorn main:app --host 127.0.0.1 --port 8086 2>&1 | tee $ROOT/logs/solo-adv-depth.log"
echo "solo-adv-depth started in tmux"

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
tmux has-session -t solo-adv-tts    2>/dev/null && echo "  solo-adv-tts:    running" || echo "  solo-adv-tts:    NOT running"
tmux has-session -t solo-adv-depth  2>/dev/null && echo "  solo-adv-depth:  running" || echo "  solo-adv-depth:  NOT running"
tmux has-session -t solo-adv-server 2>/dev/null && echo "  solo-adv-server: running" || echo "  solo-adv-server: NOT running"
tmux has-session -t solo-adv-web    2>/dev/null && echo "  solo-adv-web:    running" || echo "  solo-adv-web:    NOT running"
echo ""
echo "Backend:  http://localhost:${PORT:-8084}"
echo "Frontend: http://localhost:3004"
echo ""
echo "Logs: tail -f logs/solo-adv-server.log"
echo "      tail -f logs/solo-adv-web.log"
