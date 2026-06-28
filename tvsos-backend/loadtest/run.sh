#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_URL="${BASE_URL:-http://127.0.0.1:8088}"
API_DURATION="${API_DURATION:-15s}"
BACKLOG_DURATION="${BACKLOG_DURATION:-45s}"
RESULT_DIR="$ROOT_DIR/loadtest/results"
MODE="${1:-all}"

mkdir -p "$RESULT_DIR/raw"

run_algorithm() {
  echo "[1/3] 车辆规模与算法耗时测试"
  (
    cd "$ROOT_DIR"
    go test ./internal/task -run '^$' -bench '^BenchmarkSchedulerVehicleScale$' \
      -benchtime=1x -count=3 | tee "$RESULT_DIR/algorithm-benchmark.txt"
  )
}

require_server() {
  if ! curl -fsS "$BASE_URL/vehicles" >/dev/null 2>&1; then
    echo "错误：服务 $BASE_URL 不可访问，请先运行 loadtest/start_server.sh。" >&2
    exit 1
  fi
}

run_api() {
  require_server
  echo "[2/3] 核心查询接口并发测试"
  echo "endpoint,vus,qps,p95_ms,avg_ms,http_fail_rate,business_fail_rate,request_count" >"$RESULT_DIR/api-summary.csv"
  for endpoint_name in vehicles shipments report; do
    case "$endpoint_name" in
      vehicles) endpoint="/vehicles" ;;
      shipments) endpoint="/shipments" ;;
      report) endpoint="/report/realtime" ;;
    esac
    for vus in 10 50 100 200; do
      prefix="api-${endpoint_name}-vu${vus}"
      echo "测试 ${endpoint}，VU=${vus}"
      BASE_URL="$BASE_URL" ENDPOINT="$endpoint" VUS="$vus" DURATION="$API_DURATION" \
        k6 run --summary-export "$RESULT_DIR/raw/${prefix}.json" \
        --quiet "$ROOT_DIR/loadtest/k6/api.js" || true
      jq -r --arg endpoint "$endpoint_name" --arg vus "$vus" \
        '[$endpoint, ($vus|tonumber), .metrics.http_reqs.rate,
          .metrics.http_req_duration["p(95)"], .metrics.http_req_duration.avg,
          .metrics.http_req_failed.value, .metrics.business_failures.value,
          .metrics.http_reqs.count] | @csv' \
        "$RESULT_DIR/raw/${prefix}.json" >>"$RESULT_DIR/api-summary.csv"
      sleep 2
    done
  done
}

run_backlog() {
  require_server
  echo "[3/3] 持续发单积压测试"
  for rate in 1 2 5; do
    prefix="backlog-rate${rate}"
    echo "测试发单速率 ${rate} 单/秒"
    BASE_URL="$BASE_URL" RATE="$rate" DURATION="$BACKLOG_DURATION" \
      k6 run --summary-export "$RESULT_DIR/raw/${prefix}.json" \
      --out "csv=$RESULT_DIR/raw/${prefix}.csv" --quiet "$ROOT_DIR/loadtest/k6/backlog.js" || true
  done
}

case "$MODE" in
  algorithm) run_algorithm ;;
  api) run_api ;;
  backlog) run_backlog ;;
  all) run_algorithm; run_api; run_backlog; python3 "$ROOT_DIR/loadtest/scripts/build_report_data.py" ;;
  *) echo "用法：$0 [all|algorithm|api|backlog]" >&2; exit 2 ;;
esac
