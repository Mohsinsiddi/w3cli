package providers

import (
	"errors"
	"fmt"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
)

// ErrAllFailed is returned when every provider in the registry fails.
var ErrAllFailed = errors.New("all providers failed")

// Provider fetches transaction history for an address.
type Provider interface {
	Name() string
	GetTransactions(address string, n int) ([]*chain.Transaction, error)
}

// Registry tries providers in order and returns the first successful result.
type Registry struct {
	providers []Provider
}

// New creates a Registry from an ordered list of providers.
func New(ps ...Provider) *Registry {
	return &Registry{providers: ps}
}

// Result carries the fetched transactions and the provider that supplied them.
type Result struct {
	Txs      []*chain.Transaction
	Source   string
	Warnings []string // non-fatal provider errors
}

// GetTransactions tries each provider in order. It collects warnings from
// providers that fail, and returns on the first provider that returns data.
func (r *Registry) GetTransactions(address string, n int) (*Result, error) {
	res := &Result{}
	for _, p := range r.providers {
		txs, err := p.GetTransactions(address, n)
		if err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", p.Name(), err))
			continue
		}
		if len(txs) == 0 {
			// No error but no data — continue to next provider.
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no transactions found", p.Name()))
			continue
		}
		res.Txs = txs
		res.Source = p.Name()
		return res, nil
	}
	if len(res.Warnings) == 0 {
		return res, ErrAllFailed
	}
	return res, nil
}

// Names returns the names of all registered providers (for display).
func (r *Registry) Names() []string {
	names := make([]string, len(r.providers))
	for i, p := range r.providers {
		names[i] = p.Name()
	}
	return names
}
