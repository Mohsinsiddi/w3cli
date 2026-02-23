package contract_test

import (
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/contract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// GetBuiltin / GetBuiltinABI / AllBuiltins
// ---------------------------------------------------------------------------

// registerTestBuiltin injects a test builtin for the duration of a test.
// It uses a unique ID to avoid colliding with real builtins.
func registerTestBuiltin(t *testing.T, id, name string, abi []contract.ABIEntry) {
	t.Helper()
	contract.RegisterBuiltin(contract.BuiltinKind{
		ID:          id,
		Name:        name,
		Description: "test builtin for " + name,
		ABI:         abi,
	})
}

func TestGetBuiltinFound(t *testing.T) {
	id := "test-builtin-found"
	registerTestBuiltin(t, id, "Test Token", []contract.ABIEntry{
		{Name: "transfer", Type: "function"},
	})

	b, ok := contract.GetBuiltin(id)
	require.True(t, ok)
	assert.Equal(t, id, b.ID)
	assert.Equal(t, "Test Token", b.Name)
	assert.Len(t, b.ABI, 1)
}

func TestGetBuiltinNotFound(t *testing.T) {
	_, ok := contract.GetBuiltin("this-id-does-not-exist-xyz")
	assert.False(t, ok)
}

func TestGetBuiltinABIFound(t *testing.T) {
	id := "test-builtin-abi-found"
	abi := []contract.ABIEntry{
		{Name: "balanceOf", Type: "function", StateMutability: "view"},
		{Name: "Transfer", Type: "event"},
	}
	registerTestBuiltin(t, id, "ABI Token", abi)

	got := contract.GetBuiltinABI(id)
	require.NotNil(t, got)
	assert.Len(t, got, 2)
	assert.Equal(t, "balanceOf", got[0].Name)
	assert.Equal(t, "Transfer", got[1].Name)
}

func TestGetBuiltinABINotFound(t *testing.T) {
	got := contract.GetBuiltinABI("completely-unknown-id-abc123")
	assert.Nil(t, got)
}

func TestAllBuiltinsReturnsSorted(t *testing.T) {
	// Register a few test builtins with known IDs.
	contract.RegisterBuiltin(contract.BuiltinKind{ID: "zzz-test", Name: "ZZZ"})
	contract.RegisterBuiltin(contract.BuiltinKind{ID: "aaa-test", Name: "AAA"})
	contract.RegisterBuiltin(contract.BuiltinKind{ID: "mmm-test", Name: "MMM"})

	all := contract.AllBuiltins()
	require.NotEmpty(t, all)

	// Verify ordering is ascending by ID.
	for i := 1; i < len(all); i++ {
		assert.LessOrEqual(t, all[i-1].ID, all[i].ID,
			"AllBuiltins must be sorted by ID: %s > %s", all[i-1].ID, all[i].ID)
	}
}

func TestAllBuiltinsIncludesW3Token(t *testing.T) {
	// The real "w3token" builtin is registered via init() in the package.
	all := contract.AllBuiltins()
	var found bool
	for _, b := range all {
		if b.ID == "w3token" {
			found = true
			break
		}
	}
	assert.True(t, found, "w3token builtin should be registered")
}

func TestRegisterBuiltinOverwrites(t *testing.T) {
	id := "test-overwrite-builtin"
	contract.RegisterBuiltin(contract.BuiltinKind{ID: id, Name: "First"})
	contract.RegisterBuiltin(contract.BuiltinKind{ID: id, Name: "Second"})

	b, ok := contract.GetBuiltin(id)
	require.True(t, ok)
	assert.Equal(t, "Second", b.Name, "second RegisterBuiltin should overwrite first")
}
