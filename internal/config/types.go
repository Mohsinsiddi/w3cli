package config

// Config holds all w3cli configuration.
type Config struct {
	DefaultNetwork string              `json:"default_network" mapstructure:"default_network"`
	DefaultWallet  string              `json:"default_wallet"  mapstructure:"default_wallet"`
	NetworkMode    string              `json:"network_mode"    mapstructure:"network_mode"`    // "mainnet" | "testnet"
	RPCAlgorithm   string              `json:"rpc_algorithm"   mapstructure:"rpc_algorithm"`   // "fastest" | "round-robin" | "failover"
	PriceCurrency  string              `json:"price_currency"  mapstructure:"price_currency"`
	WatchInterval  int                 `json:"watch_interval"  mapstructure:"watch_interval"`  // seconds
	CustomRPCs     map[string][]string `json:"custom_rpcs"     mapstructure:"custom_rpcs"`

	// internal: config dir path used for Save()
	configDir string
}

// Wallet represents a stored wallet entry.
type Wallet struct {
	Name       string `json:"name"`
	Address    string `json:"address"`
	Type       string `json:"type"` // "watch-only" | "signing"
	KeyRef     string `json:"key_ref,omitempty"` // keychain reference for signing wallets
	ChainType  string `json:"chain_type,omitempty"` // "evm" | "solana" | "sui"
	IsDefault  bool   `json:"is_default"`
	CreatedAt  string `json:"created_at"`
}

// WalletsFile is the structure of wallets.json.
type WalletsFile struct {
	Wallets []Wallet `json:"wallets"`
}

// ContractEntry is a registered contract.
type ContractEntry struct {
	Name    string          `json:"name"`
	Network string          `json:"network"`
	Address string          `json:"address"`
	ABI     []ABIEntry      `json:"abi"`
	ABIUrl  string          `json:"abi_url,omitempty"`
}

// ABIEntry is a single ABI function/event entry.
type ABIEntry struct {
	Name            string      `json:"name"`
	Type            string      `json:"type"`
	Inputs          []ABIParam  `json:"inputs"`
	Outputs         []ABIParam  `json:"outputs"`
	StateMutability string      `json:"stateMutability"`
}

// ABIParam is a parameter in an ABI entry.
type ABIParam struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ContractsFile is the structure of contracts.json.
type ContractsFile struct {
	Contracts []ContractEntry `json:"contracts"`
}

// SyncConfig is the structure of sync.json.
type SyncConfig struct {
	Source      string `json:"source"`
	LastSynced  string `json:"last_synced"`
}
