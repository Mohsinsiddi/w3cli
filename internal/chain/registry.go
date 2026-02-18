package chain

import (
	"errors"
	"strings"
)

// ErrChainNotFound is returned when a chain is not in the registry.
var ErrChainNotFound = errors.New("chain not found")

// ChainType distinguishes EVM from non-EVM chains.
type ChainType string

const (
	ChainTypeEVM    ChainType = "evm"
	ChainTypeSolana ChainType = "solana"
	ChainTypeSUI    ChainType = "sui"
)

// Chain holds all metadata for a single chain.
type Chain struct {
	Name             string    `json:"name"`
	DisplayName      string    `json:"display_name"`
	ChainID          int64     `json:"chain_id"`  // 0 for non-EVM
	Type             ChainType `json:"type"`
	NativeCurrency   string    `json:"native_currency"`
	MainnetRPCs      []string  `json:"mainnet_rpcs"`
	TestnetRPCs      []string  `json:"testnet_rpcs"`
	MainnetExplorer  string    `json:"mainnet_explorer"`
	TestnetExplorer  string    `json:"testnet_explorer"`
	TestnetName      string    `json:"testnet_name"`
}

// Registry is the chain registry.
type Registry struct {
	chains []Chain
	byName map[string]*Chain
	byID   map[int64]*Chain
}

// NewRegistry creates and returns the full registry of all 26 chains.
func NewRegistry() *Registry {
	chains := allChains()
	r := &Registry{
		chains: chains,
		byName: make(map[string]*Chain, len(chains)),
		byID:   make(map[int64]*Chain, len(chains)),
	}
	for i := range r.chains {
		c := &r.chains[i]
		r.byName[c.Name] = c
		if c.ChainID != 0 {
			r.byID[c.ChainID] = c
		}
	}
	return r
}

// All returns every chain in the registry.
func (r *Registry) All() []Chain {
	return r.chains
}

// GetByName finds a chain by its slug name (e.g. "base", "ethereum").
func (r *Registry) GetByName(name string) (*Chain, error) {
	c, ok := r.byName[strings.ToLower(name)]
	if !ok {
		return nil, ErrChainNotFound
	}
	return c, nil
}

// GetByChainID finds an EVM chain by its numeric chain ID.
func (r *Registry) GetByChainID(id int64) (*Chain, error) {
	c, ok := r.byID[id]
	if !ok {
		return nil, ErrChainNotFound
	}
	return c, nil
}

// RPCs returns the RPC list for a chain in the given mode ("mainnet"/"testnet").
func (c *Chain) RPCs(mode string) []string {
	if mode == "testnet" {
		return c.TestnetRPCs
	}
	return c.MainnetRPCs
}

// Explorer returns the explorer URL for a chain in the given mode.
func (c *Chain) Explorer(mode string) string {
	if mode == "testnet" {
		return c.TestnetExplorer
	}
	return c.MainnetExplorer
}

// --- chain data ---

