package rpc

import "context"

// SelectBest picks the best RPC URL from the provided list using the named
// algorithm. It is a stateless wrapper around BestEVM designed for use by
// packages that do not hold a reference to the global config (e.g. an API
// server handler).
//
// algorithm must be one of "fastest", "round-robin", or "failover". An empty
// string defaults to "fastest".
//
// Returns ErrNoHealthyRPC when the list is empty or all endpoints fail.
func SelectBest(ctx context.Context, urls []string, algorithm string) (string, error) {
	if len(urls) == 0 {
		return "", ErrNoHealthyRPC
	}
	if len(urls) == 1 {
		return urls[0], nil
	}
	algo := Algorithm(algorithm)
	if algo == "" {
		algo = AlgorithmFastest
	}
	return BestEVM(ctx, urls, algo)
}
