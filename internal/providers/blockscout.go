package providers

import "github.com/Mohsinsiddi/w3cli/internal/chain"

// BlockScout wraps the existing Etherscan-compatible explorer API.
type BlockScout struct {
	APIURL string
	APIKey string
}

func NewBlockScout(apiURL, apiKey string) *BlockScout {
	return &BlockScout{APIURL: apiURL, APIKey: apiKey}
}

func (b *BlockScout) Name() string { return "blockscout" }

func (b *BlockScout) GetTransactions(address string, n int) ([]*chain.Transaction, error) {
	return chain.GetTransactionsFromExplorer(b.APIURL, address, n, b.APIKey)
}
