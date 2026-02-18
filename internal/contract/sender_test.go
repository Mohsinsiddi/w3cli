package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHexToBytesWithPrefix(t *testing.T) {
	result := hexToBytes("0xabcdef")
	assert.Equal(t, []byte{0xab, 0xcd, 0xef}, result)
}

func TestHexToBytesWithoutPrefix(t *testing.T) {
	result := hexToBytes("abcdef")
	assert.Equal(t, []byte{0xab, 0xcd, 0xef}, result)
}

func TestHexToBytesOddLength(t *testing.T) {
	// Odd-length hex gets a leading zero prepended.
	result := hexToBytes("0xabc")
	assert.Equal(t, []byte{0x0a, 0xbc}, result)
}

func TestHexToBytesEmpty(t *testing.T) {
	result := hexToBytes("0x")
	assert.Empty(t, result)
}

func TestHexToBytesBareEmpty(t *testing.T) {
	result := hexToBytes("")
	assert.Empty(t, result)
}

func TestHexToBytesSingleByte(t *testing.T) {
	result := hexToBytes("0xff")
	assert.Equal(t, []byte{0xff}, result)
}

func TestHexToBytesLong(t *testing.T) {
	// 20-byte address.
	result := hexToBytes("0x1234567890abcdef1234567890abcdef12345678")
	assert.Len(t, result, 20)
	assert.Equal(t, byte(0x12), result[0])
	assert.Equal(t, byte(0x78), result[19])
}

func TestBytesToHex(t *testing.T) {
	result := bytesToHex([]byte{0xab, 0xcd, 0xef})
	assert.Equal(t, "abcdef", result)
}

func TestBytesToHexEmpty(t *testing.T) {
	result := bytesToHex([]byte{})
	assert.Equal(t, "", result)
}

func TestBytesToHexSingleByte(t *testing.T) {
	result := bytesToHex([]byte{0x0a})
	assert.Equal(t, "0a", result)
}

func TestBytesToHexZeroByte(t *testing.T) {
	result := bytesToHex([]byte{0x00})
	assert.Equal(t, "00", result)
}

func TestBytesToHexLeadingZero(t *testing.T) {
	result := bytesToHex([]byte{0x00, 0xab})
	assert.Equal(t, "00ab", result)
}

func TestHexToBytesRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"simple", []byte{0xab, 0xcd, 0xef}},
		{"all zeros", []byte{0x00, 0x00, 0x00}},
		{"all ff", []byte{0xff, 0xff, 0xff}},
		{"single byte", []byte{0x42}},
		{"20 bytes", []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hexStr := bytesToHex(tt.input)
			recovered := hexToBytes(hexStr)
			assert.Equal(t, tt.input, recovered)
		})
	}
}

func TestHexToBytesKnownCalldata(t *testing.T) {
	// Known ERC20 balanceOf calldata.
	calldata := "0x70a08231000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045"
	result := hexToBytes(calldata)

	// First 4 bytes should be the function selector.
	assert.Equal(t, byte(0x70), result[0])
	assert.Equal(t, byte(0xa0), result[1])
	assert.Equal(t, byte(0x82), result[2])
	assert.Equal(t, byte(0x31), result[3])

	// Total: 4 + 32 = 36 bytes.
	assert.Len(t, result, 36)
}

func TestHexToBytesAllHexDigits(t *testing.T) {
	result := hexToBytes("0123456789abcdef")
	expected := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	assert.Equal(t, expected, result)
}

func TestBytesToHexLongValue(t *testing.T) {
	// 32-byte value (like a uint256).
	input := make([]byte, 32)
	input[31] = 0x64 // 100

	result := bytesToHex(input)
	// Each byte is printed as 2 hex chars.
	assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000000064", result)
}
