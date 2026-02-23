package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// etherscanResp wraps a standard Etherscan-compatible API response.
func etherscanResp(txs []map[string]interface{}) []byte {
	b, _ := json.Marshal(txs)
	out, _ := json.Marshal(map[string]interface{}{
		"status":  "1",
		"message": "OK",
		"result":  json.RawMessage(b),
	})
	return out
}

func etherscanErrResp(msg string) []byte {
	out, _ := json.Marshal(map[string]interface{}{
		"status":  "0",
		"message": msg,
		"result":  msg,
	})
	return out
}

func minimalTx(hash string) map[string]interface{} {
	return map[string]interface{}{
		"hash":            hash,
		"from":            "0xfrom",
		"to":              "0xto",
		"value":           "1000000000000000000",
		"gas":             "21000",
		"gasUsed":         "21000",
		"gasPrice":        "2000000000",
		"nonce":           "1",
		"blockNumber":     "100",
		"timeStamp":       "1700000000",
		"isError":         "0",
		"txreceipt_status": "1",
		"input":           "0x",
	}
}

func testServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// ---------------------------------------------------------------------------
// Etherscan — constructor
// ---------------------------------------------------------------------------

func TestNewEtherscanNilWhenNoKey(t *testing.T) {
	assert.Nil(t, NewEtherscan("ethereum", ""))
}

func TestNewEtherscanNilWhenUnsupportedChain(t *testing.T) {
	assert.Nil(t, NewEtherscan("solana", "SOMEKEY"))
	assert.Nil(t, NewEtherscan("unknownchain", "SOMEKEY"))
}

func TestNewEtherscanSucceedForAllSupportedChains(t *testing.T) {
	supportedChains := []string{
		"ethereum", "base", "polygon", "arbitrum", "optimism",
		"zksync", "scroll", "bnb", "avalanche", "gnosis",
		"linea", "mantle", "celo", "fantom",
	}
	for _, name := range supportedChains {
		p := NewEtherscan(name, "KEY")
		assert.NotNil(t, p, "expected non-nil Etherscan for chain %q", name)
		assert.Equal(t, "etherscan", p.Name())
	}
}

func TestNewEtherscanChainIDMapping(t *testing.T) {
	tests := []struct {
		chain   string
		wantID  int64
	}{
		{"ethereum", 1},
		{"base", 8453},
		{"polygon", 137},
		{"arbitrum", 42161},
		{"optimism", 10},
		{"bnb", 56},
		{"avalanche", 43114},
		{"fantom", 250},
	}
	for _, tt := range tests {
		e := NewEtherscan(tt.chain, "KEY")
		require.NotNil(t, e)
		assert.Equal(t, tt.wantID, e.chainID, "wrong chainID for %q", tt.chain)
	}
}

// ---------------------------------------------------------------------------
// Etherscan — GetTransactions
// ---------------------------------------------------------------------------

func TestEtherscanGetTransactionsSuccess(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(etherscanResp([]map[string]interface{}{minimalTx("0xhash1")})) //nolint:errcheck
	})

	e := NewEtherscan("ethereum", "MYKEY")
	e.baseURL = srv.URL

	txs, err := e.GetTransactions("0xaddr", 5)
	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.Equal(t, "0xhash1", txs[0].Hash)
	assert.True(t, txs[0].Success)
	assert.Equal(t, "1.000000000000000000", txs[0].ValueETH)
}

func TestEtherscanGetTransactionsPassesChainIDAndKey(t *testing.T) {
	var capturedQuery string
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Write(etherscanResp(nil)) //nolint:errcheck
	})

	e := NewEtherscan("bnb", "SECRETKEY")
	e.baseURL = srv.URL
	e.GetTransactions("0xaddr", 10) //nolint:errcheck

	assert.Contains(t, capturedQuery, "chainid=56")
	assert.Contains(t, capturedQuery, "apikey=SECRETKEY")
}

func TestEtherscanGetTransactionsPassesOffsetAsN(t *testing.T) {
	var capturedQuery string
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Write(etherscanResp(nil)) //nolint:errcheck
	})

	e := NewEtherscan("ethereum", "K")
	e.baseURL = srv.URL
	e.GetTransactions("0xaddr", 25) //nolint:errcheck

	assert.Contains(t, capturedQuery, "offset=25")
}

