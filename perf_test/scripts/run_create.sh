#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://localhost:8080}"
COUNT="${COUNT:-100000}"
RATE="${RATE:-200}"
TIMEOUT="${TIMEOUT:-30s}"
TITLE_PREFIX="${TITLE_PREFIX:-perf-group}"
RESULTS_DIR="${RESULTS_DIR:-perf_test/results}"

if [ -z "${AUTH_COOKIE:-}" ] || [ -z "${CSRF_TOKEN:-}" ]; then
  echo "Нужно задать AUTH_COOKIE и CSRF_TOKEN. Их можно получить через perf_test/scripts/register_user.sh" >&2
  exit 1
fi

if ! command -v vegeta >/dev/null 2>&1; then
  echo "vegeta не найден. Установите vegeta и повторите запуск." >&2
  exit 1
fi

timestamp="$(date '+%Y%m%d_%H%M%S')"
run_dir="$RESULTS_DIR/create_$timestamp"
mkdir -p "$run_dir"

targets="$run_dir/targets.json"
result_bin="$run_dir/result.bin"
report_txt="$run_dir/report.txt"
hist_txt="$run_dir/histogram.txt"

duration_seconds=$(( (COUNT + RATE - 1) / RATE ))

python3 perf_test/scripts/generate_targets.py create \
  --base-url "$BASE_URL" \
  --auth-cookie "$AUTH_COOKIE" \
  --csrf-token "$CSRF_TOKEN" \
  --count "$COUNT" \
  --title-prefix "$TITLE_PREFIX" \
  --output "$targets"

vegeta attack \
  -format=json \
  -targets="$targets" \
  -rate="${RATE}/s" \
  -duration="${duration_seconds}s" \
  -timeout="$TIMEOUT" \
  > "$result_bin"

vegeta report -type=text "$result_bin" > "$report_txt"
vegeta report -type=hist[0,50ms,100ms,200ms,500ms,1s,2s,5s,10s] "$result_bin" > "$hist_txt"

cat "$report_txt"
echo
echo "Результаты сохранены в $run_dir"

