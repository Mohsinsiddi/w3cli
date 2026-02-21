package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// LoadArtifactFull — Hardhat format
// ---------------------------------------------------------------------------

func TestLoadArtifactFullHardhatValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.json")

	artifact := map[string]interface{}{
		"abi": []map[string]interface{}{
			{"name": "balanceOf", "type": "function", "inputs": []map[string]string{{"name": "account", "type": "address"}}, "outputs": []map[string]string{{"name": "", "type": "uint256"}}, "stateMutability": "view"},
			{"name": "transfer", "type": "function", "inputs": []map[string]string{{"name": "to", "type": "address"}, {"name": "amount", "type": "uint256"}}, "outputs": []map[string]string{{"name": "", "type": "bool"}}, "stateMutability": "nonpayable"},
		},
		"bytecode": "0x608060405234801561001057600080fd5b50610150806100206000396000f3",
	}
	data, err := json.Marshal(artifact)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	result, err := LoadArtifactFull(path)
	require.NoError(t, err)
	assert.Len(t, result.ABI, 2)
	assert.Equal(t, "balanceOf", result.ABI[0].Name)
	assert.Equal(t, "transfer", result.ABI[1].Name)
	assert.NotEmpty(t, result.Bytecode)
}

func TestLoadArtifactFullHardhatBytecodeDecodes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.json")

	artifact := map[string]interface{}{
		"abi":      []map[string]interface{}{{"name": "foo", "type": "function", "stateMutability": "view"}},
		"bytecode": "0xaabbcc",
	}
	data, _ := json.Marshal(artifact)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	result, err := LoadArtifactFull(path)
	require.NoError(t, err)
	assert.Equal(t, []byte{0xaa, 0xbb, 0xcc}, result.Bytecode)
}

// ---------------------------------------------------------------------------
// LoadArtifactFull — Foundry format
// ---------------------------------------------------------------------------

func TestLoadArtifactFullFoundryValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.json")

	artifact := map[string]interface{}{
		"abi": []map[string]interface{}{
			{"name": "mint", "type": "function", "inputs": []map[string]string{{"name": "to", "type": "address"}}, "stateMutability": "nonpayable"},
		},
		"bytecode": map[string]string{
			"object": "0x608060405234801561001057600080fd",
		},
	}
	data, _ := json.Marshal(artifact)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	result, err := LoadArtifactFull(path)
	require.NoError(t, err)
	assert.Len(t, result.ABI, 1)
	assert.Equal(t, "mint", result.ABI[0].Name)
	assert.NotEmpty(t, result.Bytecode)
}

func TestLoadArtifactFullFoundryBytecodeDecodes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.json")

	artifact := map[string]interface{}{
		"abi":      []map[string]interface{}{{"name": "foo", "type": "function", "stateMutability": "view"}},
		"bytecode": map[string]string{"object": "0xdeadbeef"},
	}
	data, _ := json.Marshal(artifact)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	result, err := LoadArtifactFull(path)
	require.NoError(t, err)
	assert.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, result.Bytecode)
}

// ---------------------------------------------------------------------------
// LoadArtifactFull — with constructor
// ---------------------------------------------------------------------------

func TestLoadArtifactFullWithConstructor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.json")

	artifact := map[string]interface{}{
		"abi": []map[string]interface{}{
			{"type": "constructor", "inputs": []map[string]string{{"name": "name", "type": "string"}, {"name": "symbol", "type": "string"}}},
			{"name": "name", "type": "function", "stateMutability": "view", "inputs": []interface{}{}, "outputs": []map[string]string{{"type": "string"}}},
		},
		"bytecode": "0x6080604052",
	}
	data, _ := json.Marshal(artifact)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	result, err := LoadArtifactFull(path)
	require.NoError(t, err)
	assert.Len(t, result.ABI, 2)

	// Find constructor.
	var ctor *ABIEntry
	for i := range result.ABI {
		if result.ABI[i].Type == "constructor" {
			ctor = &result.ABI[i]
			break
		}
	}
	require.NotNil(t, ctor)
	assert.Len(t, ctor.Inputs, 2)
	assert.Equal(t, "name", ctor.Inputs[0].Name)
	assert.Equal(t, "string", ctor.Inputs[0].Type)
}

// ---------------------------------------------------------------------------
// LoadArtifactFull — error cases
// ---------------------------------------------------------------------------