func TestEtherscanGetTransactionsAPIError(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(etherscanErrResp("Max rate limit reached")) //nolint:errcheck
	})

	e := NewEtherscan("ethereum", "K")
	e.baseURL = srv.URL

	_, err := e.GetTransactions("0xaddr", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Max rate limit reached")
}

func TestEtherscanGetTransactionsConnectionRefused(t *testing.T) {
	e := NewEtherscan("ethereum", "K")
	e.baseURL = "http://127.0.0.1:19991"

	_, err := e.GetTransactions("0xaddr", 5)
	require.Error(t, err)
}

func TestEtherscanGetTransactionsEmptyResult(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(etherscanResp(nil)) //nolint:errcheck
	})

	e := NewEtherscan("ethereum", "K")
	e.baseURL = srv.URL

	txs, err := e.GetTransactions("0xaddr", 5)
	require.NoError(t, err)
	assert.Empty(t, txs)
}

func TestEtherscanGetTransactionsFailedTxMarkedFailed(t *testing.T) {
	tx := minimalTx("0xfailed")
	tx["isError"] = "1"
	tx["txreceipt_status"] = "0"

	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(etherscanResp([]map[string]interface{}{tx})) //nolint:errcheck
	})

	e := NewEtherscan("ethereum", "K")
	e.baseURL = srv.URL

	txs, err := e.GetTransactions("0xaddr", 5)
	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.False(t, txs[0].Success)
}

// ---------------------------------------------------------------------------
// Alchemy — constructor
// ---------------------------------------------------------------------------

func TestNewAlchemyNilWhenNoKey(t *testing.T) {
	assert.Nil(t, NewAlchemy("ethereum", ""))
}

func TestNewAlchemyNilWhenUnsupportedChain(t *testing.T) {
	assert.Nil(t, NewAlchemy("bnb", "KEY"))
	assert.Nil(t, NewAlchemy("fantom", "KEY"))
	assert.Nil(t, NewAlchemy("unknownchain", "KEY"))
}

func TestNewAlchemySucceedsForSupportedChains(t *testing.T) {
	chains := []string{"ethereum", "polygon", "arbitrum", "optimism", "base"}
	for _, name := range chains {
		p := NewAlchemy(name, "KEY")
		assert.NotNil(t, p, "expected non-nil Alchemy for chain %q", name)
		assert.Equal(t, "alchemy", p.Name())
	}
}

func TestNewAlchemyNetworkMapping(t *testing.T) {
	tests := []struct{ chain, network string }{
		{"ethereum", "eth-mainnet"},
		{"polygon", "polygon-mainnet"},
		{"arbitrum", "arb-mainnet"},
		{"optimism", "opt-mainnet"},
		{"base", "base-mainnet"},
	}
	for _, tt := range tests {
		a := NewAlchemy(tt.chain, "KEY")
		require.NotNil(t, a)
		assert.Equal(t, tt.network, a.network, "wrong network for %q", tt.chain)
	}
}

// alchemyOKResp builds a minimal alchemy_getAssetTransfers response.
func alchemyOKResp(transfers []map[string]interface{}) []byte {
	out, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]interface{}{
			"transfers": transfers,
		},
	})
	return out
}

func makeAlchemyTransfer(hash, from, to string, value float64, blockHex string) map[string]interface{} {
	return map[string]interface{}{
		"hash":     hash,
		"from":     from,
		"to":       to,
		"value":    value,
		"blockNum": blockHex,
		"metadata": map[string]interface{}{
			"blockTimestamp": "2024-01-01T00:00:00.000Z",
		},
	}
}

// ---------------------------------------------------------------------------
// Alchemy — GetTransactions
// ---------------------------------------------------------------------------

func TestAlchemyGetTransactionsSuccess(t *testing.T) {
	// Single server returns 1 transfer for both from and to queries.
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(alchemyOKResp([]map[string]interface{}{ //nolint:errcheck
			makeAlchemyTransfer("0xhash1", "0xfrom", "0xto", 1.0, "0x64"),
		}))
	})

	a := NewAlchemy("ethereum", "KEY")
	a.baseURL = srv.URL

	// The server returns the same tx for both from+to queries;
	// dedup should keep exactly one entry.
	txs, err := a.GetTransactions("0xfrom", 5)
	require.NoError(t, err)
	assert.Len(t, txs, 1)
	assert.Equal(t, "0xhash1", txs[0].Hash)
}

