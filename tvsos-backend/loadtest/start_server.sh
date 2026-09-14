#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_URL="${BASE_URL:-http://127.0.0.1:8088}"
LOG_FILE="${LOG_FILE:-$ROOT_DIR/loadtest/results/server.log}"
PID_FILE="$ROOT_DIR/loadtest/results/server.pid"

mkdir -p "$ROOT_DIR/loadtest/results"

if curl -fsS "$BASE_URL/vehicles" >/dev/null 2>&1; then
  echo "服务已经运行：$BASE_URL"
  exit 0
fi

(
  cd "$ROOT_DIR"
  nohup go run . >"$LOG_FILE" 2>&1 &
  echo $! >"$PID_FILE"
)

for _ in $(seq 1 30); do
  if curl -fsS "$BASE_URL/vehicles" >/dev/null 2>&1; then
    echo "服务启动成功：$BASE_URL"
    exit 0
  fi
  sleep 1
done

echo "错误：服务未在 30 秒内就绪，请检查 MySQL/Redis 配置并查看 $LOG_FILE" >&2
exit 1
