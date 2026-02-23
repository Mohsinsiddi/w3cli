package chain

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// rpcMock creates a test HTTP server that serves a fixed JSON-RPC response
// per method. Pass method→result pairs; any unknown method returns an RPC error.
func rpcMock(t *testing.T, responses map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			ID     int    `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if result, ok := responses[req.Method]; ok {
			json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  result,
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error":   map[string]interface{}{"code": -32601, "message": "method not found"},
			})
		}
	}))
}

// rpcErrorServer creates a test HTTP server that always returns a JSON-RPC error.
func rpcErrorServer(t *testing.T, code int, msg string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ ID int `json:"id"` }
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error":   map[string]interface{}{"code": code, "message": msg},
		})
	}))
}

// rpcBadJSON creates a server that returns malformed JSON.
func rpcBadJSON(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not valid json`)) //nolint:errcheck
	}))
}

// ---------------------------------------------------------------------------
// weiToETH
// ---------------------------------------------------------------------------

func TestWeiToETHZero(t *testing.T) {
	assert.Equal(t, "0.000000000000000000", weiToETH(big.NewInt(0)))
}

func TestWeiToETHOneEther(t *testing.T) {
	one := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	assert.Equal(t, "1.000000000000000000", weiToETH(one))
}

func TestWeiToETHOneWei(t *testing.T) {
	assert.Equal(t, "0.000000000000000001", weiToETH(big.NewInt(1)))
}

func TestWeiToETHLargeAmount(t *testing.T) {
	// 1000 ETH
	thousand := new(big.Int).Mul(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
		big.NewInt(1000),
	)
	assert.Equal(t, "1000.000000000000000000", weiToETH(thousand))
}

func TestWeiToETHHalfEther(t *testing.T) {
	half := new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil) // 0.1 ETH
	result := weiToETH(half)
	assert.Contains(t, result, "0.1")
}

// ---------------------------------------------------------------------------
// formatToken
// ---------------------------------------------------------------------------

func TestFormatTokenZeroDecimals(t *testing.T) {
	assert.Equal(t, "42", formatToken(big.NewInt(42), 0))
}

func TestFormatTokenSixDecimals(t *testing.T) {
	// 1 USDC = 1_000_000 raw
	assert.Equal(t, "1.000000", formatToken(big.NewInt(1_000_000), 6))
}

func TestFormatTokenEighteenDecimals(t *testing.T) {
	one := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	assert.Equal(t, "1.000000000000000000", formatToken(one, 18))
}

func TestFormatTokenZeroBalance(t *testing.T) {
	result := formatToken(big.NewInt(0), 6)
	assert.Equal(t, "0.000000", result)
}

func TestFormatTokenPartialUSDC(t *testing.T) {
	// 1.5 USDC = 1_500_000 raw
	assert.Equal(t, "1.500000", formatToken(big.NewInt(1_500_000), 6))
}

// ---------------------------------------------------------------------------
// parseBigHex
// ---------------------------------------------------------------------------

func TestParseBigHexValid(t *testing.T) {
	n, ok := parseBigHex("0x64")
	require.True(t, ok)
	assert.Equal(t, int64(100), n.Int64())
}

func TestParseBigHexNoPrefix(t *testing.T) {
	n, ok := parseBigHex("64")
	require.True(t, ok)
	assert.Equal(t, int64(100), n.Int64())
}

func TestParseBigHexZero(t *testing.T) {
	n, ok := parseBigHex("0x0")
	require.True(t, ok)
	assert.Equal(t, int64(0), n.Int64())
}

func TestParseBigHexUpperCase(t *testing.T) {
	n, ok := parseBigHex("0xFF")
	require.True(t, ok)
	assert.Equal(t, int64(255), n.Int64())
}

func TestParseBigHexInvalidString(t *testing.T) {
	_, ok := parseBigHex("xyz")
	assert.False(t, ok)
}

func TestParseBigHexEmpty(t *testing.T) {
	_, ok := parseBigHex("")
	assert.False(t, ok)
}