func TestAlchemyGetTransactionsValueConversion(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(alchemyOKResp([]map[string]interface{}{ //nolint:errcheck
			makeAlchemyTransfer("0xhash", "0xa", "0xb", 0.5, "0x1"),
		}))
	})

	a := NewAlchemy("ethereum", "KEY")
	a.baseURL = srv.URL

	txs, err := a.GetTransactions("0xa", 5)
	require.NoError(t, err)
	require.Len(t, txs, 1)

	// 0.5 ETH = 5e17 wei → ValueETH should be "0.500000000000000000"
	assert.Equal(t, "0.500000000000000000", txs[0].ValueETH)
	require.NotNil(t, txs[0].Value)
	assert.Equal(t, "500000000000000000", txs[0].Value.String())
}

func TestAlchemyGetTransactionsDeduplicatesByHash(t *testing.T) {
	// Both from + to queries return the same hash → should appear only once.
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(alchemyOKResp([]map[string]interface{}{ //nolint:errcheck
			makeAlchemyTransfer("0xDUP", "0xa", "0xb", 1.0, "0x2"),
		}))
	})

	a := NewAlchemy("ethereum", "KEY")
	a.baseURL = srv.URL

	txs, err := a.GetTransactions("0xa", 10)
	require.NoError(t, err)
	assert.Len(t, txs, 1, "duplicate hash must be deduplicated")
}

func TestAlchemyGetTransactionsSortedByBlockDesc(t *testing.T) {
	callCount := 0
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		var transfers []map[string]interface{}
		if callCount == 1 {
			// "from" query returns block 10 and 5
			transfers = []map[string]interface{}{
				makeAlchemyTransfer("0xA", "0xme", "0xother", 0.1, "0xa"),  // block 10
				makeAlchemyTransfer("0xB", "0xme", "0xother2", 0.1, "0x5"), // block 5
			}
		} else {
			// "to" query returns block 20 and 1
			transfers = []map[string]interface{}{
				makeAlchemyTransfer("0xC", "0xsender", "0xme", 0.1, "0x14"), // block 20
				makeAlchemyTransfer("0xD", "0xsender2", "0xme", 0.1, "0x1"), // block 1
			}
		}
		w.Write(alchemyOKResp(transfers)) //nolint:errcheck
	})

	a := NewAlchemy("ethereum", "KEY")
	a.baseURL = srv.URL

	txs, err := a.GetTransactions("0xme", 10)
	require.NoError(t, err)
	require.Len(t, txs, 4)

	// Should be sorted by BlockNum descending: 20, 10, 5, 1
	blockNums := make([]uint64, len(txs))
	for i, tx := range txs {
		blockNums[i] = tx.BlockNum
	}
	assert.True(t, sort.SliceIsSorted(blockNums, func(i, j int) bool {
		return blockNums[i] > blockNums[j]
	}), "transactions must be sorted by block number descending")
}

func TestAlchemyGetTransactionsTruncatesToN(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		transfers := make([]map[string]interface{}, 8)
		for i := range transfers {
			transfers[i] = makeAlchemyTransfer(fmt.Sprintf("0x%02x", i), "0xa", "0xb", 0.1, fmt.Sprintf("0x%x", i+1))
		}
		w.Write(alchemyOKResp(transfers)) //nolint:errcheck
	})

	a := NewAlchemy("ethereum", "KEY")
	a.baseURL = srv.URL

	txs, err := a.GetTransactions("0xa", 3)
	require.NoError(t, err)
	assert.Len(t, txs, 3, "result must be capped at n=3")
}

func TestAlchemyGetTransactionsAPIError(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		resp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"error":   map[string]interface{}{"message": "invalid API key"},
		})
		w.Write(resp) //nolint:errcheck
	})

	a := NewAlchemy("ethereum", "BADKEY")
	a.baseURL = srv.URL

	_, err := a.GetTransactions("0xaddr", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
}

func TestAlchemyGetTransactionsConnectionRefused(t *testing.T) {
	a := NewAlchemy("ethereum", "KEY")
	a.baseURL = "http://127.0.0.1:19992"

	_, err := a.GetTransactions("0xaddr", 5)
	require.Error(t, err)
}

