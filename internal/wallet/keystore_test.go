package wallet

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// normaliseHexKey
// ---------------------------------------------------------------------------

func TestNormaliseHexKeyStripsPrefix(t *testing.T) {
	assert.Equal(t, "abc123", normaliseHexKey("0xabc123"))
}

func TestNormaliseHexKeyStripsUpperPrefix(t *testing.T) {
	assert.Equal(t, "abc123", normaliseHexKey("0Xabc123"))
}

func TestNormaliseHexKeyNoPrefix(t *testing.T) {
	assert.Equal(t, "abc123", normaliseHexKey("abc123"))
}

func TestNormaliseHexKeyTrimsWhitespace(t *testing.T) {
	assert.Equal(t, "abc", normaliseHexKey("  0xabc  "))
}

func TestNormaliseHexKeyOnlyPrefix(t *testing.T) {
	assert.Equal(t, "", normaliseHexKey("0x"))
}

func TestNormaliseHexKeyEmpty(t *testing.T) {
	assert.Equal(t, "", normaliseHexKey(""))
}

func TestNormaliseHexKeyFullPrivateKey(t *testing.T) {
	const raw = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	got := normaliseHexKey("0x" + raw)
	assert.Equal(t, raw, got)
}

// ---------------------------------------------------------------------------
// Keystore.Retrieve — env var override
// ---------------------------------------------------------------------------

func TestKeystoreRetrieveEnvVarOverride(t *testing.T) {
	resetSession(t) // ensure no session file interference

	const testKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	t.Setenv("W3CLI_KEY", testKey)

	ks := &Keystore{ring: nil} // nil ring — must be served by env var
	got, err := ks.Retrieve("w3cli.any-ref")
	require.NoError(t, err)
	// normaliseHexKey strips the 0x prefix.
	assert.Equal(t, "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", got)
}

// ---------------------------------------------------------------------------
// Keystore.Retrieve — session file path
// ---------------------------------------------------------------------------

func TestKeystoreRetrieveFromSessionFile(t *testing.T) {
	resetSession(t)
	// Pre-populate the session file.
	PutSessionKey("w3cli.sessionwallet", "0xsessionkey")

	// Wipe in-process cache so the file path is exercised.
	sessionCache.Delete("w3cli.sessionwallet")

	ks := &Keystore{ring: nil}
	os.Unsetenv("W3CLI_KEY") //nolint:errcheck

	got, err := ks.Retrieve("w3cli.sessionwallet")
	require.NoError(t, err)
	assert.Equal(t, "0xsessionkey", got)
}

func TestKeystoreRetrieveNilRingNoSession(t *testing.T) {
	resetSession(t)
	sessionCache.Delete("w3cli.ghost")
	os.Unsetenv("W3CLI_KEY") //nolint:errcheck

	ks := &Keystore{ring: nil}
	_, err := ks.Retrieve("w3cli.ghost")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Keystore.Delete — clears in-process cache + session file
// ---------------------------------------------------------------------------

func TestKeystoreDeleteClearsSessionFile(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.todelete", "somekey")

	ks := &Keystore{ring: nil}
	err := ks.Delete("w3cli.todelete")
	require.NoError(t, err)

	_, ok := GetSessionKey("w3cli.todelete")
	assert.False(t, ok, "session file entry should be removed by Delete")
}

func TestKeystoreDeleteClearsInProcessCache(t *testing.T) {
	resetSession(t)
	sessionCache.Store("w3cli.cached", "cachedkey")

	ks := &Keystore{ring: nil}
	err := ks.Delete("w3cli.cached")
	require.NoError(t, err)

	_, inCache := sessionCache.Load("w3cli.cached")
	assert.False(t, inCache, "in-process cache entry should be removed by Delete")
}

func TestKeystoreDeleteNilRing(t *testing.T) {
	resetSession(t)
	// nil ring — should succeed (no OS keychain to touch).
	ks := &Keystore{ring: nil}
	err := ks.Delete("w3cli.anything")
	require.NoError(t, err)
}
