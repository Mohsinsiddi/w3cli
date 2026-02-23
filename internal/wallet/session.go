package wallet

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// sessionFilePath returns the per-user session cache file.
// Uses the OS cache directory so it is not wiped on reboot on all platforms,
// but has 0600 permissions so only the current user can read it.
//
//	macOS:   ~/Library/Caches/w3cli/session.json
//	Linux:   ~/.cache/w3cli/session.json
//	Windows: %LocalAppData%\w3cli\session.json
func sessionFilePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "w3cli", "session.json")
}

// loadSessionKeys reads the session file and returns the key map.
// Returns an empty map (never nil) on any error.
func loadSessionKeys() map[string]string {
	data, err := os.ReadFile(sessionFilePath())
	if err != nil {
		return make(map[string]string)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return make(map[string]string)
	}
	return m
}

// saveSessionKeys writes the key map to the session file with restrictive
// permissions (0600 on Unix; on Windows the directory is already user-scoped
// via %LocalAppData% ACLs, and we additionally call os.Chmod as a best-effort
// hardening step that is a no-op on most Windows configurations).
func saveSessionKeys(m map[string]string) error {
	path := sessionFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	// Best-effort: restrict permissions after write.
	// On Unix this is enforced. On Windows it is a no-op but harmless.
	_ = os.Chmod(path, 0600)
	return nil
}

// LoadSessionSnapshot returns a copy of the entire session map in a single
// file read. Use this when checking multiple keys at once (e.g. wallet unlock
// --all loop) to avoid re-reading the file on each iteration.
func LoadSessionSnapshot() map[string]string {
	return loadSessionKeys() // already returns a copy (fresh map)
}

// GetSessionKey returns a cached key for ref, or ("", false) if not cached.
func GetSessionKey(ref string) (string, bool) {
	m := loadSessionKeys()
	v, ok := m[ref]
	return v, ok
}

// GetSessionKeyCached reports whether a wallet name is already in the session
// file. It accepts the plain wallet name (not the full ref) for convenience.
func GetSessionKeyCached(name string) bool {
	_, ok := GetSessionKey("w3cli." + name)
	return ok
}

// PutSessionKey caches a key for ref in the session file.
func PutSessionKey(ref, hexKey string) {
	m := loadSessionKeys()
	m[ref] = hexKey
	_ = saveSessionKeys(m) // best-effort; errors are silently ignored
}

// BulkPutSessionKeys merges multiple keys into the session file in a single
// read+write. Use this instead of calling PutSessionKey in a loop.
func BulkPutSessionKeys(keys map[string]string) {
	if len(keys) == 0 {
		return
	}
	m := loadSessionKeys()
	for ref, hexKey := range keys {
		m[ref] = hexKey
	}
	_ = saveSessionKeys(m)
}

// RemoveSessionKey removes a single wallet's key from the session file.
// Called by Keystore.Delete so that a removed wallet is also evicted from
// the session cache immediately, not just from the OS keychain.
func RemoveSessionKey(ref string) {
	m := loadSessionKeys()
	if _, ok := m[ref]; !ok {
		return
	}
	delete(m, ref)
	_ = saveSessionKeys(m)
}

// ClearSession removes all cached keys by deleting the session file.
func ClearSession() error {
	err := os.Remove(sessionFilePath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// SessionActive reports whether a non-empty session file exists.
func SessionActive() bool {
	m := loadSessionKeys()
	return len(m) > 0
}