func TestAlchemyGetTransactionsEmptyResult(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(alchemyOKResp(nil)) //nolint:errcheck
	})

	a := NewAlchemy("ethereum", "KEY")
	a.baseURL = srv.URL

	txs, err := a.GetTransactions("0xaddr", 5)
	require.NoError(t, err)
	assert.Empty(t, txs)
}

func TestAlchemyGetTransactionsSuccessMarkedTrue(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(alchemyOKResp([]map[string]interface{}{ //nolint:errcheck
			makeAlchemyTransfer("0xok", "0xa", "0xb", 0.0, "0x1"),
		}))
	})

	a := NewAlchemy("ethereum", "KEY")
	a.baseURL = srv.URL

	txs, err := a.GetTransactions("0xa", 5)
	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.True(t, txs[0].Success, "alchemy confirmed transfers must be marked Success=true")
}

// ---------------------------------------------------------------------------
// Moralis — constructor
// ---------------------------------------------------------------------------

func TestNewMoralisNilWhenNoKey(t *testing.T) {
	assert.Nil(t, NewMoralis("ethereum", ""))
}

func TestNewMoralisNilWhenUnsupportedChain(t *testing.T) {
	assert.Nil(t, NewMoralis("unknownchain", "KEY"))
	assert.Nil(t, NewMoralis("solana", "KEY"))
}

func TestNewMoralisSucceedsForSupportedChains(t *testing.T) {
	chains := []string{
		"ethereum", "base", "polygon", "arbitrum", "optimism",
		"bnb", "avalanche", "fantom", "gnosis", "linea",
		"scroll", "zksync", "mantle", "celo",
	}
	for _, name := range chains {
		p := NewMoralis(name, "KEY")
		assert.NotNil(t, p, "expected non-nil Moralis for chain %q", name)
		assert.Equal(t, "moralis", p.Name())
	}
}

func TestNewMoralisHexChainIDMapping(t *testing.T) {
	tests := []struct{ chain, hex string }{
		{"ethereum", "0x1"},
		{"base", "0x2105"},
		{"bnb", "0x38"},
		{"polygon", "0x89"},
		{"fantom", "0xfa"},
	}
	for _, tt := range tests {
		m := NewMoralis(tt.chain, "KEY")
		require.NotNil(t, m)
		assert.Equal(t, tt.hex, m.hexChain, "wrong hex chain for %q", tt.chain)
	}
}

// moralisOKResp builds a Moralis transaction list response.
func moralisOKResp(txs []map[string]interface{}) []byte {
	out, _ := json.Marshal(map[string]interface{}{"result": txs})
	return out
}

func makeMoralisTx(hash string) map[string]interface{} {
	return map[string]interface{}{
		"hash":                    hash,
		"from_address":            "0xfrom",
		"to_address":              "0xto",
		"value":                   "1000000000000000000",
		"gas":                     "21000",
		"gas_price":               "2000000000",
		"receipt_gas_used":        "21000",
		"nonce":                   "5",
		"block_number":            "12345678",
		"receipt_status":          "1",
		"block_timestamp":         "2024-01-01T00:00:00.000Z",
		"method_label":            nil,
		"receipt_contract_address": nil,
	}
}

// ---------------------------------------------------------------------------
// Moralis — GetTransactions
// ---------------------------------------------------------------------------

func TestMoralisGetTransactionsSuccess(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(moralisOKResp([]map[string]interface{}{makeMoralisTx("0xhash1")})) //nolint:errcheck
	})

	m := NewMoralis("ethereum", "MYKEY")
	m.baseURL = srv.URL

	txs, err := m.GetTransactions("0xaddr", 5)
	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.Equal(t, "0xhash1", txs[0].Hash)
	assert.True(t, txs[0].Success)
}

func TestMoralisGetTransactionsSendsAPIKeyHeader(t *testing.T) {
	var capturedKey string
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedKey = r.Header.Get("X-API-Key")
		w.Write(moralisOKResp(nil)) //nolint:errcheck
	})

	m := NewMoralis("ethereum", "SECRETMORALISKEY")
	m.baseURL = srv.URL

	m.GetTransactions("0xaddr", 5) //nolint:errcheck
	assert.Equal(t, "SECRETMORALISKEY", capturedKey)
}

