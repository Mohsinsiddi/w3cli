package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// Wallet types.
const (
	TypeWatchOnly = "watch-only"
	TypeSigning   = "signing"
)

// Errors.
var (
	ErrWalletNotFound = errors.New("wallet not found")
	ErrWalletExists   = errors.New("wallet already exists")
	ErrInvalidKey     = errors.New("invalid private key")
)

// Wallet holds metadata for a single wallet.
type Wallet struct {
	Name      string
	Address   string
	Type      string
	KeyRef    string // keychain service key for signing wallets
	ChainType string // "evm" | "solana" | "sui"
	IsDefault bool
	CreatedAt string
}

// Store is an interface for persisting wallets.
type Store interface {
	Load() ([]*Wallet, error)
	Save([]*Wallet) error
}

// Manager handles wallet CRUD.
type Manager struct {
	store   Store
	wallets map[string]*Wallet
	loaded  bool
}

// Option configures a Manager.
type Option func(*Manager)

// WithInMemoryStore uses an in-memory store (useful for tests).
func WithInMemoryStore() Option {
	return func(m *Manager) {
		m.store = &memStore{}
	}
}

// WithStore sets a custom store.
func WithStore(s Store) Option {
	return func(m *Manager) {
		m.store = s
	}
}

// NewManager creates a new wallet manager.
func NewManager(opts ...Option) *Manager {
	m := &Manager{
		wallets: make(map[string]*Wallet),
		store:   &memStore{},
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Add registers a watch-only (or pre-built) wallet.
func (m *Manager) Add(name string, w *Wallet) error {
	if err := m.load(); err != nil {
		return err
	}
	if _, exists := m.wallets[name]; exists {
		return ErrWalletExists
	}
	if w.CreatedAt == "" {
		w.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	m.wallets[name] = w
	return m.persist()
}

// AddWithKey derives an EVM address from a hex private key and stores the wallet.
// The private key is stored in the keystore (encrypted).
func (m *Manager) AddWithKey(name, hexKey string) error {
	if err := m.load(); err != nil {
		return err
	}
	if _, exists := m.wallets[name]; exists {
		return ErrWalletExists
	}

	privKey, err := crypto.HexToECDSA(stripHexPrefix(hexKey))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}

	addr := crypto.PubkeyToAddress(privKey.PublicKey).Hex()

	// Store the key in the keystore.
	ks := DefaultKeystore()
	ref, err := ks.Store(name, hexKey)
	if err != nil {
		return fmt.Errorf("storing key: %w", err)
	}

	w := &Wallet{
		Name:      name,
		Address:   addr,
		Type:      TypeSigning,
		KeyRef:    ref,
		ChainType: "evm",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	m.wallets[name] = w
	return m.persist()
}

// Get returns a wallet by name.
func (m *Manager) Get(name string) (*Wallet, error) {
	if err := m.load(); err != nil {
		return nil, err
	}
	w, ok := m.wallets[name]
	if !ok {
		return nil, ErrWalletNotFound
	}
	return w, nil
}

// Remove deletes a wallet by name.
func (m *Manager) Remove(name string) error {
	if err := m.load(); err != nil {
		return err
	}
	if _, ok := m.wallets[name]; !ok {
		return ErrWalletNotFound
	}
	delete(m.wallets, name)
	return m.persist()
}

// List returns all wallets.
func (m *Manager) List() []*Wallet {
	m.load() //nolint:errcheck
	out := make([]*Wallet, 0, len(m.wallets))
	for _, w := range m.wallets {
		out = append(out, w)
	}
	return out
}

// SetDefault marks a wallet as the default.
func (m *Manager) SetDefault(name string) error {
	if err := m.load(); err != nil {
		return err
	}
	if _, ok := m.wallets[name]; !ok {
		return ErrWalletNotFound
	}
	for _, w := range m.wallets {
		w.IsDefault = w.Name == name
	}
	return m.persist()
}

// Default returns the default wallet, or nil if none.
func (m *Manager) Default() *Wallet {
	m.load() //nolint:errcheck
	for _, w := range m.wallets {
		if w.IsDefault {
			return w
		}
	}
	// Fallback: return first wallet if only one exists.
	if len(m.wallets) == 1 {
		for _, w := range m.wallets {
			return w
		}
	}
	return nil
}

// --- internal ---

func (m *Manager) load() error {
	if m.loaded {
		return nil
	}
	wallets, err := m.store.Load()
	if err != nil {
		return err
	}
	for _, w := range wallets {
		m.wallets[w.Name] = w
	}
	m.loaded = true
	return nil
}

func (m *Manager) persist() error {
	wallets := make([]*Wallet, 0, len(m.wallets))
	for _, w := range m.wallets {
		wallets = append(wallets, w)
	}
	return m.store.Save(wallets)
}

func stripHexPrefix(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}

// --- in-memory store ---

type memStore struct {
	wallets []*Wallet
}

func (s *memStore) Load() ([]*Wallet, error) {
	return s.wallets, nil
}

func (s *memStore) Save(wallets []*Wallet) error {
	s.wallets = wallets
	return nil
}

// --- JSON file store ---

// JSONStore persists wallets to a JSON file.
type JSONStore struct {
	path string
}

// NewJSONStore creates a JSON-backed wallet store.
func NewJSONStore(path string) *JSONStore {
	return &JSONStore{path: path}
}

func (s *JSONStore) Load() ([]*Wallet, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var wallets []*Wallet
	if err := json.Unmarshal(data, &wallets); err != nil {
		return nil, err
	}
	return wallets, nil
}

func (s *JSONStore) Save(wallets []*Wallet) error {
	data, err := json.MarshalIndent(wallets, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}
