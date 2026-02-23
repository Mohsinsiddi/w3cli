package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/config"
	"github.com/Mohsinsiddi/w3cli/internal/contract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func testSyncConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load(t.TempDir())
	require.NoError(t, err)
	return cfg
}

func testRegistry(t *testing.T) *contract.Registry {
	t.Helper()
	path := filepath.Join(t.TempDir(), "contracts.json")
	return contract.NewRegistry(path)
}

func testSyncer(t *testing.T) (*Syncer, *config.Config, *contract.Registry) {
	t.Helper()
	cfg := testSyncConfig(t)
	reg := testRegistry(t)
	s := New(cfg, reg)
	return s, cfg, reg
}

func manifestServer(t *testing.T, m Manifest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m) //nolint:errcheck
	}))
}

func abiServer(t *testing.T, abiJSON string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(abiJSON)) //nolint:errcheck
	}))
}

// ---------------------------------------------------------------------------
// Manifest struct JSON parsing
// ---------------------------------------------------------------------------

func TestManifestParseValid(t *testing.T) {
	data := `{
		"contracts": {
			"USDC": {
				"ethereum": {
					"address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
					"abi_url": "https://example.com/abi.json"
				}
			}
		}
	}`

	var m Manifest
	require.NoError(t, json.Unmarshal([]byte(data), &m))

	require.Contains(t, m.Contracts, "USDC")
	assert.Equal(t, "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", m.Contracts["USDC"]["ethereum"].Address)
	assert.Equal(t, "https://example.com/abi.json", m.Contracts["USDC"]["ethereum"].ABIUrl)
}

func TestManifestParseInvalid(t *testing.T) {
	var m Manifest
	err := json.Unmarshal([]byte(`{not valid json`), &m)
	require.Error(t, err)
}

func TestManifestParseEmpty(t *testing.T) {
	var m Manifest
	require.NoError(t, json.Unmarshal([]byte(`{}`), &m))
	assert.Empty(t, m.Contracts)
}

func TestManifestParseMultipleContracts(t *testing.T) {
	data := `{
		"contracts": {
			"TokenA": {
				"ethereum": {"address": "0xAAA", "abi_url": ""},
				"base":     {"address": "0xBBB", "abi_url": ""}
			},
			"TokenB": {
				"polygon": {"address": "0xCCC", "abi_url": ""}
			}
		}
	}`

	var m Manifest
	require.NoError(t, json.Unmarshal([]byte(data), &m))
	assert.Len(t, m.Contracts, 2)
	assert.Len(t, m.Contracts["TokenA"], 2)
	assert.Equal(t, "0xAAA", m.Contracts["TokenA"]["ethereum"].Address)
}

// ---------------------------------------------------------------------------
// fetchManifest
// ---------------------------------------------------------------------------

func TestFetchManifestSuccess(t *testing.T) {
	m := Manifest{
		Contracts: map[string]map[string]ManifestEntry{
			"Vault": {
				"arbitrum": {Address: "0xVault", ABIUrl: "https://example.com/vault.json"},
			},
		},
	}

	srv := manifestServer(t, m)
	defer srv.Close()

	s, _, _ := testSyncer(t)
	got, err := s.fetchManifest(context.Background(), srv.URL)
	require.NoError(t, err)
	require.Contains(t, got.Contracts, "Vault")
	assert.Equal(t, "0xVault", got.Contracts["Vault"]["arbitrum"].Address)
}

func TestFetchManifestInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{not json}`)) //nolint:errcheck
	}))
	defer srv.Close()

	s, _, _ := testSyncer(t)
	_, err := s.fetchManifest(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing manifest")
}

func TestFetchManifestConnectionError(t *testing.T) {
	s, _, _ := testSyncer(t)
	_, err := s.fetchManifest(context.Background(), "http://127.0.0.1:19993")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// New / SetSource
// ---------------------------------------------------------------------------

func TestSyncerNew(t *testing.T) {
	s, cfg, reg := testSyncer(t)
	assert.NotNil(t, s)
	assert.NotNil(t, s.cfg)
	assert.NotNil(t, s.reg)
	assert.NotNil(t, s.fetcher)
	assert.NotNil(t, s.client)
	_ = cfg
	_ = reg
}

func TestSetSourceSavesURL(t *testing.T) {
	s, cfg, _ := testSyncer(t)

	const testURL = "https://example.com/deployments.json"
	require.NoError(t, s.SetSource(testURL))

	syncCfg, err := cfg.LoadSync()
	require.NoError(t, err)
	assert.Equal(t, testURL, syncCfg.Source)
}

// ---------------------------------------------------------------------------
// Run
// ---------------------------------------------------------------------------

func TestRunNoSourceConfigured(t *testing.T) {
	// sync.json doesn't exist → LoadSync returns empty SyncConfig with Source="".
	s, _, _ := testSyncer(t)

	err := s.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no sync source configured")
}

func TestRunSuccessUpdatesRegistry(t *testing.T) {
	const abiJSON = `[{"name":"transfer","type":"function","inputs":[],"outputs":[],"stateMutability":"nonpayable"}]`
	abiSrv := abiServer(t, abiJSON)
	defer abiSrv.Close()

	m := Manifest{
		Contracts: map[string]map[string]ManifestEntry{
			"MyToken": {
				"ethereum": {Address: "0xTOKEN", ABIUrl: abiSrv.URL},
			},
		},
	}
	mSrv := manifestServer(t, m)
	defer mSrv.Close()

	s, _, reg := testSyncer(t)
	require.NoError(t, s.SetSource(mSrv.URL))
	require.NoError(t, s.Run(context.Background()))

	entry, err := reg.Get("MyToken", "ethereum")
	require.NoError(t, err)
	assert.Equal(t, "0xTOKEN", entry.Address)
	assert.Len(t, entry.ABI, 1)
	assert.Equal(t, "transfer", entry.ABI[0].Name)
}

func TestRunABIFetchFailureContinues(t *testing.T) {
	// ABIUrl is unreachable → warning is printed, but contract is still added.
	m := Manifest{
		Contracts: map[string]map[string]ManifestEntry{
			"BadContract": {
				"ethereum": {Address: "0xBAD", ABIUrl: "http://127.0.0.1:19992/no-abi"},
			},
		},
	}
	mSrv := manifestServer(t, m)
	defer mSrv.Close()

	s, _, reg := testSyncer(t)
	require.NoError(t, s.SetSource(mSrv.URL))
	require.NoError(t, s.Run(context.Background()))

	// Contract is added despite ABI fetch failure.
	entry, err := reg.Get("BadContract", "ethereum")
	require.NoError(t, err)
	assert.Equal(t, "0xBAD", entry.Address)
	assert.Nil(t, entry.ABI)
}

func TestRunUpdatesLastSynced(t *testing.T) {
	// Empty manifest — still updates LastSynced.
	mSrv := manifestServer(t, Manifest{Contracts: map[string]map[string]ManifestEntry{}})
	defer mSrv.Close()

	s, cfg, _ := testSyncer(t)
	require.NoError(t, s.SetSource(mSrv.URL))

	before := time.Now().Add(-time.Second)
	require.NoError(t, s.Run(context.Background()))

	syncCfg, err := cfg.LoadSync()
	require.NoError(t, err)
	require.NotEmpty(t, syncCfg.LastSynced)

	ts, err := time.Parse(time.RFC3339, syncCfg.LastSynced)
	require.NoError(t, err)
	assert.True(t, ts.After(before), "LastSynced should be after test start")
}

func TestRunMultipleNetworksForSameContract(t *testing.T) {
	const abiJSON = `[{"name":"balanceOf","type":"function","inputs":[],"outputs":[],"stateMutability":"view"}]`
	abiSrv := abiServer(t, abiJSON)
	defer abiSrv.Close()

	m := Manifest{
		Contracts: map[string]map[string]ManifestEntry{
			"USDC": {
				"ethereum": {Address: "0xA0b8", ABIUrl: abiSrv.URL},
				"base":     {Address: "0x833589", ABIUrl: abiSrv.URL},
			},
		},
	}
	mSrv := manifestServer(t, m)
	defer mSrv.Close()

	s, _, reg := testSyncer(t)
	require.NoError(t, s.SetSource(mSrv.URL))
	require.NoError(t, s.Run(context.Background()))

	ethEntry, err := reg.Get("USDC", "ethereum")
	require.NoError(t, err)
	assert.Equal(t, "0xA0b8", ethEntry.Address)

	baseEntry, err := reg.Get("USDC", "base")
	require.NoError(t, err)
	assert.Equal(t, "0x833589", baseEntry.Address)
}

// ---------------------------------------------------------------------------
// Watch
// ---------------------------------------------------------------------------

func TestWatchCancellation(t *testing.T) {
	// Watch should return nil when the context is cancelled.
	callCount := 0
	mSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(Manifest{Contracts: map[string]map[string]ManifestEntry{}}) //nolint:errcheck
	}))
	defer mSrv.Close()

	s, _, _ := testSyncer(t)
	require.NoError(t, s.SetSource(mSrv.URL))

	// Context with a short timeout — longer than one Run cycle but shorter than the tick interval.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		// Use a very long tick so it never fires during the test.
		done <- s.Watch(ctx, 30*time.Second)
	}()

	select {
	case err := <-done:
		assert.NoError(t, err, "Watch should return nil on context cancellation")
	case <-time.After(5 * time.Second):
		t.Fatal("Watch did not return after context deadline")
	}

	// At least one initial Run should have been triggered.
	assert.GreaterOrEqual(t, callCount, 1)
}
