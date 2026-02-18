package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	assert.Equal(t, "ethereum", cfg.DefaultNetwork)
	assert.Equal(t, "mainnet", cfg.NetworkMode)
	assert.Equal(t, "fastest", cfg.RPCAlgorithm)
	assert.Equal(t, "USD", cfg.PriceCurrency)
	assert.Equal(t, 10, cfg.WatchInterval)
}

func TestSaveAndReloadConfig(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	cfg.DefaultNetwork = "base"
	cfg.DefaultWallet = "mywallet"
	cfg.RPCAlgorithm = "round-robin"

	require.NoError(t, cfg.Save())

	reloaded, err := config.Load(dir)
	require.NoError(t, err)

	assert.Equal(t, "base", reloaded.DefaultNetwork)
	assert.Equal(t, "mywallet", reloaded.DefaultWallet)
	assert.Equal(t, "round-robin", reloaded.RPCAlgorithm)
}

func TestAddCustomRPC(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	require.NoError(t, cfg.AddRPC("base", "https://custom.base.rpc"))

	rpcs := cfg.GetRPCs("base")
	assert.Contains(t, rpcs, "https://custom.base.rpc")
}

func TestAddDuplicateRPCErrors(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	cfg.AddRPC("base", "https://custom.base.rpc") //nolint:errcheck
	err := cfg.AddRPC("base", "https://custom.base.rpc")
	assert.Error(t, err)
}

func TestRemoveCustomRPC(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	cfg.AddRPC("base", "https://rpc1.base") //nolint:errcheck
	cfg.AddRPC("base", "https://rpc2.base") //nolint:errcheck

	require.NoError(t, cfg.RemoveRPC("base", "https://rpc1.base"))

	rpcs := cfg.GetRPCs("base")
	assert.NotContains(t, rpcs, "https://rpc1.base")
	assert.Contains(t, rpcs, "https://rpc2.base")
}

func TestRemoveNonExistentRPCErrors(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	err := cfg.RemoveRPC("base", "https://nonexistent.rpc")
	assert.Error(t, err)
}

func TestConfigFileCreatedOnSave(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)
	require.NoError(t, cfg.Save())

	_, err := os.Stat(filepath.Join(dir, "config.json"))
	assert.NoError(t, err, "config.json should be created on save")
}

func TestConfigDir(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)
	assert.Equal(t, dir, cfg.Dir())
}

func TestLoadFromNonExistentDir(t *testing.T) {
	dir := t.TempDir() + "/subdir"
	cfg, err := config.Load(dir)
	require.NoError(t, err)
	// Should create dir and return defaults.
	assert.Equal(t, "ethereum", cfg.DefaultNetwork)
}

func TestMultipleCustomRPCs(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Load(dir)

	for _, url := range []string{"https://rpc1", "https://rpc2", "https://rpc3"} {
		cfg.AddRPC("ethereum", url) //nolint:errcheck
	}

	rpcs := cfg.GetRPCs("ethereum")
	assert.Len(t, rpcs, 3)
}
