package contract

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// EncodeConstructorArgs — basic cases
// ---------------------------------------------------------------------------

func TestEncodeConstructorArgsNoParams(t *testing.T) {
	result, err := EncodeConstructorArgs(nil, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEncodeConstructorArgsEmptyParams(t *testing.T) {
	result, err := EncodeConstructorArgs([]ABIParam{}, []string{})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEncodeConstructorArgsMismatchCount(t *testing.T) {
	params := []ABIParam{{Name: "a", Type: "uint256"}, {Name: "b", Type: "uint256"}}
	_, err := EncodeConstructorArgs(params, []string{"100"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expects 2 args, got 1")
}

func TestEncodeConstructorArgsMismatchCountExtra(t *testing.T) {
	params := []ABIParam{{Name: "a", Type: "uint256"}}
	_, err := EncodeConstructorArgs(params, []string{"100", "200"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expects 1 args, got 2")
}

// ---------------------------------------------------------------------------
// EncodeConstructorArgs — unsupported types
// ---------------------------------------------------------------------------

func TestEncodeConstructorArgsArrayTypeRejected(t *testing.T) {
	params := []ABIParam{{Name: "ids", Type: "uint256[]"}}
	_, err := EncodeConstructorArgs(params, []string{"[1,2,3]"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "array and tuple types are not yet supported")
}

func TestEncodeConstructorArgsFixedArrayRejected(t *testing.T) {
	params := []ABIParam{{Name: "ids", Type: "uint256[3]"}}
	_, err := EncodeConstructorArgs(params, []string{"[1,2,3]"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "array and tuple types are not yet supported")
}

func TestEncodeConstructorArgsTupleTypeRejected(t *testing.T) {
	params := []ABIParam{{Name: "data", Type: "tuple"}}
	_, err := EncodeConstructorArgs(params, []string{"(1,2)"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "array and tuple types are not yet supported")
}

// ---------------------------------------------------------------------------
// EncodeConstructorArgs — single static params
// ---------------------------------------------------------------------------

func TestEncodeConstructorArgsSingleAddress(t *testing.T) {
	params := []ABIParam{{Name: "owner", Type: "address"}}
	result, err := EncodeConstructorArgs(params, []string{"0xd8da6bf26964af9d7eed9e03e53415d37aa96045"})
	require.NoError(t, err)
	assert.Len(t, result, 32)

	// Last 20 bytes should contain the address.
	addrHex := hex.EncodeToString(result[12:32])
	assert.Equal(t, "d8da6bf26964af9d7eed9e03e53415d37aa96045", addrHex)
}

func TestEncodeConstructorArgsSingleUint256(t *testing.T) {
	params := []ABIParam{{Name: "supply", Type: "uint256"}}
	result, err := EncodeConstructorArgs(params, []string{"1000000"})
	require.NoError(t, err)
	assert.Len(t, result, 32)

	n := new(big.Int).SetBytes(result)
	assert.Equal(t, "1000000", n.String())
}

func TestEncodeConstructorArgsSingleBoolTrue(t *testing.T) {
	params := []ABIParam{{Name: "active", Type: "bool"}}
	result, err := EncodeConstructorArgs(params, []string{"true"})
	require.NoError(t, err)
	assert.Len(t, result, 32)
	assert.Equal(t, byte(1), result[31])
}

func TestEncodeConstructorArgsSingleBoolFalse(t *testing.T) {
	params := []ABIParam{{Name: "active", Type: "bool"}}
	result, err := EncodeConstructorArgs(params, []string{"false"})
	require.NoError(t, err)
	assert.Len(t, result, 32)
	assert.Equal(t, byte(0), result[31])
}

func TestEncodeConstructorArgsSingleBytes32(t *testing.T) {
	params := []ABIParam{{Name: "root", Type: "bytes32"}}
	hexVal := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	result, err := EncodeConstructorArgs(params, []string{"0x" + hexVal})
	require.NoError(t, err)
	assert.Len(t, result, 32)
	assert.Equal(t, hexVal, hex.EncodeToString(result))
}

// ---------------------------------------------------------------------------
// EncodeConstructorArgs — multiple static params
// ---------------------------------------------------------------------------

func TestEncodeConstructorArgsMultipleStatic(t *testing.T) {
	params := []ABIParam{
		{Name: "owner", Type: "address"},
		{Name: "supply", Type: "uint256"},
		{Name: "active", Type: "bool"},
	}
	result, err := EncodeConstructorArgs(params, []string{
		"0xd8da6bf26964af9d7eed9e03e53415d37aa96045",
		"1000",
		"true",
	})
	require.NoError(t, err)
	assert.Len(t, result, 96) // 3 * 32 bytes

	// Verify address (first 32 bytes).
	addrHex := hex.EncodeToString(result[12:32])
	assert.Equal(t, "d8da6bf26964af9d7eed9e03e53415d37aa96045", addrHex)

	// Verify uint256 (bytes 32-63).
	supply := new(big.Int).SetBytes(result[32:64])
	assert.Equal(t, "1000", supply.String())

	// Verify bool (byte 95).
	assert.Equal(t, byte(1), result[95])
}

func TestEncodeConstructorArgsTwoAddresses(t *testing.T) {
	params := []ABIParam{
		{Name: "owner", Type: "address"},
		{Name: "treasury", Type: "address"},
	}
	result, err := EncodeConstructorArgs(params, []string{
		"0xd8da6bf26964af9d7eed9e03e53415d37aa96045",
		"0x1234567890abcdef1234567890abcdef12345678",
	})
	require.NoError(t, err)
	assert.Len(t, result, 64)
}

// ---------------------------------------------------------------------------
// EncodeConstructorArgs — dynamic types
// ---------------------------------------------------------------------------

func TestEncodeConstructorArgsSingleString(t *testing.T) {
	params := []ABIParam{{Name: "name", Type: "string"}}
	result, err := EncodeConstructorArgs(params, []string{"MyToken"})
	require.NoError(t, err)

	// Head: 32 bytes (offset to string data).
	// Tail: 32 bytes (length) + 32 bytes (padded data for "MyToken" = 7 bytes).
	// Total: 32 + 32 + 32 = 96 bytes.
	assert.Len(t, result, 96)

	// The head should contain offset = 32 (pointing past the head).
	offset := new(big.Int).SetBytes(result[0:32])
	assert.Equal(t, int64(32), offset.Int64())

	// The length at the offset should be 7.
	length := new(big.Int).SetBytes(result[32:64])
	assert.Equal(t, int64(7), length.Int64())

	// The actual string data.
	assert.Equal(t, "MyToken", string(result[64:71]))
}

func TestEncodeConstructorArgsSingleStringEmpty(t *testing.T) {
	params := []ABIParam{{Name: "name", Type: "string"}}
	result, err := EncodeConstructorArgs(params, []string{""})
	require.NoError(t, err)

	// Head: 32 bytes (offset). Tail: 32 bytes (length=0).
	assert.Len(t, result, 64)

	length := new(big.Int).SetBytes(result[32:64])
	assert.Equal(t, int64(0), length.Int64())
}

func TestEncodeConstructorArgsSingleBytes(t *testing.T) {
	params := []ABIParam{{Name: "data", Type: "bytes"}}
	result, err := EncodeConstructorArgs(params, []string{"0xdeadbeef"})
	require.NoError(t, err)

	// Head: 32 bytes (offset). Tail: 32 bytes (length=4) + 32 bytes (padded data).
	assert.Len(t, result, 96)

	length := new(big.Int).SetBytes(result[32:64])
	assert.Equal(t, int64(4), length.Int64())

	assert.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, result[64:68])
}

// ---------------------------------------------------------------------------
// EncodeConstructorArgs — mixed static + dynamic
// ---------------------------------------------------------------------------

func TestEncodeConstructorArgsMixedStaticDynamic(t *testing.T) {
	// constructor(string name, string symbol, uint8 decimals, uint256 initialSupply)
	params := []ABIParam{
		{Name: "name", Type: "string"},
		{Name: "symbol", Type: "string"},
		{Name: "decimals", Type: "uint8"},
		{Name: "initialSupply", Type: "uint256"},
	}
	result, err := EncodeConstructorArgs(params, []string{
		"MyToken",
		"MTK",
		"18",
		"1000000",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Head section: 4 params * 32 bytes = 128 bytes.
	// First param (string): head contains offset.
	// Second param (string): head contains offset.
	// Third param (uint8): head contains value 18.
	// Fourth param (uint256): head contains value 1000000.

	// Verify decimals at offset 64 (3rd word, 0-indexed param 2).
	decimals := new(big.Int).SetBytes(result[64:96])
	assert.Equal(t, int64(18), decimals.Int64())

	// Verify initialSupply at offset 96 (4th word).
	supply := new(big.Int).SetBytes(result[96:128])
	assert.Equal(t, "1000000", supply.String())
}

func TestEncodeConstructorArgsAddressAndString(t *testing.T) {
	params := []ABIParam{
		{Name: "owner", Type: "address"},
		{Name: "name", Type: "string"},
	}
	result, err := EncodeConstructorArgs(params, []string{
		"0xd8da6bf26964af9d7eed9e03e53415d37aa96045",
		"Vitalik",
	})
	require.NoError(t, err)

	// Head: 2 * 32 = 64 bytes.
	// Param 0 (address): static value in head.
	// Param 1 (string): offset in head, data in tail.

	// Verify address.
	addrHex := hex.EncodeToString(result[12:32])
	assert.Equal(t, "d8da6bf26964af9d7eed9e03e53415d37aa96045", addrHex)

	// Verify offset points past the head.
	offset := new(big.Int).SetBytes(result[32:64])
	assert.Equal(t, int64(64), offset.Int64()) // 2 params * 32 bytes

	// Verify string data.
	length := new(big.Int).SetBytes(result[64:96])
	assert.Equal(t, int64(7), length.Int64())
	assert.Equal(t, "Vitalik", string(result[96:103]))
}

// ---------------------------------------------------------------------------
// EncodeConstructorArgs — error cases for param encoding
// ---------------------------------------------------------------------------

func TestEncodeConstructorArgsInvalidAddress(t *testing.T) {
	params := []ABIParam{{Name: "owner", Type: "address"}}
	_, err := EncodeConstructorArgs(params, []string{"not-an-address"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address")
}

func TestEncodeConstructorArgsInvalidUint(t *testing.T) {
	params := []ABIParam{{Name: "amount", Type: "uint256"}}
	_, err := EncodeConstructorArgs(params, []string{"not-a-number"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid integer")
}

func TestEncodeConstructorArgsInvalidBool(t *testing.T) {
	params := []ABIParam{{Name: "active", Type: "bool"}}
	_, err := EncodeConstructorArgs(params, []string{"maybe"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bool")
}

func TestEncodeConstructorArgsInvalidBytesHex(t *testing.T) {
	params := []ABIParam{{Name: "data", Type: "bytes"}}
	_, err := EncodeConstructorArgs(params, []string{"0xGGHH"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bytes hex")
}

// ---------------------------------------------------------------------------
// isDynamicType
// ---------------------------------------------------------------------------

func TestIsDynamicTypeString(t *testing.T) {
	assert.True(t, isDynamicType("string"))
}

func TestIsDynamicTypeBytes(t *testing.T) {
	assert.True(t, isDynamicType("bytes"))
}

func TestIsDynamicTypeAddress(t *testing.T) {
	assert.False(t, isDynamicType("address"))
}

func TestIsDynamicTypeUint256(t *testing.T) {
	assert.False(t, isDynamicType("uint256"))
}

func TestIsDynamicTypeBool(t *testing.T) {
	assert.False(t, isDynamicType("bool"))
}

func TestIsDynamicTypeBytes32(t *testing.T) {
	assert.False(t, isDynamicType("bytes32"))
}

func TestIsDynamicTypeInt256(t *testing.T) {
	assert.False(t, isDynamicType("int256"))
}

// ---------------------------------------------------------------------------
// encodeStaticParam
// ---------------------------------------------------------------------------

func TestEncodeStaticParamAddress(t *testing.T) {
	result, err := encodeStaticParam("address", "0xd8da6bf26964af9d7eed9e03e53415d37aa96045")
	require.NoError(t, err)
	assert.Len(t, result, 32)
	// First 12 bytes should be zero.
	for i := 0; i < 12; i++ {
		assert.Equal(t, byte(0), result[i])
	}
	assert.Equal(t, "d8da6bf26964af9d7eed9e03e53415d37aa96045", hex.EncodeToString(result[12:]))
}

func TestEncodeStaticParamAddressInvalidLength(t *testing.T) {
	_, err := encodeStaticParam("address", "0xaabb")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address")
}

func TestEncodeStaticParamAddressInvalidHex(t *testing.T) {
	_, err := encodeStaticParam("address", "0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address")
}

func TestEncodeStaticParamUint256Zero(t *testing.T) {
	result, err := encodeStaticParam("uint256", "0")
	require.NoError(t, err)
	assert.Len(t, result, 32)
	assert.Equal(t, make([]byte, 32), result)
}

func TestEncodeStaticParamUint256Large(t *testing.T) {
	result, err := encodeStaticParam("uint256", "1000000000000000000")
	require.NoError(t, err)
	assert.Len(t, result, 32)
	n := new(big.Int).SetBytes(result)
	assert.Equal(t, "1000000000000000000", n.String())
}

func TestEncodeStaticParamUint8(t *testing.T) {
	result, err := encodeStaticParam("uint8", "18")
	require.NoError(t, err)
	assert.Len(t, result, 32)
	assert.Equal(t, byte(18), result[31])
}

func TestEncodeStaticParamInt256Positive(t *testing.T) {
	result, err := encodeStaticParam("int256", "42")
	require.NoError(t, err)
	assert.Len(t, result, 32)
	n := new(big.Int).SetBytes(result)
	assert.Equal(t, "42", n.String())
}

func TestEncodeStaticParamInt256Negative(t *testing.T) {
	result, err := encodeStaticParam("int256", "-1")
	require.NoError(t, err)
	assert.Len(t, result, 32)
	// Two's complement: -1 = all 0xff bytes.
	for _, b := range result {
		assert.Equal(t, byte(0xff), b)
	}
}

func TestEncodeStaticParamBoolTrue(t *testing.T) {
	result, err := encodeStaticParam("bool", "true")
	require.NoError(t, err)
	assert.Equal(t, byte(1), result[31])
}

func TestEncodeStaticParamBoolFalse(t *testing.T) {
	result, err := encodeStaticParam("bool", "false")
	require.NoError(t, err)
	assert.Equal(t, byte(0), result[31])
}

func TestEncodeStaticParamBool1(t *testing.T) {
	result, err := encodeStaticParam("bool", "1")
	require.NoError(t, err)
	assert.Equal(t, byte(1), result[31])
}

func TestEncodeStaticParamBool0(t *testing.T) {
	result, err := encodeStaticParam("bool", "0")
	require.NoError(t, err)
	assert.Equal(t, byte(0), result[31])
}

func TestEncodeStaticParamBoolInvalid(t *testing.T) {
	_, err := encodeStaticParam("bool", "maybe")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bool")
}

func TestEncodeStaticParamBytes32(t *testing.T) {
	hexVal := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	result, err := encodeStaticParam("bytes32", "0x"+hexVal)
	require.NoError(t, err)
	assert.Len(t, result, 32)
	assert.Equal(t, hexVal, hex.EncodeToString(result))
}

func TestEncodeStaticParamBytes32Short(t *testing.T) {
	result, err := encodeStaticParam("bytes32", "0xdeadbeef")
	require.NoError(t, err)
	assert.Len(t, result, 32)
	// Should be right-padded with zeros.
	assert.Equal(t, "deadbeef", hex.EncodeToString(result[0:4]))
	for i := 4; i < 32; i++ {
		assert.Equal(t, byte(0), result[i])
	}
}

func TestEncodeStaticParamBytes32InvalidHex(t *testing.T) {
	_, err := encodeStaticParam("bytes32", "0xZZZZ")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bytes32")
}

func TestEncodeStaticParamUnsupportedType(t *testing.T) {
	_, err := encodeStaticParam("string", "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported static type")
}

func TestEncodeStaticParamInvalidInteger(t *testing.T) {
	_, err := encodeStaticParam("uint256", "not-a-number")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid integer")
}

// ---------------------------------------------------------------------------
// encodeDynamicParam
// ---------------------------------------------------------------------------

func TestEncodeDynamicParamString(t *testing.T) {
	result, err := encodeDynamicParam("string", "Hello")
	require.NoError(t, err)
	// 32 bytes length + 32 bytes padded data.
	assert.Len(t, result, 64)
	length := new(big.Int).SetBytes(result[0:32])
	assert.Equal(t, int64(5), length.Int64())
	assert.Equal(t, "Hello", string(result[32:37]))
}

func TestEncodeDynamicParamStringEmpty(t *testing.T) {
	result, err := encodeDynamicParam("string", "")
	require.NoError(t, err)
	assert.Len(t, result, 32) // Just the length prefix (0).
	length := new(big.Int).SetBytes(result[0:32])
	assert.Equal(t, int64(0), length.Int64())
}

func TestEncodeDynamicParamStringExactly32Bytes(t *testing.T) {
	s := "abcdefghijklmnopqrstuvwxyz012345" // 32 chars
	result, err := encodeDynamicParam("string", s)
	require.NoError(t, err)
	// 32 bytes length + 32 bytes data (exactly fits).
	assert.Len(t, result, 64)
	length := new(big.Int).SetBytes(result[0:32])
	assert.Equal(t, int64(32), length.Int64())
}

func TestEncodeDynamicParamStringOver32Bytes(t *testing.T) {
	s := "abcdefghijklmnopqrstuvwxyz0123456" // 33 chars
	result, err := encodeDynamicParam("string", s)
	require.NoError(t, err)
	// 32 bytes length + 64 bytes padded data (33 bytes rounds up to 64).
	assert.Len(t, result, 96)
	length := new(big.Int).SetBytes(result[0:32])
	assert.Equal(t, int64(33), length.Int64())
}

func TestEncodeDynamicParamBytes(t *testing.T) {
	result, err := encodeDynamicParam("bytes", "0xdeadbeef")
	require.NoError(t, err)
	assert.Len(t, result, 64)
	length := new(big.Int).SetBytes(result[0:32])
	assert.Equal(t, int64(4), length.Int64())
	assert.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, result[32:36])
}

func TestEncodeDynamicParamBytesNoPrefix(t *testing.T) {
	result, err := encodeDynamicParam("bytes", "aabb")
	require.NoError(t, err)
	length := new(big.Int).SetBytes(result[0:32])
	assert.Equal(t, int64(2), length.Int64())
}

func TestEncodeDynamicParamBytesInvalidHex(t *testing.T) {
	_, err := encodeDynamicParam("bytes", "0xGGHH")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bytes hex")
}

func TestEncodeDynamicParamUnsupportedType(t *testing.T) {
	_, err := encodeDynamicParam("address", "0x1234")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported dynamic type")
}

// ---------------------------------------------------------------------------
// encodeBytesData
// ---------------------------------------------------------------------------

func TestEncodeBytesDataShort(t *testing.T) {
	result := encodeBytesData([]byte{0xaa, 0xbb})
	// 32 bytes (length = 2) + 32 bytes (padded data).
	assert.Len(t, result, 64)
	length := new(big.Int).SetBytes(result[0:32])
	assert.Equal(t, int64(2), length.Int64())
	assert.Equal(t, byte(0xaa), result[32])
	assert.Equal(t, byte(0xbb), result[33])
	// Rest should be zero-padded.
	for i := 34; i < 64; i++ {
		assert.Equal(t, byte(0), result[i])
	}
}

func TestEncodeBytesDataEmpty(t *testing.T) {
	result := encodeBytesData([]byte{})
	assert.Len(t, result, 32) // Just length prefix.
	length := new(big.Int).SetBytes(result[0:32])
	assert.Equal(t, int64(0), length.Int64())
}

func TestEncodeBytesDataExactly32(t *testing.T) {
	data := make([]byte, 32)
	for i := range data {
		data[i] = byte(i)
	}
	result := encodeBytesData(data)
	assert.Len(t, result, 64) // 32 length + 32 data
}

func TestEncodeBytesData33Bytes(t *testing.T) {
	data := make([]byte, 33)
	result := encodeBytesData(data)
	assert.Len(t, result, 96) // 32 length + 64 padded data
}

// ---------------------------------------------------------------------------
// appendUint256Big
// ---------------------------------------------------------------------------

func TestAppendUint256BigZero(t *testing.T) {
	result := appendUint256Big(nil, big.NewInt(0))
	assert.Len(t, result, 32)
	assert.Equal(t, make([]byte, 32), result)
}

func TestAppendUint256BigSmall(t *testing.T) {
	result := appendUint256Big(nil, big.NewInt(42))
	assert.Len(t, result, 32)
	assert.Equal(t, byte(42), result[31])
}

func TestAppendUint256BigLarge(t *testing.T) {
	n, _ := new(big.Int).SetString("1000000000000000000", 10) // 1e18
	result := appendUint256Big(nil, n)
	assert.Len(t, result, 32)
	recovered := new(big.Int).SetBytes(result)
	assert.Equal(t, "1000000000000000000", recovered.String())
}

func TestAppendUint256BigAppends(t *testing.T) {
	buf := make([]byte, 10)
	result := appendUint256Big(buf, big.NewInt(1))
	assert.Len(t, result, 42) // 10 + 32
}

// ---------------------------------------------------------------------------
// padInt256
// ---------------------------------------------------------------------------

func TestPadInt256Zero(t *testing.T) {
	result := padInt256(big.NewInt(0))
	assert.Len(t, result, 32)
	assert.Equal(t, make([]byte, 32), result)
}

func TestPadInt256Positive(t *testing.T) {
	result := padInt256(big.NewInt(100))
	assert.Len(t, result, 32)
	assert.Equal(t, byte(100), result[31])
}

func TestPadInt256NegativeOne(t *testing.T) {
	result := padInt256(big.NewInt(-1))
	assert.Len(t, result, 32)
	// Two's complement of -1 is all 0xff.
	for _, b := range result {
		assert.Equal(t, byte(0xff), b)
	}
}

func TestPadInt256NegativeLarge(t *testing.T) {
	result := padInt256(big.NewInt(-100))
	assert.Len(t, result, 32)
	// Verify: 2^256 - 100
	n := new(big.Int).SetBytes(result)
	mod := new(big.Int).Lsh(big.NewInt(1), 256)
	expected := new(big.Int).Sub(mod, big.NewInt(100))
	assert.Equal(t, expected.String(), n.String())
}

func TestPadInt256MaxPositive(t *testing.T) {
	// 2^255 - 1 (max int256)
	maxInt256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1))
	result := padInt256(maxInt256)
	assert.Len(t, result, 32)
	// First byte should be 0x7f.
	assert.Equal(t, byte(0x7f), result[0])
	for i := 1; i < 32; i++ {
		assert.Equal(t, byte(0xff), result[i])
	}
}

// ---------------------------------------------------------------------------
// roundUp32Bytes
// ---------------------------------------------------------------------------

func TestRoundUp32BytesZero(t *testing.T) {
	assert.Equal(t, 0, roundUp32Bytes(0))
}

func TestRoundUp32BytesExact(t *testing.T) {
	assert.Equal(t, 32, roundUp32Bytes(32))
	assert.Equal(t, 64, roundUp32Bytes(64))
	assert.Equal(t, 96, roundUp32Bytes(96))
}

func TestRoundUp32BytesRoundsUp(t *testing.T) {
	assert.Equal(t, 32, roundUp32Bytes(1))
	assert.Equal(t, 32, roundUp32Bytes(7))
	assert.Equal(t, 32, roundUp32Bytes(31))
	assert.Equal(t, 64, roundUp32Bytes(33))
	assert.Equal(t, 64, roundUp32Bytes(63))
	assert.Equal(t, 96, roundUp32Bytes(65))
}

// ---------------------------------------------------------------------------
// EncodeConstructorArgs — round-trip / integration
// ---------------------------------------------------------------------------

func TestEncodeConstructorArgsERC20Pattern(t *testing.T) {
	// Typical ERC20: constructor(string name, string symbol, uint8 decimals, uint256 totalSupply)
	params := []ABIParam{
		{Name: "name", Type: "string"},
		{Name: "symbol", Type: "string"},
		{Name: "decimals", Type: "uint8"},
		{Name: "totalSupply", Type: "uint256"},
	}

	result, err := EncodeConstructorArgs(params, []string{
		"MyToken",
		"MTK",
		"18",
		"1000000000000000000000000",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Head is 4 * 32 = 128 bytes.
	require.True(t, len(result) >= 128)

	// Word 2 (decimals) should be 18.
	decimals := new(big.Int).SetBytes(result[64:96])
	assert.Equal(t, int64(18), decimals.Int64())

	// Word 3 (totalSupply) should be 1e24.
	supply := new(big.Int).SetBytes(result[96:128])
	expected, _ := new(big.Int).SetString("1000000000000000000000000", 10)
	assert.Equal(t, expected.String(), supply.String())
}

func TestEncodeConstructorArgsBoolVariants(t *testing.T) {
	params := []ABIParam{{Name: "flag", Type: "bool"}}

	tests := []struct {
		input    string
		expected byte
	}{
		{"true", 1},
		{"false", 0},
		{"1", 1},
		{"0", 0},
		{"TRUE", 1},
		{"FALSE", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := EncodeConstructorArgs(params, []string{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result[31])
		})
	}
}

func TestEncodeConstructorArgsUintVariants(t *testing.T) {
	types := []string{"uint8", "uint16", "uint32", "uint64", "uint128", "uint256"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			params := []ABIParam{{Name: "val", Type: typ}}
			result, err := EncodeConstructorArgs(params, []string{"255"})
			require.NoError(t, err)
			assert.Len(t, result, 32)
			n := new(big.Int).SetBytes(result)
			assert.Equal(t, "255", n.String())
		})
	}
}

func TestEncodeConstructorArgsIntVariants(t *testing.T) {
	types := []string{"int8", "int16", "int32", "int64", "int128", "int256"}
	for _, typ := range types {
		t.Run(typ+"_positive", func(t *testing.T) {
			params := []ABIParam{{Name: "val", Type: typ}}
			result, err := EncodeConstructorArgs(params, []string{"42"})
			require.NoError(t, err)
			assert.Len(t, result, 32)
			n := new(big.Int).SetBytes(result)
			assert.Equal(t, "42", n.String())
		})
	}
}
