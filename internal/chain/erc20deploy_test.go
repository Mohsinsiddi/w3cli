package chain

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// roundUp32
// ---------------------------------------------------------------------------

func TestRoundUp32Exact(t *testing.T) {
	assert.Equal(t, 32, roundUp32(32))
	assert.Equal(t, 64, roundUp32(64))
	assert.Equal(t, 0, roundUp32(0))
}

func TestRoundUp32Partial(t *testing.T) {
	assert.Equal(t, 32, roundUp32(1))
	assert.Equal(t, 32, roundUp32(31))
	assert.Equal(t, 64, roundUp32(33))
	assert.Equal(t, 64, roundUp32(63))
}

func TestRoundUp32LargeValue(t *testing.T) {
	assert.Equal(t, 128, roundUp32(100))
	assert.Equal(t, 256, roundUp32(225))
}

// ---------------------------------------------------------------------------
// appendUint256
// ---------------------------------------------------------------------------

func TestAppendUint256Zero(t *testing.T) {
	buf := appendUint256(nil, 0)
	require.Len(t, buf, 32)
	for _, b := range buf {
		assert.Equal(t, byte(0), b)
	}
}

func TestAppendUint256Small(t *testing.T) {
	buf := appendUint256(nil, 128)
	require.Len(t, buf, 32)
	assert.Equal(t, byte(0x80), buf[31])
	// leading bytes must be zero
	for i := 0; i < 31; i++ {
		assert.Equal(t, byte(0), buf[i])
	}
}

func TestAppendUint256BigEndian(t *testing.T) {
	val := uint64(0x0102030405060708)
	buf := appendUint256(nil, val)
	require.Len(t, buf, 32)
	// Last 8 bytes should be big-endian representation.
	var got uint64
	binary.BigEndian.PutUint64(buf[24:], binary.BigEndian.Uint64(buf[24:]))
	got = binary.BigEndian.Uint64(buf[24:])
	assert.Equal(t, val, got)
}

func TestAppendUint256Appends(t *testing.T) {
	base := []byte{0xFF}
	buf := appendUint256(base, 1)
	require.Len(t, buf, 33)
	assert.Equal(t, byte(0xFF), buf[0])
}

// ---------------------------------------------------------------------------
// appendBigInt
// ---------------------------------------------------------------------------

func TestAppendBigIntZero(t *testing.T) {
	buf := appendBigInt(nil, big.NewInt(0))
	require.Len(t, buf, 32)
	for _, b := range buf {
		assert.Equal(t, byte(0), b)
	}
}

func TestAppendBigIntOne(t *testing.T) {
	buf := appendBigInt(nil, big.NewInt(1))
	require.Len(t, buf, 32)
	assert.Equal(t, byte(1), buf[31])
}

func TestAppendBigIntLarge(t *testing.T) {
	// 1 ETH = 10^18
	one := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	buf := appendBigInt(nil, one)
	require.Len(t, buf, 32)
	// Reconstruct and compare.
	recovered := new(big.Int).SetBytes(buf)
	assert.Equal(t, 0, recovered.Cmp(one))
}

func TestAppendBigIntMaxUint256(t *testing.T) {
	// 2^256 - 1
	max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	buf := appendBigInt(nil, max)
	require.Len(t, buf, 32)
	for _, b := range buf {
		assert.Equal(t, byte(0xFF), b)
	}
}

// ---------------------------------------------------------------------------
// appendString
// ---------------------------------------------------------------------------

func TestAppendStringEmpty(t *testing.T) {
	buf := appendString(nil, []byte{})
	require.Len(t, buf, 32) // just the length word (0)
	for _, b := range buf {
		assert.Equal(t, byte(0), b)
	}
}

func TestAppendStringShort(t *testing.T) {
	data := []byte("hello") // 5 bytes
	buf := appendString(nil, data)
	// length word (32) + data padded to next 32 = 64 bytes total
	require.Len(t, buf, 64)
	// Length in the first word = 5
	assert.Equal(t, byte(5), buf[31])
	// Data starts at offset 32
	assert.Equal(t, data, buf[32:37])
	// Padding is zeros
	assert.Equal(t, make([]byte, 27), buf[37:64])
}

func TestAppendStringExactly32(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 32)
	buf := appendString(nil, data)
	// length word (32) + 32 data bytes = 64 bytes (no extra padding)
	require.Len(t, buf, 64)
	assert.Equal(t, data, buf[32:64])
}

func TestAppendStringOver32(t *testing.T) {
	data := bytes.Repeat([]byte("a"), 33)
	buf := appendString(nil, data)
	// length(32) + data(33) padded to 64 = 96 total
	require.Len(t, buf, 96)
}

// ---------------------------------------------------------------------------
// BuildERC20DeployData — structure validation
// ---------------------------------------------------------------------------

