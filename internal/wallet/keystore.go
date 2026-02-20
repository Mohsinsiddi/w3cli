package wallet

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/99designs/keyring"
)

const keychainService = "w3cli"

// KeystoreBackend is the interface satisfied by both Keystore and InMemoryKeystore.
// Using this interface lets Manager accept either backend, making tests keystroke-free.
type KeystoreBackend interface {
	Store(name, hexKey string) (ref string, err error)
	Retrieve(ref string) (hexKey string, err error)
	Delete(ref string) error
}

// sessionCache holds private keys for the lifetime of the current process.
// After the first keychain access, subsequent signing calls are served from
// memory — no repeated OS prompts within a single CLI session.
var sessionCache sync.Map // ref → hexKey string

// Keystore wraps OS keychain access.
type Keystore struct {
	ring keyring.Keyring
}

// DefaultKeystore returns a keystore backed by the OS keychain.
func DefaultKeystore() *Keystore {
	cfg := keyring.Config{
		ServiceName:              keychainService,
		KeychainTrustApplication: true,
	}

	// On Linux without a GUI, fall back to file-based storage.
	if runtime.GOOS == "linux" {
		cfg.AllowedBackends = []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KWalletBackend,
			keyring.FileBackend,
		}
	}

	ring, err := keyring.Open(cfg)
	if err != nil {
		ring, _ = keyring.Open(keyring.Config{
			ServiceName:     keychainService,
			AllowedBackends: []keyring.BackendType{keyring.FileBackend},
		})
	}

	return &Keystore{ring: ring}
}

// Store saves a private key for a wallet name and returns a reference key.
func (k *Keystore) Store(name, hexKey string) (string, error) {
	if k.ring == nil {
		return name, nil
	}
	ref := keychainService + "." + name
	err := k.ring.Set(keyring.Item{
		Key:  ref,
		Data: []byte(hexKey),
	})
	if err != nil {
		return "", fmt.Errorf("keychain store: %w", err)
	}
	return ref, nil
}

// Retrieve fetches a private key by its reference.
//
// Priority order (highest → lowest):
//  1. W3CLI_KEY env var              — global CI/CD override
//  2. In-process memory cache        — already retrieved this process
//  3. Session file                   — unlocked via `w3cli wallet unlock`
//  4. OS keychain                    — prompts once on macOS, then auto-cached
//
// Run `w3cli wallet unlock` once to cache all keys into the session file.
// After that, every command runs without any OS keychain dialog until
// `w3cli wallet lock` is called.
func (k *Keystore) Retrieve(ref string) (string, error) {
	// 1. Global env var override (CI/CD, throwaway dev keys).
	if key := os.Getenv("W3CLI_KEY"); key != "" {
		return normaliseHexKey(key), nil
	}

	// 2. In-process memory cache — fastest path for repeated calls within
	//    a single process (e.g. contract studio function loop).
	if cached, ok := sessionCache.Load(ref); ok {
		return cached.(string), nil
	}

	// 3. Session file — written by `w3cli wallet unlock`, survives across
	//    process invocations so the OS keychain is never hit again until
	//    `w3cli wallet lock` clears the file.
	if hexKey, ok := GetSessionKey(ref); ok {
		sessionCache.Store(ref, hexKey) // promote to in-process cache
		return hexKey, nil
	}

	// 4. OS keychain (may show a macOS dialog on first access).
	if k.ring == nil {
		return "", fmt.Errorf("keystore not available")
	}
	item, err := k.ring.Get(ref)
	if err != nil {
		return "", fmt.Errorf("keychain retrieve: %w", err)
	}
	hexKey := string(item.Data)
	sessionCache.Store(ref, hexKey) // promote to in-process cache
	PutSessionKey(ref, hexKey)      // persist so next invocation skips keychain
	return hexKey, nil
}

// Delete removes a stored key from all layers: in-process cache, session file,
// and OS keychain. Call this when removing a wallet so the key is fully gone.
func (k *Keystore) Delete(ref string) error {
	sessionCache.Delete(ref)  // in-process memory
	RemoveSessionKey(ref)     // session file (survives across processes)
	if k.ring == nil {
		return nil
	}
	return k.ring.Remove(ref) // OS keychain
}

// normaliseHexKey strips whitespace and ensures the key has no 0x prefix
// (go-ethereum's crypto.HexToECDSA expects the raw hex string).
func normaliseHexKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.TrimPrefix(key, "0x")
	key = strings.TrimPrefix(key, "0X")
	return key
}

// ── In-memory keystore (tests) ────────────────────────────────────────────────

// InMemoryKeystore stores keys in memory — used by tests.
type InMemoryKeystore struct {
	data map[string]string
}

func NewInMemoryKeystore() *InMemoryKeystore {
	return &InMemoryKeystore{data: make(map[string]string)}
}

func (k *InMemoryKeystore) Store(name, hexKey string) (string, error) {
	ref := keychainService + "." + name
	k.data[ref] = hexKey
	return ref, nil
}

func (k *InMemoryKeystore) Retrieve(ref string) (string, error) {
	v, ok := k.data[ref]
	if !ok {
		return "", fmt.Errorf("key not found: %s", ref)
	}
	return v, nil
}

func (k *InMemoryKeystore) Delete(ref string) error {
	delete(k.data, ref)
	return nil
}
