# W3CLI — Comprehensive Test Plan

> Auto-generated test coverage roadmap for `github.com/Mohsinsiddi/w3cli`

---

## ✅ Already Tested

| Package | File | Tests |
|---------|------|-------|
| `internal/chain` | `registry_test.go` | 11 tests — chain lookup, RPC/explorer by mode, non-EVM chains |
| `internal/config` | `config_test.go` | 12 tests — load/save, custom RPCs, network mode |
| `internal/rpc` | `picker_test.go` | 10 tests — fastest/round-robin/failover, stale nodes, caching |
| `internal/wallet` | `manager_test.go` | 11 tests — add/remove, signing, default wallet, duplicate errors |
| `internal/contract` | `registry_test.go` | 22 tests — CRUD, file I/O, save/reload, ABI helpers |
| `internal/contract` | `caller_test.go` | 35 tests — functionSelector, encodeParam, decodeResult, round-trips |
| `internal/contract` | `fetcher_test.go` | 20 tests — parseABI, LoadFromFile, HTTP fetching with httptest |
| `internal/contract` | `sender_test.go` | 15 tests — hexToBytes, bytesToHex, round-trips |
| `internal/ui` | `styles_test.go` | 18 tests — formatters, TruncateAddr, Banner |
| `internal/ui` | `table_test.go` | 14 tests — Table render, KeyValueBlock, row ordering |

---

## 🔴 Missing Tests — By Package

---

### 1. `internal/chain/explorer.go` — 0 tests

#### 1.1 `decodeMethod` (pure function)

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestDecodeMethodEmptyInput` | `""` → `"transfer"` |
| 2 | `TestDecodeMethodBareHexPrefix` | `"0x"` → `"transfer"` |
| 3 | `TestDecodeMethodShortInput` | `"0xabcd"` (< 8 hex chars) → `"call"` |
| 4 | `TestDecodeMethodKnownTransfer` | `"0xa9059cbb..."` → `"transfer"` |
| 5 | `TestDecodeMethodKnownApprove` | `"0x095ea7b3..."` → `"approve"` |
| 6 | `TestDecodeMethodKnownSwap` | `"0x7ff36ab5..."` → `"swapExactETHForTokens"` |
| 7 | `TestDecodeMethodKnownDeposit` | `"0xd0e30db0"` → `"deposit"` |
| 8 | `TestDecodeMethodUnknownSelector` | `"0xdeadbeef..."` → `"0xdeadbeef"` |
| 9 | `TestDecodeMethodUpperCase` | `"0xA9059CBB..."` → `"transfer"` (case-insensitive) |
| 10 | `TestDecodeMethodAllKnownSelectors` | Loop all `knownMethods` entries |

#### 1.2 `GetTransactionsFromExplorer` (httptest mock)

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 11 | `TestGetTransactionsSuccess` | Valid response with 2 txs → parsed correctly |
| 12 | `TestGetTransactionsErrorStatus` | Status `"0"` with error string → error |
| 13 | `TestGetTransactionsInvalidJSON` | Malformed body → error |
| 14 | `TestGetTransactionsEmptyResult` | Status `"1"`, empty array → empty slice |
| 15 | `TestGetTransactionsContractDeploy` | `To=""`, `ContractAddress` set → `deploy` method |
| 16 | `TestGetTransactionsWithAPIKey` | API key appended to URL |
| 17 | `TestGetTransactionsParseValues` | Wei, gas, gasUsed, nonce, blockNum, timestamp parsed |
| 18 | `TestGetTransactionsConnectionRefused` | Unreachable URL → error |

#### 1.3 `FetchContractNames` (httptest mock)

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 19 | `TestFetchContractNamesSuccess` | Known contract → name returned |
| 20 | `TestFetchContractNamesDedup` | Duplicate addresses → single request |
| 21 | `TestFetchContractNamesUnknown` | EOA address → omitted from map |
| 22 | `TestFetchContractNamesEmpty` | Empty input → empty map |
| 23 | `TestFetchContractNamesPartialFailure` | One address fails, others succeed |

---

### 2. `internal/chain/evm.go` — 0 unit tests (only integration)

#### 2.1 Pure Functions

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestWeiToETHZero` | `0` wei → `"0.000000000000000000"` |
| 2 | `TestWeiToETHOneEther` | `1e18` wei → `"1.000000000000000000"` |
| 3 | `TestWeiToETHSmallAmount` | `1` wei → `"0.000000000000000001"` |
| 4 | `TestWeiToETHLargeAmount` | `1000 ETH` in wei → correct |
| 5 | `TestFormatTokenZeroDecimals` | `decimals=0` → raw integer string |
| 6 | `TestFormatTokenSixDecimals` | `1000000` with `decimals=6` → `"1.000000"` |
| 7 | `TestFormatTokenEighteenDecimals` | `1e18` with `decimals=18` → `"1.000000000000000000"` |
| 8 | `TestFormatTokenZeroBalance` | `0` → `"0.000000"` |
| 9 | `TestParseBigHexValid` | `"0x64"` → `100, true` |
| 10 | `TestParseBigHexNoPrefix` | `"64"` → `100, true` |
| 11 | `TestParseBigHexZero` | `"0x0"` → `0, true` |
| 12 | `TestParseBigHexInvalid` | `"xyz"` → `nil, false` |
| 13 | `TestParseBigHexEmpty` | `""` → `nil, false` |

