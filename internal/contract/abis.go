package contract

import "sort"

// BuiltinKind describes a built-in contract type whose ABI is embedded in the
// binary. New built-ins register themselves via init() in their own file —
// just create internal/contract/<name>_abi.go and call RegisterBuiltin().
type BuiltinKind struct {
	ID          string     // machine key, e.g. "w3token", "erc20"
	Name        string     // human label, e.g. "W3Token (Mintable+Burnable ERC-20)"
	Description string     // one-line summary shown in `contract builtins`
	ABI         []ABIEntry // full ABI, ready to use
}

var builtinRegistry = map[string]BuiltinKind{}

// RegisterBuiltin adds a built-in ABI to the global registry.
// Call this from init() in the file that defines the ABI.
func RegisterBuiltin(b BuiltinKind) {
	builtinRegistry[b.ID] = b
}

// GetBuiltin returns a built-in by ID. ok is false if not found.
func GetBuiltin(id string) (BuiltinKind, bool) {
	b, ok := builtinRegistry[id]
	return b, ok
}

// GetBuiltinABI returns the ABI entries for a built-in ID, or nil if unknown.
func GetBuiltinABI(id string) []ABIEntry {
	b, ok := builtinRegistry[id]
	if !ok {
		return nil
	}
	return b.ABI
}

// AllBuiltins returns all registered built-ins sorted by ID.
func AllBuiltins() []BuiltinKind {
	out := make([]BuiltinKind, 0, len(builtinRegistry))
	for _, b := range builtinRegistry {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
