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
	Name                string    `json:"name"`
	DisplayName         string    `json:"display_name"`
	ChainID             int64     `json:"chain_id"`  // 0 for non-EVM
	Type                ChainType `json:"type"`
	NativeCurrency      string    `json:"native_currency"`
	MainnetRPCs         []string  `json:"mainnet_rpcs"`
	TestnetRPCs         []string  `json:"testnet_rpcs"`
	MainnetExplorer     string    `json:"mainnet_explorer"`
	TestnetExplorer     string    `json:"testnet_explorer"`
	TestnetName         string    `json:"testnet_name"`
	// Etherscan-compatible tx API endpoints (no key required for basic use).
	MainnetExplorerAPI  string    `json:"mainnet_explorer_api,omitempty"`
	TestnetExplorerAPI  string    `json:"testnet_explorer_api,omitempty"`
	// FaucetURL is the official testnet faucet (empty = bridge from parent chain).
	FaucetURL           string    `json:"faucet_url,omitempty"`
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

// ExplorerAPIURL returns the Etherscan-compatible API endpoint for the given
// mode, or an empty string if no API is registered for this chain.
func (c *Chain) ExplorerAPIURL(mode string) string {
	if mode == "testnet" {
		return c.TestnetExplorerAPI
	}
	return c.MainnetExplorerAPI
}

// --- chain data ---

