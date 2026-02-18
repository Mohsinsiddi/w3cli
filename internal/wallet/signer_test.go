package wallet

import (
	"math/big"
	"testing"

	"github.com/99designs/keyring"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Well-known Hardhat/Anvil test account #0 — never fund on mainnet.
const (
	testPrivKeyHex = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	testSignerAddr = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
)

// testKeystore returns a file-backed Keystore isolated to a temp directory.
// Using the FileBackend avoids OS keychain prompts in CI.
func testKeystore(t *testing.T) *Keystore {
	t.Helper()
	ring, err := keyring.Open(keyring.Config{
		ServiceName:      "w3cli-test",
		AllowedBackends:  []keyring.BackendType{keyring.FileBackend},
		FileDir:          t.TempDir(),
		FilePasswordFunc: func(string) (string, error) { return "testpass", nil },
	})
	require.NoError(t, err)
	return &Keystore{ring: ring}
}

// nullKeystore has ring=nil — Retrieve always fails with "keystore not available".
func nullKeystore() *Keystore { return &Keystore{ring: nil} }

// ---------------------------------------------------------------------------
// Signer.Address
// ---------------------------------------------------------------------------

func TestSignerAddress(t *testing.T) {
	w := &Wallet{Name: "w", Address: testSignerAddr, Type: TypeSigning}
	s := NewSigner(w, nullKeystore())
	assert.Equal(t, testSignerAddr, s.Address())
}

// ---------------------------------------------------------------------------
// Signer.SignTx — error paths
// ---------------------------------------------------------------------------

func TestSignTxWatchOnlyError(t *testing.T) {
	w := &Wallet{Name: "watcher", Address: testSignerAddr, Type: TypeWatchOnly}
	s := NewSigner(w, nullKeystore())

	tx := types.NewTransaction(0, [20]byte{}, big.NewInt(0), 21000, big.NewInt(1e9), nil)
	_, err := s.SignTx(tx, big.NewInt(1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "watch-only")
}

func TestSignTxKeystoreNotAvailable(t *testing.T) {
	// ring=nil → "keystore not available" wrapped in "retrieving key".
	w := &Wallet{Name: "w", Address: testSignerAddr, Type: TypeSigning, KeyRef: "w3cli.w"}
	s := NewSigner(w, nullKeystore())

	tx := types.NewTransaction(0, [20]byte{}, big.NewInt(0), 21000, big.NewInt(1e9), nil)
	_, err := s.SignTx(tx, big.NewInt(1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retrieving key")
}

func TestSignTxKeyNotFound(t *testing.T) {
	// Keystore exists but KeyRef has no stored key → error.
	ks := testKeystore(t)
	w := &Wallet{Name: "missing", Address: testSignerAddr, Type: TypeSigning, KeyRef: "w3cli.doesnotexist"}
	s := NewSigner(w, ks)

	tx := types.NewTransaction(0, [20]byte{}, big.NewInt(0), 21000, big.NewInt(1e9), nil)
	_, err := s.SignTx(tx, big.NewInt(1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retrieving key")
}

// ---------------------------------------------------------------------------
// Signer.SignTx — success paths
// ---------------------------------------------------------------------------

func TestSignTxSuccess(t *testing.T) {
	ks := testKeystore(t)
	ref, err := ks.Store("testwal", testPrivKeyHex)
	require.NoError(t, err)

	w := &Wallet{Name: "testwal", Address: testSignerAddr, Type: TypeSigning, KeyRef: ref}
	s := NewSigner(w, ks)

	tx := types.NewTransaction(0, [20]byte{1}, big.NewInt(1e18), 21000, big.NewInt(1e9), nil)
	raw, err := s.SignTx(tx, big.NewInt(1)) // mainnet
	require.NoError(t, err)
	assert.NotEmpty(t, raw, "signed bytes must not be empty")
}

func TestSignTxDifferentChainIDs(t *testing.T) {
	ks := testKeystore(t)
	ref, err := ks.Store("testwal2", testPrivKeyHex)
	require.NoError(t, err)

	w := &Wallet{Name: "testwal2", Address: testSignerAddr, Type: TypeSigning, KeyRef: ref}
	s := NewSigner(w, ks)

	tx := types.NewTransaction(0, [20]byte{1}, big.NewInt(0), 21000, big.NewInt(1e9), nil)

	rawMainnet, err := s.SignTx(tx, big.NewInt(1))    // Ethereum mainnet
	require.NoError(t, err)

	rawBase, err := s.SignTx(tx, big.NewInt(8453))    // Base mainnet
	require.NoError(t, err)

	assert.NotEqual(t, rawMainnet, rawBase, "same tx signed on different chains must differ")
}

// ---------------------------------------------------------------------------
// InMemoryKeystore
// ---------------------------------------------------------------------------

func TestInMemoryKeystoreStoreAndRetrieve(t *testing.T) {
	iks := NewInMemoryKeystore()
	ref, err := iks.Store("mykey", "0xdeadbeef")
	require.NoError(t, err)
	assert.Equal(t, "w3cli.mykey", ref)

	val, err := iks.Retrieve(ref)
	require.NoError(t, err)
	assert.Equal(t, "0xdeadbeef", val)
}

func TestInMemoryKeystoreRetrieveNotFound(t *testing.T) {
	iks := NewInMemoryKeystore()
	_, err := iks.Retrieve("w3cli.ghost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInMemoryKeystoreDelete(t *testing.T) {
	iks := NewInMemoryKeystore()
	ref, _ := iks.Store("del", "secret")

	err := iks.Delete(ref)
	require.NoError(t, err)

	_, err = iks.Retrieve(ref)
	require.Error(t, err, "key should be gone after delete")
}

func TestInMemoryKeystoreDeleteNonExistent(t *testing.T) {
	iks := NewInMemoryKeystore()
	assert.NoError(t, iks.Delete("w3cli.ghost"), "deleting missing key must not error")
}

func TestInMemoryKeystoreOverwrite(t *testing.T) {
	iks := NewInMemoryKeystore()
	iks.Store("k", "first")  //nolint:errcheck
	iks.Store("k", "second") //nolint:errcheck

	val, err := iks.Retrieve("w3cli.k")
	require.NoError(t, err)
	assert.Equal(t, "second", val, "second store should overwrite first")
}

func TestInMemoryKeystoreRefFormat(t *testing.T) {
	iks := NewInMemoryKeystore()
	ref, err := iks.Store("mywallet", "hex")
	require.NoError(t, err)
	assert.Equal(t, "w3cli.mywallet", ref)
}

func TestInMemoryKeystoreMultipleKeys(t *testing.T) {
	iks := NewInMemoryKeystore()
	names := []string{"alice", "bob", "carol"}
	vals := map[string]string{"alice": "0xaaa", "bob": "0xbbb", "carol": "0xccc"}

	for _, name := range names {
		ref, err := iks.Store(name, vals[name])
		require.NoError(t, err)

		got, err := iks.Retrieve(ref)
		require.NoError(t, err)
		assert.Equal(t, vals[name], got)
	}
}
