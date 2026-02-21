# Plan: 8 New Developer Commands for w3cli

## Context
w3cli already covers balance, send, txs, contracts, tokens, wallets. This plan adds 8 commands (10 total with subcommands) that fill daily-workflow gaps for Web3 developers: allowance/approve, calldata decoding, one-shot contract reads, message signing/verify, unit conversion, nonce inspection, tx simulation, and ENS resolution.

---

## Architecture: What Goes Where

### New internal methods (reusable by future API server)

**`internal/chain/evm.go`** — 3 new methods on `EVMClient`:
```go
GetAllowance(tokenAddr, owner, spender string) (*big.Int, error)
// selector 0xdd62ed3e + ABI-encode (owner, spender) → parse uint256

GetPendingNonce(address string) (uint64, error)
// eth_getTransactionCount with "pending" block tag

SimulateCall(from, to, data string, value *big.Int) (ok bool, result string, err error)
// eth_call with from field; revert → (false, revertReason, nil); network err → (false,"",err)
```

**`internal/wallet/sign.go`** — new file, EIP-191 utilities:
```go
SignMessage(w *Wallet, ks KeystoreBackend, message []byte) ([]byte, error)
// "\x19Ethereum Signed Message:\n" + len + message, then crypto.Sign

VerifyMessage(message, sig []byte) (common.Address, error)
// recover signer address from EIP-191 sig using crypto.SigToPub
```

**`internal/ens/resolver.go`** — new package:
```go
// ENS Registry: 0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e (mainnet + Sepolia)
Resolve(name string, client *chain.EVMClient) (string, error)
// namehash(name) → resolver(node) [0x0178b8bf] → addr(node) [0x3b3b57de]

ReverseLookup(address string, client *chain.EVMClient) (string, error)
// namehash("<addr>.addr.reverse") → resolver → name(node) [0x691f3431]
```
`namehash` implemented in-package using `crypto.Keccak256Hash` (already a go-ethereum dep).

---

## New cmd files (8 files, 10 commands)

| File | Command(s) | Key flags |
|------|-----------|-----------|
| `cmd/allowance.go` | `w3cli allowance` + `w3cli approve` | `--token --owner --spender --amount --wallet --network` |
| `cmd/decode.go` | `w3cli decode <calldata>` | positional hex arg, no RPC |
| `cmd/call.go` | `w3cli call <addr> <func> [args...]` | `--network` |
| `cmd/sign.go` | `w3cli sign <msg>` + `w3cli verify <msg>` | `--wallet` / `--sig --address` |
| `cmd/convert.go` | `w3cli convert <amount> <unit>` | pure math, no flags |
| `cmd/nonce.go` | `w3cli nonce` | `--wallet --network` |
| `cmd/simulate.go` | `w3cli simulate` | `--from --to --data --value --network` |
| `cmd/ens.go` | `w3cli ens <name-or-address>` | `--network` |

### Command behaviours

**`w3cli allowance`** — calls `EVMClient.GetAllowance()`, fetches decimals via `0x313ce567`, formats amount.

**`w3cli approve`** — reuses `loadSigningWallet()`. Builds approve calldata (selector `0x095ea7b3` + padded spender + scaled amount). Full sign→send→wait flow matching `cmd/send.go` pattern. Uses `config.GasLimitERC20Transfer` fallback.

**`w3cli decode <calldata>`** — reuses `chain.DecodeMethod()` from `internal/chain/explorer.go:90`. For unknown selectors: shows hex + "(unknown)". No RPC call.

**`w3cli call <address> <func> [args...]`** — uses `contract.NewCallerFromEntries(rpcURL, erc20ABI)` for known ERC-20 functions (reuses `internal/contract/erc20_abi.go`). For unregistered functions: user supplies signature like `foo(uint256)`, encode manually.

**`w3cli sign <message>`** — uses new `wallet.SignMessage`. Prints hex signature.

**`w3cli verify <message> --sig 0x... --address 0x...`** — uses `wallet.VerifyMessage`, compares recovered address.

