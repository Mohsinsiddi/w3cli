#!/usr/bin/env bash
# test-wallets.sh
# Builds w3cli and runs balance + txs checks for a set of wallets
# across all EVM chains on both mainnet and testnet.
#
# Usage:
#   ./scripts/test-wallets.sh
#
# Output:
#   - Live progress to stdout
#   - Full error log: /tmp/w3cli-test/errors.log
#   - Summary report: /tmp/w3cli-test/summary.txt

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────

WALLETS=(
  "0x802D8097eC1D49808F3c2c866020442891adde57"
  "0x315a352720E52EaDCB62f5e0879D5Fea82B959A4"
  "0x5d1D0b1d5790B1c88cC1e94366D3B242991DC05d"
)

EVM_CHAINS=(
  ethereum base polygon arbitrum optimism
  bnb avalanche fantom linea zksync
  scroll mantle celo gnosis blast
  mode zora moonbeam cronos klaytn
  aurora polygon-zkevm boba
)

MODES=(mainnet testnet)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MODULE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="/tmp/w3cli-testbin"
CONFIG_DIR="/tmp/w3cli-test/config"
LOG_DIR="/tmp/w3cli-test"
ERROR_LOG="$LOG_DIR/errors.log"
SUMMARY="$LOG_DIR/summary.txt"

# ── Colors ────────────────────────────────────────────────────────────────────

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
RESET='\033[0m'

# ── Counters ──────────────────────────────────────────────────────────────────

PASS=0
FAIL=0
SKIP=0

# ── Helpers ───────────────────────────────────────────────────────────────────

log_error() {
  local context="$1"
  local msg="$2"
  echo "[ERROR] $context" >> "$ERROR_LOG"
  echo "        $msg" >> "$ERROR_LOG"
  echo "" >> "$ERROR_LOG"
}

short_addr() {
  local addr="$1"
  echo "${addr:0:6}...${addr: -4}"
}

section() {
  echo ""
  echo -e "${BOLD}${CYAN}══════════════════════════════════════════════════${RESET}"
  echo -e "${BOLD}${CYAN}  $1${RESET}"
  echo -e "${BOLD}${CYAN}══════════════════════════════════════════════════${RESET}"
}

run_cmd() {
  local label="$1"
  shift
  local output
  local exit_code=0

  # Strip spinner output, capture only the final result
  output=$("$@" 2>&1) || exit_code=$?

  if [ $exit_code -eq 0 ]; then
    echo -e "  ${GREEN}✓${RESET} $label"
    # Print the output (indent each line)
    echo "$output" | grep -v "^⠋\|^⠙\|^⠹\|^⠸\|^⠼\|^⠴\|^⠦\|^⠧\|^⠇\|^⠏" \
      | sed 's/^/     /' | head -20
    PASS=$((PASS + 1))
    return 0
  else
    echo -e "  ${RED}✗${RESET} $label"
    # Log the error
    log_error "$label" "$output"
    # Print short error inline
    local short_err
    short_err=$(echo "$output" | grep -i "error\|Error\|failed\|Failed" | head -1 | sed 's/^[[:space:]]*//')
    if [ -n "$short_err" ]; then
      echo -e "    ${DIM}↳ $short_err${RESET}"
    else
      echo -e "    ${DIM}↳ (see $ERROR_LOG for details)${RESET}"
    fi
    FAIL=$((FAIL + 1))
    return 1
  fi
}

# ── Step 1: Build ─────────────────────────────────────────────────────────────

section "Step 1: Building w3cli"

# Clean up from previous runs
rm -rf "$CONFIG_DIR" "$ERROR_LOG" "$SUMMARY"
mkdir -p "$LOG_DIR" "$CONFIG_DIR"
> "$ERROR_LOG"
> "$SUMMARY"

echo -e "  ${DIM}Module root: $MODULE_ROOT${RESET}"
echo -e "  ${DIM}Binary:      $BINARY${RESET}"
echo -e "  ${DIM}Config dir:  $CONFIG_DIR${RESET}"
echo ""

if (cd "$MODULE_ROOT" && go build -o "$BINARY" .); then
  echo -e "  ${GREEN}✓${RESET} Build successful"
else
  echo -e "  ${RED}✗${RESET} Build failed — aborting"
  exit 1
fi

CLI="$BINARY --config $CONFIG_DIR"

# ── Step 2: Basic sanity checks ───────────────────────────────────────────────

section "Step 2: Sanity Checks"

run_cmd "version flag"         $CLI --version
run_cmd "help flag"            $CLI --help
run_cmd "network list"         $CLI network list
run_cmd "config list (fresh)"  $CLI config list

# ── Step 3: Add wallets ───────────────────────────────────────────────────────

section "Step 3: Adding Wallets"

