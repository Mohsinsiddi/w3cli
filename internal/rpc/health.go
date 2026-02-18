package rpc

import (
	"context"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
)

// HealthCheck pings a single EVM RPC and returns whether it's healthy.
// A node is considered healthy if it responds within timeout and its block
// is within staleBlockThreshold of bestBlock (pass 0 to skip recency check).
func HealthCheck(ctx context.Context, url string, bestBlock uint64) (Endpoint, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	c := chain.NewEVMClient(url)
	latency, blockNum, err := c.Ping(timeoutCtx)

	ep := Endpoint{
		URL:         url,
		Latency:     latency,
		BlockNumber: blockNum,
		Healthy:     err == nil,
	}

	// Apply stale-block check.
	if err == nil && bestBlock > 0 && bestBlock-blockNum > staleBlockThreshold {
		ep.Healthy = false
	}

	return ep, err
}