**`w3cli convert <amount> <unit>`** — pure math with `big.Float`. Units: `eth`, `gwei`, `wei`, `hex`, `decimal`.
```
w3cli convert 1.5 eth    →  1500000000 gwei  /  1500000000000000000 wei
w3cli convert 50 gwei    →  0.000000050 eth  /  50000000000 wei
w3cli convert 0xff       →  255 (decimal)
w3cli convert 255 hex    →  0xff
```

**`w3cli nonce`** — calls `GetNonce` + `GetPendingNonce` in parallel. Shows confirmed + pending; warns if they differ (stuck/pending txs).

**`w3cli simulate`** — calls `EVMClient.SimulateCall()`. Shows succeed/revert + gas estimate. Parses revert reason string from RPC error.

**`w3cli ens`** — auto-detects direction (name vs 0x address). Forces `ethereum` chain. Shows both forward and reverse results side-by-side.

---

## Register in cmd/root.go
Add to `rootCmd.AddCommand(...)`:
```go
allowanceCmd, approveCmd,
decodeCmd,
callCmd,
signCmd, verifyCmd,
convertCmd,
nonceCmd,
simulateCmd,
ensCmd,
```

---

## Test Files

### Pure unit tests (no RPC, no keystore)
**`cmd/convert_test.go`** — eth↔gwei↔wei, hex↔decimal, edge cases (0, very large, 1e18)

**`cmd/decode_test.go`** — known selectors (transfer, approve, balanceOf), unknown selector, empty calldata

### Mock RPC tests (follow `rpcMock()` pattern from `internal/chain/evm_test.go`)
**`internal/chain/evm_extra_test.go`**
- `GetAllowance`: success (0x1388=5000), zero allowance, RPC error
- `GetPendingNonce`: equal to confirmed, greater than confirmed, RPC error
- `SimulateCall`: success with return data, revert with reason, network error

### Wallet signing tests (use `WithInMemoryStore()` + `NewInMemoryKeystore()`)
**`internal/wallet/sign_test.go`**
- `TestSignMessage_RoundTrip` — sign then verify, recovered == wallet address
- `TestVerifyMessage_WrongSig` — tampered sig → wrong recovered address
- `TestVerifyMessage_WrongMessage` — same sig, different msg → mismatch
- Uses Hardhat account #0 test key: `0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80`

### ENS tests
**`internal/ens/resolver_test.go`**
- `TestNamehash_KnownVectors` — EIP-137 spec vectors: `""`, `"eth"`, `"foo.eth"`, `"vitalik.eth"`
- `TestResolve_MockRPC` — mock registry + resolver responses
- `TestReverseLookup_MockRPC`

---

## Critical Files Modified
- `internal/chain/evm.go` — add 3 methods
- `cmd/root.go` — register 10 new commands

## Critical Files Created
- `internal/wallet/sign.go` + `internal/wallet/sign_test.go`
- `internal/ens/resolver.go` + `internal/ens/resolver_test.go`
- `internal/chain/evm_extra_test.go`
- `cmd/allowance.go`, `cmd/decode.go`, `cmd/call.go`, `cmd/sign.go`
- `cmd/convert.go`, `cmd/nonce.go`, `cmd/simulate.go`, `cmd/ens.go`
- `cmd/convert_test.go`, `cmd/decode_test.go`

## Reused Existing Code
- `chain.DecodeMethod()` — `internal/chain/explorer.go:90`
- `contract.NewCallerFromEntries()` + ERC-20 ABI — `internal/contract/`
- `loadSigningWallet()` — `cmd/wallet.go:379`
- `pickBestRPC()` — `cmd/balance.go:190`
- `resolveWalletAndChain()` — `cmd/balance.go:207`
- `rpcMock()` — `internal/chain/evm_test.go`
- `WithInMemoryStore()` + `NewInMemoryKeystore()` — `internal/wallet/manager.go`
- `config.GasLimitERC20Transfer`, `config.TxConfirmTimeout` — `internal/config/constants.go`

## Verification
```bash
/opt/homebrew/bin/go build ./...
/opt/homebrew/bin/go test ./...

# Smoke (no wallet needed)
w3cli convert 1 eth
w3cli convert 0xff
w3cli decode 0xa9059cbb0000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa960450000000000000000000000000000000000000000000000000de0b6b3a7640000
w3cli ens vitalik.eth
```