#### 2.2 `rawTx.toTx()` (struct conversion)

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 14 | `TestRawTxToTxFullFields` | All fields populated → Transaction correct |
| 15 | `TestRawTxToTxEmptyFields` | Empty strings → zero values |
| 16 | `TestRawTxToTxValueETH` | Value parsed → ValueETH computed |

#### 2.3 EVM Client with httptest mock RPC

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 17 | `TestGetBalanceSuccess` | Mock `eth_getBalance` → Balance parsed |
| 18 | `TestGetBalanceRPCError` | RPC error response → error |
| 19 | `TestGetBlockNumberSuccess` | Mock `eth_blockNumber` → uint64 |
| 20 | `TestChainIDSuccess` | Mock `eth_chainId` → int64 |
| 21 | `TestGasPriceSuccess` | Mock `eth_gasPrice` → big.Int |
| 22 | `TestGetNonceSuccess` | Mock `eth_getTransactionCount` → uint64 |
| 23 | `TestCallContractSuccess` | Mock `eth_call` → hex result |
| 24 | `TestSendRawTransactionSuccess` | Mock `eth_sendRawTransaction` → hash |
| 25 | `TestEstimateGasSuccess` | Mock `eth_estimateGas` → uint64 |
| 26 | `TestEstimateGasFallback` | Error → fallback `100000` |
| 27 | `TestGetTransactionByHashSuccess` | Mock response → Transaction |
| 28 | `TestGetTransactionByHashNotFound` | `null` result → error |
| 29 | `TestRPCConnectionRefused` | Bad URL → error |
| 30 | `TestRPCInvalidJSON` | Malformed response → error |

---

### 3. `internal/chain/solana.go` — 0 tests

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestSolanaLamportsToSOLZero` | `0` → `"0.000000000"` |
| 2 | `TestSolanaLamportsToSOLOneSol` | `1e9` → `"1.000000000"` |
| 3 | `TestSolanaLamportsToSOLSmall` | `1` lamport → `"0.000000001"` |
| 4 | `TestSolanaLamportsToSOLLarge` | `100e9` → `"100.000000000"` |
| 5 | `TestSolanaClientGetBalanceMock` | httptest mock → Balance parsed |
| 6 | `TestSolanaClientGetSlotMock` | httptest mock → slot number |
| 7 | `TestSolanaClientRPCError` | Error response → error |

---

### 4. `internal/chain/sui.go` — 0 tests

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestSuiMistToSUIZero` | `0` → `"0.000000000"` |
| 2 | `TestSuiMistToSUIOneSUI` | `1e9` → `"1.000000000"` |
| 3 | `TestSuiMistToSUISmall` | `1` mist → `"0.000000001"` |
| 4 | `TestSUIClientGetBalanceMock` | httptest mock → Balance parsed |
| 5 | `TestSUIClientGetCheckpointMock` | httptest mock → checkpoint number |
| 6 | `TestSUIClientRPCError` | Error response → error |

---

