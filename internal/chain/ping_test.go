package chain

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// SolanaClient.Ping
// ---------------------------------------------------------------------------

func TestSolanaPingSuccess(t *testing.T) {
	srv := solanaServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(solanaSlotResp(300_000_000))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSolanaClient(srv.URL)
	latency, slot, err := c.Ping(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(300_000_000), slot)
	assert.Greater(t, latency, time.Duration(0))
}

func TestSolanaPingRPCError(t *testing.T) {
	srv := solanaServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(solanaErrorResp(-32000, "node unavailable"))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSolanaClient(srv.URL)
	_, _, err := c.Ping(context.Background())
	require.Error(t, err)
}

func TestSolanaPingConnectionRefused(t *testing.T) {
	c := NewSolanaClient("http://127.0.0.1:19992")
	_, _, err := c.Ping(context.Background())
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// SUIClient.Ping
// ---------------------------------------------------------------------------

func TestSUIPingSuccess(t *testing.T) {
	srv := suiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(suiCheckpointResp("123456789"))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSUIClient(srv.URL)
	latency, cp, err := c.Ping(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(123456789), cp)
	assert.Greater(t, latency, time.Duration(0))
}

func TestSUIPingRPCError(t *testing.T) {
	srv := suiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(suiErrorResp(-32600, "internal error"))) //nolint:errcheck
	})
	defer srv.Close()

	c := NewSUIClient(srv.URL)
	_, _, err := c.Ping(context.Background())
	require.Error(t, err)
}

func TestSUIPingConnectionRefused(t *testing.T) {
	c := NewSUIClient("http://127.0.0.1:19993")
	_, _, err := c.Ping(context.Background())
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// EVMClient.Ping — bad JSON and non-string result
// ---------------------------------------------------------------------------

func TestEVMPingBadJSON(t *testing.T) {
	srv := rpcBadJSON(t)
	defer srv.Close()

	client := NewEVMClient(srv.URL)
	_, _, err := client.Ping(context.Background())
	require.Error(t, err)
}
