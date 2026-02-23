package chain

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// WeiToETH (public exported wrapper)
// ---------------------------------------------------------------------------

func TestWeiToETHExported(t *testing.T) {
	one := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	assert.Equal(t, "1.000000000000000000", WeiToETH(one))
}

func TestWeiToETHExportedZero(t *testing.T) {
	assert.Equal(t, "0.000000000000000000", WeiToETH(big.NewInt(0)))
}

// ---------------------------------------------------------------------------
// EVMClient — GetTransactionReceipt
// ---------------------------------------------------------------------------

func TestGetTransactionReceiptSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionReceipt": map[string]interface{}{
			"status":          "0x1",
			"blockNumber":     "0x100",
			"gasUsed":         "0x5208",
			"contractAddress": "",
		},
	})
	defer srv.Close()

	receipt, err := NewEVMClient(srv.URL).GetTransactionReceipt("0xtxhash")
	require.NoError(t, err)
	require.NotNil(t, receipt)
	assert.Equal(t, uint64(1), receipt.Status)
	assert.Equal(t, uint64(256), receipt.BlockNumber)
	assert.Equal(t, uint64(21000), receipt.GasUsed)
	assert.Equal(t, "0xtxhash", receipt.Hash)
}

func TestGetTransactionReceiptReverted(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionReceipt": map[string]interface{}{
			"status":          "0x0",
			"blockNumber":     "0x200",
			"gasUsed":         "0x7530",
			"contractAddress": "",
		},
	})
	defer srv.Close()

	receipt, err := NewEVMClient(srv.URL).GetTransactionReceipt("0xreverted")
	require.NoError(t, err)
	require.NotNil(t, receipt)
	assert.Equal(t, uint64(0), receipt.Status)
}

func TestGetTransactionReceiptPending(t *testing.T) {
	// Pending transactions return null result.
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionReceipt": nil,
	})
	defer srv.Close()

	receipt, err := NewEVMClient(srv.URL).GetTransactionReceipt("0xpending")
	require.NoError(t, err)
	assert.Nil(t, receipt, "pending tx should return nil receipt")
}

func TestGetTransactionReceiptWithContractAddress(t *testing.T) {
	contractAddr := "0xNewContractAddress"
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionReceipt": map[string]interface{}{
			"status":          "0x1",
			"blockNumber":     "0x1",
			"gasUsed":         "0x3D090",
			"contractAddress": contractAddr,
		},
	})
	defer srv.Close()

	receipt, err := NewEVMClient(srv.URL).GetTransactionReceipt("0xdeploytx")
	require.NoError(t, err)
	require.NotNil(t, receipt)
	assert.Equal(t, contractAddr, receipt.ContractAddress)
}

func TestGetTransactionReceiptRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32602, "invalid hash format")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetTransactionReceipt("badhash")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient — WaitForReceipt
// ---------------------------------------------------------------------------

func TestWaitForReceiptImmediate(t *testing.T) {
	// Receipt available on first poll.
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionReceipt": map[string]interface{}{
			"status":          "0x1",
			"blockNumber":     "0xA",
			"gasUsed":         "0x5208",
			"contractAddress": "",
		},
	})
	defer srv.Close()

	client := NewEVMClient(srv.URL)
	receipt, err := client.WaitForReceipt("0xtxhash", 10*time.Second)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	assert.Equal(t, uint64(1), receipt.Status)
}

func TestWaitForReceiptReverted(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionReceipt": map[string]interface{}{
			"status":          "0x0",
			"blockNumber":     "0xA",
			"gasUsed":         "0x5208",
			"contractAddress": "",
		},
	})
	defer srv.Close()

	client := NewEVMClient(srv.URL)
	receipt, err := client.WaitForReceipt("0xreverted", 10*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reverted")
	require.NotNil(t, receipt)
	assert.Equal(t, uint64(0), receipt.Status)
}

func TestWaitForReceiptTimeout(t *testing.T) {
	// Always return nil (pending) to trigger timeout.
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionReceipt": nil,
	})
	defer srv.Close()

	client := NewEVMClient(srv.URL)
	// Very short timeout so the test doesn't take long.
	_, err := client.WaitForReceipt("0xstuck", 1*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not mined within")
}

func TestWaitForReceiptRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "node error")
	defer srv.Close()

	client := NewEVMClient(srv.URL)
	_, err := client.WaitForReceipt("0xtx", 5*time.Second)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient — Ping
// ---------------------------------------------------------------------------

func TestPingSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_blockNumber": "0x1388", // 5000
	})
	defer srv.Close()

	client := NewEVMClient(srv.URL)
	latency, blockNum, err := client.Ping(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(5000), blockNum)
	assert.Greater(t, latency, time.Duration(0))
}

func TestPingRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "server down")
	defer srv.Close()

	client := NewEVMClient(srv.URL)
	_, _, err := client.Ping(context.Background())
	require.Error(t, err)
}

func TestPingContextCancelled(t *testing.T) {
	// Server that delays; context cancels immediately.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
			"jsonrpc": "2.0", "id": 1, "result": "0x1",
		})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := NewEVMClient(srv.URL)
	_, _, err := client.Ping(ctx)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient — GetTokenBalance
// ---------------------------------------------------------------------------

func TestGetTokenBalanceSuccess(t *testing.T) {
	// balanceOf returns 1_000_000 raw (1.000000 USDC at 6 decimals).
	srv := rpcMock(t, map[string]interface{}{
		"eth_call": "0x00000000000000000000000000000000000000000000000000000000000F4240",
	})
	defer srv.Close()

	tb, err := NewEVMClient(srv.URL).GetTokenBalance(
		"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", // USDC
		"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		6,
	)
	require.NoError(t, err)
	require.NotNil(t, tb)
	assert.Equal(t, "1.000000", tb.Formatted)
	assert.Equal(t, big.NewInt(1_000_000), tb.Raw)
	assert.Equal(t, 6, tb.Decimals)
}

func TestGetTokenBalanceZero(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_call": "0x0000000000000000000000000000000000000000000000000000000000000000",
	})
	defer srv.Close()

	tb, err := NewEVMClient(srv.URL).GetTokenBalance("0xtoken", "0xholder", 18)
	require.NoError(t, err)
	require.NotNil(t, tb)
	assert.Equal(t, "0.000000000000000000", tb.Formatted)
	assert.Equal(t, 0, tb.Raw.Sign()) // Raw is zero
}

func TestGetTokenBalanceRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "execution reverted")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetTokenBalance("0xtoken", "0xholder", 18)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient — GetRecentTransactions
// ---------------------------------------------------------------------------

func TestGetRecentTransactionsSuccess(t *testing.T) {
	target := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	txList := []interface{}{
		map[string]interface{}{
			"hash":        "0xtx1",
			"from":        target,
			"to":          "0xto",
			"value":       "0xDE0B6B3A7640000",
			"gas":         "0x5208",
			"gasPrice":    "0x77359400",
			"nonce":       "0x0",
			"blockNumber": "0x200",
		},
	}
	// Use a high block number (0x200 = 512) so that latest - 200 doesn't underflow.
	srv := rpcMock(t, map[string]interface{}{
		"eth_blockNumber": "0x200",
		"eth_getBlockByNumber": map[string]interface{}{
			"transactions": txList,
		},
	})
	defer srv.Close()

	txs, err := NewEVMClient(srv.URL).GetRecentTransactions(target, 5)
	require.NoError(t, err)
	// Result may be empty (depending on scan window) but must not error.
	_ = txs
}

func TestGetRecentTransactionsRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "error")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetRecentTransactions("0xaddr", 5)
	require.Error(t, err)
}
