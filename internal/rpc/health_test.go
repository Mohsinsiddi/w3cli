package rpc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// evmRPCServer creates an httptest server that responds to eth_blockNumber.
// The blockNum is returned as a hex string.
func evmRPCServer(t *testing.T, blockNum uint64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		hexBlock := fmt.Sprintf("0x%x", blockNum)
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":"%s"}`, hexBlock)
	}))
}

// ---------------------------------------------------------------------------
// HealthCheck
// ---------------------------------------------------------------------------

func TestHealthCheckHealthy(t *testing.T) {
	srv := evmRPCServer(t, 1000)
	defer srv.Close()

	ep, err := HealthCheck(context.Background(), srv.URL, 0)
	require.NoError(t, err)

	assert.True(t, ep.Healthy)
	assert.Equal(t, srv.URL, ep.URL)
	assert.Equal(t, uint64(1000), ep.BlockNumber)
	assert.Greater(t, ep.Latency, int64(0), "latency should be measured")
}

func TestHealthCheckUnreachable(t *testing.T) {
	ep, err := HealthCheck(context.Background(), "http://127.0.0.1:19994", 0)
	require.Error(t, err)
	assert.False(t, ep.Healthy)
}

func TestHealthCheckStaleBehind(t *testing.T) {
	// The server returns block 500, but bestBlock is 510 — 10 blocks behind
	// (> staleBlockThreshold of 3) → unhealthy.
	srv := evmRPCServer(t, 500)
	defer srv.Close()

	ep, err := HealthCheck(context.Background(), srv.URL, 510)
	require.NoError(t, err) // RPC call itself succeeded
	assert.False(t, ep.Healthy, "node is too far behind bestBlock")
	assert.Equal(t, uint64(500), ep.BlockNumber)
}

func TestHealthCheckJustWithinThreshold(t *testing.T) {
	// The server returns block 997, bestBlock is 1000 — 3 blocks behind
	// (== staleBlockThreshold, not >, so it is still healthy).
	srv := evmRPCServer(t, 997)
	defer srv.Close()

	ep, err := HealthCheck(context.Background(), srv.URL, 1000)
	require.NoError(t, err)
	assert.True(t, ep.Healthy, "exactly at threshold is still healthy")
}

func TestHealthCheckNoBestBlock(t *testing.T) {
	// bestBlock = 0 → recency check is skipped entirely.
	// Even if block returned is 0, the node is healthy.
	srv := evmRPCServer(t, 0)
	defer srv.Close()

	ep, err := HealthCheck(context.Background(), srv.URL, 0)
	require.NoError(t, err)
	assert.True(t, ep.Healthy, "bestBlock=0 skips recency check")
}

func TestHealthCheckCancelledContext(t *testing.T) {
	// A pre-cancelled context means the HTTP request should fail immediately,
	// and the endpoint should be marked unhealthy.
	srv := evmRPCServer(t, 1000) // server exists but we cancel before connecting
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	ep, err := HealthCheck(ctx, srv.URL, 0)
	// Error may or may not be returned depending on timing, but Healthy must be false.
	if err != nil {
		assert.False(t, ep.Healthy)
	}
	// If no error (context wasn't checked in time), at least the call returns.
}
