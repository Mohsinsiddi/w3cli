package chain_test

import (
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Chain.ExplorerAPIURL
// ---------------------------------------------------------------------------

func TestExplorerAPIURLMainnet(t *testing.T) {
	reg := chain.NewRegistry()
	eth, _ := reg.GetByName("ethereum")

	url := eth.ExplorerAPIURL("mainnet")
	assert.NotEmpty(t, url, "ethereum should have a mainnet explorer API URL")
}

func TestExplorerAPIURLTestnet(t *testing.T) {
	reg := chain.NewRegistry()
	eth, _ := reg.GetByName("ethereum")

	url := eth.ExplorerAPIURL("testnet")
	assert.NotEmpty(t, url, "ethereum should have a testnet explorer API URL")
}

func TestExplorerAPIURLMainnetVsTestnet(t *testing.T) {
	reg := chain.NewRegistry()
	eth, _ := reg.GetByName("ethereum")

	mainnet := eth.ExplorerAPIURL("mainnet")
	testnet := eth.ExplorerAPIURL("testnet")

	// Mainnet and testnet explorer API URLs should be different.
	if mainnet != "" && testnet != "" {
		assert.NotEqual(t, mainnet, testnet)
	}
}

func TestExplorerAPIURLUnknownModeDefaultsToMainnet(t *testing.T) {
	reg := chain.NewRegistry()
	eth, _ := reg.GetByName("ethereum")

	// Any mode other than "testnet" returns mainnet URL.
	url := eth.ExplorerAPIURL("anything")
	assert.Equal(t, eth.ExplorerAPIURL("mainnet"), url)
}

func TestAllEVMChainsExplorerAPIURLConsistency(t *testing.T) {
	reg := chain.NewRegistry()
	for _, c := range reg.All() {
		if c.Type != chain.ChainTypeEVM {
			continue
		}
		t.Run(c.Name, func(t *testing.T) {
			// ExplorerAPIURL must not panic regardless of mode.
			_ = c.ExplorerAPIURL("mainnet")
			_ = c.ExplorerAPIURL("testnet")
		})
	}
}