for ADDR in "${WALLETS[@]}"; do
  LABEL="$(short_addr "$ADDR")"
  NAME="wallet_${ADDR: -6}"
  run_cmd "wallet add $LABEL" \
    $CLI wallet add "$NAME" "$ADDR"
done

run_cmd "wallet list" $CLI wallet list

# ── Step 4: Balance + TXs per chain per mode ──────────────────────────────────

for MODE in "${MODES[@]}"; do

  MODE_UPPER=$(echo "$MODE" | tr '[:lower:]' '[:upper:]')
  section "Step 4 [$MODE_UPPER]: Balance & Transactions"

  if [ "$MODE" = "testnet" ]; then
    MODE_FLAG="--testnet"
  else
    MODE_FLAG=""
  fi

  for CHAIN in "${EVM_CHAINS[@]}"; do

    echo ""
    echo -e "  ${BOLD}${YELLOW}▶ $CHAIN ($MODE)${RESET}"

    # Set default network + mode
    if [ "$MODE" = "testnet" ]; then
      $CLI network use "$CHAIN" --testnet > /dev/null 2>&1 || {
        echo -e "    ${DIM}↳ skipped (network use failed)${RESET}"
        SKIP=$((SKIP + 1))
        continue
      }
    else
      $CLI network use "$CHAIN" > /dev/null 2>&1 || {
        echo -e "    ${DIM}↳ skipped (network use failed)${RESET}"
        SKIP=$((SKIP + 1))
        continue
      }
    fi

    for ADDR in "${WALLETS[@]}"; do
      SHORT="$(short_addr "$ADDR")"

      # Balance check
      run_cmd "balance  $SHORT  [$CHAIN/$MODE]" \
        $CLI balance --wallet "$ADDR" --network "$CHAIN" || true

      # Recent transactions
      run_cmd "txs      $SHORT  [$CHAIN/$MODE]" \
        $CLI txs --wallet "$ADDR" --network "$CHAIN" || true

    done

  done

done

# ── Step 5: TX detail (grab a real hash from ethereum mainnet first) ───────────

section "Step 5: Transaction Detail Lookup"

$CLI network use ethereum > /dev/null 2>&1

# Try to grab a real recent tx hash from ethereum to test `tx` command
echo -e "  ${DIM}Fetching a recent Ethereum tx hash...${RESET}"
RECENT_HASH=$(curl -s -X POST https://eth.llamarpc.com \
  -H 'Content-Type: application/json' \
  --max-time 10 \
  -d '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest",true],"id":1}' \
  2>/dev/null \
  | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    txs = d.get('result', {}).get('transactions', [])
    print(txs[0]['hash'] if txs else '')
except:
    print('')
" 2>/dev/null || echo "")

if [ -n "$RECENT_HASH" ]; then
  echo -e "  ${DIM}Hash: $RECENT_HASH${RESET}"
  run_cmd "tx detail  [ethereum/mainnet]" \
    $CLI tx "$RECENT_HASH" --network ethereum || true
else
  echo -e "  ${YELLOW}⚠${RESET}  Could not fetch recent tx hash — skipping tx detail test"
  SKIP=$((SKIP + 1))
fi

# ── Step 6: RPC commands ──────────────────────────────────────────────────────

section "Step 6: RPC Management"

run_cmd "rpc add custom base RPC" \
  $CLI rpc add base "https://mainnet.base.org"

run_cmd "rpc list base" \
  $CLI rpc list base

run_cmd "rpc algorithm set fastest" \
  $CLI rpc algorithm set fastest

run_cmd "config list (final)" \
  $CLI config list

# ── Summary ───────────────────────────────────────────────────────────────────

TOTAL=$((PASS + FAIL + SKIP))

{
  echo "w3cli Test Summary"
  echo "=================="
  echo "Date:    $(date)"
  echo "Total:   $TOTAL"
  echo "Pass:    $PASS"
  echo "Fail:    $FAIL"
  echo "Skip:    $SKIP"
  echo ""
  echo "Wallets tested:"
  for W in "${WALLETS[@]}"; do echo "  $W"; done
  echo ""
  echo "Chains tested: ${EVM_CHAINS[*]}"
  echo ""
  if [ -s "$ERROR_LOG" ]; then
    echo "Errors (see $ERROR_LOG):"
    cat "$ERROR_LOG"
  else
    echo "No errors logged."
  fi
} | tee "$SUMMARY"

echo ""
echo -e "${BOLD}Results:  ${GREEN}$PASS passed${RESET}  ${RED}$FAIL failed${RESET}  ${YELLOW}$SKIP skipped${RESET}  (total: $TOTAL)"
echo -e "${DIM}Summary saved to: $SUMMARY${RESET}"
echo -e "${DIM}Error log:        $ERROR_LOG${RESET}"
echo ""
