package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ResultsToEndpoints — pure function
// ---------------------------------------------------------------------------

func TestResultsToEndpointsEmpty(t *testing.T) {
	out := ResultsToEndpoints(nil)
	assert.Empty(t, out)

	out2 := ResultsToEndpoints([]BenchmarkResult{})
	assert.Empty(t, out2)
}

func TestResultsToEndpointsHealthy(t *testing.T) {
	results := []BenchmarkResult{
		{URL: "https://rpc1.example.com", Latency: 50 * time.Millisecond, BlockNumber: 100, Err: nil},
	}
	endpoints := ResultsToEndpoints(results)
	require.Len(t, endpoints, 1)

	ep := endpoints[0]
	assert.Equal(t, "https://rpc1.example.com", ep.URL)
	assert.Equal(t, 50*time.Millisecond, ep.Latency)
	assert.Equal(t, uint64(100), ep.BlockNumber)
	assert.True(t, ep.Healthy)
	assert.True(t, ep.Checked)
}

func TestResultsToEndpointsUnhealthy(t *testing.T) {
	results := []BenchmarkResult{
		{URL: "https://dead.rpc.example.com", Err: errors.New("connection refused")},
	}
	endpoints := ResultsToEndpoints(results)
	require.Len(t, endpoints, 1)

	ep := endpoints[0]
	assert.False(t, ep.Healthy)
	assert.True(t, ep.Checked, "Checked must always be true after ResultsToEndpoints")
}

func TestResultsToEndpointsMixed(t *testing.T) {
	results := []BenchmarkResult{
		{URL: "https://rpc1.example.com", Err: nil},
		{URL: "https://rpc2.example.com", Err: errors.New("timeout")},
		{URL: "https://rpc3.example.com", Err: nil},
	}
	endpoints := ResultsToEndpoints(results)
	require.Len(t, endpoints, 3)

	assert.True(t, endpoints[0].Healthy)
	assert.False(t, endpoints[1].Healthy)
	assert.True(t, endpoints[2].Healthy)

	// All must have Checked=true.
	for _, ep := range endpoints {
		assert.True(t, ep.Checked)
	}
}

func TestResultsToEndpointsPreservesOrder(t *testing.T) {
	urls := []string{"https://a.com", "https://b.com", "https://c.com"}
	results := make([]BenchmarkResult, len(urls))
	for i, u := range urls {
		results[i] = BenchmarkResult{URL: u}
	}

	endpoints := ResultsToEndpoints(results)
	require.Len(t, endpoints, len(urls))
	for i, u := range urls {
		assert.Equal(t, u, endpoints[i].URL, "order must be preserved at index %d", i)
	}
}

func TestResultsToEndpointsPreservesLatency(t *testing.T) {
	results := []BenchmarkResult{
		{URL: "https://fast.rpc", Latency: 10 * time.Millisecond},
		{URL: "https://slow.rpc", Latency: 500 * time.Millisecond},
	}
	endpoints := ResultsToEndpoints(results)
	require.Len(t, endpoints, 2)

	assert.Equal(t, 10*time.Millisecond, endpoints[0].Latency)
	assert.Equal(t, 500*time.Millisecond, endpoints[1].Latency)
}

func TestResultsToEndpointsCheckedAlwaysTrue(t *testing.T) {
	// Even for zero-value results, Checked must be true.
	results := []BenchmarkResult{{}, {}, {}}
	endpoints := ResultsToEndpoints(results)
	for _, ep := range endpoints {
		assert.True(t, ep.Checked)
	}
}

// ---------------------------------------------------------------------------
// BestEVM — single URL shortcut
// ---------------------------------------------------------------------------

func TestBestEVMSingleURL(t *testing.T) {
	// When only one URL is given, BestEVM returns it immediately without
	// running any benchmark (so no real network call is needed).
	ctx := context.Background()
	url, err := BestEVM(ctx, []string{"https://only.rpc.example.com"}, AlgorithmFastest)
	require.NoError(t, err)
	assert.Equal(t, "https://only.rpc.example.com", url)
}

func TestBestEVMNoURLs(t *testing.T) {
	// Zero URLs — Pick returns ErrNoHealthyRPC.
	ctx := context.Background()
	_, err := BestEVM(ctx, []string{}, AlgorithmFastest)
	require.Error(t, err)
}