func allChains() []Chain {
	return []Chain{
		// 1. Ethereum
		{
			Name: "ethereum", DisplayName: "Ethereum", ChainID: 1, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://eth.llamarpc.com", "https://ethereum-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://rpc.sepolia.org", "https://sepolia.gateway.tenderly.co"},
			MainnetExplorer: "https://etherscan.io",
			TestnetExplorer: "https://sepolia.etherscan.io",
			TestnetName:    "Sepolia",
			MainnetExplorerAPI: "https://eth.blockscout.com/api",
			TestnetExplorerAPI: "https://eth-sepolia.blockscout.com/api",
			FaucetURL:      "https://sepoliafaucet.com",
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
			MainnetExplorerAPI: "https://base.blockscout.com/api",
			TestnetExplorerAPI: "https://base-sepolia.blockscout.com/api",
			FaucetURL:      "https://www.alchemy.com/faucets/base-sepolia",
		},
		// 3. Polygon
		{
			Name: "polygon", DisplayName: "Polygon", ChainID: 137, Type: ChainTypeEVM,
			NativeCurrency: "MATIC",
			MainnetRPCs:    []string{"https://polygon-bor-rpc.publicnode.com", "https://polygon-pokt.nodies.app"},
			TestnetRPCs:    []string{"https://rpc-amoy.polygon.technology"},
			MainnetExplorer: "https://polygonscan.com",
			TestnetExplorer: "https://amoy.polygonscan.com",
			TestnetName:    "Amoy",
			MainnetExplorerAPI: "https://polygon.blockscout.com/api",
			TestnetExplorerAPI: "https://polygon-amoy.blockscout.com/api",
			FaucetURL:      "https://faucet.polygon.technology",
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
			MainnetExplorerAPI: "https://arbitrum.blockscout.com/api",
			TestnetExplorerAPI: "https://arbitrum-sepolia.blockscout.com/api",
			FaucetURL:      "https://www.alchemy.com/faucets/arbitrum-sepolia",
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
			MainnetExplorerAPI: "https://optimism.blockscout.com/api",
			TestnetExplorerAPI: "https://optimism-sepolia.blockscout.com/api",
			FaucetURL:      "https://www.alchemy.com/faucets/optimism-sepolia",
		},
		// 6. BNB Chain
		{
			Name: "bnb", DisplayName: "BNB Chain", ChainID: 56, Type: ChainTypeEVM,
			NativeCurrency: "BNB",
			MainnetRPCs:    []string{"https://bsc-dataseed.binance.org", "https://bsc-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://data-seed-prebsc-1-s1.binance.org:8545"},
			MainnetExplorer: "https://bscscan.com",
			TestnetExplorer: "https://testnet.bscscan.com",
			TestnetName:    "BSC Testnet",
			MainnetExplorerAPI: "https://bsc.blockscout.com/api",
			TestnetExplorerAPI: "https://bsc-testnet.blockscout.com/api",
			FaucetURL:      "https://www.bnbchain.org/en/testnet-faucet",
		},
		// 7. Avalanche
		{
			Name: "avalanche", DisplayName: "Avalanche", ChainID: 43114, Type: ChainTypeEVM,
			NativeCurrency: "AVAX",
			MainnetRPCs:    []string{"https://api.avax.network/ext/bc/C/rpc", "https://avalanche-c-chain-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://api.avax-test.network/ext/bc/C/rpc"},
			MainnetExplorer: "https://snowtrace.io",
			TestnetExplorer: "https://testnet.snowtrace.io",
			TestnetName:    "Fuji",
			MainnetExplorerAPI: "https://avalanche.blockscout.com/api",
			TestnetExplorerAPI: "https://avalanche-fuji.blockscout.com/api",
			FaucetURL:      "https://faucet.avax.network",
		},
		// 8. Fantom
		{
			Name: "fantom", DisplayName: "Fantom", ChainID: 250, Type: ChainTypeEVM,
			NativeCurrency: "FTM",
			MainnetRPCs:    []string{"https://rpcapi.fantom.network", "https://fantom-pokt.nodies.app"},
			TestnetRPCs:    []string{"https://rpc.testnet.fantom.network"},
			MainnetExplorer: "https://ftmscan.com",
			TestnetExplorer: "https://testnet.ftmscan.com",
			TestnetName:    "FTM Testnet",
			MainnetExplorerAPI: "https://fantom.blockscout.com/api",
			TestnetExplorerAPI: "https://fantom-testnet.blockscout.com/api",
			FaucetURL:      "https://faucet.fantom.network",
		},
		// 9. Linea
		{
			Name: "linea", DisplayName: "Linea", ChainID: 59144, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://rpc.linea.build", "https://linea-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://rpc.sepolia.linea.build"},
			MainnetExplorer: "https://lineascan.build",
			TestnetExplorer: "https://sepolia.lineascan.build",
			TestnetName:    "Linea Sepolia",
			MainnetExplorerAPI: "https://linea.blockscout.com/api",
			TestnetExplorerAPI: "https://linea-sepolia.blockscout.com/api",
			FaucetURL:      "https://www.infura.io/faucet/linea", // official MetaMask/Consensys faucet
		},
		// 10. zkSync Era
		{
			Name: "zksync", DisplayName: "zkSync Era", ChainID: 324, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://mainnet.era.zksync.io", "https://zksync-era-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://sepolia.era.zksync.dev"},
			MainnetExplorer: "https://explorer.zksync.io",
			TestnetExplorer: "https://sepolia.explorer.zksync.io",
			TestnetName:    "zkSync Sepolia",
			MainnetExplorerAPI: "https://zksync.blockscout.com/api",
			TestnetExplorerAPI: "https://zksync-sepolia.blockscout.com/api",
			FaucetURL:      "https://faucet.quicknode.com/zksync/sepolia",
		},
		// 11. Scroll
		{
			Name: "scroll", DisplayName: "Scroll", ChainID: 534352, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://rpc.scroll.io", "https://scroll-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://sepolia-rpc.scroll.io"},
			MainnetExplorer: "https://scrollscan.com",
			TestnetExplorer: "https://sepolia.scrollscan.com",
			TestnetName:    "Scroll Sepolia",
			MainnetExplorerAPI: "https://scroll.blockscout.com/api",
			TestnetExplorerAPI: "https://scroll-sepolia.blockscout.com/api",
			FaucetURL:      "https://faucet.quicknode.com/scroll/sepolia",
		},
		// 12. Mantle
		{
			Name: "mantle", DisplayName: "Mantle", ChainID: 5000, Type: ChainTypeEVM,
			NativeCurrency: "MNT",
			MainnetRPCs:    []string{"https://rpc.mantle.xyz", "https://mantle-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://rpc.sepolia.mantle.xyz"},
			MainnetExplorer: "https://mantlescan.xyz",
			TestnetExplorer: "https://sepolia.mantlescan.xyz",
			TestnetName:    "Mantle Sepolia",
			MainnetExplorerAPI: "https://mantle.blockscout.com/api",
			TestnetExplorerAPI: "https://mantle-sepolia.blockscout.com/api",
			FaucetURL:      "https://faucet.sepolia.mantle.xyz",
		},
		// 13. Celo
		{
			Name: "celo", DisplayName: "Celo", ChainID: 42220, Type: ChainTypeEVM,
			NativeCurrency: "CELO",
			MainnetRPCs:    []string{"https://forno.celo.org", "https://celo-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://alfajores-forno.celo-testnet.org"},
			MainnetExplorer: "https://celoscan.io",
			TestnetExplorer: "https://alfajores.celoscan.io",
			TestnetName:    "Alfajores",
			MainnetExplorerAPI: "https://celo.blockscout.com/api",
			TestnetExplorerAPI: "https://celo-alfajores.blockscout.com/api",
			FaucetURL:      "https://faucet.celo.org",
		},
		// 14. Gnosis
		{
			Name: "gnosis", DisplayName: "Gnosis", ChainID: 100, Type: ChainTypeEVM,
			NativeCurrency: "xDAI",
			MainnetRPCs:    []string{"https://rpc.gnosischain.com", "https://gnosis-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://rpc.chiadochain.net"},
			MainnetExplorer: "https://gnosisscan.io",
			TestnetExplorer: "https://gnosis-chiado.blockscout.com",
			TestnetName:    "Chiado",
			MainnetExplorerAPI: "https://gnosis.blockscout.com/api",
			TestnetExplorerAPI: "https://gnosis-chiado.blockscout.com/api",
			FaucetURL:      "https://faucet.chiadochain.net", // official Chiado testnet faucet
		},
		// 15. Blast
		{
			Name: "blast", DisplayName: "Blast", ChainID: 81457, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://rpc.blast.io", "https://blast-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://sepolia.blast.io"},
			MainnetExplorer: "https://blastscan.io",
			TestnetExplorer: "https://testnet.blastscan.io",
			TestnetName:    "Blast Sepolia",
			MainnetExplorerAPI: "https://blast.blockscout.com/api",
			TestnetExplorerAPI: "https://blast-sepolia.blockscout.com/api",
			FaucetURL:      "https://faucet.quicknode.com/blast/sepolia",
		},
		// 16. Mode
		{
			Name: "mode", DisplayName: "Mode", ChainID: 34443, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://mainnet.mode.network", "https://mode-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://sepolia.mode.network"},
			MainnetExplorer: "https://explorer.mode.network",
			TestnetExplorer: "https://sepolia.explorer.mode.network",
			TestnetName:    "Mode Sepolia",
			MainnetExplorerAPI: "https://explorer.mode.network/api",
			TestnetExplorerAPI: "https://sepolia.explorer.mode.network/api",
			FaucetURL:      "https://app.optimism.io/faucet",
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
			MainnetExplorerAPI: "https://explorer.zora.energy/api",
			TestnetExplorerAPI: "https://sepolia.explorer.zora.energy/api",
			FaucetURL:      "https://www.l2faucet.com/zora",
		},
		// 18. Moonbeam
		{
			Name: "moonbeam", DisplayName: "Moonbeam", ChainID: 1284, Type: ChainTypeEVM,
			NativeCurrency: "GLMR",
			MainnetRPCs:    []string{"https://rpc.api.moonbeam.network", "https://moonbeam-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://rpc.api.moonbase.moonbeam.network"},
			MainnetExplorer: "https://moonscan.io",
			TestnetExplorer: "https://moonbase.moonscan.io",
			TestnetName:    "Moonbase Alpha",
			MainnetExplorerAPI: "https://moonbeam.blockscout.com/api",
			TestnetExplorerAPI: "https://moonbase.blockscout.com/api",
			FaucetURL:      "https://apps.moonbeam.network/moonbase-alpha/faucet",
		},
		// 19. Cronos
		{
			Name: "cronos", DisplayName: "Cronos", ChainID: 25, Type: ChainTypeEVM,
			NativeCurrency: "CRO",
			MainnetRPCs:    []string{"https://evm.cronos.org", "https://cronos-evm-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://evm-t3.cronos.org"},
			MainnetExplorer: "https://cronoscan.com",
			TestnetExplorer: "https://testnet.cronoscan.com",
			TestnetName:    "Cronos Testnet",
			MainnetExplorerAPI: "https://cronos.blockscout.com/api",
			TestnetExplorerAPI: "https://cronos-testnet.blockscout.com/api",
			FaucetURL:      "https://cronos.org/faucet",
		},
		// 20. Klaytn (now Kaia) — rebranded in 2024
		{
			Name: "klaytn", DisplayName: "Klaytn (Kaia)", ChainID: 8217, Type: ChainTypeEVM,
			NativeCurrency: "KLAY",
			MainnetRPCs:    []string{"https://public-en.node.kaia.io", "https://kaia.blockpi.network/v1/rpc/public"},
			TestnetRPCs:    []string{"https://public-en-kairos.node.kaia.io"},
			MainnetExplorer: "https://kaiascan.io",
			TestnetExplorer: "https://kairos.kaiascan.io",
			TestnetName:    "Kairos",
			FaucetURL:      "https://faucet.kaia.io", // official post-rebrand Kaia faucet
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
			MainnetExplorerAPI: "https://explorer.aurora.dev/api",
			TestnetExplorerAPI: "https://explorer.testnet.aurora.dev/api",
			FaucetURL:      "https://aurora.dev/faucet",
		},
		// 22. Polygon zkEVM
		{
			Name: "polygon-zkevm", DisplayName: "Polygon zkEVM", ChainID: 1101, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://zkevm-rpc.com", "https://polygon-zkevm-rpc.publicnode.com"},
			TestnetRPCs:    []string{"https://rpc.cardona.zkevm-rpc.com"},
			MainnetExplorer: "https://zkevm.polygonscan.com",
			TestnetExplorer: "https://cardona-zkevm.polygonscan.com",
			TestnetName:    "Cardona",
			MainnetExplorerAPI: "https://zkevm.blockscout.com/api",
			TestnetExplorerAPI: "https://zkevm-cardona.blockscout.com/api",
			FaucetURL:      "https://faucet.polygon.technology",
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
			FaucetURL:      "https://app.hyperliquid-testnet.xyz/drip",
		},
		// 24. Boba Network
		{
			Name: "boba", DisplayName: "Boba Network", ChainID: 288, Type: ChainTypeEVM,
			NativeCurrency: "ETH",
			MainnetRPCs:    []string{"https://mainnet.boba.network", "https://boba-ethereum.gateway.tenderly.co"},
			TestnetRPCs:    []string{"https://sepolia.boba.network"},
			MainnetExplorer: "https://bobascan.com",
			TestnetExplorer: "https://testnet.bobascan.com",
			TestnetName:    "Boba Sepolia",
			MainnetExplorerAPI: "https://blockexplorer.boba.network/api",
			TestnetExplorerAPI: "https://blockexplorer.sepolia.boba.network/api",
			FaucetURL:      "https://hub.boba.network", // Boba Hub (gateway.boba.network redirects here)
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
			FaucetURL:      "https://faucet.solana.com",
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
			FaucetURL:      "https://faucet.sui.io",
		},
	}
}
