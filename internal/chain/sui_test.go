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
// suiMistToSUI — pure function
// ---------------------------------------------------------------------------

func TestSuiMistToSUIZero(t *testing.T) {
	assert.Equal(t, "0.000000000", suiMistToSUI(big.NewInt(0)))
}

func TestSuiMistToSUIOneSUI(t *testing.T) {
	// 1 SUI = 1,000,000,000 MIST
	mist := new(big.Int).SetUint64(1_000_000_000)
	assert.Equal(t, "1.000000000", suiMistToSUI(mist))
}

func TestSuiMistToSUISmall(t *testing.T) {
	// 1 MIST → 0.000000001 SUI
	assert.Equal(t, "0.000000001", suiMistToSUI(big.NewInt(1)))
}

func TestSuiMistToSUILarge(t *testing.T) {
	// 250 SUI
	mist := new(big.Int).Mul(big.NewInt(250), new(big.Int).SetUint64(1_000_000_000))
	assert.Equal(t, "250.000000000", suiMistToSUI(mist))
}

func TestSuiMistToSUIFractional(t *testing.T) {
	// 0.5 SUI = 500,000,000 MIST
	mist := new(big.Int).SetUint64(500_000_000)
	assert.Equal(t, "0.500000000", suiMistToSUI(mist))
}

// ---------------------------------------------------------------------------
// helpers for mock SUI RPC server
// ---------------------------------------------------------------------------

func suiServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func suiBalanceResp(balance string) string {
	return fmt.Sprintf(
		`{"jsonrpc":"2.0","id":1,"result":{"coinType":"0x2::sui::SUI","coinObjectCount":1,"totalBalance":"%s","lockedBalance":{}}}`,
		balance,
	)
}

func suiCheckpointResp(seq string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"%s"}`, seq)
}

func suiErrorResp(code int, msg string) string {
	return fmt.Sprintf(
		`{"jsonrpc":"2.0","id":1,"error":{"code":%d,"message":"%s"}}`,
		code, msg,
	)
}

// ---------------------------------------------------------------------------
// SUIClient.GetBalance — httptest mock
// ---------------------------------------------------------------------------

func TestSUIClientGetBalanceMock(t *testing.T) {
	const mistStr = "3000000000" // 3 SUI

	srv := suiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(suiBalanceResp(mistStr))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSUIClient(srv.URL)
	bal, err := c.GetBalance("0xFakeAddr")
	require.NoError(t, err)

	require.NotNil(t, bal)
	assert.Equal(t, "3.000000000", bal.ETH)

	expectedMist, _ := new(big.Int).SetString(mistStr, 10)
	assert.Equal(t, expectedMist, bal.Wei)
}

func TestSUIClientGetBalanceZero(t *testing.T) {
	srv := suiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(suiBalanceResp("0"))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSUIClient(srv.URL)
	bal, err := c.GetBalance("0xFakeAddr")
	require.NoError(t, err)
	assert.Equal(t, "0.000000000", bal.ETH)
	assert.Equal(t, big.NewInt(0), bal.Wei)
}

// ---------------------------------------------------------------------------
// SUIClient.GetLatestCheckpoint — httptest mock
// ---------------------------------------------------------------------------

func TestSUIClientGetCheckpointMock(t *testing.T) {
	const expectedSeq = "987654"

	srv := suiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(suiCheckpointResp(expectedSeq))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSUIClient(srv.URL)
	cp, err := c.GetLatestCheckpoint()
	require.NoError(t, err)
	assert.Equal(t, uint64(987654), cp)
}

// ---------------------------------------------------------------------------
// SUIClient — error paths
// ---------------------------------------------------------------------------

func TestSUIClientRPCError(t *testing.T) {
	srv := suiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(suiErrorResp(-32700, "Parse error"))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSUIClient(srv.URL)
	_, err := c.GetLatestCheckpoint()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SUI RPC error")
	assert.Contains(t, err.Error(), "Parse error")
}

func TestSUIClientConnectionRefused(t *testing.T) {
	c := NewSUIClient("http://127.0.0.1:19990")
	_, err := c.GetLatestCheckpoint()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SUI RPC request")
}

func TestSUIClientInvalidJSON(t *testing.T) {
	srv := suiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{not valid`)) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSUIClient(srv.URL)
	_, err := c.GetLatestCheckpoint()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing SUI response")
}

func TestSUIClientBalanceInvalidMist(t *testing.T) {
	// "totalBalance" is not a valid number → falls back to zero.
	srv := suiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(suiBalanceResp("not-a-number"))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSUIClient(srv.URL)
	bal, err := c.GetBalance("0xFakeAddr")
	require.NoError(t, err) // falls back to 0, no error
	assert.Equal(t, "0.000000000", bal.ETH)
}