### 5. `internal/rpc/benchmark.go` — 0 tests

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestResultsToEndpointsEmpty` | Empty input → empty output |
| 2 | `TestResultsToEndpointsHealthy` | No error → `Healthy: true, Checked: true` |
| 3 | `TestResultsToEndpointsUnhealthy` | With error → `Healthy: false, Checked: true` |
| 4 | `TestResultsToEndpointsMixed` | Mix of healthy/unhealthy |
| 5 | `TestResultsToEndpointsPreservesOrder` | Order matches input |
| 6 | `TestResultsToEndpointsPreservesLatency` | Latency values carried over |
| 7 | `TestBestEVMSingleURL` | Single URL → returned immediately (no benchmark) |

---

### 6. `internal/rpc/health.go` — 0 tests

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestHealthCheckHealthy` | Valid RPC → `Healthy: true` |
| 2 | `TestHealthCheckUnreachable` | Bad URL → `Healthy: false` |
| 3 | `TestHealthCheckStaleBehind` | Block behind threshold → `Healthy: false` |
| 4 | `TestHealthCheckNoBestBlock` | `bestBlock=0` → skip recency check |
| 5 | `TestHealthCheckTimeout` | Slow server → context deadline error |

---

### 7. `internal/price/fetcher.go` — 0 tests

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestNewFetcherDefaultCurrency` | `""` → `"usd"` |
| 2 | `TestNewFetcherCustomCurrency` | `"EUR"` → `"eur"` (lowercased) |
| 3 | `TestGetPriceKnownChain` | httptest mock → price returned |
| 4 | `TestGetPriceUnknownChain` | `"fakechain"` → error |
| 5 | `TestGetPricesCaseInsensitive` | `"Ethereum"` → matches |
| 6 | `TestGetPricesMultipleChains` | 3 chains → map with all prices |
| 7 | `TestGetPricesDedupsIDs` | `ethereum` + `base` + `arbitrum` → single `"ethereum"` ID |
| 8 | `TestGetPricesMixedKnownUnknown` | Unknown chains silently skipped |
| 9 | `TestFetchBatchHTTPError` | Connection refused → error |
| 10 | `TestFetchBatchInvalidJSON` | Bad response → error |
| 11 | `TestFetchBatchMissingCurrency` | Response missing currency key → 0 |
| 12 | `TestCoinGeckoIDsMappingComplete` | All 26 chains have a mapping |

---

### 8. `internal/wallet/keystore.go` — 0 tests

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestInMemoryKeystoreStoreAndRetrieve` | Store + Retrieve round-trip |
| 2 | `TestInMemoryKeystoreRetrieveNotFound` | Missing key → error |
| 3 | `TestInMemoryKeystoreDelete` | Delete → subsequent Retrieve fails |
| 4 | `TestInMemoryKeystoreDeleteNonExistent` | Delete missing key → no panic |
| 5 | `TestInMemoryKeystoreOverwrite` | Store same name twice → latest wins |
| 6 | `TestInMemoryKeystoreRefFormat` | Ref = `"w3cli.<name>"` |
| 7 | `TestInMemoryKeystoreMultipleKeys` | Store 3 keys, retrieve all |

---

### 9. `internal/wallet/signer.go` — 0 tests

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestSignTxSuccess` | Known private key → signed bytes non-empty |
| 2 | `TestSignTxWatchOnlyError` | Watch-only wallet → error |
| 3 | `TestSignerAddress` | Returns wallet address |
| 4 | `TestSignTxInvalidKeyRef` | Missing key in keystore → error |
| 5 | `TestSignTxDifferentChainIDs` | Sign on chainID 1 vs 8453 → different output |

---

### 10. `internal/wallet/manager.go` — Additional edge cases

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestSetDefaultNonExistent` | Non-existent wallet → `ErrWalletNotFound` |
| 2 | `TestSetDefaultClearsPrevious` | Setting new default clears old default |
| 3 | `TestDefaultNoWallets` | Empty manager → `nil` |
| 4 | `TestDefaultMultipleNoExplicit` | >1 wallets, none default → `nil` |
| 5 | `TestJSONStoreLoadNonExistent` | Missing file → `nil, nil` |
| 6 | `TestJSONStoreSaveAndReload` | Save + Load round-trip |
| 7 | `TestJSONStoreCorruptFile` | Invalid JSON → error |
| 8 | `TestStripHexPrefix` | `"0xabc"` → `"abc"`, `"abc"` → `"abc"` |

---

