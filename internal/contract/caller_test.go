package contract

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunctionSelector(t *testing.T) {
	tests := []struct {
		name     string
		fn       ABIEntry
		expected string
	}{
		{
			"balanceOf(address)",
			ABIEntry{Name: "balanceOf", Inputs: []ABIParam{{Type: "address"}}},
			"0x70a08231",
		},
		{
			"transfer(address,uint256)",
			ABIEntry{Name: "transfer", Inputs: []ABIParam{{Type: "address"}, {Type: "uint256"}}},
			"0xa9059cbb",
		},
		{
			"name()",
			ABIEntry{Name: "name", Inputs: []ABIParam{}},
			"0x06fdde03",
		},
		{
			"symbol()",
			ABIEntry{Name: "symbol", Inputs: []ABIParam{}},
			"0x95d89b41",
		},
		{
			"decimals()",
			ABIEntry{Name: "decimals", Inputs: nil},
			"0x313ce567",
		},
		{
			"totalSupply()",
			ABIEntry{Name: "totalSupply", Inputs: nil},
			"0x18160ddd",
		},
		{
			"approve(address,uint256)",
			ABIEntry{Name: "approve", Inputs: []ABIParam{{Type: "address"}, {Type: "uint256"}}},
			"0x095ea7b3",
		},
		{
			"allowance(address,address)",
			ABIEntry{Name: "allowance", Inputs: []ABIParam{{Type: "address"}, {Type: "address"}}},
			"0xdd62ed3e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := functionSelector(&tt.fn)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestEncodeParamAddress(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		expected string
	}{
		{
			"with 0x prefix",
			"0x1234567890abcdef1234567890abcdef12345678",
			"0000000000000000000000001234567890abcdef1234567890abcdef12345678",
		},
		{
			"without 0x prefix",
			"1234567890abcdef1234567890abcdef12345678",
			"0000000000000000000000001234567890abcdef1234567890abcdef12345678",
		},
		{
			"zero address",
			"0x0000000000000000000000000000000000000000",
			"0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"short address",
			"0x1",
			"0000000000000000000000000000000000000000000000000000000000000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encodeParam("address", tt.val)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
			assert.Len(t, result, 64)
		})
	}
}

func TestEncodeParamUint256(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		expected string
		wantErr  bool
	}{
		{
			"zero",
			"0",
			"0000000000000000000000000000000000000000000000000000000000000000",
			false,
		},
		{
			"small number",
			"100",
			"0000000000000000000000000000000000000000000000000000000000000064",
			false,
		},
		{
			"one",
			"1",
			"0000000000000000000000000000000000000000000000000000000000000001",
			false,
		},
		{
			"large number",
			"1000000000000000000",
			"0000000000000000000000000000000000000000000000000de0b6b3a7640000",
			false,
		},
		{
			"invalid non-numeric",
			"not-a-number",
			"",
			true,
		},
		{
			"invalid empty",
			"",
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encodeParam("uint256", tt.val)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
				assert.Len(t, result, 64)
			}
		})
	}
}

func TestEncodeParamIntTypes(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		val  string
	}{
		{"uint8", "uint8", "255"},
		{"uint128", "uint128", "100"},
		{"int256", "int256", "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encodeParam(tt.typ, tt.val)
			require.NoError(t, err)
			assert.Len(t, result, 64)
		})
	}
}

