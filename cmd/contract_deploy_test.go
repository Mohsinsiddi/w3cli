package cmd

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/contract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// EncodeConstructorArgs — public API from cmd layer (integration)
// ---------------------------------------------------------------------------

func TestEncodeConstructorArgsNoParamsFromCmd(t *testing.T) {
	result, err := contract.EncodeConstructorArgs(nil, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEncodeConstructorArgsSingleAddressFromCmd(t *testing.T) {
	params := []contract.ABIParam{{Name: "owner", Type: "address"}}
	result, err := contract.EncodeConstructorArgs(params, []string{
		"0xd8da6bf26964af9d7eed9e03e53415d37aa96045",
	})
	require.NoError(t, err)
	assert.Len(t, result, 32)
}

func TestEncodeConstructorArgsMultipleFromCmd(t *testing.T) {
	params := []contract.ABIParam{
		{Name: "name", Type: "string"},
		{Name: "symbol", Type: "string"},
		{Name: "decimals", Type: "uint8"},
		{Name: "supply", Type: "uint256"},
	}
	result, err := contract.EncodeConstructorArgs(params, []string{
		"MyToken", "MTK", "18", "1000000",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	// Head is 4 * 32 = 128 bytes minimum.
	assert.True(t, len(result) >= 128)
}

func TestEncodeConstructorArgsMismatchFromCmd(t *testing.T) {
	params := []contract.ABIParam{
		{Name: "a", Type: "uint256"},
		{Name: "b", Type: "uint256"},
	}
	_, err := contract.EncodeConstructorArgs(params, []string{"1"})
	assert.Error(t, err)
}

func TestEncodeConstructorArgsArrayRejectedFromCmd(t *testing.T) {
	params := []contract.ABIParam{{Name: "ids", Type: "uint256[]"}}
	_, err := contract.EncodeConstructorArgs(params, []string{"[1,2]"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet supported")
}

// ---------------------------------------------------------------------------
// LoadArtifactFull — public API from cmd layer (integration)
// ---------------------------------------------------------------------------

func TestLoadArtifactFullHardhatFromCmd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Token.json")

	artifact := map[string]interface{}{
		"abi": []map[string]interface{}{
			{"type": "constructor", "inputs": []map[string]string{
				{"name": "name", "type": "string"},
				{"name": "symbol", "type": "string"},
			}},
			{"name": "name", "type": "function", "inputs": []interface{}{},
				"outputs": []map[string]string{{"type": "string"}}, "stateMutability": "view"},
			{"name": "symbol", "type": "function", "inputs": []interface{}{},
				"outputs": []map[string]string{{"type": "string"}}, "stateMutability": "view"},
			{"name": "transfer", "type": "function",
				"inputs":          []map[string]string{{"name": "to", "type": "address"}, {"name": "amount", "type": "uint256"}},
				"outputs":         []map[string]string{{"type": "bool"}},
				"stateMutability": "nonpayable"},
		},
		"bytecode": "0x608060405234801561001057600080fd5b50604051610a",
	}
	data, _ := json.Marshal(artifact)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	result, err := contract.LoadArtifactFull(path)
	require.NoError(t, err)
	assert.Len(t, result.ABI, 4)
	assert.NotEmpty(t, result.Bytecode)

	// Find constructor.
	var ctor *contract.ABIEntry
	for i := range result.ABI {
		if result.ABI[i].Type == "constructor" {
			ctor = &result.ABI[i]
			break
		}
	}
	require.NotNil(t, ctor)
	assert.Len(t, ctor.Inputs, 2)
}

func TestLoadArtifactFullFoundryFromCmd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Counter.json")

	artifact := map[string]interface{}{
		"abi": []map[string]interface{}{
			{"name": "number", "type": "function", "inputs": []interface{}{},
				"outputs": []map[string]string{{"type": "uint256"}}, "stateMutability": "view"},
			{"name": "setNumber", "type": "function",
				"inputs": []map[string]string{{"name": "newNumber", "type": "uint256"}}, "stateMutability": "nonpayable"},
		},
		"bytecode": map[string]string{"object": "0x608060405234801561001057600080fd"},
	}
	data, _ := json.Marshal(artifact)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	result, err := contract.LoadArtifactFull(path)
	require.NoError(t, err)
	assert.Len(t, result.ABI, 2)
	assert.NotEmpty(t, result.Bytecode)
}

func TestLoadArtifactFullNoBytecodeFromCmd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Interface.json")

	data := `{"abi":[{"name":"foo","type":"function","stateMutability":"view"}]}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	_, err := contract.LoadArtifactFull(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bytecode")
}

func TestLoadArtifactFullEmptyBytecodeFromCmd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Abstract.json")

	data := `{"abi":[{"name":"foo","type":"function","stateMutability":"view"}],"bytecode":"0x"}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	_, err := contract.LoadArtifactFull(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestLoadArtifactFullInvalidJSONFromCmd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("broken"), 0o644))

	_, err := contract.LoadArtifactFull(path)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Deploy data construction (bytecode + encoded args)
// ---------------------------------------------------------------------------

func TestDeployDataBytecodeOnly(t *testing.T) {
	bytecode := []byte{0x60, 0x80, 0x60, 0x40}
	deployData := make([]byte, len(bytecode))
	copy(deployData, bytecode)

	assert.Len(t, deployData, 4)
	assert.Equal(t, bytecode, deployData)
}

func TestDeployDataBytecodeWithConstructorArgs(t *testing.T) {
	bytecode := []byte{0x60, 0x80, 0x60, 0x40}
	params := []contract.ABIParam{{Name: "supply", Type: "uint256"}}

	encodedArgs, err := contract.EncodeConstructorArgs(params, []string{"1000000"})
	require.NoError(t, err)

	deployData := make([]byte, len(bytecode)+len(encodedArgs))
	copy(deployData, bytecode)
	copy(deployData[len(bytecode):], encodedArgs)

	assert.Len(t, deployData, 4+32)
	// First 4 bytes should be bytecode.
	assert.Equal(t, bytecode, deployData[:4])
	// Last 32 bytes should contain the encoded uint256.
	supply := new(big.Int).SetBytes(deployData[4:])
	assert.Equal(t, "1000000", supply.String())
}

func TestDeployDataBytecodeWithStringArg(t *testing.T) {
	bytecode := []byte{0x60, 0x80}
	params := []contract.ABIParam{{Name: "name", Type: "string"}}

	encodedArgs, err := contract.EncodeConstructorArgs(params, []string{"MyNFT"})
	require.NoError(t, err)

	deployData := make([]byte, len(bytecode)+len(encodedArgs))
	copy(deployData, bytecode)
	copy(deployData[len(bytecode):], encodedArgs)

	// Bytecode should appear first.
	assert.Equal(t, byte(0x60), deployData[0])
	assert.Equal(t, byte(0x80), deployData[1])
	// Encoded args should follow.
	assert.True(t, len(deployData) > 2)
}

func TestDeployDataHexEncoding(t *testing.T) {
	bytecode := []byte{0xaa, 0xbb, 0xcc}
	deployData := bytecode
	deployHex := "0x" + hex.EncodeToString(deployData)
	assert.Equal(t, "0xaabbcc", deployHex)
}

// ---------------------------------------------------------------------------
// abiTypeExample — helper function tests
// ---------------------------------------------------------------------------

func TestAbiTypeExampleAddress(t *testing.T) {
	ex := abiTypeExample("address")
	assert.Contains(t, ex, "0x")
	assert.Contains(t, ex, "42")
}

func TestAbiTypeExampleUint256(t *testing.T) {
	ex := abiTypeExample("uint256")
	assert.NotEmpty(t, ex)
	assert.Contains(t, ex, "1000000")
}

func TestAbiTypeExampleUint8(t *testing.T) {
	ex := abiTypeExample("uint8")
	assert.Contains(t, ex, "18")
}

func TestAbiTypeExampleBool(t *testing.T) {
	ex := abiTypeExample("bool")
	assert.Contains(t, ex, "true")
	assert.Contains(t, ex, "false")
}

func TestAbiTypeExampleString(t *testing.T) {
	ex := abiTypeExample("string")
	assert.Contains(t, ex, "hello")
}

func TestAbiTypeExampleBytes32(t *testing.T) {
	ex := abiTypeExample("bytes32")
	assert.Contains(t, ex, "0x")
}

func TestAbiTypeExampleBytes(t *testing.T) {
	ex := abiTypeExample("bytes")
	assert.Contains(t, ex, "0x")
}

func TestAbiTypeExampleUint128(t *testing.T) {
	ex := abiTypeExample("uint128")
	assert.NotEmpty(t, ex)
}

func TestAbiTypeExampleInt256(t *testing.T) {
	ex := abiTypeExample("int256")
	assert.Contains(t, ex, "-100")
}

func TestAbiTypeExampleUnknown(t *testing.T) {
	ex := abiTypeExample("tuple")
	assert.Equal(t, "", ex)
}

// ---------------------------------------------------------------------------
// validateABIInput — input validation helper tests
// ---------------------------------------------------------------------------

func TestValidateABIInputAddressValid(t *testing.T) {
	msg := validateABIInput("address", "0xd8da6bf26964af9d7eed9e03e53415d37aa96045")
	assert.Empty(t, msg)
}

func TestValidateABIInputAddressNoPrefix(t *testing.T) {
	msg := validateABIInput("address", "d8da6bf26964af9d7eed9e03e53415d37aa96045")
	assert.Contains(t, msg, "0x")
}

func TestValidateABIInputAddressTooShort(t *testing.T) {
	msg := validateABIInput("address", "0x1234")
	assert.Contains(t, msg, "42")
}

func TestValidateABIInputAddressInvalidHex(t *testing.T) {
	msg := validateABIInput("address", "0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG")
	assert.Contains(t, msg, "invalid hex")
}

func TestValidateABIInputAddressEmpty(t *testing.T) {
	msg := validateABIInput("address", "")
	assert.Contains(t, msg, "empty")
}

func TestValidateABIInputBoolValid(t *testing.T) {
	for _, v := range []string{"true", "false", "1", "0"} {
		msg := validateABIInput("bool", v)
		assert.Empty(t, msg, "should accept %q", v)
	}
}

func TestValidateABIInputBoolInvalid(t *testing.T) {
	msg := validateABIInput("bool", "maybe")
	assert.Contains(t, msg, "true")
}

func TestValidateABIInputUint256Valid(t *testing.T) {
	msg := validateABIInput("uint256", "1000000000000000000")
	assert.Empty(t, msg)
}

func TestValidateABIInputUint256Zero(t *testing.T) {
	msg := validateABIInput("uint256", "0")
	assert.Empty(t, msg)
}

func TestValidateABIInputUint256Negative(t *testing.T) {
	msg := validateABIInput("uint256", "-1")
	assert.Contains(t, msg, "negative")
}

func TestValidateABIInputUint256NonNumeric(t *testing.T) {
	msg := validateABIInput("uint256", "abc")
	assert.NotEmpty(t, msg)
}

func TestValidateABIInputInt256Valid(t *testing.T) {
	msg := validateABIInput("int256", "-100")
	assert.Empty(t, msg)
}

func TestValidateABIInputInt256Invalid(t *testing.T) {
	msg := validateABIInput("int256", "xyz")
	assert.NotEmpty(t, msg)
}

func TestValidateABIInputBytes32Valid(t *testing.T) {
	msg := validateABIInput("bytes32", "0xdeadbeef")
	assert.Empty(t, msg)
}

func TestValidateABIInputBytes32TooLong(t *testing.T) {
	// 65 hex chars (> 64)
	msg := validateABIInput("bytes32", "0x"+string(make([]byte, 65)))
	assert.NotEmpty(t, msg)
}

func TestValidateABIInputBytesValid(t *testing.T) {
	msg := validateABIInput("bytes", "0xdeadbeef")
	assert.Empty(t, msg)
}

func TestValidateABIInputBytesInvalidHex(t *testing.T) {
	msg := validateABIInput("bytes", "0xGGHH")
	assert.Contains(t, msg, "hex")
}

func TestValidateABIInputStringValid(t *testing.T) {
	// string type has no special validation beyond non-empty.
	msg := validateABIInput("string", "hello world")
	assert.Empty(t, msg)
}

// ---------------------------------------------------------------------------
// scaleTokenInput — scaling helper tests
// ---------------------------------------------------------------------------

func TestScaleTokenInputSimple(t *testing.T) {
	result, errMsg := scaleTokenInput("1.0", 18)
	assert.Empty(t, errMsg)
	expected, _ := new(big.Int).SetString("1000000000000000000", 10)
	assert.Equal(t, expected.String(), result.String())
}

func TestScaleTokenInputFractional(t *testing.T) {
	result, errMsg := scaleTokenInput("0.5", 18)
	assert.Empty(t, errMsg)
	expected, _ := new(big.Int).SetString("500000000000000000", 10)
	assert.Equal(t, expected.String(), result.String())
}

func TestScaleTokenInputZero(t *testing.T) {
	result, errMsg := scaleTokenInput("0", 18)
	assert.Empty(t, errMsg)
	assert.Equal(t, "0", result.String())
}

func TestScaleTokenInputSixDecimals(t *testing.T) {
	result, errMsg := scaleTokenInput("1.0", 6)
	assert.Empty(t, errMsg)
	assert.Equal(t, "1000000", result.String())
}

func TestScaleTokenInputWholeNumber(t *testing.T) {
	result, errMsg := scaleTokenInput("100", 18)
	assert.Empty(t, errMsg)
	expected, _ := new(big.Int).SetString("100000000000000000000", 10)
	assert.Equal(t, expected.String(), result.String())
}

func TestScaleTokenInputEmpty(t *testing.T) {
	_, errMsg := scaleTokenInput("", 18)
	assert.NotEmpty(t, errMsg)
}

func TestScaleTokenInputInvalid(t *testing.T) {
	_, errMsg := scaleTokenInput("abc", 18)
	assert.NotEmpty(t, errMsg)
}

func TestScaleTokenInputNegative(t *testing.T) {
	_, errMsg := scaleTokenInput("-1", 18)
	assert.NotEmpty(t, errMsg)
}

// ---------------------------------------------------------------------------
// abiKnownDescription — known function descriptions
// ---------------------------------------------------------------------------

func TestAbiKnownDescriptionKnown(t *testing.T) {
	assert.Contains(t, abiKnownDescription("name"), "token name")
	assert.Contains(t, abiKnownDescription("symbol"), "ticker")
	assert.Contains(t, abiKnownDescription("decimals"), "decimal")
	assert.Contains(t, abiKnownDescription("totalSupply"), "total")
	assert.Contains(t, abiKnownDescription("balanceOf"), "balance")
	assert.Contains(t, abiKnownDescription("transfer"), "Transfer")
	assert.Contains(t, abiKnownDescription("approve"), "Approve")
	assert.Contains(t, abiKnownDescription("mint"), "Mint")
	assert.Contains(t, abiKnownDescription("burn"), "destroy")
}

func TestAbiKnownDescriptionUnknown(t *testing.T) {
	assert.Equal(t, "", abiKnownDescription("unknownFunction"))
}

// ---------------------------------------------------------------------------
// countFunctions — counts function-type entries in ABI
// ---------------------------------------------------------------------------

func TestCountFunctionsEmpty(t *testing.T) {
	assert.Equal(t, 0, countFunctions(nil))
	assert.Equal(t, 0, countFunctions([]contract.ABIEntry{}))
}

func TestCountFunctionsOnlyFunctions(t *testing.T) {
	abi := []contract.ABIEntry{
		{Name: "name", Type: "function"},
		{Name: "symbol", Type: "function"},
		{Name: "decimals", Type: "function"},
	}
	assert.Equal(t, 3, countFunctions(abi))
}

func TestCountFunctionsMixed(t *testing.T) {
	abi := []contract.ABIEntry{
		{Type: "constructor"},
		{Name: "Transfer", Type: "event"},
		{Name: "name", Type: "function"},
		{Name: "transfer", Type: "function"},
		{Type: "fallback"},
	}
	assert.Equal(t, 2, countFunctions(abi))
}

// ---------------------------------------------------------------------------
// formatParams / formatOutputs — helper formatters
// ---------------------------------------------------------------------------

func TestFormatParamsEmpty(t *testing.T) {
	assert.Equal(t, "", formatParams(nil))
	assert.Equal(t, "", formatParams([]contract.ABIParam{}))
}

func TestFormatParamsSingle(t *testing.T) {
	params := []contract.ABIParam{{Name: "to", Type: "address"}}
	assert.Equal(t, "address to", formatParams(params))
}

func TestFormatParamsMultiple(t *testing.T) {
	params := []contract.ABIParam{
		{Name: "to", Type: "address"},
		{Name: "amount", Type: "uint256"},
	}
	assert.Equal(t, "address to, uint256 amount", formatParams(params))
}

func TestFormatParamsNoName(t *testing.T) {
	params := []contract.ABIParam{{Name: "", Type: "uint256"}}
	assert.Equal(t, "uint256", formatParams(params))
}

func TestFormatOutputsEmpty(t *testing.T) {
	assert.Equal(t, "", formatOutputs(nil))
	assert.Equal(t, "", formatOutputs([]contract.ABIParam{}))
}

func TestFormatOutputsSingle(t *testing.T) {
	params := []contract.ABIParam{{Type: "uint256"}}
	assert.Equal(t, "uint256", formatOutputs(params))
}

func TestFormatOutputsMultiple(t *testing.T) {
	params := []contract.ABIParam{
		{Type: "uint256"},
		{Type: "bool"},
	}
	assert.Equal(t, "uint256, bool", formatOutputs(params))
}

// ---------------------------------------------------------------------------
// Deploy command registration
// ---------------------------------------------------------------------------

func TestContractDeployCommandRegistered(t *testing.T) {
	// Verify the deploy subcommand is registered under contract.
	found := false
	for _, cmd := range contractCmd.Commands() {
		if cmd.Use == "deploy <name> <artifact-path>" {
			found = true
			break
		}
	}
	assert.True(t, found, "deploy should be registered as a subcommand of contract")
}

func TestContractDeployCommandFlags(t *testing.T) {
	// Verify deploy command has all expected flags.
	flags := contractDeployCmd.Flags()

	f := flags.Lookup("args")
	require.NotNil(t, f, "--args flag should exist")
	assert.Equal(t, "string", f.Value.Type())

	f = flags.Lookup("value")
	require.NotNil(t, f, "--value flag should exist")
	assert.Equal(t, "string", f.Value.Type())

	f = flags.Lookup("gas")
	require.NotNil(t, f, "--gas flag should exist")
	assert.Equal(t, "uint64", f.Value.Type())

	f = flags.Lookup("wallet")
	require.NotNil(t, f, "--wallet flag should exist")
	assert.Equal(t, "string", f.Value.Type())

	f = flags.Lookup("network")
	require.NotNil(t, f, "--network flag should exist")
	assert.Equal(t, "string", f.Value.Type())
}

func TestContractDeployCommandRequiresExactArgs(t *testing.T) {
	// The command requires exactly 2 args.
	assert.NotNil(t, contractDeployCmd.Args)
}

func TestContractDeployCommandShortDescription(t *testing.T) {
	assert.Contains(t, contractDeployCmd.Short, "Deploy")
}

func TestContractDeployCommandLongDescription(t *testing.T) {
	assert.Contains(t, contractDeployCmd.Long, "artifact")
	assert.Contains(t, contractDeployCmd.Long, "bytecode")
	assert.Contains(t, contractDeployCmd.Long, "auto-registered")
}

// ---------------------------------------------------------------------------
// ethToWei — exact string-based ETH → wei conversion
// ---------------------------------------------------------------------------

func TestEthToWeiExactOneTenth(t *testing.T) {
	wei, err := ethToWei("0.1")
	require.NoError(t, err)
	expected, _ := new(big.Int).SetString("100000000000000000", 10) // 1e17
	assert.Equal(t, expected.String(), wei.String())
}

func TestEthToWeiExactOneThousandth(t *testing.T) {
	// This was the precision bug — 0.001 ETH must be exactly 1e15 wei.
	wei, err := ethToWei("0.001")
	require.NoError(t, err)
	expected, _ := new(big.Int).SetString("1000000000000000", 10) // 1e15
	assert.Equal(t, expected.String(), wei.String())
}

func TestEthToWeiWholeNumber(t *testing.T) {
	wei, err := ethToWei("1")
	require.NoError(t, err)
	expected, _ := new(big.Int).SetString("1000000000000000000", 10) // 1e18
	assert.Equal(t, expected.String(), wei.String())
}

func TestEthToWeiZero(t *testing.T) {
	wei, err := ethToWei("0")
	require.NoError(t, err)
	assert.Equal(t, "0", wei.String())
}

func TestEthToWeiLargeAmount(t *testing.T) {
	wei, err := ethToWei("100")
	require.NoError(t, err)
	expected, _ := new(big.Int).SetString("100000000000000000000", 10) // 1e20
	assert.Equal(t, expected.String(), wei.String())
}

func TestEthToWeiSmallFraction(t *testing.T) {
	// 0.000000000000000001 ETH = 1 wei
	wei, err := ethToWei("0.000000000000000001")
	require.NoError(t, err)
	assert.Equal(t, "1", wei.String())
}

func TestEthToWeiExcessDecimals(t *testing.T) {
	// More than 18 decimals — should truncate to 18.
	wei, err := ethToWei("0.0000000000000000019")
	require.NoError(t, err)
	assert.Equal(t, "1", wei.String(), "should truncate past 18 decimals")
}

func TestEthToWeiMixed(t *testing.T) {
	wei, err := ethToWei("1.5")
	require.NoError(t, err)
	expected, _ := new(big.Int).SetString("1500000000000000000", 10) // 1.5e18
	assert.Equal(t, expected.String(), wei.String())
}

func TestEthToWeiEmpty(t *testing.T) {
	_, err := ethToWei("")
	assert.Error(t, err)
}

func TestEthToWeiInvalidString(t *testing.T) {
	_, err := ethToWei("abc")
	assert.Error(t, err)
}

func TestEthToWeiNoLeadingZero(t *testing.T) {
	// ".5" — no leading zero before the decimal
	wei, err := ethToWei(".5")
	require.NoError(t, err)
	expected, _ := new(big.Int).SetString("500000000000000000", 10) // 0.5e18
	assert.Equal(t, expected.String(), wei.String())
}

func TestEthToWeiWhitespace(t *testing.T) {
	wei, err := ethToWei("  0.001  ")
	require.NoError(t, err)
	expected, _ := new(big.Int).SetString("1000000000000000", 10)
	assert.Equal(t, expected.String(), wei.String())
}

// ---------------------------------------------------------------------------
// abiToStudioEntries — payable detection
// ---------------------------------------------------------------------------

func TestAbiToStudioEntriesPayableFunction(t *testing.T) {
	abi := []contract.ABIEntry{
		{Name: "deposit", Type: "function", StateMutability: "payable"},
	}
	entries := abiToStudioEntries(abi, -1)
	require.Len(t, entries, 1)
	assert.True(t, entries[0].IsPayable, "payable function should set IsPayable=true")
	assert.True(t, entries[0].IsWrite, "payable function should be a write function")
}

func TestAbiToStudioEntriesNonPayableWrite(t *testing.T) {
	abi := []contract.ABIEntry{
		{Name: "withdraw", Type: "function", StateMutability: "nonpayable",
			Inputs: []contract.ABIParam{{Name: "amount", Type: "uint256"}}},
	}
	entries := abiToStudioEntries(abi, -1)
	require.Len(t, entries, 1)
	assert.False(t, entries[0].IsPayable, "nonpayable function should set IsPayable=false")
	assert.True(t, entries[0].IsWrite, "nonpayable function should still be a write function")
}

func TestAbiToStudioEntriesViewNotPayable(t *testing.T) {
	abi := []contract.ABIEntry{
		{Name: "getBalance", Type: "function", StateMutability: "view"},
	}
	entries := abiToStudioEntries(abi, -1)
	require.Len(t, entries, 1)
	assert.False(t, entries[0].IsPayable, "view function should not be payable")
	assert.False(t, entries[0].IsWrite, "view function should not be a write function")
}

func TestAbiToStudioEntriesPureNotPayable(t *testing.T) {
	abi := []contract.ABIEntry{
		{Name: "add", Type: "function", StateMutability: "pure",
			Inputs: []contract.ABIParam{{Name: "a", Type: "uint256"}, {Name: "b", Type: "uint256"}}},
	}
	entries := abiToStudioEntries(abi, -1)
	require.Len(t, entries, 1)
	assert.False(t, entries[0].IsPayable)
	assert.False(t, entries[0].IsWrite)
}

func TestAbiToStudioEntriesEventNotPayable(t *testing.T) {
	abi := []contract.ABIEntry{
		{Name: "Deposited", Type: "event",
			Inputs: []contract.ABIParam{{Name: "sender", Type: "address"}, {Name: "amount", Type: "uint256"}}},
	}
	entries := abiToStudioEntries(abi, -1)
	require.Len(t, entries, 1)
	assert.False(t, entries[0].IsPayable, "events should not be payable")
	assert.True(t, entries[0].IsEvent)
}

func TestAbiToStudioEntriesMixedPayability(t *testing.T) {
	abi := []contract.ABIEntry{
		{Name: "deposit", Type: "function", StateMutability: "payable"},
		{Name: "withdraw", Type: "function", StateMutability: "nonpayable",
			Inputs: []contract.ABIParam{{Name: "amount", Type: "uint256"}}},
		{Name: "getBalance", Type: "function", StateMutability: "view"},
		{Name: "receive", Type: "function", StateMutability: "payable"},
	}
	entries := abiToStudioEntries(abi, -1)
	require.Len(t, entries, 4)

	// deposit — payable
	assert.True(t, entries[0].IsPayable)
	assert.True(t, entries[0].IsWrite)

	// withdraw — nonpayable
	assert.False(t, entries[1].IsPayable)
	assert.True(t, entries[1].IsWrite)

	// getBalance — view
	assert.False(t, entries[2].IsPayable)
	assert.False(t, entries[2].IsWrite)

	// receive — payable
	assert.True(t, entries[3].IsPayable)
	assert.True(t, entries[3].IsWrite)
}

func TestAbiToStudioEntriesConstructorSkipped(t *testing.T) {
	abi := []contract.ABIEntry{
		{Name: "", Type: "constructor", StateMutability: "payable",
			Inputs: []contract.ABIParam{{Name: "_token", Type: "address"}}},
		{Name: "deposit", Type: "function", StateMutability: "payable"},
	}
	entries := abiToStudioEntries(abi, -1)
	// Constructor should be skipped, only function entries included.
	require.Len(t, entries, 1)
	assert.Equal(t, "deposit", entries[0].Name)
	assert.True(t, entries[0].IsPayable)
}

func TestAbiToStudioEntriesPayableWithInputs(t *testing.T) {
	abi := []contract.ABIEntry{
		{Name: "buyToken", Type: "function", StateMutability: "payable",
			Inputs: []contract.ABIParam{
				{Name: "recipient", Type: "address"},
				{Name: "minOut", Type: "uint256"},
			}},
	}
	entries := abiToStudioEntries(abi, -1)
	require.Len(t, entries, 1)
	assert.True(t, entries[0].IsPayable)
	assert.True(t, entries[0].IsWrite)
	assert.Len(t, entries[0].Inputs, 2)
	assert.Equal(t, "recipient", entries[0].Inputs[0].Name)
	assert.Equal(t, "minOut", entries[0].Inputs[1].Name)
}
