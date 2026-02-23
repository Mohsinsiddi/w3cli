package contract_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/contract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryEmpty(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))
	assert.Empty(t, reg.All())
}

func TestRegistryAddAndGet(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))

	entry := &contract.Entry{
		Name:    "usdc",
		Network: "ethereum",
		Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
		ABI:     []contract.ABIEntry{{Name: "balanceOf", Type: "function"}},
	}
	reg.Add(entry)

	got, err := reg.Get("usdc", "ethereum")
	require.NoError(t, err)
	assert.Equal(t, "usdc", got.Name)
	assert.Equal(t, "ethereum", got.Network)
	assert.Equal(t, "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", got.Address)
	assert.Len(t, got.ABI, 1)
}

func TestRegistryGetNotFound(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))

	_, err := reg.Get("nonexistent", "ethereum")
	assert.ErrorIs(t, err, contract.ErrContractNotFound)
}

func TestRegistryGetDifferentNetwork(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))

	reg.Add(&contract.Entry{Name: "usdc", Network: "ethereum", Address: "0xaaa"})

	_, err := reg.Get("usdc", "polygon")
	assert.ErrorIs(t, err, contract.ErrContractNotFound)
}

func TestRegistryAddOverwritesExisting(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))

	reg.Add(&contract.Entry{Name: "usdc", Network: "ethereum", Address: "0xOLD"})
	reg.Add(&contract.Entry{Name: "usdc", Network: "ethereum", Address: "0xNEW"})

	got, err := reg.Get("usdc", "ethereum")
	require.NoError(t, err)
	assert.Equal(t, "0xNEW", got.Address)
	assert.Len(t, reg.All(), 1)
}

func TestRegistryGetByNameMultipleNetworks(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))

	reg.Add(&contract.Entry{Name: "usdc", Network: "ethereum", Address: "0xETH"})
	reg.Add(&contract.Entry{Name: "usdc", Network: "base", Address: "0xBASE"})
	reg.Add(&contract.Entry{Name: "usdc", Network: "polygon", Address: "0xPOLY"})
	reg.Add(&contract.Entry{Name: "dai", Network: "ethereum", Address: "0xDAI"})

	entries := reg.GetByName("usdc")
	assert.Len(t, entries, 3)

	addresses := make(map[string]bool)
	for _, e := range entries {
		assert.Equal(t, "usdc", e.Name)
		addresses[e.Address] = true
	}
	assert.True(t, addresses["0xETH"])
	assert.True(t, addresses["0xBASE"])
	assert.True(t, addresses["0xPOLY"])
}

func TestRegistryGetByNameNoMatch(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))

	reg.Add(&contract.Entry{Name: "usdc", Network: "ethereum", Address: "0x1"})

	entries := reg.GetByName("unknown")
	assert.Empty(t, entries)
}

func TestRegistryAllEmpty(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))
	assert.Empty(t, reg.All())
}

func TestRegistryAllReturnsAll(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))

	reg.Add(&contract.Entry{Name: "usdc", Network: "ethereum"})
	reg.Add(&contract.Entry{Name: "dai", Network: "ethereum"})
	reg.Add(&contract.Entry{Name: "usdc", Network: "base"})

	assert.Len(t, reg.All(), 3)
}

func TestRegistryRemoveExisting(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))

	reg.Add(&contract.Entry{Name: "usdc", Network: "ethereum", Address: "0x1"})

	err := reg.Remove("usdc", "ethereum")
	require.NoError(t, err)

	_, err = reg.Get("usdc", "ethereum")
	assert.ErrorIs(t, err, contract.ErrContractNotFound)
	assert.Empty(t, reg.All())
}

func TestRegistryRemoveNotFound(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))

	err := reg.Remove("ghost", "ethereum")
	assert.ErrorIs(t, err, contract.ErrContractNotFound)
}

func TestRegistryRemoveOnlyTargeted(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))

	reg.Add(&contract.Entry{Name: "usdc", Network: "ethereum"})
	reg.Add(&contract.Entry{Name: "usdc", Network: "base"})

	require.NoError(t, reg.Remove("usdc", "ethereum"))

	_, err := reg.Get("usdc", "ethereum")
	assert.ErrorIs(t, err, contract.ErrContractNotFound)

	got, err := reg.Get("usdc", "base")
	require.NoError(t, err)
	assert.Equal(t, "base", got.Network)
}

func TestRegistryLoadNonExistentFile(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "does-not-exist.json"))

	err := reg.Load()
	assert.NoError(t, err)
	assert.Empty(t, reg.All())
}

func TestRegistryLoadCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "contracts.json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid json"), 0o600))

	reg := contract.NewRegistry(path)
	err := reg.Load()
	assert.Error(t, err)
}

func TestRegistryLoadEmptyArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "contracts.json")
	require.NoError(t, os.WriteFile(path, []byte("[]"), 0o600))

	reg := contract.NewRegistry(path)
	require.NoError(t, reg.Load())
	assert.Empty(t, reg.All())
}

func TestRegistrySaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "contracts.json")

	reg := contract.NewRegistry(path)
	reg.Add(&contract.Entry{
		Name:    "usdc",
		Network: "ethereum",
		Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
		ABI: []contract.ABIEntry{
			{Name: "balanceOf", Type: "function", StateMutability: "view",
				Inputs:  []contract.ABIParam{{Name: "account", Type: "address"}},
				Outputs: []contract.ABIParam{{Name: "", Type: "uint256"}}},
		},
		ABIUrl: "https://example.com/abi",
	})
	reg.Add(&contract.Entry{
		Name:    "dai",
		Network: "base",
		Address: "0xDAI",
	})

	require.NoError(t, reg.Save())

	reg2 := contract.NewRegistry(path)
	require.NoError(t, reg2.Load())

	assert.Len(t, reg2.All(), 2)

	usdc, err := reg2.Get("usdc", "ethereum")
	require.NoError(t, err)
	assert.Equal(t, "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", usdc.Address)
	assert.Equal(t, "https://example.com/abi", usdc.ABIUrl)
	assert.Len(t, usdc.ABI, 1)
	assert.Equal(t, "balanceOf", usdc.ABI[0].Name)
	assert.Len(t, usdc.ABI[0].Inputs, 1)
	assert.Len(t, usdc.ABI[0].Outputs, 1)

	dai, err := reg2.Get("dai", "base")
	require.NoError(t, err)
	assert.Equal(t, "0xDAI", dai.Address)
}

func TestRegistrySaveCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "contracts.json")

	reg := contract.NewRegistry(path)
	reg.Add(&contract.Entry{Name: "test", Network: "ethereum", Address: "0x1"})

	require.NoError(t, reg.Save())

	_, err := os.Stat(path)
	assert.NoError(t, err)
}

func TestRegistrySaveProducesValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "contracts.json")

	reg := contract.NewRegistry(path)
	reg.Add(&contract.Entry{Name: "test", Network: "ethereum", Address: "0x1"})
	require.NoError(t, reg.Save())

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var entries []contract.Entry
	assert.NoError(t, json.Unmarshal(data, &entries))
	assert.Len(t, entries, 1)
}

func TestRegistryAddEmptyABI(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))

	reg.Add(&contract.Entry{Name: "test", Network: "ethereum", Address: "0x1", ABI: nil})

	got, err := reg.Get("test", "ethereum")
	require.NoError(t, err)
	assert.Nil(t, got.ABI)
}

func TestRegistryKeySpecialChars(t *testing.T) {
	reg := contract.NewRegistry(filepath.Join(t.TempDir(), "contracts.json"))

	reg.Add(&contract.Entry{Name: "my-contract.v2", Network: "base-sepolia", Address: "0x1"})

	got, err := reg.Get("my-contract.v2", "base-sepolia")
	require.NoError(t, err)
	assert.Equal(t, "my-contract.v2", got.Name)
}

func TestABIEntryIsReadFunction(t *testing.T) {
	tests := []struct {
		name     string
		entry    contract.ABIEntry
		expected bool
	}{
		{"view function", contract.ABIEntry{Type: "function", StateMutability: "view"}, true},
		{"pure function", contract.ABIEntry{Type: "function", StateMutability: "pure"}, true},
		{"nonpayable function", contract.ABIEntry{Type: "function", StateMutability: "nonpayable"}, false},
		{"payable function", contract.ABIEntry{Type: "function", StateMutability: "payable"}, false},
		{"event type", contract.ABIEntry{Type: "event", StateMutability: "view"}, false},
		{"constructor", contract.ABIEntry{Type: "constructor", StateMutability: "nonpayable"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.entry.IsReadFunction())
		})
	}
}

func TestABIEntryIsWriteFunction(t *testing.T) {
	tests := []struct {
		name     string
		entry    contract.ABIEntry
		expected bool
	}{
		{"nonpayable function", contract.ABIEntry{Type: "function", StateMutability: "nonpayable"}, true},
		{"payable function", contract.ABIEntry{Type: "function", StateMutability: "payable"}, true},
		{"view function", contract.ABIEntry{Type: "function", StateMutability: "view"}, false},
		{"pure function", contract.ABIEntry{Type: "function", StateMutability: "pure"}, false},
		{"event type", contract.ABIEntry{Type: "event", StateMutability: "nonpayable"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.entry.IsWriteFunction())
		})
	}
}

func TestRegistryLoadPreservesABIUrl(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "contracts.json")

	data := `[{"name":"test","network":"eth","address":"0x1","abi":[],"abi_url":"https://example.com"}]`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))

	reg := contract.NewRegistry(path)
	require.NoError(t, reg.Load())

	got, err := reg.Get("test", "eth")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", got.ABIUrl)
}

func TestRegistrySaveEmptyRegistry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "contracts.json")

	reg := contract.NewRegistry(path)
	require.NoError(t, reg.Save())

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var entries []contract.Entry
	require.NoError(t, json.Unmarshal(data, &entries))
	assert.Empty(t, entries)
}
