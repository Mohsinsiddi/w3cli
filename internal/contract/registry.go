package contract

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// ErrContractNotFound is returned when a contract is not found.
var ErrContractNotFound = errors.New("contract not found")

// ABIEntry is one ABI entry (function, event, etc.).
type ABIEntry struct {
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	Inputs          []ABIParam `json:"inputs"`
	Outputs         []ABIParam `json:"outputs"`
	StateMutability string     `json:"stateMutability"`
}

// ABIParam is a parameter in an ABI entry.
type ABIParam struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// IsReadFunction returns true if the function is read-only (view/pure).
func (e ABIEntry) IsReadFunction() bool {
	return e.Type == "function" &&
		(e.StateMutability == "view" || e.StateMutability == "pure")
}

// IsWriteFunction returns true if the function modifies state.
func (e ABIEntry) IsWriteFunction() bool {
	return e.Type == "function" &&
		(e.StateMutability == "nonpayable" || e.StateMutability == "payable")
}

// Entry is a stored contract.
type Entry struct {
	Name    string     `json:"name"`
	Network string     `json:"network"`
	Address string     `json:"address"`
	ABI     []ABIEntry `json:"abi"`
	ABIUrl  string     `json:"abi_url,omitempty"`
}

// Registry stores and retrieves contract entries.
type Registry struct {
	path     string
	contracts map[string]*Entry // key: "name@network"
}

// NewRegistry creates a Registry backed by a JSON file.
func NewRegistry(path string) *Registry {
	return &Registry{
		path:      path,
		contracts: make(map[string]*Entry),
	}
}

// Load reads stored contracts from disk.
func (r *Registry) Load() error {
	data, err := os.ReadFile(r.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	for i := range entries {
		e := &entries[i]
		r.contracts[key(e.Name, e.Network)] = e
	}
	return nil
}

// Save writes all contracts to disk.
func (r *Registry) Save() error {
	entries := make([]Entry, 0, len(r.contracts))
	for _, e := range r.contracts {
		entries = append(entries, *e)
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0o600)
}

// Add adds or updates a contract entry.
func (r *Registry) Add(e *Entry) {
	r.contracts[key(e.Name, e.Network)] = e
}

// Get returns a contract by name and network.
func (r *Registry) Get(name, network string) (*Entry, error) {
	e, ok := r.contracts[key(name, network)]
	if !ok {
		return nil, fmt.Errorf("%w: %s on %s", ErrContractNotFound, name, network)
	}
	return e, nil
}

// GetByName returns all entries for a contract name across all networks.
func (r *Registry) GetByName(name string) []*Entry {
	var out []*Entry
	for _, e := range r.contracts {
		if e.Name == name {
			out = append(out, e)
		}
	}
	return out
}

// All returns all registered contracts.
func (r *Registry) All() []*Entry {
	out := make([]*Entry, 0, len(r.contracts))
	for _, e := range r.contracts {
		out = append(out, e)
	}
	return out
}

// Remove deletes a contract entry.
func (r *Registry) Remove(name, network string) error {
	k := key(name, network)
	if _, ok := r.contracts[k]; !ok {
		return fmt.Errorf("%w: %s on %s", ErrContractNotFound, name, network)
	}
	delete(r.contracts, k)
	return nil
}

func key(name, network string) string {
	return name + "@" + network
}