func TestMoralisGetTransactionsSendsCorrectChainParam(t *testing.T) {
	var capturedQuery string
	var capturedPath string
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		capturedPath = r.URL.Path
		w.Write(moralisOKResp(nil)) //nolint:errcheck
	})

	m := NewMoralis("bnb", "KEY")
	m.baseURL = srv.URL
	m.GetTransactions("0xaddr", 10) //nolint:errcheck

	assert.Contains(t, capturedPath, "/wallets/0xaddr/history", "must use /wallets/{addr}/history endpoint")
	assert.Contains(t, capturedQuery, "chain=0x38")
	assert.Contains(t, capturedQuery, "limit=10")
}

func TestMoralisGetTransactionsFieldParsing(t *testing.T) {
	tx := makeMoralisTx("0xparsed")
	tx["value"] = "500000000000000000" // 0.5 ETH
	tx["gas"] = "42000"
	tx["receipt_gas_used"] = "30000"
	tx["gas_price"] = "3000000000"
	tx["nonce"] = "7"
	tx["block_number"] = "19000000"

	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(moralisOKResp([]map[string]interface{}{tx})) //nolint:errcheck
	})

	m := NewMoralis("ethereum", "KEY")
	m.baseURL = srv.URL

	txs, err := m.GetTransactions("0xaddr", 5)
	require.NoError(t, err)
	require.Len(t, txs, 1)

	got := txs[0]
	assert.Equal(t, "0.500000000000000000", got.ValueETH)
	assert.Equal(t, uint64(42000), got.Gas)
	assert.Equal(t, uint64(30000), got.GasUsed)
	assert.Equal(t, uint64(7), got.Nonce)
	assert.Equal(t, uint64(19000000), got.BlockNum)
	require.NotNil(t, got.GasPrice)
	assert.Equal(t, "3000000000", got.GasPrice.String())
}

func TestMoralisGetTransactionsFailedTxStatus(t *testing.T) {
	tx := makeMoralisTx("0xfailed")
	tx["receipt_status"] = "0"

	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(moralisOKResp([]map[string]interface{}{tx})) //nolint:errcheck
	})

	m := NewMoralis("ethereum", "KEY")
	m.baseURL = srv.URL

	txs, err := m.GetTransactions("0xaddr", 5)
	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.False(t, txs[0].Success)
}

func TestMoralisGetTransactionsContractCall(t *testing.T) {
	tx := makeMoralisTx("0xcontract")
	label := "swapExactETHForTokens"
	tx["method_label"] = label

	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(moralisOKResp([]map[string]interface{}{tx})) //nolint:errcheck
	})

	m := NewMoralis("ethereum", "KEY")
	m.baseURL = srv.URL

	txs, err := m.GetTransactions("0xaddr", 5)
	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.True(t, txs[0].IsContract)
	assert.Equal(t, "swapExactETHForTokens", txs[0].FunctionName)
}

func TestMoralisGetTransactionsHTTPError(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	m := NewMoralis("ethereum", "BADKEY")
	m.baseURL = srv.URL

	_, err := m.GetTransactions("0xaddr", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestMoralisGetTransactionsConnectionRefused(t *testing.T) {
	m := NewMoralis("ethereum", "KEY")
	m.baseURL = "http://127.0.0.1:19993"

	_, err := m.GetTransactions("0xaddr", 5)
	require.Error(t, err)
}

func TestMoralisGetTransactionsEmptyResult(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write(moralisOKResp(nil)) //nolint:errcheck
	})

	m := NewMoralis("ethereum", "KEY")
	m.baseURL = srv.URL

	txs, err := m.GetTransactions("0xaddr", 5)
	require.NoError(t, err)
	assert.Empty(t, txs)
}

