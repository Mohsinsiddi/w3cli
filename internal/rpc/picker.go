package rpc

import (
	"errors"
	"sync"
	"time"
)

// ErrNoHealthyRPC is returned when no healthy RPC endpoint is available.
var ErrNoHealthyRPC = errors.New("no healthy RPC endpoint available")

// Algorithm defines how an RPC endpoint is selected.
type Algorithm string

const (
	AlgorithmFastest    Algorithm = "fastest"
	AlgorithmRoundRobin Algorithm = "round-robin"
	AlgorithmFailover   Algorithm = "failover"

	// Discard nodes more than this many blocks behind the best.
	staleBlockThreshold = 3
	// Cache winner for this duration before re-benchmarking.
	cacheTTL = 5 * time.Minute
)

// Endpoint represents a single RPC endpoint with its measured attributes.
type Endpoint struct {
	URL         string
	Latency     time.Duration
	BlockNumber uint64
	Healthy     bool // meaningful only when Checked == true
	Checked     bool // true when the endpoint has been health-checked
}

// Picker selects an RPC endpoint according to the configured algorithm.
type Picker struct {
	algo        Algorithm
	mu          sync.Mutex
	rrIndex     int
	cachedURL   string
	cacheExpiry time.Time
	onBenchmark func()
}

// NewPicker creates a new Picker with the given algorithm.
func NewPicker(algo Algorithm) *Picker {
	return &Picker{algo: algo}
}

// OnBenchmark registers a hook called each time a benchmark run occurs (useful for testing).
func (p *Picker) OnBenchmark(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onBenchmark = fn
}

// Pick selects an endpoint from the provided list according to the algorithm.
func (p *Picker) Pick(endpoints []Endpoint) (*Endpoint, error) {
	if len(endpoints) == 0 {
		return nil, ErrNoHealthyRPC
	}

	switch p.algo {
	case AlgorithmRoundRobin:
		return p.pickRoundRobin(endpoints)
	case AlgorithmFailover:
		return p.pickFailover(endpoints)
	default:
		return p.pickFastest(endpoints)
	}
}

// pickFastest selects the fastest healthy endpoint, caching the result for cacheTTL.
func (p *Picker) pickFastest(endpoints []Endpoint) (*Endpoint, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Return cached winner if still fresh.
	if p.cachedURL != "" && time.Now().Before(p.cacheExpiry) {
		for i := range endpoints {
			if endpoints[i].URL == p.cachedURL {
				return &endpoints[i], nil
			}
		}
	}

	// Run benchmark.
	if p.onBenchmark != nil {
		p.onBenchmark()
	}

	// Find the best block number so we can discard stale nodes.
	var bestBlock uint64
	for _, e := range endpoints {
		if e.BlockNumber > bestBlock {
			bestBlock = e.BlockNumber
		}
	}

	// Filter by health state.
	candidates := healthyEndpoints(endpoints)
	if len(candidates) == 0 {
		return nil, ErrNoHealthyRPC
	}

	var winner *Endpoint
	var bestScore float64

	for _, e := range candidates {
		// Discard stale nodes.
		if bestBlock > 0 && bestBlock-e.BlockNumber > staleBlockThreshold {
			continue
		}

		s := score(e, bestBlock)
		if winner == nil || s > bestScore {
			winner = e
			bestScore = s
		}
	}

	if winner == nil {
		return nil, ErrNoHealthyRPC
	}

	p.cachedURL = winner.URL
	p.cacheExpiry = time.Now().Add(cacheTTL)
	return winner, nil
}

// pickRoundRobin cycles through all healthy endpoints.
func (p *Picker) pickRoundRobin(endpoints []Endpoint) (*Endpoint, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	healthy := healthyEndpoints(endpoints)
	if len(healthy) == 0 {
		return nil, ErrNoHealthyRPC
	}

	idx := p.rrIndex % len(healthy)
	p.rrIndex = (idx + 1) % len(healthy)
	return healthy[idx], nil
}

// pickFailover always tries endpoints in order, skipping explicitly unhealthy ones.
func (p *Picker) pickFailover(endpoints []Endpoint) (*Endpoint, error) {
	for i := range endpoints {
		e := &endpoints[i]
		// Skip checked endpoints that are unhealthy.
		if e.Checked && !e.Healthy {
			continue
		}
		return e, nil
	}
	return nil, ErrNoHealthyRPC
}

// --- scoring ---

func score(e *Endpoint, bestBlock uint64) float64 {
	var s float64

	// Latency score: higher = faster.
	if e.Latency > 0 {
		s += 1000.0 / float64(e.Latency.Milliseconds())
	}

	// Block recency bonus: closer to best block = +10.
	if bestBlock > 0 {
		behind := bestBlock - e.BlockNumber
		s += float64(10-behind) // loses 1 point per block behind
	}

	return s
}

// healthyEndpoints returns endpoints eligible for selection.
// When Checked == true, only Healthy endpoints are returned.
// When Checked == false, the endpoint is treated as a candidate regardless of Healthy.
func healthyEndpoints(endpoints []Endpoint) []*Endpoint {
	// Check whether any endpoint has been health-checked.
	anyChecked := false
	for _, e := range endpoints {
		if e.Checked {
			anyChecked = true
			break
		}
	}

	if !anyChecked {
		// No health data — treat all as candidates.
		all := make([]*Endpoint, len(endpoints))
		for i := range endpoints {
			all[i] = &endpoints[i]
		}
		return all
	}

	// Filter: include only checked+healthy endpoints.
	var out []*Endpoint
	for i := range endpoints {
		e := &endpoints[i]
		if !e.Checked || e.Healthy {
			out = append(out, e)
		}
	}
	return out
}