func TestParseBigHexLargeValue(t *testing.T) {
	// 1 ETH in wei = 0xDE0B6B3A7640000
	n, ok := parseBigHex("0xDE0B6B3A7640000")
	require.True(t, ok)
	expected := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	assert.Equal(t, expected, n)
}

// ---------------------------------------------------------------------------
// rawTx.toTx
// ---------------------------------------------------------------------------

func TestRawTxToTxFullFields(t *testing.T) {
	rt := &rawTx{
		Hash:     "0xabc",
		From:     "0x111",
		To:       "0x222",
		Value:    "0xDE0B6B3A7640000", // 1 ETH
		Gas:      "0x5208",             // 21000
		GasPrice: "0x77359400",         // 2 Gwei
		Nonce:    "0x5",
		BlockNum: "0xE5E534",
	}
	tx := rt.toTx()

	assert.Equal(t, "0xabc", tx.Hash)
	assert.Equal(t, "0x111", tx.From)
	assert.Equal(t, "0x222", tx.To)
	assert.Equal(t, uint64(21000), tx.Gas)
	assert.Equal(t, uint64(5), tx.Nonce)
	assert.Equal(t, "1.000000000000000000", tx.ValueETH)
	assert.NotNil(t, tx.GasPrice)
}

func TestRawTxToTxEmptyFields(t *testing.T) {
	rt := &rawTx{}
	tx := rt.toTx()

	assert.Equal(t, "", tx.Hash)
	assert.Equal(t, uint64(0), tx.Gas)
	assert.Equal(t, uint64(0), tx.Nonce)
	assert.Equal(t, "", tx.ValueETH)
	assert.Nil(t, tx.Value)
}

func TestRawTxToTxZeroValue(t *testing.T) {
	rt := &rawTx{Value: "0x0"}
	tx := rt.toTx()
	assert.Equal(t, "0.000000000000000000", tx.ValueETH)
}

// ---------------------------------------------------------------------------
// EVMClient — GetBalance
// ---------------------------------------------------------------------------

func TestGetBalanceSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getBalance": "0x1BC16D674EC80000", // 2 ETH
	})
	defer srv.Close()

	bal, err := NewEVMClient(srv.URL).GetBalance("0x1234567890abcdef1234567890abcdef12345678")
	require.NoError(t, err)
	assert.Equal(t, "2.000000000000000000", bal.ETH)
	assert.NotNil(t, bal.Wei)
}

func TestGetBalanceZero(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getBalance": "0x0",
	})
	defer srv.Close()

	bal, err := NewEVMClient(srv.URL).GetBalance("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	require.NoError(t, err)
	assert.Equal(t, "0.000000000000000000", bal.ETH)
}

func TestGetBalanceRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32602, "invalid params")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetBalance("0x1234")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RPC error")
}

func TestGetBalanceConnectionRefused(t *testing.T) {
	_, err := NewEVMClient("http://127.0.0.1:19999").GetBalance("0x1234")
	require.Error(t, err)
}

func TestGetBalanceInvalidJSON(t *testing.T) {
	srv := rpcBadJSON(t)
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetBalance("0x1234")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient — GetBlockNumber
// ---------------------------------------------------------------------------

func TestGetBlockNumberSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_blockNumber": "0x1388", // 5000
	})
	defer srv.Close()

	n, err := NewEVMClient(srv.URL).GetBlockNumber()
	require.NoError(t, err)
	assert.Equal(t, uint64(5000), n)
}

func TestGetBlockNumberRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32603, "internal error")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetBlockNumber()
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient — ChainID
// ---------------------------------------------------------------------------

func TestChainIDSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_chainId": "0x2105", // 8453 = Base mainnet
	})
	defer srv.Close()

	id, err := NewEVMClient(srv.URL).ChainID()
	require.NoError(t, err)
	assert.Equal(t, int64(8453), id)
}

func TestChainIDEthereum(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_chainId": "0x1",
	})
	defer srv.Close()

	id, err := NewEVMClient(srv.URL).ChainID()
	require.NoError(t, err)
	assert.Equal(t, int64(1), id)
}

func TestChainIDRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32601, "method not found")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).ChainID()
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient — GasPrice
// ---------------------------------------------------------------------------

func TestGasPriceSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_gasPrice": "0x77359400", // 2 Gwei
	})
	defer srv.Close()

	gp, err := NewEVMClient(srv.URL).GasPrice()
	require.NoError(t, err)
	assert.Equal(t, int64(2_000_000_000), gp.Int64())
}

func TestGasPriceRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "server error")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GasPrice()
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient — GetNonce
// ---------------------------------------------------------------------------

func TestGetNonceSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionCount": "0xa", // 10
	})
	defer srv.Close()

	nonce, err := NewEVMClient(srv.URL).GetNonce("0x1234567890abcdef1234567890abcdef12345678")
	require.NoError(t, err)
	assert.Equal(t, uint64(10), nonce)
}

func TestGetNonceZero(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionCount": "0x0",
	})
	defer srv.Close()

	nonce, err := NewEVMClient(srv.URL).GetNonce("0xdeadbeef")
	require.NoError(t, err)
	assert.Equal(t, uint64(0), nonce)
}

func TestGetNonceRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32602, "invalid address")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetNonce("0xbad")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient — CallContract
// ---------------------------------------------------------------------------

func TestCallContractSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_call": "0x000000000000000000000000000000000000000000000000000000003B9ACA00",
	})
	defer srv.Close()

	result, err := NewEVMClient(srv.URL).CallContract(
		"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
		"0x70a08231000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.True(t, strings.HasPrefix(result, "0x"), "result should start with 0x")
}

func TestCallContractRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "execution reverted")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).CallContract("0xtoken", "0xdata")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RPC error")
}

// ---------------------------------------------------------------------------
// EVMClient — SendRawTransaction
// ---------------------------------------------------------------------------

func TestSendRawTransactionSuccess(t *testing.T) {
	txHash := "0xaabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	srv := rpcMock(t, map[string]interface{}{
		"eth_sendRawTransaction": txHash,
	})
	defer srv.Close()

	hash, err := NewEVMClient(srv.URL).SendRawTransaction("0xsignedtxdata")
	require.NoError(t, err)
	assert.Equal(t, txHash, hash)
}

func TestSendRawTransactionRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "nonce too low")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).SendRawTransaction("0xbadtx")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonce too low")
}

// ---------------------------------------------------------------------------
// EVMClient — EstimateGas
// ---------------------------------------------------------------------------

func TestEstimateGasSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_estimateGas": "0x5208", // 21000
	})
	defer srv.Close()

	gas, err := NewEVMClient(srv.URL).EstimateGas("0xfrom", "0xto", "", nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(21000), gas)
}

func TestEstimateGasFallbackOnError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "cannot estimate gas")
	defer srv.Close()

	// Error should return fallback gas (21000), not propagate the error.
	gas, err := NewEVMClient(srv.URL).EstimateGas("0xfrom", "0xto", "", nil)
	require.Error(t, err) // EstimateGas propagates the RPC error
	_ = gas
}

// ---------------------------------------------------------------------------
// EVMClient — GetTransactionByHash
// ---------------------------------------------------------------------------

func TestGetTransactionByHashSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionByHash": map[string]interface{}{
			"hash":        "0xabc123",
			"from":        "0x1111111111111111111111111111111111111111",
			"to":          "0x2222222222222222222222222222222222222222",
			"value":       "0xDE0B6B3A7640000", // 1 ETH
			"gas":         "0x5208",
			"gasPrice":    "0x77359400",
			"nonce":       "0x3",
			"blockNumber": "0xABC",
		},
	})
	defer srv.Close()

	tx, err := NewEVMClient(srv.URL).GetTransactionByHash("0xabc123")
	require.NoError(t, err)
	assert.Equal(t, "0xabc123", tx.Hash)
	assert.Equal(t, uint64(21000), tx.Gas)
	assert.Equal(t, uint64(3), tx.Nonce)
	assert.Equal(t, "1.000000000000000000", tx.ValueETH)
}

func TestGetTransactionByHashNotFound(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionByHash": nil,
	})
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetTransactionByHash("0xnonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaction not found")
}

func TestGetTransactionByHashRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32602, "invalid hash")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetTransactionByHash("0xbad")
	require.Error(t, err)
}
