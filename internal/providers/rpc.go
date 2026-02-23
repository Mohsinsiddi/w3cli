package providers

import "github.com/Mohsinsiddi/w3cli/internal/chain"

// RPC scans recent blocks via JSON-RPC as a last-resort fallback.
type RPC struct {
	rpcURL string
}

func NewRPC(rpcURL string) *RPC {
	return &RPC{rpcURL: rpcURL}
}

func (r *RPC) Name() string { return "rpc" }

func (r *RPC) GetTransactions(address string, n int) ([]*chain.Transaction, error) {
	return chain.NewEVMClient(r.rpcURL).GetRecentTransactions(address, n)
}
