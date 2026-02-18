package config_test

import (
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// SyncConfig — load/save
// ---------------------------------------------------------------------------

func TestLoadSyncDefault(t *testing.T) {
	// No sync.json exists → LoadSync returns a zero-value SyncConfig, no error.
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	sc, err := cfg.LoadSync()
	require.NoError(t, err)
	assert.Empty(t, sc.Source)
	assert.Empty(t, sc.LastSynced)
}

func TestSaveSyncAndReload(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	sc := &config.SyncConfig{
		Source:     "https://example.com/deployments.json",
		LastSynced: "2024-01-01T00:00:00Z",
	}
	require.NoError(t, cfg.SaveSync(sc))

	reloaded, err := cfg.LoadSync()
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/deployments.json", reloaded.Source)
	assert.Equal(t, "2024-01-01T00:00:00Z", reloaded.LastSynced)
}

func TestSaveSyncOverwrites(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	first := &config.SyncConfig{Source: "https://first.example.com"}
	require.NoError(t, cfg.SaveSync(first))

	second := &config.SyncConfig{Source: "https://second.example.com"}
	require.NoError(t, cfg.SaveSync(second))

	reloaded, err := cfg.LoadSync()
	require.NoError(t, err)
	assert.Equal(t, "https://second.example.com", reloaded.Source)
}

// ---------------------------------------------------------------------------
// Explorer API keys
// ---------------------------------------------------------------------------

func TestGetExplorerAPIKeyGlobal(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	cfg.SetExplorerAPIKey("", "GLOBALKEY123")
	assert.Equal(t, "GLOBALKEY123", cfg.GetExplorerAPIKey("ethereum"))
	assert.Equal(t, "GLOBALKEY123", cfg.GetExplorerAPIKey("base"))
	assert.Equal(t, "GLOBALKEY123", cfg.GetExplorerAPIKey(""))
}

func TestGetExplorerAPIKeyPerChainOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	cfg.SetExplorerAPIKey("", "GLOBALKEY")
	cfg.SetExplorerAPIKey("base", "BASE_SPECIFIC_KEY")

	// Chain-specific key takes priority.
	assert.Equal(t, "BASE_SPECIFIC_KEY", cfg.GetExplorerAPIKey("base"))
	// Other chains still get global key.
	assert.Equal(t, "GLOBALKEY", cfg.GetExplorerAPIKey("ethereum"))
}

func TestSetExplorerAPIKeyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	cfg.SetExplorerAPIKey("polygon", "POLY_KEY")

	// Save and reload to confirm persistence.
	require.NoError(t, cfg.Save())

	reloaded, err := config.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "POLY_KEY", reloaded.GetExplorerAPIKey("polygon"))
}

func TestGetExplorerAPIKeyNoKeyConfigured(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	// No key set → returns "".
	assert.Empty(t, cfg.GetExplorerAPIKey("ethereum"))
	assert.Empty(t, cfg.GetExplorerAPIKey(""))
}

func TestSetExplorerAPIKeyMultipleChains(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	cfg.SetExplorerAPIKey("ethereum", "ETH_KEY")
	cfg.SetExplorerAPIKey("base", "BASE_KEY")
	cfg.SetExplorerAPIKey("polygon", "POLY_KEY")

	assert.Equal(t, "ETH_KEY", cfg.GetExplorerAPIKey("ethereum"))
	assert.Equal(t, "BASE_KEY", cfg.GetExplorerAPIKey("base"))
	assert.Equal(t, "POLY_KEY", cfg.GetExplorerAPIKey("polygon"))
}

func TestSetExplorerAPIKeyGlobalPersistsOnSave(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	require.NoError(t, err)

	cfg.SetExplorerAPIKey("", "SAVED_GLOBAL")
	require.NoError(t, cfg.Save())

	reloaded, err := config.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "SAVED_GLOBAL", reloaded.GetExplorerAPIKey("anything"))
}
