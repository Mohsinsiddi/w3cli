# w3cli — Multi-Provider Architecture for Transaction History

## Overview

`w3cli txs` uses a **priority-ordered provider registry** to fetch transaction history. Each provider is tried in sequence; the first one that returns data wins. Providers that require an API key are automatically skipped when no key is configured.

---

## Provider Priority Order

```
1. Etherscan V2   — key-gated (etherscan.io/apis)
2. Alchemy        — key-gated (alchemy.com)
3. Moralis        — key-gated (moralis.io)
4. BlockScout     — free, no key (chain must have an explorer API)
5. Ankr           — free tier (higher limits with key)
6. RPC            — always available fallback (last 200 blocks only)
```

**Free providers (4–6) always work by default.** Paid providers (1–3) activate only when you configure their key.

---

## Provider Details

### 1. Etherscan V2 _(key-gated)_

| Attribute | Value |
|---|---|
| API | `https://api.etherscan.io/v2/api?chainid={chainID}&...` |
| One key covers | 14+ EVM chains via a single key |
| Get key | [etherscan.io/apis](https://etherscan.io/apis) |
| Enable | `w3cli config set-key etherscan <key>` |

**Supported chains:** ethereum (1), base (8453), polygon (137), arbitrum (42161), optimism (10), zksync (324), scroll (534352), bnb (56), avalanche (43114), gnosis (100), linea (59144), mantle (5000), celo (42220), fantom (250)

---

### 2. Alchemy _(key-gated)_

| Attribute | Value |
|---|---|
| API | `https://{network}.g.alchemy.com/v2/{key}` — `alchemy_getAssetTransfers` |
| Get key | [alchemy.com](https://alchemy.com) (free tier available) |
| Enable | `w3cli config set-key alchemy <key>` |

**Supported chains:** ethereum (`eth-mainnet`), polygon (`polygon-mainnet`), arbitrum (`arb-mainnet`), optimism (`opt-mainnet`), base (`base-mainnet`)

**Notes:**
- Makes two requests per call: one for `fromAddress`, one for `toAddress`
- Deduplicates by hash, sorts by block number descending
- Value is returned as a float (ETH, not wei) — converted internally to wei then `WeiToETH`

---

### 3. Moralis _(key-gated)_

| Attribute | Value |
|---|---|
| API | `GET https://deep-index.moralis.io/api/v2.2/{address}/transactions?chain={hex}&limit={n}` |
| Header | `X-API-Key: {key}` |
| Get key | [moralis.io](https://moralis.io) (free tier available) |
| Enable | `w3cli config set-key moralis <key>` |

**Supported chains:** ethereum (`0x1`), base (`0x2105`), polygon (`0x89`), arbitrum (`0xa4b1`), optimism (`0xa`), bnb (`0x38`), avalanche (`0xa86a`), fantom (`0xfa`), gnosis (`0x64`), linea (`0xe708`), scroll (`0x82750`), zksync (`0x144`), mantle (`0x1388`), celo (`0xa4ec`)

---

### 4. BlockScout _(free, no key)_

| Attribute | Value |
|---|---|
| API | Etherscan-compatible `/api?module=account&action=txlist` |
| Availability | Chains that have `mainnet_explorer_api` set in the registry |

**Working endpoints (mainnet):**

| Chain | Status | URL |
|---|---|---|
| ethereum | ✅ | `eth.blockscout.com/api` |
| base | ✅ | `base.blockscout.com/api` |
| polygon | ✅ | `polygon.blockscout.com/api` |
| arbitrum | ✅ | `arbitrum.blockscout.com/api` |
| zksync | ✅ | `zksync.blockscout.com/api` |
| scroll | ✅ | `scroll.blockscout.com/api` |
| avalanche | ✅ | `api.snowtrace.io/api` |
| optimism | ✅ | `explorer.optimism.io/api` |
| bnb | ❌ | No free BlockScout endpoint — use Etherscan/Moralis/Ankr |
| fantom | ❌ | No free BlockScout endpoint — use Etherscan/Moralis/Ankr |
| linea | ❌ | Dead endpoint — use Etherscan/Moralis/Ankr |
| mantle | ❌ | Dead endpoint — use Etherscan/Moralis/Ankr |

---

### 5. Ankr _(free tier, optional key)_

| Attribute | Value |
|---|---|
| API | `POST https://rpc.ankr.com/multichain` — `ankr_getTransactionsByAddress` |
| Free tier | 30 req/s, 200M credits/month — **no key required** |
| Get key | [ankr.com/rpc/apps](https://ankr.com/rpc/apps) |
| Enable key | `w3cli config set-key ankr <key>` |

**Supported chains:** ethereum (`eth`), base (`base`), polygon (`polygon`), arbitrum (`arbitrum`), optimism (`optimism`), zksync (`zksync_era`), scroll (`scroll`), bnb (`bsc`), fantom (`fantom`), linea (`linea`), mantle (`mantle`), avalanche (`avalanche`), gnosis (`gnosis`), celo (`celo`)

---

### 6. RPC _(always available)_

Scans the **last 200 blocks** on the chain's JSON-RPC endpoint. Always available as a last-resort fallback.

**Limitations:** Only shows transactions from the last ~200 blocks. For older history, configure at least one indexed provider above.

---

## Setting API Keys

```bash
w3cli config set-key etherscan <key>   # Etherscan V2 — covers 14+ chains
w3cli config set-key alchemy <key>     # Alchemy — ETH, Base, Polygon, ARB, OP
w3cli config set-key moralis <key>     # Moralis — 14+ chains incl. BNB, FTM
w3cli config set-key ankr <key>        # Ankr — higher rate limits (free already works)
```

Verify saved keys:
```bash
w3cli config list --verbose
```

---

## Per-Chain Provider Coverage

| Chain | Free (no key) | With key |
|---|---|---|
| ethereum | BlockScout + Ankr + RPC | + Etherscan + Alchemy + Moralis |
| base | BlockScout + Ankr + RPC | + Etherscan + Alchemy + Moralis |
| polygon | BlockScout + Ankr + RPC | + Etherscan + Alchemy + Moralis |
| arbitrum | BlockScout + Ankr + RPC | + Etherscan + Alchemy + Moralis |
| optimism | BlockScout + Ankr + RPC | + Etherscan + Alchemy + Moralis |
| zksync | BlockScout + Ankr + RPC | + Etherscan + Moralis |
| scroll | BlockScout + Ankr + RPC | + Etherscan + Moralis |
| avalanche | BlockScout + Ankr + RPC | + Etherscan + Moralis |
| bnb | Ankr + RPC | + Etherscan + Moralis |
| fantom | Ankr + RPC | + Etherscan + Moralis |
| linea | Ankr + RPC | + Etherscan + Moralis |
| mantle | Ankr + RPC | + Etherscan + Moralis |
| gnosis | Ankr + RPC | + Etherscan + Moralis |
| celo | Ankr + RPC | + Etherscan + Moralis |

---

## Architecture

### Interface

```go
// internal/providers/provider.go
type Provider interface {
    Name() string
    GetTransactions(address string, n int) ([]*chain.Transaction, error)
}
```

### Registry (fallback chain)

```go
// First provider that returns data wins. Others add warnings.
result, err := registry.GetTransactions(address, n)
// result.Source   — name of the provider that succeeded
// result.Txs      — the transactions
// result.Warnings — non-fatal errors from providers that were tried first
```

### Factory

```go
// internal/providers/factory.go
reg := providers.BuildRegistry(chainName, c, networkMode, rpcURL, cfg)
```

`BuildRegistry` assembles the ordered provider list based on:
- Which chains each provider supports
- Which API keys are configured in `~/.w3cli/config.json`
- `nil` returns from constructors (unsupported chain / no key) are automatically filtered

### File Layout

```
internal/providers/
  provider.go       — Provider interface, Registry, ErrAllFailed
  blockscout.go     — BlockScout / Etherscan-compatible explorer API (free)
  ankr.go           — Ankr Advanced API (JSON-RPC, free tier)
  etherscan.go      — Etherscan V2 unified gateway (key-gated)
  alchemy.go        — Alchemy alchemy_getAssetTransfers (key-gated)
  moralis.go        — Moralis Deep Index (key-gated)
  factory.go        — BuildRegistry() assembles providers in priority order
  rpc.go            — RPC block scan fallback (always last)
  providers_test.go — 54 unit tests covering all providers + factory + registry
```
