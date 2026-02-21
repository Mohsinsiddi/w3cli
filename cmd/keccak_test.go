package cmd

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestKeccak_TransferSignature(t *testing.T) {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte("transfer(address,uint256)"))
	hash := hex.EncodeToString(h.Sum(nil))
	selector := hash[:8]
	assert.Equal(t, "a9059cbb", selector)
}

func TestKeccak_ApproveSignature(t *testing.T) {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte("approve(address,uint256)"))
	hash := hex.EncodeToString(h.Sum(nil))
	selector := hash[:8]
	assert.Equal(t, "095ea7b3", selector)
}

func TestKeccak_BalanceOfSignature(t *testing.T) {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte("balanceOf(address)"))
	hash := hex.EncodeToString(h.Sum(nil))
	selector := hash[:8]
	assert.Equal(t, "70a08231", selector)
}

func TestKeccak_EmptyString(t *testing.T) {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(""))
	hash := hex.EncodeToString(h.Sum(nil))
	// Known keccak of empty string.
	assert.Equal(t, "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470", hash)
}

func TestKeccak_HexInput(t *testing.T) {
	// Hash of raw 0xdeadbeef bytes.
	data, err := hex.DecodeString("deadbeef")
	require.NoError(t, err)
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	hash := hex.EncodeToString(h.Sum(nil))
	assert.Len(t, hash, 64)
}

func TestKeccak_Deterministic(t *testing.T) {
	h1 := sha3.NewLegacyKeccak256()
	h1.Write([]byte("test"))
	hash1 := hex.EncodeToString(h1.Sum(nil))

	h2 := sha3.NewLegacyKeccak256()
	h2.Write([]byte("test"))
	hash2 := hex.EncodeToString(h2.Sum(nil))

	assert.Equal(t, hash1, hash2)
}

func TestKeccak_DifferentInputs(t *testing.T) {
	h1 := sha3.NewLegacyKeccak256()
	h1.Write([]byte("hello"))
	hash1 := hex.EncodeToString(h1.Sum(nil))

	h2 := sha3.NewLegacyKeccak256()
	h2.Write([]byte("world"))
	hash2 := hex.EncodeToString(h2.Sum(nil))

	assert.NotEqual(t, hash1, hash2)
}
