package chain_test

import (
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryHasAllChains(t *testing.T) {
	registry := chain.NewRegistry()
	assert.Equal(t, 26, len(registry.All()))
}

func TestRegistryGetByName(t *testing.T) {
	registry := chain.NewRegistry()

	tests := []struct {
		name    string
		chainID int64
	}{
		{"ethereum", 1},
		{"base", 8453},
		{"polygon", 137},
		{"arbitrum", 42161},
		{"optimism", 10},
		{"bnb", 56},
		{"avalanche", 43114},
		{"solana", 0}, // non-EVM
		{"sui", 0},    // non-EVM
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := registry.GetByName(tt.name)
			require.NoError(t, err)
			assert.Equal(t, tt.name, c.Name)
			if tt.chainID != 0 {
				assert.Equal(t, tt.chainID, c.ChainID)
			}
		})
	}
}

func TestRegistryGetUnknownChain(t *testing.T) {
	registry := chain.NewRegistry()
	_, err := registry.GetByName("unknownchain")
	assert.ErrorIs(t, err, chain.ErrChainNotFound)
}

func TestAllChainsHaveRPC(t *testing.T) {
	registry := chain.NewRegistry()
	for _, c := range registry.All() {
		t.Run(c.Name, func(t *testing.T) {
			assert.NotEmpty(t, c.MainnetRPCs, "chain %s has no mainnet RPCs", c.Name)
			assert.NotEmpty(t, c.TestnetRPCs, "chain %s has no testnet RPCs", c.Name)
		})
	}
}

func TestAllChainsHaveExplorer(t *testing.T) {
	registry := chain.NewRegistry()
	for _, c := range registry.All() {
		t.Run(c.Name, func(t *testing.T) {
			assert.NotEmpty(t, c.MainnetExplorer, "chain %s missing mainnet explorer", c.Name)
			assert.NotEmpty(t, c.TestnetExplorer, "chain %s missing testnet explorer", c.Name)
		})
	}
}

func TestGetByChainID(t *testing.T) {
	registry := chain.NewRegistry()
	c, err := registry.GetByChainID(8453)
	require.NoError(t, err)
	assert.Equal(t, "base", c.Name)
}

func TestGetByChainIDUnknown(t *testing.T) {
	registry := chain.NewRegistry()
	_, err := registry.GetByChainID(99999999)
	assert.ErrorIs(t, err, chain.ErrChainNotFound)
}

func TestChainRPCsByMode(t *testing.T) {
	registry := chain.NewRegistry()
	eth, _ := registry.GetByName("ethereum")

	assert.NotEmpty(t, eth.RPCs("mainnet"))
	assert.NotEmpty(t, eth.RPCs("testnet"))
	assert.Equal(t, eth.MainnetRPCs, eth.RPCs("mainnet"))
	assert.Equal(t, eth.TestnetRPCs, eth.RPCs("testnet"))
}

func TestChainExplorerByMode(t *testing.T) {
	registry := chain.NewRegistry()
	eth, _ := registry.GetByName("ethereum")

	assert.Equal(t, "https://etherscan.io", eth.Explorer("mainnet"))
	assert.Equal(t, "https://sepolia.etherscan.io", eth.Explorer("testnet"))
}

func TestNonEVMChains(t *testing.T) {
	registry := chain.NewRegistry()

	sol, err := registry.GetByName("solana")
	require.NoError(t, err)
	assert.Equal(t, chain.ChainTypeSolana, sol.Type)
	assert.Equal(t, int64(0), sol.ChainID)

	sui, err := registry.GetByName("sui")
	require.NoError(t, err)
	assert.Equal(t, chain.ChainTypeSUI, sui.Type)
}

func TestAllChainsHaveNativeCurrency(t *testing.T) {
	registry := chain.NewRegistry()
	for _, c := range registry.All() {
		t.Run(c.Name, func(t *testing.T) {
			assert.NotEmpty(t, c.NativeCurrency, "chain %s has no native currency", c.Name)
		})
	}
}
