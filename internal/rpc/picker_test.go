package rpc_test

import (
	"testing"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// checked builds an endpoint that has been health-checked (Checked: true).
func checked(url string, latency time.Duration, block uint64, healthy bool) rpc.Endpoint {
	return rpc.Endpoint{URL: url, Latency: latency, BlockNumber: block, Healthy: healthy, Checked: true}
}

// unchecked builds an endpoint with latency/block data but no health-check status.
func unchecked(url string, latency time.Duration, block uint64) rpc.Endpoint {
	return rpc.Endpoint{URL: url, Latency: latency, BlockNumber: block}
}

func TestPickerSelectsFastest(t *testing.T) {
	endpoints := []rpc.Endpoint{
		unchecked("http://slow.rpc", 200*time.Millisecond, 100),
		unchecked("http://fast.rpc", 30*time.Millisecond, 100),
		unchecked("http://medium.rpc", 80*time.Millisecond, 100),
	}

	picker := rpc.NewPicker(rpc.AlgorithmFastest)
	winner, err := picker.Pick(endpoints)

	require.NoError(t, err)
	assert.Equal(t, "http://fast.rpc", winner.URL)
}

func TestPickerDiscardsStaleNodes(t *testing.T) {
	endpoints := []rpc.Endpoint{
		checked("http://fresh.rpc", 50*time.Millisecond, 1000, true),
		checked("http://stale.rpc", 10*time.Millisecond, 990, true), // 10 blocks behind
	}

	picker := rpc.NewPicker(rpc.AlgorithmFastest)
	winner, err := picker.Pick(endpoints)

	require.NoError(t, err)
	assert.Equal(t, "http://fresh.rpc", winner.URL, "stale node should be discarded even if faster")
}

func TestPickerRoundRobin(t *testing.T) {
	endpoints := []rpc.Endpoint{
		checked("http://rpc1", 0, 100, true),
		checked("http://rpc2", 0, 100, true),
		checked("http://rpc3", 0, 100, true),
	}

	picker := rpc.NewPicker(rpc.AlgorithmRoundRobin)

	urls := make([]string, 3)
	for i := range 3 {
		e, err := picker.Pick(endpoints)
		require.NoError(t, err)
		urls[i] = e.URL
	}

	assert.Contains(t, urls, "http://rpc1")
	assert.Contains(t, urls, "http://rpc2")
	assert.Contains(t, urls, "http://rpc3")
}

func TestPickerFailover(t *testing.T) {
	endpoints := []rpc.Endpoint{
		checked("http://primary", 0, 100, false),
		checked("http://secondary", 0, 100, true),
		checked("http://tertiary", 0, 100, true),
	}

	picker := rpc.NewPicker(rpc.AlgorithmFailover)
	winner, err := picker.Pick(endpoints)

	require.NoError(t, err)
	assert.Equal(t, "http://secondary", winner.URL, "should failover to secondary when primary is unhealthy")
}

func TestPickerErrorsWhenAllUnhealthy(t *testing.T) {
	endpoints := []rpc.Endpoint{
		checked("http://rpc1", 100*time.Millisecond, 0, false),
		checked("http://rpc2", 200*time.Millisecond, 0, false),
	}

	picker := rpc.NewPicker(rpc.AlgorithmFastest)
	_, err := picker.Pick(endpoints)

	assert.ErrorIs(t, err, rpc.ErrNoHealthyRPC)
}

func TestPickerCachesWinner(t *testing.T) {
	callCount := 0
	endpoints := []rpc.Endpoint{
		unchecked("http://fast.rpc", 30*time.Millisecond, 100),
	}

	picker := rpc.NewPicker(rpc.AlgorithmFastest)
	picker.OnBenchmark(func() { callCount++ })

	picker.Pick(endpoints) //nolint:errcheck — first call benchmarks
	picker.Pick(endpoints) //nolint:errcheck — should use cache
	picker.Pick(endpoints) //nolint:errcheck — should use cache

	assert.Equal(t, 1, callCount, "benchmark should only run once within cache TTL")
}

func TestPickerEmptyEndpoints(t *testing.T) {
	picker := rpc.NewPicker(rpc.AlgorithmFastest)
	_, err := picker.Pick([]rpc.Endpoint{})
	assert.ErrorIs(t, err, rpc.ErrNoHealthyRPC)
}

func TestPickerRoundRobinCycles(t *testing.T) {
	endpoints := []rpc.Endpoint{
		checked("http://rpc1", 0, 100, true),
		checked("http://rpc2", 0, 100, true),
	}

	picker := rpc.NewPicker(rpc.AlgorithmRoundRobin)
	first, _ := picker.Pick(endpoints)
	second, _ := picker.Pick(endpoints)
	third, _ := picker.Pick(endpoints) // should loop back

	assert.NotEqual(t, first.URL, second.URL)
	assert.Equal(t, first.URL, third.URL)
}

func TestPickerUncheckedAllTreatedAsCandidates(t *testing.T) {
	// Unchecked endpoints (no Checked flag) should all be included as candidates.
	endpoints := []rpc.Endpoint{
		unchecked("http://rpc1", 100*time.Millisecond, 50),
		unchecked("http://rpc2", 50*time.Millisecond, 50),
	}

	picker := rpc.NewPicker(rpc.AlgorithmFastest)
	winner, err := picker.Pick(endpoints)

	require.NoError(t, err)
	assert.Equal(t, "http://rpc2", winner.URL)
}
