package chain

import (
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// solanaLamportsToSOL — pure function
// ---------------------------------------------------------------------------

func TestSolanaLamportsToSOLZero(t *testing.T) {
	assert.Equal(t, "0.000000000", solanaLamportsToSOL(big.NewInt(0)))
}

func TestSolanaLamportsToSOLOneSol(t *testing.T) {
	// 1 SOL = 1,000,000,000 lamports
	lamports := new(big.Int).SetUint64(1_000_000_000)
	assert.Equal(t, "1.000000000", solanaLamportsToSOL(lamports))
}

func TestSolanaLamportsToSOLSmall(t *testing.T) {
	// 1 lamport → 0.000000001 SOL
	assert.Equal(t, "0.000000001", solanaLamportsToSOL(big.NewInt(1)))
}

func TestSolanaLamportsToSOLLarge(t *testing.T) {
	// 100 SOL
	lamports := new(big.Int).Mul(big.NewInt(100), new(big.Int).SetUint64(1_000_000_000))
	assert.Equal(t, "100.000000000", solanaLamportsToSOL(lamports))
}

func TestSolanaLamportsToSOLFractional(t *testing.T) {
	// 1.5 SOL = 1,500,000,000 lamports
	lamports := new(big.Int).SetUint64(1_500_000_000)
	assert.Equal(t, "1.500000000", solanaLamportsToSOL(lamports))
}

// ---------------------------------------------------------------------------
// helpers for mock Solana RPC server
// ---------------------------------------------------------------------------

func solanaServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func solanaBalanceResp(value uint64) string {
	return fmt.Sprintf(
		`{"jsonrpc":"2.0","id":1,"result":{"context":{"slot":123},"value":%d}}`,
		value,
	)
}

func solanaSlotResp(slot uint64) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":%d}`, slot)
}

func solanaErrorResp(code int, msg string) string {
	return fmt.Sprintf(
		`{"jsonrpc":"2.0","id":1,"error":{"code":%d,"message":"%s"}}`,
		code, msg,
	)
}

// ---------------------------------------------------------------------------
// SolanaClient.GetBalance — httptest mock
// ---------------------------------------------------------------------------

func TestSolanaClientGetBalanceMock(t *testing.T) {
	const lamports = 5_000_000_000 // 5 SOL

	srv := solanaServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(solanaBalanceResp(lamports))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSolanaClient(srv.URL)
	bal, err := c.GetBalance("FakeAddr1111111111111111111111111111111")
	require.NoError(t, err)

	require.NotNil(t, bal)
	assert.Equal(t, "5.000000000", bal.ETH)
	assert.Equal(t, new(big.Int).SetUint64(lamports), bal.Wei)
}

func TestSolanaClientGetBalanceZero(t *testing.T) {
	srv := solanaServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(solanaBalanceResp(0))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSolanaClient(srv.URL)
	bal, err := c.GetBalance("FakeAddr1111111111111111111111111111111")
	require.NoError(t, err)
	assert.Equal(t, "0.000000000", bal.ETH)
}

// ---------------------------------------------------------------------------
// SolanaClient.GetSlot — httptest mock
// ---------------------------------------------------------------------------

func TestSolanaClientGetSlotMock(t *testing.T) {
	const expectedSlot = uint64(42_000_000)

	srv := solanaServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(solanaSlotResp(expectedSlot))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSolanaClient(srv.URL)
	slot, err := c.GetSlot()
	require.NoError(t, err)
	assert.Equal(t, expectedSlot, slot)
}

// ---------------------------------------------------------------------------
// SolanaClient — error paths
// ---------------------------------------------------------------------------

func TestSolanaClientRPCError(t *testing.T) {
	srv := solanaServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(solanaErrorResp(-32600, "Invalid request"))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSolanaClient(srv.URL)
	_, err := c.GetSlot()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "solana RPC error")
	assert.Contains(t, err.Error(), "Invalid request")
}

func TestSolanaClientConnectionRefused(t *testing.T) {
	c := NewSolanaClient("http://127.0.0.1:19991")
	_, err := c.GetSlot()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "solana RPC request")
}

func TestSolanaClientInvalidJSON(t *testing.T) {
	srv := solanaServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{not valid`)) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSolanaClient(srv.URL)
	_, err := c.GetSlot()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing solana response")
}
