package chain

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// EVMClient — GetAllowance
// ---------------------------------------------------------------------------

func TestGetAllowanceSuccess(t *testing.T) {
	// 5000 = 0x1388
	srv := rpcMock(t, map[string]interface{}{
		"eth_call": "0x0000000000000000000000000000000000000000000000000000000000001388",
	})
	defer srv.Close()

	allowance, err := NewEVMClient(srv.URL).GetAllowance(
		"0xTokenAddress",
		"0xOwnerAddress",
		"0xSpenderAddress",
	)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), allowance.Int64())
}

func TestGetAllowanceZero(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_call": "0x0000000000000000000000000000000000000000000000000000000000000000",
	})
	defer srv.Close()

	allowance, err := NewEVMClient(srv.URL).GetAllowance(
		"0xToken", "0xOwner", "0xSpender",
	)
	require.NoError(t, err)
	assert.Equal(t, int64(0), allowance.Int64())
}

func TestGetAllowanceLargeValue(t *testing.T) {
	// max uint256 (unlimited approval)
	srv := rpcMock(t, map[string]interface{}{
		"eth_call": "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	})
	defer srv.Close()

	allowance, err := NewEVMClient(srv.URL).GetAllowance(
		"0xToken", "0xOwner", "0xSpender",
	)
	require.NoError(t, err)
	maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	assert.Equal(t, maxUint256, allowance)
}

func TestGetAllowanceRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "execution reverted")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetAllowance(
		"0xToken", "0xOwner", "0xSpender",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RPC error")
}

// ---------------------------------------------------------------------------
// EVMClient — GetPendingNonce
// ---------------------------------------------------------------------------

func TestGetPendingNonceSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionCount": "0xf", // 15
	})
	defer srv.Close()

	nonce, err := NewEVMClient(srv.URL).GetPendingNonce("0x1234567890abcdef1234567890abcdef12345678")
	require.NoError(t, err)
	assert.Equal(t, uint64(15), nonce)
}

func TestGetPendingNonceZero(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getTransactionCount": "0x0",
	})
	defer srv.Close()

	nonce, err := NewEVMClient(srv.URL).GetPendingNonce("0xdeadbeef")
	require.NoError(t, err)
	assert.Equal(t, uint64(0), nonce)
}

func TestGetPendingNonceRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32602, "invalid address")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetPendingNonce("0xbad")
	require.Error(t, err)
}

func TestGetPendingNonceConnectionRefused(t *testing.T) {
	_, err := NewEVMClient("http://127.0.0.1:19999").GetPendingNonce("0x1234")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient — SimulateCall
// ---------------------------------------------------------------------------

func TestSimulateCallSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_call": "0x0000000000000000000000000000000000000000000000000000000000000001",
	})
	defer srv.Close()

	ok, result, err := NewEVMClient(srv.URL).SimulateCall(
		"0xFromAddress",
		"0xToAddress",
		"0xa9059cbb",
		nil,
	)
	require.NoError(t, err)
	assert.True(t, ok, "simulation should succeed")
	assert.NotEmpty(t, result)
}

func TestSimulateCallSuccessWithValue(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_call": "0x",
	})
	defer srv.Close()

	ok, _, err := NewEVMClient(srv.URL).SimulateCall(
		"0xFrom", "0xTo", "",
		big.NewInt(1e18),
	)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestSimulateCallRevert(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "execution reverted: insufficient balance")
	defer srv.Close()

	ok, reason, err := NewEVMClient(srv.URL).SimulateCall(
		"0xFrom", "0xTo", "0xdata", nil,
	)
	require.NoError(t, err, "revert should not return an error")
	assert.False(t, ok, "simulation should report failure")
	assert.Contains(t, reason, "revert")
}

func TestSimulateCallNetworkError(t *testing.T) {
	ok, _, err := NewEVMClient("http://127.0.0.1:19999").SimulateCall(
		"0xFrom", "0xTo", "0xdata", nil,
	)
	require.Error(t, err)
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// extractRevertReason
// ---------------------------------------------------------------------------

func TestExtractRevertReason_Standard(t *testing.T) {
	msg := "RPC error -32000: execution reverted: ERC20: insufficient allowance"
	reason := extractRevertReason(msg)
	assert.Contains(t, reason, "execution reverted")
	assert.Contains(t, reason, "insufficient allowance")
}

func TestExtractRevertReason_Simple(t *testing.T) {
	msg := "revert"
	reason := extractRevertReason(msg)
	assert.Equal(t, "revert", reason)
}

func TestExtractRevertReason_NoMatch(t *testing.T) {
	msg := "some other error"
	reason := extractRevertReason(msg)
	assert.Equal(t, msg, reason)
}

// ---------------------------------------------------------------------------
// EVMClient — GetStorageAt
// ---------------------------------------------------------------------------

func TestGetStorageAtSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getStorageAt": "0x000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045",
	})
	defer srv.Close()

	val, err := NewEVMClient(srv.URL).GetStorageAt("0xContract", "0x0")
	require.NoError(t, err)
	assert.Contains(t, val, "d8da6bf26964af9d7eed9e03e53415d37aa96045")
}

func TestGetStorageAtZero(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getStorageAt": "0x0000000000000000000000000000000000000000000000000000000000000000",
	})
	defer srv.Close()

	val, err := NewEVMClient(srv.URL).GetStorageAt("0xContract", "0x0")
	require.NoError(t, err)
	assert.NotEmpty(t, val)
}

func TestGetStorageAtRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "execution error")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetStorageAt("0xContract", "0x0")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient — GetCode
// ---------------------------------------------------------------------------

func TestGetCodeContract(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getCode": "0x6080604052600436106100a05760003560e01c",
	})
	defer srv.Close()

	code, err := NewEVMClient(srv.URL).GetCode("0xContract")
	require.NoError(t, err)
	assert.True(t, len(code) > 2, "contract should have bytecode")
	assert.NotEqual(t, "0x", code)
}

func TestGetCodeEOA(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getCode": "0x",
	})
	defer srv.Close()

	code, err := NewEVMClient(srv.URL).GetCode("0xEOA")
	require.NoError(t, err)
	assert.Equal(t, "0x", code)
}

func TestGetCodeRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32602, "invalid address")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetCode("0xbad")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient — GetLogs
// ---------------------------------------------------------------------------

func TestGetLogsSuccess(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getLogs": []interface{}{
			map[string]interface{}{
				"address":          "0xtoken",
				"topics":           []interface{}{"0xddf252ad"},
				"data":             "0x0000000000000000000000000000000000000000000000000de0b6b3a7640000",
				"blockNumber":      "0x100",
				"transactionHash":  "0xabc123",
				"logIndex":         "0x0",
			},
		},
	})
	defer srv.Close()

	logs, err := NewEVMClient(srv.URL).GetLogs("0xtoken", nil, "0x0", "latest")
	require.NoError(t, err)
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, "0xtoken", logs[0].Address)
	assert.Equal(t, "0xabc123", logs[0].TxHash)
}

func TestGetLogsEmpty(t *testing.T) {
	srv := rpcMock(t, map[string]interface{}{
		"eth_getLogs": []interface{}{},
	})
	defer srv.Close()

	logs, err := NewEVMClient(srv.URL).GetLogs("0xtoken", nil, "0x0", "latest")
	require.NoError(t, err)
	assert.Equal(t, 0, len(logs))
}

func TestGetLogsRPCError(t *testing.T) {
	srv := rpcErrorServer(t, -32000, "too many results")
	defer srv.Close()

	_, err := NewEVMClient(srv.URL).GetLogs("0xtoken", nil, "0x0", "latest")
	require.Error(t, err)
}