func TestEncodeParamBool(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		expected string
	}{
		{"true", "true", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"1", "1", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"false", "false", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0", "0", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"empty", "", "0000000000000000000000000000000000000000000000000000000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encodeParam("bool", tt.val)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEncodeParamBytes32(t *testing.T) {
	result, err := encodeParam("bytes32", "abcdef")
	require.NoError(t, err)
	assert.Len(t, result, 64)
	assert.True(t, result[:6] == "abcdef")
}

func TestEncodeParamUnknownType(t *testing.T) {
	result, err := encodeParam("string", "hello")
	require.NoError(t, err)
	assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000000000", result)
}

func TestEncodeCallNoArgs(t *testing.T) {
	fn := &ABIEntry{
		Name:   "name",
		Type:   "function",
		Inputs: nil,
	}

	result, err := encodeCall(fn, nil)
	require.NoError(t, err)
	assert.Equal(t, "0x06fdde03", result)
}

func TestEncodeCallWithArgs(t *testing.T) {
	fn := &ABIEntry{
		Name:   "balanceOf",
		Type:   "function",
		Inputs: []ABIParam{{Name: "account", Type: "address"}},
	}

	result, err := encodeCall(fn, []string{"0x1234567890abcdef1234567890abcdef12345678"})
	require.NoError(t, err)
	assert.Equal(t, "0x70a08231", result[:10])
	assert.Equal(t, "0000000000000000000000001234567890abcdef1234567890abcdef12345678", result[10:])
}

func TestEncodeCallMultipleArgs(t *testing.T) {
	fn := &ABIEntry{
		Name:   "transfer",
		Type:   "function",
		Inputs: []ABIParam{{Name: "to", Type: "address"}, {Name: "amount", Type: "uint256"}},
	}

	result, err := encodeCall(fn, []string{"0x1234567890abcdef1234567890abcdef12345678", "1000"})
	require.NoError(t, err)

	assert.Equal(t, "0xa9059cbb", result[:10])
	// 4 byte selector + 2 * 64-char words = 10 + 128 = 138 chars
	assert.Len(t, result, 10+128)
}

func TestEncodeCallFewerArgsThanInputs(t *testing.T) {
	fn := &ABIEntry{
		Name:   "transfer",
		Type:   "function",
		Inputs: []ABIParam{{Name: "to", Type: "address"}, {Name: "amount", Type: "uint256"}},
	}

	// Only provide first arg; missing uint256 arg defaults to empty string which is invalid.
	_, err := encodeCall(fn, []string{"0x1234567890abcdef1234567890abcdef12345678"})
	assert.Error(t, err)
}

func TestEncodeCallInvalidArg(t *testing.T) {
	fn := &ABIEntry{
		Name:   "transfer",
		Type:   "function",
		Inputs: []ABIParam{{Name: "to", Type: "address"}, {Name: "amount", Type: "uint256"}},
	}

	_, err := encodeCall(fn, []string{"0xAddr", "not-a-number"})
	assert.Error(t, err)
}

func TestDecodeWordAddress(t *testing.T) {
	// 32-byte word with address in last 20 bytes.
	word := make([]byte, 32)
	addrBytes, _ := hex.DecodeString("1234567890abcdef1234567890abcdef12345678")
	copy(word[12:], addrBytes)

	result, err := decodeWord("address", word, nil)
	require.NoError(t, err)
	assert.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", result)
}

func TestDecodeWordAddressZero(t *testing.T) {
	word := make([]byte, 32)

	result, err := decodeWord("address", word, nil)
	require.NoError(t, err)
	assert.Equal(t, "0x0000000000000000000000000000000000000000", result)
}

func TestDecodeWordUint256(t *testing.T) {
	tests := []struct {
		name     string
		word     []byte
		expected string
	}{
		{
			"zero",
			make([]byte, 32),
			"0",
		},
		{
			"one",
			func() []byte {
				w := make([]byte, 32)
				w[31] = 1
				return w
			}(),
			"1",
		},
		{
			"100",
			func() []byte {
				w := make([]byte, 32)
				w[31] = 100
				return w
			}(),
			"100",
		},
		{
			"1 ether in wei",
			func() []byte {
				w := make([]byte, 32)
				// 1e18 = 0x0de0b6b3a7640000
				b, _ := hex.DecodeString("0de0b6b3a7640000")
				copy(w[24:], b)
				return w
			}(),
			"1000000000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeWord("uint256", tt.word, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDecodeWordBool(t *testing.T) {
	tests := []struct {
		name     string
		lastByte byte
		expected string
	}{
		{"true", 1, "true"},
		{"false", 0, "false"},
		{"non-one truthy", 2, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			word := make([]byte, 32)
			word[31] = tt.lastByte

			result, err := decodeWord("bool", word, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDecodeWordString(t *testing.T) {
	// Construct ABI-encoded string "Hello".
	// Word at offset 0: offset = 32 (0x20)
	// Data at offset 32: length = 5
	// Data at offset 64: "Hello" + padding
	fullData := make([]byte, 96)

	// Offset word: value = 32
	fullData[31] = 32

	// Length at position 32-63: value = 5
	fullData[63] = 5

	// String data at position 64
	copy(fullData[64:], []byte("Hello"))

	word := fullData[0:32]
	result, err := decodeWord("string", word, fullData)
	require.NoError(t, err)
	assert.Equal(t, "Hello", result)
}

func TestDecodeWordStringEmpty(t *testing.T) {
	fullData := make([]byte, 64)
	fullData[31] = 32 // offset = 32
	// length at position 32-63 = 0

	word := fullData[0:32]
	result, err := decodeWord("string", word, fullData)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestDecodeWordStringOffsetBeyondData(t *testing.T) {
	// Offset points beyond available data.
	word := make([]byte, 32)
	word[31] = 200 // offset = 200

	fullData := make([]byte, 32) // only 32 bytes available
	copy(fullData[0:32], word)

	result, err := decodeWord("string", word, fullData)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestDecodeWordStringLengthBeyondData(t *testing.T) {
	// Offset is valid but length exceeds available data.
	fullData := make([]byte, 64)
	fullData[31] = 32  // offset = 32
	fullData[63] = 100 // length = 100, but only 0 bytes of data after

	word := fullData[0:32]
	result, err := decodeWord("string", word, fullData)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestDecodeWordUnknownType(t *testing.T) {
	word := make([]byte, 32)
	word[0] = 0xab
	word[1] = 0xcd

	result, err := decodeWord("tuple", word, nil)
	require.NoError(t, err)
	assert.Equal(t, "0x"+hex.EncodeToString(word), result)
}

func TestDecodeResultNoOutputs(t *testing.T) {
	fn := &ABIEntry{
		Name:    "doSomething",
		Outputs: nil,
	}

	result, err := decodeResult(fn, "0x")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDecodeResultEmptyOutputs(t *testing.T) {
	fn := &ABIEntry{
		Name:    "doSomething",
		Outputs: []ABIParam{},
	}

	result, err := decodeResult(fn, "0x")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDecodeResultSingleUint256(t *testing.T) {
	fn := &ABIEntry{
		Name:    "balanceOf",
		Outputs: []ABIParam{{Name: "", Type: "uint256"}},
	}

	// Encode value 1000 as 32-byte hex.
	hexData := "0x00000000000000000000000000000000000000000000000000000000000003e8"

	result, err := decodeResult(fn, hexData)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "1000", result[0])
}

func TestDecodeResultMultipleOutputs(t *testing.T) {
	fn := &ABIEntry{
		Name: "getInfo",
		Outputs: []ABIParam{
			{Name: "amount", Type: "uint256"},
			{Name: "active", Type: "bool"},
		},
	}

	// 1000 (uint256) + true (bool)
	hexData := "0x" +
		"00000000000000000000000000000000000000000000000000000000000003e8" +
		"0000000000000000000000000000000000000000000000000000000000000001"

	result, err := decodeResult(fn, hexData)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "1000", result[0])
	assert.Equal(t, "true", result[1])
}

func TestDecodeResultTruncatedData(t *testing.T) {
	fn := &ABIEntry{
		Name: "getInfo",
		Outputs: []ABIParam{
			{Name: "a", Type: "uint256"},
			{Name: "b", Type: "uint256"},
		},
	}

	// Only one 32-byte word, but two outputs expected.
	hexData := "0x00000000000000000000000000000000000000000000000000000000000003e8"

	result, err := decodeResult(fn, hexData)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "1000", result[0])
	assert.Equal(t, "", result[1])
}

func TestDecodeResultInvalidHex(t *testing.T) {
	fn := &ABIEntry{
		Name:    "test",
		Outputs: []ABIParam{{Name: "", Type: "uint256"}},
	}

	_, err := decodeResult(fn, "0xNOTHEX")
	assert.Error(t, err)
}

func TestDecodeResultNoPrefix(t *testing.T) {
	fn := &ABIEntry{
		Name:    "balanceOf",
		Outputs: []ABIParam{{Name: "", Type: "uint256"}},
	}

	hexData := "00000000000000000000000000000000000000000000000000000000000003e8"

	result, err := decodeResult(fn, hexData)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "1000", result[0])
}

func TestCallerFindFunction(t *testing.T) {
	abi := []ABIEntry{
		{Name: "balanceOf", Type: "function", StateMutability: "view"},
		{Name: "transfer", Type: "function", StateMutability: "nonpayable"},
		{Name: "Transfer", Type: "event"},
	}

	caller := &Caller{abi: abi}

	assert.NotNil(t, caller.findFunction("balanceOf"))
	assert.NotNil(t, caller.findFunction("transfer"))
	assert.Nil(t, caller.findFunction("nonexistent"))
	// Events should not be found as functions.
	assert.Nil(t, caller.findFunction("Transfer"))
}

func TestCallerFindFunctionMatchesTypeFunction(t *testing.T) {
	// Same name as event, but only function type should match.
	abi := []ABIEntry{
		{Name: "Transfer", Type: "event"},
		{Name: "Transfer", Type: "function", StateMutability: "nonpayable"},
	}

	caller := &Caller{abi: abi}
	fn := caller.findFunction("Transfer")
	require.NotNil(t, fn)
	assert.Equal(t, "function", fn.Type)
}

func TestEncodeCallEmptyArgs(t *testing.T) {
	fn := &ABIEntry{
		Name:   "totalSupply",
		Type:   "function",
		Inputs: []ABIParam{},
	}

	result, err := encodeCall(fn, []string{})
	require.NoError(t, err)
	assert.Equal(t, "0x18160ddd", result)
}

func TestDecodeResultAddressOutput(t *testing.T) {
	fn := &ABIEntry{
		Name:    "owner",
		Outputs: []ABIParam{{Name: "", Type: "address"}},
	}

	hexData := "0x000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045"

	result, err := decodeResult(fn, hexData)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "0xd8da6bf26964af9d7eed9e03e53415d37aa96045", result[0])
}

func TestDecodeResultBoolOutput(t *testing.T) {
	fn := &ABIEntry{
		Name:    "isApproved",
		Outputs: []ABIParam{{Name: "", Type: "bool"}},
	}

	tests := []struct {
		name     string
		hexData  string
		expected string
	}{
		{
			"true",
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"true",
		},
		{
			"false",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeResult(fn, tt.hexData)
			require.NoError(t, err)
			require.Len(t, result, 1)
			assert.Equal(t, tt.expected, result[0])
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	// Encode a uint256 value, then decode it to verify consistency.
	enc, err := encodeParam("uint256", "42")
	require.NoError(t, err)

	data, err := hex.DecodeString(enc)
	require.NoError(t, err)

	result, err := decodeWord("uint256", data, nil)
	require.NoError(t, err)
	assert.Equal(t, "42", result)
}

func TestEncodeDecodeRoundTripAddress(t *testing.T) {
	addr := "d8da6bf26964af9d7eed9e03e53415d37aa96045"
	enc, err := encodeParam("address", addr)
	require.NoError(t, err)

	data, err := hex.DecodeString(enc)
	require.NoError(t, err)

	// decodeWord extracts the last 20 bytes (the address) from the 32-byte word.
	result, err := decodeWord("address", data, nil)
	require.NoError(t, err)
	assert.Equal(t, "0x"+addr, result)
}

func TestEncodeDecodeRoundTripBool(t *testing.T) {
	for _, val := range []string{"true", "false"} {
		enc, err := encodeParam("bool", val)
		require.NoError(t, err)

		data, err := hex.DecodeString(enc)
		require.NoError(t, err)

		result, err := decodeWord("bool", data, nil)
		require.NoError(t, err)
		assert.Equal(t, val, result)
	}
}
