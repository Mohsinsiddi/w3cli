package wallet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// overrideSessionPath redirects the session file to a temp dir for isolation.
// It temporarily overrides UserCacheDir via an env var pattern by monkey-patching
// the sessionFilePath function output through the environment.
// Since sessionFilePath uses os.UserCacheDir(), we redirect by setting
// XDG_CACHE_HOME (Linux) or by changing HOME (macOS fallback):
// simplest approach — use t.TempDir and a wrapper that replaces the file path.
// However, since sessionFilePath is not exported, we test via the exported API
// which will use whatever path is returned. To isolate tests we clear the session
// before and after each test.

func resetSession(t *testing.T) {
	t.Helper()
	_ = ClearSession()
	t.Cleanup(func() { _ = ClearSession() })
}

// ---------------------------------------------------------------------------
// SessionActive
// ---------------------------------------------------------------------------

func TestSessionActiveEmpty(t *testing.T) {
	resetSession(t)
	assert.False(t, SessionActive())
}

func TestSessionActiveAfterPut(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.test", "0xdeadbeef")
	assert.True(t, SessionActive())
}

// ---------------------------------------------------------------------------
// PutSessionKey / GetSessionKey
// ---------------------------------------------------------------------------

func TestPutAndGetSessionKey(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.mywallet", "0xprivatekey")

	got, ok := GetSessionKey("w3cli.mywallet")
	require.True(t, ok)
	assert.Equal(t, "0xprivatekey", got)
}

func TestGetSessionKeyMissing(t *testing.T) {
	resetSession(t)
	_, ok := GetSessionKey("w3cli.nonexistent")
	assert.False(t, ok)
}

func TestPutSessionKeyOverwrites(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.wallet1", "firstkey")
	PutSessionKey("w3cli.wallet1", "secondkey")

	got, ok := GetSessionKey("w3cli.wallet1")
	require.True(t, ok)
	assert.Equal(t, "secondkey", got)
}

func TestPutMultipleKeys(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.alice", "key_alice")
	PutSessionKey("w3cli.bob", "key_bob")
	PutSessionKey("w3cli.carol", "key_carol")

	gotA, okA := GetSessionKey("w3cli.alice")
	gotB, okB := GetSessionKey("w3cli.bob")
	gotC, okC := GetSessionKey("w3cli.carol")

	require.True(t, okA)
	require.True(t, okB)
	require.True(t, okC)
	assert.Equal(t, "key_alice", gotA)
	assert.Equal(t, "key_bob", gotB)
	assert.Equal(t, "key_carol", gotC)
}

// ---------------------------------------------------------------------------
// BulkPutSessionKeys
// ---------------------------------------------------------------------------

func TestBulkPutSessionKeysEmpty(t *testing.T) {
	resetSession(t)
	BulkPutSessionKeys(map[string]string{})
	assert.False(t, SessionActive())
}

func TestBulkPutSessionKeysMerges(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.existing", "existingkey")

	BulkPutSessionKeys(map[string]string{
		"w3cli.new1": "key1",
		"w3cli.new2": "key2",
	})

	// All three should be present.
	_, okE := GetSessionKey("w3cli.existing")
	_, ok1 := GetSessionKey("w3cli.new1")
	_, ok2 := GetSessionKey("w3cli.new2")
	assert.True(t, okE)
	assert.True(t, ok1)
	assert.True(t, ok2)
}

func TestBulkPutSessionKeysOverwrites(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.wallet", "oldkey")

	BulkPutSessionKeys(map[string]string{
		"w3cli.wallet": "newkey",
	})

	got, ok := GetSessionKey("w3cli.wallet")
	require.True(t, ok)
	assert.Equal(t, "newkey", got)
}

func TestBulkPutManyKeys(t *testing.T) {
	resetSession(t)
	keys := make(map[string]string)
	for i := 0; i < 10; i++ {
		keys[string(rune('a'+i))] = string(rune('A' + i))
	}
	BulkPutSessionKeys(keys)
	snap := LoadSessionSnapshot()
	assert.Len(t, snap, 10)
}

// ---------------------------------------------------------------------------
// LoadSessionSnapshot
// ---------------------------------------------------------------------------

