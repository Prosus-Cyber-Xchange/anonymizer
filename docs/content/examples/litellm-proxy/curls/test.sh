#!/usr/bin/env bash
set -euo pipefail

BASE="http://localhost:4000"
BOLD="\033[1m"
GREEN="\033[32m"
BLUE="\033[34m"
RED="\033[31m"
NC="\033[0m"

pass()  { printf "${GREEN}%s${NC}\n" "  PASS: $*"; }
fail()  { printf "${RED}%s${NC}\n" "  FAIL: $*"; exit 1; }
info()  { printf "${BLUE}%s${NC}\n" "$*"; }
header(){ printf "\n${BOLD}%s${NC}\n" "$*"; }

wait_for() {
  local url="$1"
  local label="$2"
  info "Waiting for $label at $url ..."
  for i in $(seq 1 30); do
    if curl -sf -o /dev/null "$url"; then
      info "$label is ready"
      return 0
    fi
    sleep 2
  done
  fail "$label did not become healthy"
}

run_test() {
  local name="$1"
  local body="$2"
  local expect_contains="$3"
  local expect_not_contains="$4"

  header "Test: $name"

  local response
  response=$(curl -s --location "$BASE/chat/completions" \
    -H "Authorization: Bearer sk-local-demo-key" \
    -H "Content-Type: application/json" \
    -d "$body")

  local content
  content=$(echo "$response" | python3 -c "import sys,json; print(json.load(sys.stdin)['choices'][0]['message']['content'])" 2>/dev/null || echo "PARSE_ERROR")

  if [[ "$response" == *"PARSE_ERROR"* ]] || [[ "$content" == "PARSE_ERROR" ]]; then
    printf "Raw response:\n%s\n" "$response"
    fail "Failed to parse response"
  fi

  if echo "$content" | grep -qi "$expect_not_contains"; then
    printf "Response: %s\n" "$content"
    fail "Response should NOT contain '$expect_not_contains' but it does"
  fi

  if ! echo "$content" | grep -qi "$expect_contains"; then
    printf "Response: %s\n" "$content"
    fail "Response should contain '$expect_contains' but it doesn't"
  fi

  pass "$name"
  printf "  Response: %s\n" "$(echo "$content" | head -c 200)"
}

# ---- wait for services ----
wait_for "http://localhost:8080/health" "anonymizer"
wait_for "http://localhost:4000/health" "litellm"

# ---- Test 1: PII gets redacted before reaching the LLM ----
run_test \
  "PII redaction in user message" \
  '{
    "model": "gpt-4o",
    "max_tokens": 60,
    "messages": [
      {"role": "user", "content": "My email is bob@example.com and my phone is +55 11 99999-9999. What is prompt injection?"}
    ]
  }' \
  "prompt" \
  "bob@example.com"

# ---- Test 2: LLM never sees PII, response keeps placeholders ----
run_test \
  "LLM response contains placeholder, not original PII" \
  '{
    "model": "gpt-4o",
    "max_tokens": 60,
    "messages": [
      {"role": "user", "content": "Repeat exactly: My contact is alice@foo.com"}
    ]
  }' \
  "\\[EMAIL\\]" \
  "alice@foo.com"

# ---- Test 3: No PII passes through unchanged ----
run_test \
  "No PII passes through unchanged" \
  '{
    "model": "gpt-4o",
    "max_tokens": 60,
    "messages": [
      {"role": "user", "content": "Say hello in exactly 3 words."}
    ]
  }' \
  "." \
  "DOES_NOT_MATTER_PLACEHOLDER_XYZ"

header "All tests passed!"