func TestBuildERC20DeployDataReturnsData(t *testing.T) {
	supply := new(big.Int).Mul(big.NewInt(1_000_000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	data, err := BuildERC20DeployData("MyToken", "MTK", 18, supply)
	require.NoError(t, err)
	assert.Greater(t, len(data), 1000, "should include bytecode + ABI args")
}

func TestBuildERC20DeployDataDecimals(t *testing.T) {
	supply := big.NewInt(1000)
	data, err := BuildERC20DeployData("T", "T", 6, supply)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestBuildERC20DeployDataZeroSupply(t *testing.T) {
	data, err := BuildERC20DeployData("Zero", "ZRO", 18, big.NewInt(0))
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestBuildERC20DeployDataLongName(t *testing.T) {
	longName := strings.Repeat("A", 100)
	data, err := BuildERC20DeployData(longName, "LONG", 18, big.NewInt(1))
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestBuildERC20DeployDataLargeSupply(t *testing.T) {
	// 1 billion tokens with 18 decimals
	supply := new(big.Int).Mul(
		big.NewInt(1_000_000_000),
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
	)
	data, err := BuildERC20DeployData("BigToken", "BIG", 18, supply)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestBuildERC20DeployDataConsistency(t *testing.T) {
	supply := big.NewInt(42)
	d1, err := BuildERC20DeployData("Foo", "FOO", 18, supply)
	require.NoError(t, err)
	d2, err := BuildERC20DeployData("Foo", "FOO", 18, supply)
	require.NoError(t, err)
	assert.Equal(t, d1, d2, "same inputs must produce same output")
}

func TestBuildERC20DeployDataDifferentNames(t *testing.T) {
	supply := big.NewInt(1000)
	d1, _ := BuildERC20DeployData("Alpha", "ALP", 18, supply)
	d2, _ := BuildERC20DeployData("Beta", "BET", 18, supply)
	assert.NotEqual(t, d1, d2, "different names should produce different calldata")
}

// ---------------------------------------------------------------------------
// W3TokenMintCalldata
// ---------------------------------------------------------------------------

func TestMintCalldataSelector(t *testing.T) {
	addr := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	amount := big.NewInt(1000)
	data := W3TokenMintCalldata(addr, amount)

	require.GreaterOrEqual(t, len(data), 4)
	// Selector for mint(address,uint256) = 0x40c10f19
	assert.Equal(t, []byte{0x40, 0xc1, 0x0f, 0x19}, data[:4])
}

func TestMintCalldataLength(t *testing.T) {
	data := W3TokenMintCalldata("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", big.NewInt(1))
	// 4 (selector) + 32 (address) + 32 (amount) = 68
	assert.Len(t, data, 68)
}

func TestMintCalldataAmountEncoded(t *testing.T) {
	amount := big.NewInt(255) // 0xFF
	data := W3TokenMintCalldata("0x0000000000000000000000000000000000000001", amount)
	// Amount is the last 32 bytes.
	amountBytes := data[36:68]
	recovered := new(big.Int).SetBytes(amountBytes)
	assert.Equal(t, amount.Int64(), recovered.Int64())
}

func TestMintCalldataLargeAmount(t *testing.T) {
	large := new(big.Int).Mul(big.NewInt(1_000_000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	data := W3TokenMintCalldata("0x1234567890123456789012345678901234567890", large)
	assert.Len(t, data, 68)
	recovered := new(big.Int).SetBytes(data[36:])
	assert.Equal(t, 0, recovered.Cmp(large))
}

func TestMintCalldataAddressWithoutPrefix(t *testing.T) {
	// Should handle address without 0x prefix gracefully (hex.DecodeString fails but no panic).
	data := W3TokenMintCalldata("1234567890abcdef1234567890abcdef12345678", big.NewInt(1))
	assert.Len(t, data, 68)
}

// ---------------------------------------------------------------------------
// W3TokenBurnCalldata
// ---------------------------------------------------------------------------

func TestBurnCalldataSelector(t *testing.T) {
	data := W3TokenBurnCalldata(big.NewInt(500))
	require.GreaterOrEqual(t, len(data), 4)
	// Selector for burn(uint256) = 0x42966c68
	assert.Equal(t, []byte{0x42, 0x96, 0x6c, 0x68}, data[:4])
}

func TestBurnCalldataLength(t *testing.T) {
	data := W3TokenBurnCalldata(big.NewInt(1))
	// 4 (selector) + 32 (amount) = 36
	assert.Len(t, data, 36)
}

func TestBurnCalldataAmountEncoded(t *testing.T) {
	amount := big.NewInt(1000)
	data := W3TokenBurnCalldata(amount)
	amountBytes := data[4:36]
	recovered := new(big.Int).SetBytes(amountBytes)
	assert.Equal(t, amount.Int64(), recovered.Int64())
}

func TestBurnCalldataZeroAmount(t *testing.T) {
	data := W3TokenBurnCalldata(big.NewInt(0))
	assert.Len(t, data, 36)
	for _, b := range data[4:] {
		assert.Equal(t, byte(0), b)
	}
}

func TestBurnCalldataLargeAmount(t *testing.T) {
	large := new(big.Int).Mul(big.NewInt(1_000_000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	data := W3TokenBurnCalldata(large)
	assert.Len(t, data, 36)
	recovered := new(big.Int).SetBytes(data[4:])
	assert.Equal(t, 0, recovered.Cmp(large))
}
