#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://localhost:8080}"
LOGIN="${LOGIN:-perf_user}"
EMAIL="${EMAIL:-perf_user@example.com}"
PASSWORD="${PASSWORD:-PerfPass1!}"
OUT_DIR="${OUT_DIR:-perf_test/results/auth}"

mkdir -p "$OUT_DIR"

headers_file="$OUT_DIR/register_headers.txt"
body_file="$OUT_DIR/register_body.json"

curl -i -sS \
  -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"login\":\"$LOGIN\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" \
  -o "$body_file" \
  -D "$headers_file"

session_id="$(awk 'BEGIN{IGNORECASE=1} /^Set-Cookie:/ { sub(/^Set-Cookie:[[:space:]]*/, ""); split($0, parts, ";"); print parts[1]; exit }' "$headers_file")"
csrf_token="$(awk 'BEGIN{IGNORECASE=1} /^X-NEW-CSRF-TOKEN:/ { sub(/\r$/, ""); print $2; exit }' "$headers_file")"

if [ -z "$session_id" ] || [ -z "$csrf_token" ]; then
  echo "Не удалось получить session_id или CSRF token. Ответ сохранен в $body_file, заголовки в $headers_file" >&2
  exit 1
fi

cat <<EOF
export AUTH_COOKIE='$session_id'
export CSRF_TOKEN='$csrf_token'
EOF

