package wallet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewJSONStore / Load / Save
// ---------------------------------------------------------------------------

func TestJSONStoreSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallets.json")
	store := NewJSONStore(path)

	wallets := []*Wallet{
		{Name: "alice", Address: "0x1111", Type: TypeWatchOnly},
		{Name: "bob", Address: "0x2222", Type: TypeSigning, KeyRef: "w3cli.bob"},
	}
	require.NoError(t, store.Save(wallets))

	loaded, err := store.Load()
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	assert.Equal(t, "alice", loaded[0].Name)
	assert.Equal(t, "bob", loaded[1].Name)
	assert.Equal(t, TypeSigning, loaded[1].Type)
	assert.Equal(t, "w3cli.bob", loaded[1].KeyRef)
}

func TestJSONStoreLoadNoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	store := NewJSONStore(path)

	wallets, err := store.Load()
	require.NoError(t, err)
	assert.Nil(t, wallets, "loading a missing file should return nil, nil")
}

func TestJSONStoreSaveCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallets.json")
	store := NewJSONStore(path)

	require.NoError(t, store.Save([]*Wallet{}))

	_, err := os.Stat(path)
	assert.NoError(t, err, "Save should create the file")
}

func TestJSONStoreSaveRestrictivePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallets.json")
	store := NewJSONStore(path)
	require.NoError(t, store.Save([]*Wallet{{Name: "w", Address: "0x1"}}))

	info, err := os.Stat(path)
	require.NoError(t, err)
	if info.Mode().Perm() != 0 { // Unix only
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestJSONStoreRoundTripAllFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallets.json")
	store := NewJSONStore(path)

	w := &Wallet{
		Name:      "full",
		Address:   "0xABCD",
		Type:      TypeSigning,
		KeyRef:    "w3cli.full",
		ChainType: "evm",
		IsDefault: true,
		CreatedAt: "2024-01-01T00:00:00Z",
	}
	require.NoError(t, store.Save([]*Wallet{w}))

	loaded, err := store.Load()
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, w.Name, loaded[0].Name)
	assert.Equal(t, w.Address, loaded[0].Address)
	assert.Equal(t, w.Type, loaded[0].Type)
	assert.Equal(t, w.KeyRef, loaded[0].KeyRef)
	assert.Equal(t, w.ChainType, loaded[0].ChainType)
	assert.Equal(t, w.IsDefault, loaded[0].IsDefault)
	assert.Equal(t, w.CreatedAt, loaded[0].CreatedAt)
}

func TestJSONStoreSaveOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallets.json")
	store := NewJSONStore(path)

	first := []*Wallet{{Name: "first", Address: "0x111", Type: TypeWatchOnly}}
	require.NoError(t, store.Save(first))

	second := []*Wallet{
		{Name: "second-a", Address: "0x222", Type: TypeWatchOnly},
		{Name: "second-b", Address: "0x333", Type: TypeWatchOnly},
	}
	require.NoError(t, store.Save(second))

	loaded, err := store.Load()
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	assert.Equal(t, "second-a", loaded[0].Name)
}

func TestJSONStoreLoadCorrupt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.json")
	require.NoError(t, os.WriteFile(path, []byte("{not valid json"), 0600))

	store := NewJSONStore(path)
	_, err := store.Load()
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// WithStore option
// ---------------------------------------------------------------------------

func TestWithStoreOption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallets.json")
	store := NewJSONStore(path)

	mgr := NewManager(WithStore(store))
	require.NoError(t, mgr.Add("test-ws", &Wallet{
		Name: "test-ws", Address: "0xABC", Type: TypeWatchOnly,
	}))

	// Reload from same path — wallet should persist.
	mgr2 := NewManager(WithStore(NewJSONStore(path)))
	w, err := mgr2.Get("test-ws")
	require.NoError(t, err)
	assert.Equal(t, "test-ws", w.Name)
}
