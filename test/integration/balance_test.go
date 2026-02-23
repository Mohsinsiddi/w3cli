package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRPCServer creates a test HTTP server that mimics EVM JSON-RPC responses.
func mockRPCServer(t *testing.T, responses map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string        `json:"method"`
			Params []interface{} `json:"params"`
			ID     int           `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck

		resp, ok := responses[req.Method]
		if !ok {
			http.Error(w, "method not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  resp,
		})
	}))
}

func TestGetNativeBalance(t *testing.T) {
	server := mockRPCServer(t, map[string]interface{}{
		"eth_getBalance":  "0x1BC16D674EC80000", // 2 ETH in wei (hex)
		"eth_blockNumber": "0xE5E534",
	})
	defer server.Close()

	client := chain.NewEVMClient(server.URL)
	balance, err := client.GetBalance("0x1234567890abcdef1234567890abcdef12345678")

	require.NoError(t, err)
	assert.Equal(t, "2.000000000000000000", balance.ETH)
}

func TestGetBalanceZero(t *testing.T) {
	server := mockRPCServer(t, map[string]interface{}{
		"eth_getBalance":  "0x0",
		"eth_blockNumber": "0xE5E534",
	})
	defer server.Close()

	client := chain.NewEVMClient(server.URL)
	balance, err := client.GetBalance("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	require.NoError(t, err)
	assert.Equal(t, "0.000000000000000000", balance.ETH)
}

func TestGetERC20Balance(t *testing.T) {
	// balanceOf returns 1_000_000_000 raw (= 1000 USDC with 6 decimals)
	// 1000 * 10^6 = 1,000,000,000 = 0x3B9ACA00
	server := mockRPCServer(t, map[string]interface{}{
		"eth_call":        "0x000000000000000000000000000000000000000000000000000000003B9ACA00",
		"eth_blockNumber": "0xE5E534",
	})
	defer server.Close()

	client := chain.NewEVMClient(server.URL)
	balance, err := client.GetTokenBalance(
		"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
		"0x1234567890abcdef1234567890abcdef12345678",
		6,
	)

	require.NoError(t, err)
	assert.Equal(t, "1000.000000", balance.Formatted)
}

func TestGetBlockNumber(t *testing.T) {
	server := mockRPCServer(t, map[string]interface{}{
		"eth_blockNumber": "0x1388", // 5000
	})
	defer server.Close()

	client := chain.NewEVMClient(server.URL)
	block, err := client.GetBlockNumber()

	require.NoError(t, err)
	assert.Equal(t, uint64(5000), block)
}

func TestGetTransactionByHash(t *testing.T) {
	server := mockRPCServer(t, map[string]interface{}{
		"eth_getTransactionByHash": map[string]interface{}{
			"hash":        "0xabc123def456",
			"from":        "0x1234567890abcdef1234567890abcdef12345678",
			"to":          "0x9876543210fedcba9876543210fedcba98765432",
			"value":       "0xDE0B6B3A7640000", // 1 ETH
			"gas":         "0x5208",            // 21000
			"gasPrice":    "0x77359400",        // 2 Gwei
			"nonce":       "0x5",
			"blockNumber": "0xE5E534",
		},
	})
	defer server.Close()

	client := chain.NewEVMClient(server.URL)
	tx, err := client.GetTransactionByHash("0xabc123def456")

	require.NoError(t, err)
	assert.Equal(t, "0xabc123def456", tx.Hash)
	assert.Equal(t, uint64(21000), tx.Gas)
	assert.Equal(t, uint64(5), tx.Nonce)
}

func TestGetNonce(t *testing.T) {
	server := mockRPCServer(t, map[string]interface{}{
		"eth_getTransactionCount": "0xa", // 10
	})
	defer server.Close()

	client := chain.NewEVMClient(server.URL)
	nonce, err := client.GetNonce("0x1234567890abcdef1234567890abcdef12345678")

	require.NoError(t, err)
	assert.Equal(t, uint64(10), nonce)
}

func TestGetChainID(t *testing.T) {
	server := mockRPCServer(t, map[string]interface{}{
		"eth_chainId": "0x2105", // 8453 = Base mainnet
	})
	defer server.Close()

	client := chain.NewEVMClient(server.URL)
	id, err := client.ChainID()

	require.NoError(t, err)
	assert.Equal(t, int64(8453), id)
}

func TestGasPrice(t *testing.T) {
	server := mockRPCServer(t, map[string]interface{}{
		"eth_gasPrice": "0x77359400", // 2 Gwei
	})
	defer server.Close()

	client := chain.NewEVMClient(server.URL)
	gp, err := client.GasPrice()

	require.NoError(t, err)
	assert.Equal(t, int64(2000000000), gp.Int64()) // 2 Gwei
}