func allChains() []Chain {
	return []Chain{
		// 1. Ethereum
		{
			Name: "ethereum", DisplayName: "Ethereum", ChainID: 1, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://eth.llamarpc.com", "https://cloudflare-eth.com"},
			TestnetRPCs:    []string{"https://rpc.sepolia.org", "https://sepolia.gateway.tenderly.co"},
			MainnetExplorer: "https://etherscan.io",
			TestnetExplorer: "https://sepolia.etherscan.io",
			TestnetName:    "Sepolia",
		},
		// 2. Base
		{
			Name: "base", DisplayName: "Base", ChainID: 8453, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://mainnet.base.org", "https://base.llamarpc.com"},
			TestnetRPCs:    []string{"https://sepolia.base.org"},
			MainnetExplorer: "https://basescan.org",
			TestnetExplorer: "https://sepolia.basescan.org",
			TestnetName:    "Base Sepolia",
		},
		// 3. Polygon
		{
			Name: "polygon", DisplayName: "Polygon", ChainID: 137, Type: ChainTypeEVM,
			NativeCurrency: "MATIC",
			MainnetRPCs:    []string{"https://polygon-rpc.com", "https://rpc-mainnet.maticvigil.com"},
			TestnetRPCs:    []string{"https://rpc-amoy.polygon.technology"},
			MainnetExplorer: "https://polygonscan.com",
			TestnetExplorer: "https://amoy.polygonscan.com",
			TestnetName:    "Amoy",
		},
		// 4. Arbitrum
		{
			Name: "arbitrum", DisplayName: "Arbitrum", ChainID: 42161, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://arb1.arbitrum.io/rpc", "https://arbitrum.llamarpc.com"},
			TestnetRPCs:    []string{"https://sepolia-rollup.arbitrum.io/rpc"},
			MainnetExplorer: "https://arbiscan.io",
			TestnetExplorer: "https://sepolia.arbiscan.io",
			TestnetName:    "Arb Sepolia",
		},
		// 5. Optimism
		{
			Name: "optimism", DisplayName: "Optimism", ChainID: 10, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://mainnet.optimism.io", "https://optimism.llamarpc.com"},
			TestnetRPCs:    []string{"https://sepolia.optimism.io"},
			MainnetExplorer: "https://optimistic.etherscan.io",
			TestnetExplorer: "https://sepolia-optimism.etherscan.io",
			TestnetName:    "OP Sepolia",
		},
		// 6. BNB Chain
		{
			Name: "bnb", DisplayName: "BNB Chain", ChainID: 56, Type: ChainTypeEVM,
			NativeCurrency: "BNB",
			MainnetRPCs:    []string{"https://bsc-dataseed.binance.org", "https://bsc-dataseed1.defibit.io"},
			TestnetRPCs:    []string{"https://data-seed-prebsc-1-s1.binance.org:8545"},
			MainnetExplorer: "https://bscscan.com",
			TestnetExplorer: "https://testnet.bscscan.com",
			TestnetName:    "BSC Testnet",
		},
		// 7. Avalanche
		{
			Name: "avalanche", DisplayName: "Avalanche", ChainID: 43114, Type: ChainTypeEVM,
			NativeCurrency: "AVAX",
			MainnetRPCs:    []string{"https://api.avax.network/ext/bc/C/rpc"},
			TestnetRPCs:    []string{"https://api.avax-test.network/ext/bc/C/rpc"},
			MainnetExplorer: "https://snowtrace.io",
			TestnetExplorer: "https://testnet.snowtrace.io",
			TestnetName:    "Fuji",
		},
		// 8. Fantom
		{
			Name: "fantom", DisplayName: "Fantom", ChainID: 250, Type: ChainTypeEVM,
			NativeCurrency: "FTM",
			MainnetRPCs:    []string{"https://rpc.ftm.tools", "https://fantom.publicnode.com"},
			TestnetRPCs:    []string{"https://rpc.testnet.fantom.network"},
			MainnetExplorer: "https://ftmscan.com",
			TestnetExplorer: "https://testnet.ftmscan.com",
			TestnetName:    "FTM Testnet",
		},
		// 9. Linea
		{
			Name: "linea", DisplayName: "Linea", ChainID: 59144, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://rpc.linea.build"},
			TestnetRPCs:    []string{"https://rpc.sepolia.linea.build"},
			MainnetExplorer: "https://lineascan.build",
			TestnetExplorer: "https://sepolia.lineascan.build",
			TestnetName:    "Linea Sepolia",
		},
		// 10. zkSync Era
		{
			Name: "zksync", DisplayName: "zkSync Era", ChainID: 324, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://mainnet.era.zksync.io"},
			TestnetRPCs:    []string{"https://sepolia.era.zksync.dev"},
			MainnetExplorer: "https://explorer.zksync.io",
			TestnetExplorer: "https://sepolia.explorer.zksync.io",
			TestnetName:    "zkSync Sepolia",
		},
		// 11. Scroll
		{
			Name: "scroll", DisplayName: "Scroll", ChainID: 534352, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://rpc.scroll.io"},
			TestnetRPCs:    []string{"https://sepolia-rpc.scroll.io"},
			MainnetExplorer: "https://scrollscan.com",
			TestnetExplorer: "https://sepolia.scrollscan.com",
			TestnetName:    "Scroll Sepolia",
		},
		// 12. Mantle
		{
			Name: "mantle", DisplayName: "Mantle", ChainID: 5000, Type: ChainTypeEVM,
			NativeCurrency: "MNT",
			MainnetRPCs:    []string{"https://rpc.mantle.xyz"},
			TestnetRPCs:    []string{"https://rpc.sepolia.mantle.xyz"},
			MainnetExplorer: "https://mantlescan.xyz",
			TestnetExplorer: "https://sepolia.mantlescan.xyz",
			TestnetName:    "Mantle Sepolia",
		},
		// 13. Celo
		{
			Name: "celo", DisplayName: "Celo", ChainID: 42220, Type: ChainTypeEVM,
			NativeCurrency: "CELO",
			MainnetRPCs:    []string{"https://forno.celo.org"},
			TestnetRPCs:    []string{"https://alfajores-forno.celo-testnet.org"},
			MainnetExplorer: "https://celoscan.io",
			TestnetExplorer: "https://alfajores.celoscan.io",
			TestnetName:    "Alfajores",
		},
		// 14. Gnosis
		{
			Name: "gnosis", DisplayName: "Gnosis", ChainID: 100, Type: ChainTypeEVM,
			NativeCurrency: "xDAI",
			MainnetRPCs:    []string{"https://rpc.gnosischain.com"},
			TestnetRPCs:    []string{"https://rpc.chiadochain.net"},
			MainnetExplorer: "https://gnosisscan.io",
			TestnetExplorer: "https://gnosis-chiado.blockscout.com",
			TestnetName:    "Chiado",
		},
		// 15. Blast
		{
			Name: "blast", DisplayName: "Blast", ChainID: 81457, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://rpc.blast.io"},
			TestnetRPCs:    []string{"https://sepolia.blast.io"},
			MainnetExplorer: "https://blastscan.io",
			TestnetExplorer: "https://testnet.blastscan.io",
			TestnetName:    "Blast Sepolia",
		},
		// 16. Mode
		{
			Name: "mode", DisplayName: "Mode", ChainID: 34443, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://mainnet.mode.network"},
			TestnetRPCs:    []string{"https://sepolia.mode.network"},
			MainnetExplorer: "https://explorer.mode.network",
			TestnetExplorer: "https://sepolia.explorer.mode.network",
			TestnetName:    "Mode Sepolia",
		},
		// 17. Zora
		{
			Name: "zora", DisplayName: "Zora", ChainID: 7777777, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://rpc.zora.energy"},
			TestnetRPCs:    []string{"https://sepolia.rpc.zora.energy"},
			MainnetExplorer: "https://explorer.zora.energy",
			TestnetExplorer: "https://sepolia.explorer.zora.energy",
			TestnetName:    "Zora Sepolia",
		},
		// 18. Moonbeam
		{
			Name: "moonbeam", DisplayName: "Moonbeam", ChainID: 1284, Type: ChainTypeEVM,
			NativeCurrency: "GLMR",
			MainnetRPCs:    []string{"https://rpc.api.moonbeam.network"},
			TestnetRPCs:    []string{"https://rpc.api.moonbase.moonbeam.network"},
			MainnetExplorer: "https://moonscan.io",
			TestnetExplorer: "https://moonbase.moonscan.io",
			TestnetName:    "Moonbase Alpha",
		},
		// 19. Cronos
		{
			Name: "cronos", DisplayName: "Cronos", ChainID: 25, Type: ChainTypeEVM,
			NativeCurrency: "CRO",
			MainnetRPCs:    []string{"https://evm.cronos.org"},
			TestnetRPCs:    []string{"https://evm-t3.cronos.org"},
			MainnetExplorer: "https://cronoscan.com",
			TestnetExplorer: "https://testnet.cronoscan.com",
			TestnetName:    "Cronos Testnet",
		},
		// 20. Klaytn
		{
			Name: "klaytn", DisplayName: "Klaytn", ChainID: 8217, Type: ChainTypeEVM,
			NativeCurrency: "KLAY",
			MainnetRPCs:    []string{"https://public-en-cypress.klaytn.net"},
			TestnetRPCs:    []string{"https://public-en-baobab.klaytn.net"},
			MainnetExplorer: "https://scope.klaytn.com",
			TestnetExplorer: "https://baobab.scope.klaytn.com",
			TestnetName:    "Baobab",
		},
		// 21. Aurora
		{
			Name: "aurora", DisplayName: "Aurora", ChainID: 1313161554, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://mainnet.aurora.dev"},
			TestnetRPCs:    []string{"https://testnet.aurora.dev"},
			MainnetExplorer: "https://aurorascan.dev",
			TestnetExplorer: "https://testnet.aurorascan.dev",
			TestnetName:    "Aurora Testnet",
		},
		// 22. Polygon zkEVM
		{
			Name: "polygon-zkevm", DisplayName: "Polygon zkEVM", ChainID: 1101, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://zkevm-rpc.com"},
			TestnetRPCs:    []string{"https://rpc.cardona.zkevm-rpc.com"},
			MainnetExplorer: "https://zkevm.polygonscan.com",
			TestnetExplorer: "https://cardona.zkevm-rpc.com",
			TestnetName:    "Cardona",
		},
		// 23. Hyperliquid EVM
		{
			Name: "hyperliquid", DisplayName: "Hyperliquid EVM", ChainID: 999, Type: ChainTypeEVM,
			NativeCurrency: "HYPE",
			MainnetRPCs:    []string{"https://api.hyperliquid.xyz/evm"},
			TestnetRPCs:    []string{"https://api.hyperliquid-testnet.xyz/evm"},
			MainnetExplorer: "https://app.hyperliquid.xyz/explorer",
			TestnetExplorer: "https://app.hyperliquid-testnet.xyz/explorer",
			TestnetName:    "HyperEVM Testnet",
		},
		// 24. Boba Network
		{
			Name: "boba", DisplayName: "Boba Network", ChainID: 288, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://mainnet.boba.network"},
			TestnetRPCs:    []string{"https://sepolia.boba.network"},
			MainnetExplorer: "https://bobascan.com",
			TestnetExplorer: "https://testnet.bobascan.com",
			TestnetName:    "Boba Sepolia",
		},
		// 25. Solana
		{
			Name: "solana", DisplayName: "Solana", ChainID: 0, Type: ChainTypeSolana,
			NativeCurrency: "SOL",
			MainnetRPCs:    []string{"https://api.mainnet-beta.solana.com"},
			TestnetRPCs:    []string{"https://api.devnet.solana.com"},
			MainnetExplorer: "https://solscan.io",
			TestnetExplorer: "https://solscan.io/?cluster=devnet",
			TestnetName:    "Devnet",
		},
		// 26. SUI
		{
			Name: "sui", DisplayName: "SUI", ChainID: 0, Type: ChainTypeSUI,
			NativeCurrency: "SUI",
			MainnetRPCs:    []string{"https://fullnode.mainnet.sui.io"},
			TestnetRPCs:    []string{"https://fullnode.testnet.sui.io"},
			MainnetExplorer: "https://suiscan.xyz",
			TestnetExplorer: "https://suiscan.xyz/testnet",
			TestnetName:    "SUI Testnet",
		},
	}
}
