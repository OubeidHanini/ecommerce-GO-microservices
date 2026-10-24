#!/usr/bin/env bash
# End-to-end check through the gateway: one order that gets confirmed, one that gets
# cancelled for lack of stock. Safe to re-run: it creates fresh data each time.
#
#   scripts/smoke-test.sh [gateway-url]     (default: http://localhost:8080)
set -euo pipefail

GATEWAY="${1:-http://localhost:8080}"
RUN_ID="$(date +%s)-$RANDOM"

fail() { echo "FAIL: $*" >&2; exit 1; }

# post PATH BODY EXPECTED_STATUS -> prints the response body
post() {
  local out status
  out="$(curl -sS -w '\n%{http_code}' -X POST "$GATEWAY$1" -H 'Content-Type: application/json' -d "$2")"
  status="${out##*$'\n'}"
  [[ "$status" == "$3" ]] || fail "POST $1 returned $status, want $3: ${out%$'\n'*}"
  echo "${out%$'\n'*}"
}

json_field() { sed -nE "s/.*\"$1\":\"?([^\",}]*)\"?.*/\1/p" <<<"$2"; }

# wait_for_status ORDER_ID WANTED: stock reservation is asynchronous
wait_for_status() {
  local body status
  for _ in $(seq 1 30); do
    body="$(curl -sS "$GATEWAY/orders/$1")"
    status="$(json_field status "$body")"
    [[ "$status" == "$2" ]] && return 0
    sleep 1
  done
  fail "order $1 is '$status', want '$2'"
}

echo "Waiting for the gateway at $GATEWAY..."
for _ in $(seq 1 60); do
  curl -sf "$GATEWAY/health" >/dev/null && break
  sleep 2
done
curl -sf "$GATEWAY/health" >/dev/null || fail "gateway not reachable"

user_id="$(json_field id "$(post /users "{\"name\":\"Smoke\",\"email\":\"smoke-$RUN_ID@example.com\"}" 201)")"
product_id="$(json_field id "$(post /products '{"name":"Keyboard","price_cents":4990,"stock":3}' 201)")"

confirmed="$(post /orders "{\"user_id\":$user_id,\"items\":[{\"product_id\":$product_id,\"quantity\":2}]}" 202)"
[[ "$(json_field total_cents "$confirmed")" == "9980" ]] || fail "wrong total: $confirmed"
cancelled="$(post /orders "{\"user_id\":$user_id,\"items\":[{\"product_id\":$product_id,\"quantity\":5}]}" 202)"

wait_for_status "$(json_field id "$confirmed")" confirmed
wait_for_status "$(json_field id "$cancelled")" cancelled

stock="$(json_field stock "$(curl -sS "$GATEWAY/products/$product_id")")"
[[ "$stock" == "1" ]] || fail "stock is $stock, want 1"

post /orders '{"user_id":999999999,"items":[{"product_id":1,"quantity":1}]}' 400 >/dev/null

echo "PASS: orders confirmed and cancelled as expected, stock consistent"
