package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToChecksumAddress_VitalikAddress(t *testing.T) {
	// Known EIP-55 checksum for vitalik's address.
	addr := "d8da6bf26964af9d7eed9e03e53415d37aa96045"
	result := toChecksumAddress(addr)
	assert.Equal(t, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", result)
}

func TestToChecksumAddress_AllLowercase(t *testing.T) {
	addr := "0000000000000000000000000000000000000001"
	result := toChecksumAddress(addr)
	assert.Equal(t, "0x0000000000000000000000000000000000000001", result)
}

func TestToChecksumAddress_USDC(t *testing.T) {
	addr := "a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	result := toChecksumAddress(addr)
	assert.Equal(t, "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", result)
}

func TestToChecksumAddress_Deterministic(t *testing.T) {
	addr := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	r1 := toChecksumAddress(addr)
	r2 := toChecksumAddress(addr)
	assert.Equal(t, r1, r2)
}

func TestToChecksumAddress_ZeroAddress(t *testing.T) {
	addr := "0000000000000000000000000000000000000000"
	result := toChecksumAddress(addr)
	assert.Equal(t, "0x0000000000000000000000000000000000000000", result)
}

func TestToChecksumAddress_AllFs(t *testing.T) {
	addr := "ffffffffffffffffffffffffffffffffffffffff"
	result := toChecksumAddress(addr)
	// Should have mixed case based on keccak hash.
	assert.True(t, len(result) == 42, "result should be 42 chars (0x + 40)")
	assert.Equal(t, "0x", result[:2])
}