func TestLoadSessionSnapshotEmpty(t *testing.T) {
	resetSession(t)
	snap := LoadSessionSnapshot()
	assert.NotNil(t, snap)
	assert.Empty(t, snap)
}

func TestLoadSessionSnapshotContents(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.a", "keyA")
	PutSessionKey("w3cli.b", "keyB")

	snap := LoadSessionSnapshot()
	assert.Equal(t, "keyA", snap["w3cli.a"])
	assert.Equal(t, "keyB", snap["w3cli.b"])
}

func TestLoadSessionSnapshotIsACopy(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.x", "original")

	snap := LoadSessionSnapshot()
	snap["w3cli.x"] = "mutated"

	// Original session must be unaffected.
	got, ok := GetSessionKey("w3cli.x")
	require.True(t, ok)
	assert.Equal(t, "original", got)
}

// ---------------------------------------------------------------------------
// GetSessionKeyCached
// ---------------------------------------------------------------------------

func TestGetSessionKeyCachedTrue(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.mywallet", "somekey")
	assert.True(t, GetSessionKeyCached("mywallet"))
}

func TestGetSessionKeyCachedFalse(t *testing.T) {
	resetSession(t)
	assert.False(t, GetSessionKeyCached("ghost"))
}

// ---------------------------------------------------------------------------
// RemoveSessionKey
// ---------------------------------------------------------------------------

func TestRemoveSessionKeyExists(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.target", "somekey")
	PutSessionKey("w3cli.other", "otherkey")

	RemoveSessionKey("w3cli.target")

	_, ok := GetSessionKey("w3cli.target")
	assert.False(t, ok, "removed key should be gone")

	_, okOther := GetSessionKey("w3cli.other")
	assert.True(t, okOther, "unrelated key must survive")
}

func TestRemoveSessionKeyMissing(t *testing.T) {
	resetSession(t)
	// Should not panic or error when key does not exist.
	assert.NotPanics(t, func() { RemoveSessionKey("w3cli.ghost") })
}

func TestRemoveSessionKeyLastEntry(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.last", "lastkey")
	RemoveSessionKey("w3cli.last")
	assert.False(t, SessionActive())
}

// ---------------------------------------------------------------------------
// ClearSession
// ---------------------------------------------------------------------------

func TestClearSessionWhenEmpty(t *testing.T) {
	resetSession(t)
	// Should succeed even when no file exists.
	err := ClearSession()
	require.NoError(t, err)
}

func TestClearSessionRemovesAllKeys(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.a", "ka")
	PutSessionKey("w3cli.b", "kb")

	require.NoError(t, ClearSession())
	assert.False(t, SessionActive())
}

func TestClearSessionIdempotent(t *testing.T) {
	resetSession(t)
	require.NoError(t, ClearSession())
	require.NoError(t, ClearSession()) // second call must also succeed
}

// ---------------------------------------------------------------------------
// saveSessionKeys file permissions
// ---------------------------------------------------------------------------

func TestSessionFilePermissions(t *testing.T) {
	resetSession(t)
	PutSessionKey("w3cli.perm", "testkey")

	path := sessionFilePath()
	info, err := os.Stat(path)
	require.NoError(t, err)

	// On Unix the file must be owner-only (0600).
	if info.Mode().Perm() != 0 { // skip check on Windows where Chmod is a no-op
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

// ---------------------------------------------------------------------------
// sessionFilePath
// ---------------------------------------------------------------------------

func TestSessionFilePathContainsW3cli(t *testing.T) {
	path := sessionFilePath()
	assert.Contains(t, filepath.Base(path), "session.json")
	assert.Contains(t, path, "w3cli")
}

// ---------------------------------------------------------------------------
// Corrupt session file (loadSessionKeys robustness)
// ---------------------------------------------------------------------------

func TestLoadSessionKeysCorruptFile(t *testing.T) {
	resetSession(t)
	// Write invalid JSON to the session file.
	path := sessionFilePath()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, []byte("{corrupt:json"), 0600))

	// Should return empty map, not panic.
	m := loadSessionKeys()
	assert.Empty(t, m)
}
