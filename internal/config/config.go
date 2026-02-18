package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const (
	defaultNetwork   = "ethereum"
	defaultMode      = "mainnet"
	defaultAlgorithm = "fastest"
	defaultCurrency  = "USD"
	defaultInterval  = 10

	configFile    = "config.json"
	walletsFile   = "wallets.json"
	contractsFile = "contracts.json"
	syncFile      = "sync.json"
)

// Load reads config from dir (or creates defaults). dir defaults to ~/.w3cli.
func Load(dir string) (*Config, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not determine home dir: %w", err)
		}
		dir = filepath.Join(home, ".w3cli")
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("could not create config dir: %w", err)
	}

	cfg := defaults(dir)

	path := filepath.Join(dir, configFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.configDir = dir
	if cfg.CustomRPCs == nil {
		cfg.CustomRPCs = make(map[string][]string)
	}

	return cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	if err := os.MkdirAll(c.configDir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.configDir, configFile), data, 0o600)
}

// AddRPC adds a custom RPC URL for a chain.
func (c *Config) AddRPC(chain, url string) error {
	if c.CustomRPCs == nil {
		c.CustomRPCs = make(map[string][]string)
	}
	if slices.Contains(c.CustomRPCs[chain], url) {
		return fmt.Errorf("RPC %s already exists for chain %s", url, chain)
	}
	c.CustomRPCs[chain] = append(c.CustomRPCs[chain], url)
	return nil
}

// RemoveRPC removes a custom RPC URL for a chain.
func (c *Config) RemoveRPC(chain, url string) error {
	rpcs := c.CustomRPCs[chain]
	idx := slices.Index(rpcs, url)
	if idx == -1 {
		return fmt.Errorf("RPC %s not found for chain %s", url, chain)
	}
	c.CustomRPCs[chain] = slices.Delete(rpcs, idx, idx+1)
	return nil
}

// GetRPCs returns custom RPCs for a chain.
func (c *Config) GetRPCs(chain string) []string {
	return c.CustomRPCs[chain]
}

// Dir returns the config directory.
func (c *Config) Dir() string {
	return c.configDir
}

// LoadWallets reads wallets.json.
func (c *Config) LoadWallets() (*WalletsFile, error) {
	return loadJSON[WalletsFile](filepath.Join(c.configDir, walletsFile))
}

// SaveWallets writes wallets.json.
func (c *Config) SaveWallets(wf *WalletsFile) error {
	return saveJSON(filepath.Join(c.configDir, walletsFile), wf)
}

// LoadContracts reads contracts.json.
func (c *Config) LoadContracts() (*ContractsFile, error) {
	return loadJSON[ContractsFile](filepath.Join(c.configDir, contractsFile))
}

// SaveContracts writes contracts.json.
func (c *Config) SaveContracts(cf *ContractsFile) error {
	return saveJSON(filepath.Join(c.configDir, contractsFile), cf)
}

// LoadSync reads sync.json.
func (c *Config) LoadSync() (*SyncConfig, error) {
	return loadJSON[SyncConfig](filepath.Join(c.configDir, syncFile))
}

// SaveSync writes sync.json.
func (c *Config) SaveSync(sc *SyncConfig) error {
	return saveJSON(filepath.Join(c.configDir, syncFile), sc)
}

// --- helpers ---

func defaults(dir string) *Config {
	return &Config{
		DefaultNetwork: defaultNetwork,
		NetworkMode:    defaultMode,
		RPCAlgorithm:   defaultAlgorithm,
		PriceCurrency:  defaultCurrency,
		WatchInterval:  defaultInterval,
		CustomRPCs:     make(map[string][]string),
		configDir:      dir,
	}
}

func loadJSON[T any](path string) (*T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &zero, nil
	}
	if err != nil {
		return nil, err
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func saveJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
