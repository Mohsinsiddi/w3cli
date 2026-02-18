package rpc

import (
	"context"
	"sync"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
)

// BenchmarkResult holds the result of a single endpoint benchmark.
type BenchmarkResult struct {
	URL         string
	Latency     time.Duration
	BlockNumber uint64
	Err         error
}

// BenchmarkEVM pings all EVM RPC URLs in parallel and returns results.
func BenchmarkEVM(ctx context.Context, urls []string) []BenchmarkResult {
	results := make([]BenchmarkResult, len(urls))
	var wg sync.WaitGroup

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, u string) {
			defer wg.Done()
			c := chain.NewEVMClient(u)
			latency, block, err := c.Ping(ctx)
			results[idx] = BenchmarkResult{
				URL:         u,
				Latency:     latency,
				BlockNumber: block,
				Err:         err,
			}
		}(i, url)
	}

	wg.Wait()
	return results
}

// ResultsToEndpoints converts benchmark results to picker Endpoints.
// All returned endpoints have Checked: true since they have been actively tested.
func ResultsToEndpoints(results []BenchmarkResult) []Endpoint {
	endpoints := make([]Endpoint, 0, len(results))
	for _, r := range results {
		endpoints = append(endpoints, Endpoint{
			URL:         r.URL,
			Latency:     r.Latency,
			BlockNumber: r.BlockNumber,
			Healthy:     r.Err == nil,
			Checked:     true,
		})
	}
	return endpoints
}

// BestEVM runs a benchmark and returns the best EVM endpoint URL using the given algorithm.
func BestEVM(ctx context.Context, urls []string, algo Algorithm) (string, error) {
	if len(urls) == 1 {
		return urls[0], nil
	}

	results := BenchmarkEVM(ctx, urls)
	endpoints := ResultsToEndpoints(results)

	picker := NewPicker(algo)
	winner, err := picker.Pick(endpoints)
	if err != nil {
		return "", err
	}
	return winner.URL, nil
}