func TestLoadArtifactFullFileNotFound(t *testing.T) {
	_, err := LoadArtifactFull("/nonexistent/path/artifact.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read artifact file")
}

func TestLoadArtifactFullEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

	_, err := LoadArtifactFull(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestLoadArtifactFullInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("not json at all"), 0o644))

	_, err := LoadArtifactFull(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid artifact JSON")
}

func TestLoadArtifactFullNoABIKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-abi.json")

	data := `{"bytecode":"0x6080","contractName":"Test"}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	_, err := LoadArtifactFull(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid \"abi\" array")
}

func TestLoadArtifactFullEmptyABIArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty-abi.json")

	data := `{"abi":[],"bytecode":"0x6080"}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	_, err := LoadArtifactFull(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestLoadArtifactFullNoBytecode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-bytecode.json")

	data := `{"abi":[{"name":"balanceOf","type":"function","stateMutability":"view"}]}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	_, err := LoadArtifactFull(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bytecode")
}

func TestLoadArtifactFullEmptyBytecodeString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty-bc.json")

	data := `{"abi":[{"name":"balanceOf","type":"function","stateMutability":"view"}],"bytecode":"0x"}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	_, err := LoadArtifactFull(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bytecode is empty")
}

func TestLoadArtifactFullEmptyBytecodeFoundry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty-bc-foundry.json")

	data := `{"abi":[{"name":"balanceOf","type":"function","stateMutability":"view"}],"bytecode":{"object":"0x"}}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	_, err := LoadArtifactFull(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bytecode is empty")
}

func TestLoadArtifactFullInvalidBytecodeHex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-hex.json")

	data := `{"abi":[{"name":"balanceOf","type":"function","stateMutability":"view"}],"bytecode":"0xGGHH"}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	_, err := LoadArtifactFull(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bytecode hex")
}

func TestLoadArtifactFullRawABIArrayRejects(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw-abi.json")

	// A raw ABI array (no wrapping object) — no "abi" key.
	data := `[{"name":"balanceOf","type":"function","stateMutability":"view"}]`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	_, err := LoadArtifactFull(path)
	assert.Error(t, err)
}

func TestLoadArtifactFullABIWithOnlyFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fallback-only.json")

	// ABI with only a fallback function — no real functions or events.
	data := `{"abi":[{"type":"fallback","stateMutability":"payable"}],"bytecode":"0x6080"}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	_, err := LoadArtifactFull(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "none are functions or events")
}

// ---------------------------------------------------------------------------
// LoadArtifactFull — bytecode without 0x prefix
// ---------------------------------------------------------------------------

func TestLoadArtifactFullBytecodeNoPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-prefix.json")

	data := `{"abi":[{"name":"foo","type":"function","stateMutability":"view"}],"bytecode":"aabbcc"}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	result, err := LoadArtifactFull(path)
	require.NoError(t, err)
	assert.Equal(t, []byte{0xaa, 0xbb, 0xcc}, result.Bytecode)
}

// ---------------------------------------------------------------------------
// extractBytecodeHex
// ---------------------------------------------------------------------------

func TestExtractBytecodeHexPlainString(t *testing.T) {
	raw := json.RawMessage(`"0x608060405234801561001057600080fd"`)
	result, err := extractBytecodeHex(raw)
	require.NoError(t, err)
	assert.Equal(t, "0x608060405234801561001057600080fd", result)
}

func TestExtractBytecodeHexPlainStringNoPrefix(t *testing.T) {
	raw := json.RawMessage(`"aabbcc"`)
	result, err := extractBytecodeHex(raw)
	require.NoError(t, err)
	assert.Equal(t, "aabbcc", result)
}

func TestExtractBytecodeHexFoundryObject(t *testing.T) {
	raw := json.RawMessage(`{"object":"0x608060405234801561001057600080fd"}`)
	result, err := extractBytecodeHex(raw)
	require.NoError(t, err)
	assert.Equal(t, "0x608060405234801561001057600080fd", result)
}

func TestExtractBytecodeHexFoundryObjectNoPrefix(t *testing.T) {
	raw := json.RawMessage(`{"object":"aabbcc"}`)
	result, err := extractBytecodeHex(raw)
	require.NoError(t, err)
	assert.Equal(t, "aabbcc", result)
}

