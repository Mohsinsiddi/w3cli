package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// GetProviderKey / SetProviderKey
// ---------------------------------------------------------------------------

func TestGetProviderKeyEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	assert.Equal(t, "", cfg.GetProviderKey("alchemy"))
	assert.Equal(t, "", cfg.GetProviderKey("ankr"))
}

func TestSetAndGetProviderKey(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	cfg.SetProviderKey("alchemy", "test-alchemy-key")

	assert.Equal(t, "test-alchemy-key", cfg.GetProviderKey("alchemy"))
}

func TestSetProviderKeyMultiple(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	cfg.SetProviderKey("alchemy", "key-alchemy")
	cfg.SetProviderKey("ankr", "key-ankr")
	cfg.SetProviderKey("moralis", "key-moralis")

	assert.Equal(t, "key-alchemy", cfg.GetProviderKey("alchemy"))
	assert.Equal(t, "key-ankr", cfg.GetProviderKey("ankr"))
	assert.Equal(t, "key-moralis", cfg.GetProviderKey("moralis"))
}

func TestProviderKeyPersistence(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	cfg.SetProviderKey("etherscan", "persisted-key")
	require.NoError(t, cfg.Save())

	reloaded, err := config.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "persisted-key", reloaded.GetProviderKey("etherscan"))
}

func TestSetProviderKeyOverwrites(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	cfg.SetProviderKey("alchemy", "old-key")
	cfg.SetProviderKey("alchemy", "new-key")

	assert.Equal(t, "new-key", cfg.GetProviderKey("alchemy"))
}

func TestGetProviderKeyUnknownProvider(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)
	cfg.SetProviderKey("alchemy", "some-key")

	// An unknown provider returns "".
	assert.Equal(t, "", cfg.GetProviderKey("unknown-provider"))
}

func TestProviderKeyAllFour(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	providers := []string{"ankr", "alchemy", "moralis", "etherscan"}
	for _, p := range providers {
		cfg.SetProviderKey(p, "key-"+p)
	}
	require.NoError(t, cfg.Save())

	reloaded, err := config.Load(dir)
	require.NoError(t, err)
	for _, p := range providers {
		assert.Equal(t, "key-"+p, reloaded.GetProviderKey(p))
	}
}

// ---------------------------------------------------------------------------
// LoadWallets / SaveWallets
// ---------------------------------------------------------------------------

func TestLoadWalletsEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	wf, err := cfg.LoadWallets()
	require.NoError(t, err)
	assert.NotNil(t, wf)
	assert.Empty(t, wf.Wallets)
}

func TestSaveAndLoadWallets(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	wf := &config.WalletsFile{
		Wallets: []config.Wallet{
			{Name: "alice", Address: "0x1111", Type: "signing", IsDefault: true},
			{Name: "bob", Address: "0x2222", Type: "watch-only"},
		},
	}
	require.NoError(t, cfg.SaveWallets(wf))

	loaded, err := cfg.LoadWallets()
	require.NoError(t, err)
	assert.Len(t, loaded.Wallets, 2)
	assert.Equal(t, "alice", loaded.Wallets[0].Name)
	assert.Equal(t, "bob", loaded.Wallets[1].Name)
	assert.True(t, loaded.Wallets[0].IsDefault)
}

func TestWalletsFileCreated(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	wf := &config.WalletsFile{Wallets: []config.Wallet{}}
	require.NoError(t, cfg.SaveWallets(wf))

	_, err := os.Stat(filepath.Join(dir, "wallets.json"))
	assert.NoError(t, err, "wallets.json should be created")
}

func TestWalletRoundTripWithKeyRef(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	wf := &config.WalletsFile{
		Wallets: []config.Wallet{
			{Name: "signer", Address: "0xAAA", Type: "signing", KeyRef: "w3cli.signer", ChainType: "evm"},
		},
	}
	require.NoError(t, cfg.SaveWallets(wf))

	loaded, err := cfg.LoadWallets()
	require.NoError(t, err)
	require.Len(t, loaded.Wallets, 1)
	assert.Equal(t, "w3cli.signer", loaded.Wallets[0].KeyRef)
	assert.Equal(t, "evm", loaded.Wallets[0].ChainType)
}

// ---------------------------------------------------------------------------
// LoadContracts / SaveContracts
// ---------------------------------------------------------------------------

func TestLoadContractsEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	cf, err := cfg.LoadContracts()
	require.NoError(t, err)
	assert.NotNil(t, cf)
	assert.Empty(t, cf.Contracts)
}

func TestSaveAndLoadContracts(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	cf := &config.ContractsFile{
		Contracts: []config.ContractEntry{
			{Name: "mytoken", Address: "0xtoken", Network: "ethereum"},
		},
	}
	require.NoError(t, cfg.SaveContracts(cf))

	loaded, err := cfg.LoadContracts()
	require.NoError(t, err)
	assert.Len(t, loaded.Contracts, 1)
	assert.Equal(t, "mytoken", loaded.Contracts[0].Name)
	assert.Equal(t, "0xtoken", loaded.Contracts[0].Address)
}

func TestContractsFileCreated(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	cf := &config.ContractsFile{}
	require.NoError(t, cfg.SaveContracts(cf))

	_, err := os.Stat(filepath.Join(dir, "contracts.json"))
	assert.NoError(t, err, "contracts.json should be created")
}

func TestSaveContractsMultiple(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	entries := make([]config.ContractEntry, 5)
	for i := range entries {
		entries[i] = config.ContractEntry{Name: string(rune('a' + i)), Network: "ethereum"}
	}
	cf := &config.ContractsFile{Contracts: entries}
	require.NoError(t, cfg.SaveContracts(cf))

	loaded, _ := cfg.LoadContracts()
	assert.Len(t, loaded.Contracts, 5)
}

func TestContractEntryWithABI(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	entry := config.ContractEntry{
		Name:    "erc20",
		Network: "base",
		Address: "0x1234",
		ABI: []config.ABIEntry{
			{Name: "transfer", Type: "function", StateMutability: "nonpayable"},
			{Name: "balanceOf", Type: "function", StateMutability: "view"},
		},
	}
	cf := &config.ContractsFile{Contracts: []config.ContractEntry{entry}}
	require.NoError(t, cfg.SaveContracts(cf))

	loaded, err := cfg.LoadContracts()
	require.NoError(t, err)
	assert.Len(t, loaded.Contracts[0].ABI, 2)
	assert.Equal(t, "transfer", loaded.Contracts[0].ABI[0].Name)
}
