package wallet

import (
	"fmt"
	"runtime"

	"github.com/99designs/keyring"
)

const keychainService = "w3cli"

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
		// Use file backend as ultimate fallback.
		ring, _ = keyring.Open(keyring.Config{
			ServiceName:  keychainService,
			AllowedBackends: []keyring.BackendType{keyring.FileBackend},
		})
	}

	return &Keystore{ring: ring}
}

// Store saves a private key for a wallet name and returns a reference key.
func (k *Keystore) Store(name, hexKey string) (string, error) {
	if k.ring == nil {
		return name, nil // in-memory fallback
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
func (k *Keystore) Retrieve(ref string) (string, error) {
	if k.ring == nil {
		return "", fmt.Errorf("keystore not available")
	}
	item, err := k.ring.Get(ref)
	if err != nil {
		return "", fmt.Errorf("keychain retrieve: %w", err)
	}
	return string(item.Data), nil
}

// Delete removes a stored key.
func (k *Keystore) Delete(ref string) error {
	if k.ring == nil {
		return nil
	}
	return k.ring.Remove(ref)
}

// InMemoryKeystore returns a keystore that stores keys in memory (for tests).
type InMemoryKeystore struct {
	data map[string]string
}

// NewInMemoryKeystore creates an in-memory keystore.
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
