package wallet_test

import (
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddWatchOnlyWallet(t *testing.T) {
	mgr := wallet.NewManager(wallet.WithInMemoryStore())

	err := mgr.Add("mywallet", &wallet.Wallet{
		Name:    "mywallet",
		Address: "0x1234567890abcdef1234567890abcdef12345678",
		Type:    wallet.TypeWatchOnly,
	})

	require.NoError(t, err)

	w, err := mgr.Get("mywallet")
	require.NoError(t, err)
	assert.Equal(t, "mywallet", w.Name)
	assert.Equal(t, wallet.TypeWatchOnly, w.Type)
}

func TestAddDuplicateWalletErrors(t *testing.T) {
	mgr := wallet.NewManager(wallet.WithInMemoryStore())

	w := &wallet.Wallet{Name: "dup", Address: "0x123...", Type: wallet.TypeWatchOnly}
	err := mgr.Add("dup", w)
	require.NoError(t, err)

	err = mgr.Add("dup", w)
	assert.ErrorIs(t, err, wallet.ErrWalletExists)
}

func TestAddSigningWallet(t *testing.T) {
	mgr := wallet.NewManager(wallet.WithInMemoryStore())

	err := mgr.AddWithKey("signer", "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)

	w, err := mgr.Get("signer")
	require.NoError(t, err)
	assert.Equal(t, wallet.TypeSigning, w.Type)
	assert.Equal(t, "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", w.Address) // known address for test key
}

func TestInvalidPrivateKey(t *testing.T) {
	mgr := wallet.NewManager(wallet.WithInMemoryStore())
	err := mgr.AddWithKey("bad", "not-a-valid-key")
	assert.Error(t, err)
}

func TestListWallets(t *testing.T) {
	mgr := wallet.NewManager(wallet.WithInMemoryStore())
	mgr.Add("w1", &wallet.Wallet{Name: "w1", Address: "0x111...", Type: wallet.TypeWatchOnly}) //nolint:errcheck
	mgr.Add("w2", &wallet.Wallet{Name: "w2", Address: "0x222...", Type: wallet.TypeWatchOnly}) //nolint:errcheck
	mgr.Add("w3", &wallet.Wallet{Name: "w3", Address: "0x333...", Type: wallet.TypeWatchOnly}) //nolint:errcheck

	wallets := mgr.List()
	assert.Len(t, wallets, 3)
}

func TestRemoveWallet(t *testing.T) {
	mgr := wallet.NewManager(wallet.WithInMemoryStore())
	mgr.Add("w1", &wallet.Wallet{Name: "w1", Address: "0x111...", Type: wallet.TypeWatchOnly}) //nolint:errcheck

	err := mgr.Remove("w1")
	require.NoError(t, err)

	_, err = mgr.Get("w1")
	assert.ErrorIs(t, err, wallet.ErrWalletNotFound)
}

func TestRemoveNonExistentWallet(t *testing.T) {
	mgr := wallet.NewManager(wallet.WithInMemoryStore())
	err := mgr.Remove("ghost")
	assert.ErrorIs(t, err, wallet.ErrWalletNotFound)
}

func TestGetNonExistentWallet(t *testing.T) {
	mgr := wallet.NewManager(wallet.WithInMemoryStore())
	_, err := mgr.Get("ghost")
	assert.ErrorIs(t, err, wallet.ErrWalletNotFound)
}

func TestSetDefault(t *testing.T) {
	mgr := wallet.NewManager(wallet.WithInMemoryStore())
	mgr.Add("w1", &wallet.Wallet{Name: "w1", Address: "0x111...", Type: wallet.TypeWatchOnly}) //nolint:errcheck
	mgr.Add("w2", &wallet.Wallet{Name: "w2", Address: "0x222...", Type: wallet.TypeWatchOnly}) //nolint:errcheck

	require.NoError(t, mgr.SetDefault("w2"))

	def := mgr.Default()
	require.NotNil(t, def)
	assert.Equal(t, "w2", def.Name)
}

func TestDefaultWalletWithSingleWallet(t *testing.T) {
	mgr := wallet.NewManager(wallet.WithInMemoryStore())
	mgr.Add("only", &wallet.Wallet{Name: "only", Address: "0x111...", Type: wallet.TypeWatchOnly}) //nolint:errcheck

	def := mgr.Default()
	require.NotNil(t, def)
	assert.Equal(t, "only", def.Name)
}

func TestCreatedAtIsSet(t *testing.T) {
	mgr := wallet.NewManager(wallet.WithInMemoryStore())
	mgr.Add("w", &wallet.Wallet{Name: "w", Address: "0x111...", Type: wallet.TypeWatchOnly}) //nolint:errcheck

	w, _ := mgr.Get("w")
	assert.NotEmpty(t, w.CreatedAt)
}