### 11. `internal/sync/syncer.go` — 0 tests

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestManifestParseValid` | JSON → Manifest struct |
| 2 | `TestManifestParseInvalid` | Bad JSON → error |
| 3 | `TestManifestParseEmpty` | `{}` → empty contracts map |
| 4 | `TestFetchManifestSuccess` | httptest mock → Manifest returned |
| 5 | `TestFetchManifestInvalidJSON` | Bad response → error |
| 6 | `TestFetchManifestConnectionError` | Bad URL → error |
| 7 | `TestSyncerNew` | Creates Syncer with client, registry, fetcher |
| 8 | `TestSetSourceSavesURL` | SetSource → URL persisted in sync config |
| 9 | `TestRunNoSourceConfigured` | Empty source → error with helpful message |
| 10 | `TestRunSuccessUpdatesRegistry` | Mock manifest + ABI → contracts added to registry |
| 11 | `TestRunABIFetchFailureContinues` | ABI fetch fails → warning, contract still added with nil ABI |
| 12 | `TestRunUpdatesLastSynced` | After Run → `LastSynced` timestamp updated |
| 13 | `TestWatchCancellation` | Cancelled context → returns `nil` |

---

### 12. `internal/config/config.go` — Additional edge cases

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestLoadSyncDefault` | No sync.json → empty SyncConfig |
| 2 | `TestSaveSyncAndReload` | Save + Load round-trip |
| 3 | `TestGetExplorerAPIKey` | Global key returned |
| 4 | `TestGetExplorerAPIKeyPerChain` | Per-chain override takes priority |
| 5 | `TestSetExplorerAPIKey` | Set + Get round-trip |

---

### 13. `test/e2e/` — Additional CLI edge cases

| # | Test Name | What It Tests |
|---|-----------|---------------|
| 1 | `TestConfigSetDefaultNetworkInvalid` | `w3cli config set-network fakechain` → error |
| 2 | `TestWalletUseNonExistent` | `w3cli wallet use ghost` → error |
| 3 | `TestContractAddWithNetworkFlag` | `--network base` flag works |
| 4 | `TestSendHelpShowsRequiredArgs` | `w3cli send --help` → shows arguments |
| 5 | `TestConfigSetDefaultWalletNonExistent` | Non-existent wallet → error |
| 6 | `TestBalanceNoWalletConfigured` | No default wallet → helpful error |
| 7 | `TestTxsNoExplorerAPI` | Chain without explorer API → error |

---

## 📊 Summary

| Package | Existing Tests | Missing Tests | Priority |
|---------|---------------|---------------|----------|
| `internal/contract/` | ✅ 92 | 0 | Done |
| `internal/chain/explorer.go` | ❌ 0 | **23** | 🔴 High |
| `internal/chain/evm.go` | ❌ 0 | **30** | 🔴 High |
| `internal/chain/solana.go` | ❌ 0 | **7** | 🟡 Medium |
| `internal/chain/sui.go` | ❌ 0 | **6** | 🟡 Medium |
| `internal/rpc/benchmark.go` | ❌ 0 | **7** | 🟡 Medium |
| `internal/rpc/health.go` | ❌ 0 | **5** | 🟡 Medium |
| `internal/price/fetcher.go` | ❌ 0 | **12** | 🟡 Medium |
| `internal/wallet/keystore.go` | ❌ 0 | **7** | 🟡 Medium |
| `internal/wallet/signer.go` | ❌ 0 | **5** | 🔴 High |
| `internal/wallet/manager.go` | ✅ 11 | **8** | 🟢 Low |
| `internal/sync/syncer.go` | ❌ 0 | **13** | 🔴 High |
| `internal/config/config.go` | ✅ 12 | **5** | 🟢 Low |
| `test/e2e/` | ✅ exists | **7** | 🟢 Low |
| **TOTAL** | **~135** | **~135** | |

---

## 🏗️ Implementation Order

1. **`chain/evm.go`** — Pure functions first (`weiToETH`, `formatToken`, `parseBigHex`), then httptest mocks
2. **`chain/explorer.go`** — `decodeMethod` pure function, then httptest for API calls
3. **`wallet/signer.go`** — Critical for signing correctness
4. **`sync/syncer.go`** — Full sync flow with mocked HTTP
5. **`price/fetcher.go`** — httptest mock for CoinGecko
6. **`rpc/benchmark.go`** — `ResultsToEndpoints` pure function
7. **`rpc/health.go`** — Health check with mock RPC
8. **`wallet/keystore.go`** — `InMemoryKeystore` edge cases
9. **`chain/solana.go`** — Lamports conversion + mock RPC
10. **`chain/sui.go`** — MIST conversion + mock RPC
11. **`wallet/manager.go`** — Additional edge cases
12. **`config/config.go`** — Sync config, explorer keys
13. **`test/e2e/`** — CLI-level edge cases

---

## 🧪 Running Tests

```bash
# Run all tests
go test ./... -v

# Run specific package
go test ./internal/chain/ -v
go test ./internal/contract/ -v
go test ./internal/rpc/ -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```