func TestMoralisGetTransactionsInvalidJSON(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{not json}`)) //nolint:errcheck
	})

	m := NewMoralis("ethereum", "KEY")
	m.baseURL = srv.URL

	_, err := m.GetTransactions("0xaddr", 5)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Factory — BuildRegistry
// ---------------------------------------------------------------------------

func newTestChain(name string, chainID int64, explorerAPI string) *chain.Chain {
	return &chain.Chain{
		Name:               name,
		ChainID:            chainID,
		MainnetExplorerAPI: explorerAPI,
	}
}

func newTestConfig(keys map[string]string) *config.Config {
	return &config.Config{
		ProviderKeys: keys,
	}
}

func TestBuildRegistryNoKeysEthereumChain(t *testing.T) {
	c := newTestChain("ethereum", 1, "https://eth.blockscout.com/api")
	cfg := newTestConfig(nil)

	reg := BuildRegistry("ethereum", c, "mainnet", "http://rpc", cfg)
	names := reg.Names()

	// With no keys: BlockScout (has explorer) + Ankr (chain supported) + RPC
	assert.Equal(t, []string{"blockscout", "ankr", "rpc"}, names)
}

func TestBuildRegistryNoKeysChainWithoutExplorer(t *testing.T) {
	// BNB has no free BlockScout API
	c := newTestChain("bnb", 56, "")
	cfg := newTestConfig(nil)

	reg := BuildRegistry("bnb", c, "mainnet", "http://rpc", cfg)
	names := reg.Names()

	// No blockscout (no explorerAPI), ankr supported for bnb, rpc fallback
	assert.Equal(t, []string{"ankr", "rpc"}, names)
}

func TestBuildRegistryNoKeysUnsupportedAnkrChain(t *testing.T) {
	// Use a chain unsupported by Ankr (e.g. "hyperliquid")
	c := newTestChain("hyperliquid", 998, "")
	cfg := newTestConfig(nil)

	reg := BuildRegistry("hyperliquid", c, "mainnet", "http://rpc", cfg)
	names := reg.Names()

	// No explorer, no ankr → only rpc
	assert.Equal(t, []string{"rpc"}, names)
}

func TestBuildRegistryEtherscanKeyAdded(t *testing.T) {
	c := newTestChain("ethereum", 1, "https://eth.blockscout.com/api")
	cfg := newTestConfig(map[string]string{"etherscan": "ESKEY"})

	reg := BuildRegistry("ethereum", c, "mainnet", "http://rpc", cfg)
	names := reg.Names()

	// Etherscan is first, before blockscout
	assert.Equal(t, "etherscan", names[0])
	assert.Contains(t, names, "blockscout")
	assert.Contains(t, names, "rpc")
}

func TestBuildRegistryAlchemyKeyAdded(t *testing.T) {
	c := newTestChain("ethereum", 1, "https://eth.blockscout.com/api")
	cfg := newTestConfig(map[string]string{"alchemy": "ALKKEY"})

	reg := BuildRegistry("ethereum", c, "mainnet", "http://rpc", cfg)
	names := reg.Names()

	assert.Contains(t, names, "alchemy")
	assert.Less(t, indexOf(names, "alchemy"), indexOf(names, "blockscout"),
		"alchemy must come before blockscout")
}

func TestBuildRegistryMoralisKeyAdded(t *testing.T) {
	c := newTestChain("bnb", 56, "") // bnb: no free explorer
	cfg := newTestConfig(map[string]string{"moralis": "MORKEY"})

	reg := BuildRegistry("bnb", c, "mainnet", "http://rpc", cfg)
	names := reg.Names()

	assert.Contains(t, names, "moralis")
	assert.Contains(t, names, "rpc")
}

func TestBuildRegistryAllKeysSetEthereumOrder(t *testing.T) {
	c := newTestChain("ethereum", 1, "https://eth.blockscout.com/api")
	cfg := newTestConfig(map[string]string{
		"etherscan": "ESKEY",
		"alchemy":   "ALKKEY",
		"moralis":   "MORKEY",
		"ankr":      "ANKRKEY",
	})

	reg := BuildRegistry("ethereum", c, "mainnet", "http://rpc", cfg)
	names := reg.Names()

	// Expected order: etherscan, alchemy, moralis, blockscout, ankr, rpc
	assert.Equal(t, []string{"etherscan", "alchemy", "moralis", "blockscout", "ankr", "rpc"}, names)
}

func TestBuildRegistryAllKeysSetBNBOrder(t *testing.T) {
	// BNB: supported by etherscan + moralis + ankr, NOT alchemy (no bnb in alchemyNetwork)
	c := newTestChain("bnb", 56, "")
	cfg := newTestConfig(map[string]string{
		"etherscan": "ESKEY",
		"alchemy":   "ALKKEY",
		"moralis":   "MORKEY",
		"ankr":      "ANKRKEY",
	})

	reg := BuildRegistry("bnb", c, "mainnet", "http://rpc", cfg)
	names := reg.Names()

	// No alchemy (bnb unsupported), no blockscout (no explorerAPI)
	assert.Equal(t, []string{"etherscan", "moralis", "ankr", "rpc"}, names)
}

func TestBuildRegistryNilProvidersFiltered(t *testing.T) {
	// A chain that is not supported by any keyed provider and has no free explorer.
	c := newTestChain("celo", 42220, "")
	cfg := newTestConfig(map[string]string{
		"alchemy": "ALKKEY", // celo not in alchemy map → nil
	})

	reg := BuildRegistry("celo", c, "mainnet", "http://rpc", cfg)
	names := reg.Names()

	// alchemy is nil (celo unsupported), no blockscout, ankr is supported, rpc
	assert.NotContains(t, names, "alchemy")
	assert.Contains(t, names, "rpc")
}

func TestBuildRegistryRPCAlwaysLast(t *testing.T) {
	for _, chainName := range []string{"ethereum", "bnb", "fantom", "scroll"} {
		c := newTestChain(chainName, 1, "")
		cfg := newTestConfig(map[string]string{
			"etherscan": "KEY",
			"alchemy":   "KEY",
			"moralis":   "KEY",
		})
		reg := BuildRegistry(chainName, c, "mainnet", "http://rpc", cfg)
		names := reg.Names()
		require.NotEmpty(t, names)
		assert.Equal(t, "rpc", names[len(names)-1], "rpc must always be last for chain %q", chainName)
	}
}

// ---------------------------------------------------------------------------
// Registry — GetTransactions fallback behaviour
// ---------------------------------------------------------------------------

func TestRegistryReturnsFirstSuccessfulProvider(t *testing.T) {
	// p1 fails, p2 succeeds → result.Source should be p2's name.
	p1 := &stubProvider{name: "fail", err: fmt.Errorf("unavailable")}
	p2 := &stubProvider{name: "ok", txs: []*chain.Transaction{{Hash: "0xabc"}}}

	reg := New(p1, p2)
	result, err := reg.GetTransactions("0xaddr", 5)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Source)
	assert.Len(t, result.Txs, 1)
	assert.Contains(t, result.Warnings, "fail: unavailable")
}

func TestRegistrySkipsEmptyResultAndTriesNext(t *testing.T) {
	// p1 returns 0 txs (not an error), p2 has data.
	p1 := &stubProvider{name: "empty", txs: nil}
	p2 := &stubProvider{name: "data", txs: []*chain.Transaction{{Hash: "0xdef"}}}

	reg := New(p1, p2)
	result, err := reg.GetTransactions("0xaddr", 5)
	require.NoError(t, err)
	assert.Equal(t, "data", result.Source)
	assert.Contains(t, result.Warnings, "empty: no transactions found")
}

func TestRegistryAllFailReturnsWarnings(t *testing.T) {
	p1 := &stubProvider{name: "a", err: fmt.Errorf("timeout")}
	p2 := &stubProvider{name: "b", err: fmt.Errorf("rate limited")}

	reg := New(p1, p2)
	result, _ := reg.GetTransactions("0xaddr", 5)
	assert.Contains(t, result.Warnings, "a: timeout")
	assert.Contains(t, result.Warnings, "b: rate limited")
}

func TestRegistryNamesReturnedInOrder(t *testing.T) {
	p1 := &stubProvider{name: "first"}
	p2 := &stubProvider{name: "second"}
	p3 := &stubProvider{name: "third"}

	reg := New(p1, p2, p3)
	assert.Equal(t, []string{"first", "second", "third"}, reg.Names())
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type stubProvider struct {
	name string
	txs  []*chain.Transaction
	err  error
}

func (s *stubProvider) Name() string { return s.name }
func (s *stubProvider) GetTransactions(_ string, _ int) ([]*chain.Transaction, error) {
	return s.txs, s.err
}

func indexOf(slice []string, v string) int {
	for i, s := range slice {
		if s == v {
			return i
		}
	}
	return -1
}
