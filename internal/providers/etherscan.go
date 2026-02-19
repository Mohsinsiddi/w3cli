package providers

import (
	"fmt"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
)

// etherscanChainID maps chain slugs to Etherscan V2 chain IDs.
var etherscanChainID = map[string]int64{
	"ethereum":  1,
	"base":      8453,
	"polygon":   137,
	"arbitrum":  42161,
	"optimism":  10,
	"zksync":    324,
	"scroll":    534352,
	"bnb":       56,
	"avalanche": 43114,
	"gnosis":    100,
	"linea":     59144,
	"mantle":    5000,
	"celo":      42220,
	"fantom":    250,
}

const etherscanBaseURL = "https://api.etherscan.io/v2/api"

// Etherscan is a provider backed by the Etherscan V2 unified API.
// It requires an API key and is nil-guarded: NewEtherscan returns nil if no key is set.
type Etherscan struct {
	chainName string
	chainID   int64
	apiKey    string
	baseURL   string // defaults to etherscanBaseURL; overridable in tests
}

// NewEtherscan creates an Etherscan provider.
// Returns nil if apiKey is empty or the chain is not supported.
func NewEtherscan(chainName, apiKey string) *Etherscan {
	if apiKey == "" {
		return nil
	}
	id, ok := etherscanChainID[chainName]
	if !ok {
		return nil
	}
	return &Etherscan{chainName: chainName, chainID: id, apiKey: apiKey, baseURL: etherscanBaseURL}
}

func (e *Etherscan) Name() string { return "etherscan" }

func (e *Etherscan) GetTransactions(address string, n int) ([]*chain.Transaction, error) {
	url := fmt.Sprintf(
		"%s?chainid=%d&module=account&action=txlist&address=%s&sort=desc&offset=%d&page=1&apikey=%s",
		e.baseURL, e.chainID, address, n, e.apiKey,
	)
	return chain.GetTransactionsFromExplorer(url, address, n, e.apiKey)
}
