package wallet

import (
	"encoding/hex"
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
	Name      string `json:"name"`
	Address   string `json:"address"`
	Type      string `json:"type"`
	KeyRef    string `json:"key_ref,omitempty"`  // keychain service key for signing wallets
	ChainType string `json:"chain_type,omitempty"` // "evm" | "solana" | "sui"
	IsDefault bool   `json:"is_default"`
	CreatedAt string `json:"created_at"`
}

// Store is an interface for persisting wallets.
type Store interface {
	Load() ([]*Wallet, error)
	Save([]*Wallet) error
}

// Manager handles wallet CRUD.
type Manager struct {
	store    Store
	keystore KeystoreBackend
	wallets  map[string]*Wallet
	loaded   bool
}

// Option configures a Manager.
type Option func(*Manager)

// WithInMemoryStore uses an in-memory store and in-memory keystore (useful for
// tests). Both the wallet metadata and the private keys are kept in process
// memory — no filesystem writes and no OS keychain prompts.
func WithInMemoryStore() Option {
	return func(m *Manager) {
		m.store = &memStore{}
		m.keystore = NewInMemoryKeystore()
	}
}

// WithStore sets a custom store.
func WithStore(s Store) Option {
	return func(m *Manager) {
		m.store = s
	}
}

// WithKeystore overrides the key backend (useful for tests — avoids OS keychain prompts).
func WithKeystore(ks KeystoreBackend) Option {
	return func(m *Manager) {
		m.keystore = ks
	}
}

// NewManager creates a new wallet manager.
func NewManager(opts ...Option) *Manager {
	m := &Manager{
		wallets:  make(map[string]*Wallet),
		store:    &memStore{},
		keystore: DefaultKeystore(),
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

// Generate creates a brand-new EVM keypair, stores the private key in the OS
// keychain, and returns both the wallet metadata and the raw hex private key.
// The caller is responsible for displaying the key to the user exactly once.
func (m *Manager) Generate(name string) (*Wallet, string, error) {
	if err := m.load(); err != nil {
		return nil, "", err
	}
	if _, exists := m.wallets[name]; exists {
		return nil, "", ErrWalletExists
	}

	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, "", fmt.Errorf("generating key: %w", err)
	}

	hexKey := "0x" + hex.EncodeToString(crypto.FromECDSA(privKey))
	addr := crypto.PubkeyToAddress(privKey.PublicKey).Hex()

	ref, err := m.keystore.Store(name, hexKey)
	if err != nil {
		return nil, "", fmt.Errorf("storing key: %w", err)
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
	if err := m.persist(); err != nil {
		return nil, "", err
	}
	return w, hexKey, nil
}

// ExportKey retrieves the raw hex private key for a signing wallet from the keystore.
func (m *Manager) ExportKey(name string) (string, error) {
	if err := m.load(); err != nil {
		return "", err
	}
	w, ok := m.wallets[name]
	if !ok {
		return "", ErrWalletNotFound
	}
	if w.Type != TypeSigning {
		return "", fmt.Errorf("wallet %q is watch-only — no private key stored", name)
	}
	return m.keystore.Retrieve(w.KeyRef)
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
	ref, err := m.keystore.Store(name, hexKey)
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