func TestExtractBytecodeHexEmptyString(t *testing.T) {
	raw := json.RawMessage(`""`)
	result, err := extractBytecodeHex(raw)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestExtractBytecodeHexEmptyObject(t *testing.T) {
	raw := json.RawMessage(`{"object":""}`)
	// Empty object field — falls through to error.
	_, err := extractBytecodeHex(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "neither a hex string nor")
}

func TestExtractBytecodeHexInvalidType(t *testing.T) {
	raw := json.RawMessage(`12345`)
	_, err := extractBytecodeHex(raw)
	assert.Error(t, err)
}

func TestExtractBytecodeHexArray(t *testing.T) {
	raw := json.RawMessage(`[1,2,3]`)
	_, err := extractBytecodeHex(raw)
	assert.Error(t, err)
}

func TestExtractBytecodeHexWhitespace(t *testing.T) {
	raw := json.RawMessage(`"  0xaabbcc  "`)
	result, err := extractBytecodeHex(raw)
	require.NoError(t, err)
	assert.Equal(t, "0xaabbcc", result)
}

// ---------------------------------------------------------------------------
// validateABI
// ---------------------------------------------------------------------------

func TestValidateABIEmptySlice(t *testing.T) {
	err := validateABI([]ABIEntry{}, "test.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestValidateABINilSlice(t *testing.T) {
	err := validateABI(nil, "test.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestValidateABIOnlyFallbackAndReceive(t *testing.T) {
	abi := []ABIEntry{
		{Type: "fallback", StateMutability: "payable"},
		{Type: "receive", StateMutability: "payable"},
	}
	err := validateABI(abi, "test.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "none are functions or events")
}

func TestValidateABIWithFunction(t *testing.T) {
	abi := []ABIEntry{
		{Name: "balanceOf", Type: "function", StateMutability: "view"},
	}
	err := validateABI(abi, "test.json")
	assert.NoError(t, err)
}

func TestValidateABIWithEvent(t *testing.T) {
	abi := []ABIEntry{
		{Name: "Transfer", Type: "event"},
	}
	err := validateABI(abi, "test.json")
	assert.NoError(t, err)
}

func TestValidateABIWithConstructor(t *testing.T) {
	abi := []ABIEntry{
		{Type: "constructor", Inputs: []ABIParam{{Name: "name", Type: "string"}}},
	}
	err := validateABI(abi, "test.json")
	assert.NoError(t, err)
}

func TestValidateABIMixedTypes(t *testing.T) {
	abi := []ABIEntry{
		{Type: "constructor"},
		{Type: "fallback"},
		{Name: "Transfer", Type: "event"},
		{Name: "balanceOf", Type: "function", StateMutability: "view"},
	}
	err := validateABI(abi, "test.json")
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// LoadFromArtifact — existing function (test additional artifact formats)
// ---------------------------------------------------------------------------

func TestLoadFromArtifactHardhatFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hardhat.json")

	data := `{
		"abi": [
			{"name":"name","type":"function","inputs":[],"outputs":[{"type":"string"}],"stateMutability":"view"},
			{"name":"transfer","type":"function","inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"stateMutability":"nonpayable"}
		],
		"bytecode": "0x608060405234801561001057600080fd",
		"contractName": "MyToken"
	}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	abi, err := LoadFromArtifact(path)
	require.NoError(t, err)
	assert.Len(t, abi, 2)
	assert.Equal(t, "name", abi[0].Name)
}

func TestLoadFromArtifactFoundryFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "foundry.json")

	data := `{
		"abi": [
			{"name":"mint","type":"function","inputs":[{"name":"to","type":"address"}],"stateMutability":"nonpayable"}
		],
		"bytecode": {"object": "0x608060405234801561001057600080fd"}
	}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	abi, err := LoadFromArtifact(path)
	require.NoError(t, err)
	assert.Len(t, abi, 1)
	assert.Equal(t, "mint", abi[0].Name)
}

func TestLoadFromArtifactRawABIArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.json")

	data := `[
		{"name":"balanceOf","type":"function","inputs":[{"name":"account","type":"address"}],"outputs":[{"type":"uint256"}],"stateMutability":"view"}
	]`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	abi, err := LoadFromArtifact(path)
	require.NoError(t, err)
	assert.Len(t, abi, 1)
	assert.Equal(t, "balanceOf", abi[0].Name)
}

func TestLoadFromArtifactEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

	_, err := LoadFromArtifact(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestLoadFromArtifactFileNotFound(t *testing.T) {
	_, err := LoadFromArtifact("/nonexistent/path.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read ABI file")
}

func TestLoadFromArtifactEmptyABI(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty-abi.json")

	data := `{"abi":[],"bytecode":"0x6080"}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	_, err := LoadFromArtifact(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestLoadFromArtifactObjectNoABIKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-abi.json")

	data := `{"bytecode":"0x6080","contractName":"Test"}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	_, err := LoadFromArtifact(path)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// LoadArtifactFull — integration: full realistic artifacts
// ---------------------------------------------------------------------------

func TestLoadArtifactFullRealisticHardhat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ERC20.json")

	artifact := map[string]interface{}{
		"_format":      "hh-sol-artifact-1",
		"contractName": "ERC20",
		"sourceName":   "contracts/ERC20.sol",
		"abi": []map[string]interface{}{
			{"type": "constructor", "inputs": []map[string]string{{"name": "name_", "type": "string"}, {"name": "symbol_", "type": "string"}}},
			{"name": "name", "type": "function", "inputs": []interface{}{}, "outputs": []map[string]string{{"type": "string"}}, "stateMutability": "view"},
			{"name": "symbol", "type": "function", "inputs": []interface{}{}, "outputs": []map[string]string{{"type": "string"}}, "stateMutability": "view"},
			{"name": "decimals", "type": "function", "inputs": []interface{}{}, "outputs": []map[string]string{{"type": "uint8"}}, "stateMutability": "view"},
			{"name": "totalSupply", "type": "function", "inputs": []interface{}{}, "outputs": []map[string]string{{"type": "uint256"}}, "stateMutability": "view"},
			{"name": "balanceOf", "type": "function", "inputs": []map[string]string{{"name": "account", "type": "address"}}, "outputs": []map[string]string{{"type": "uint256"}}, "stateMutability": "view"},
			{"name": "transfer", "type": "function", "inputs": []map[string]string{{"name": "to", "type": "address"}, {"name": "amount", "type": "uint256"}}, "outputs": []map[string]string{{"type": "bool"}}, "stateMutability": "nonpayable"},
			{"name": "Transfer", "type": "event", "inputs": []map[string]interface{}{{"name": "from", "type": "address", "indexed": true}, {"name": "to", "type": "address", "indexed": true}, {"name": "value", "type": "uint256", "indexed": false}}},
		},
		"bytecode":         "0x60806040523480156200001157600080fd5b50604051620010073803806200100783398101604081905262000035916200011f565b",
		"deployedBytecode": "0x608060405234801561001057600080fd",
		"linkReferences":   map[string]interface{}{},
	}
	data, _ := json.Marshal(artifact)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	result, err := LoadArtifactFull(path)
	require.NoError(t, err)
	assert.Len(t, result.ABI, 8)
	assert.NotEmpty(t, result.Bytecode)

	// Verify constructor is present.
	var ctorFound bool
	for _, e := range result.ABI {
		if e.Type == "constructor" {
			ctorFound = true
			assert.Len(t, e.Inputs, 2)
		}
	}
	assert.True(t, ctorFound, "constructor should be in the parsed ABI")
}

func TestLoadArtifactFullRealisticFoundry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Counter.json")

	artifact := map[string]interface{}{
		"abi": []map[string]interface{}{
			{"name": "number", "type": "function", "inputs": []interface{}{}, "outputs": []map[string]string{{"type": "uint256"}}, "stateMutability": "view"},
			{"name": "setNumber", "type": "function", "inputs": []map[string]string{{"name": "newNumber", "type": "uint256"}}, "stateMutability": "nonpayable"},
			{"name": "increment", "type": "function", "inputs": []interface{}{}, "stateMutability": "nonpayable"},
		},
		"bytecode": map[string]interface{}{
			"object":         "0x608060405234801561001057600080fd5b5060f78061001f6000396000f3fe6080604052",
			"sourceMap":      "66:193:0:-:0;;;;;;;",
			"linkReferences": map[string]interface{}{},
		},
		"deployedBytecode": map[string]interface{}{
			"object": "0x6080604052",
		},
	}
	data, _ := json.Marshal(artifact)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	result, err := LoadArtifactFull(path)
	require.NoError(t, err)
	assert.Len(t, result.ABI, 3)
	assert.NotEmpty(t, result.Bytecode)
}
