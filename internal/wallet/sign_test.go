package wallet

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPrivKeyHex and testSignerAddr are defined in signer_test.go.

// ---------------------------------------------------------------------------
// SignMessage + VerifyMessage — round-trip
// ---------------------------------------------------------------------------

func TestSignMessageRoundTrip(t *testing.T) {
	iks := NewInMemoryKeystore()
	ref, err := iks.Store("signer", testPrivKeyHex)
	require.NoError(t, err)

	w := &Wallet{
		Name:    "signer",
		Address: testSignerAddr,
		Type:    TypeSigning,
		KeyRef:  ref,
	}

	message := []byte("hello web3")

	sig, err := SignMessage(w, iks, message)
	require.NoError(t, err)
	assert.Len(t, sig, 65, "EIP-191 signature must be 65 bytes")

	recovered, err := VerifyMessage(message, sig)
	require.NoError(t, err)
	assert.Equal(t, testSignerAddr, recovered.Hex(), "recovered address must match signer")
}

func TestSignMessageEmptyMessage(t *testing.T) {
	iks := NewInMemoryKeystore()
	ref, err := iks.Store("signer2", testPrivKeyHex)
	require.NoError(t, err)

	w := &Wallet{Name: "signer2", Address: testSignerAddr, Type: TypeSigning, KeyRef: ref}

	sig, err := SignMessage(w, iks, []byte(""))
	require.NoError(t, err)
	assert.Len(t, sig, 65)

	recovered, err := VerifyMessage([]byte(""), sig)
	require.NoError(t, err)
	assert.Equal(t, testSignerAddr, recovered.Hex())
}

func TestSignMessageLongMessage(t *testing.T) {
	iks := NewInMemoryKeystore()
	ref, err := iks.Store("signer3", testPrivKeyHex)
	require.NoError(t, err)

	w := &Wallet{Name: "signer3", Address: testSignerAddr, Type: TypeSigning, KeyRef: ref}

	longMsg := make([]byte, 1024)
	for i := range longMsg {
		longMsg[i] = byte(i % 256)
	}

	sig, err := SignMessage(w, iks, longMsg)
	require.NoError(t, err)

	recovered, err := VerifyMessage(longMsg, sig)
	require.NoError(t, err)
	assert.Equal(t, testSignerAddr, recovered.Hex())
}

// ---------------------------------------------------------------------------
// VerifyMessage — wrong signature
// ---------------------------------------------------------------------------

func TestVerifyMessageWrongSig(t *testing.T) {
	iks := NewInMemoryKeystore()
	ref, _ := iks.Store("s", testPrivKeyHex)
	w := &Wallet{Name: "s", Address: testSignerAddr, Type: TypeSigning, KeyRef: ref}

	msg := []byte("original message")
	sig, err := SignMessage(w, iks, msg)
	require.NoError(t, err)

	// Tamper with the signature.
	sig[0] ^= 0xff

	recovered, err := VerifyMessage(msg, sig)
	// ecrecover may succeed but return a different address.
	if err == nil {
		assert.NotEqual(t, testSignerAddr, recovered.Hex(), "tampered sig should not match signer")
	}
}

// ---------------------------------------------------------------------------
// VerifyMessage — wrong message
// ---------------------------------------------------------------------------

func TestVerifyMessageWrongMessage(t *testing.T) {
	iks := NewInMemoryKeystore()
	ref, _ := iks.Store("s2", testPrivKeyHex)
	w := &Wallet{Name: "s2", Address: testSignerAddr, Type: TypeSigning, KeyRef: ref}

	sig, err := SignMessage(w, iks, []byte("correct message"))
	require.NoError(t, err)

	recovered, err := VerifyMessage([]byte("wrong message"), sig)
	if err == nil {
		assert.NotEqual(t, testSignerAddr, recovered.Hex(), "wrong message should not match signer")
	}
}

// ---------------------------------------------------------------------------
// SignMessage — error paths
// ---------------------------------------------------------------------------

func TestSignMessageWatchOnlyError(t *testing.T) {
	iks := NewInMemoryKeystore()
	w := &Wallet{Name: "watcher", Address: testSignerAddr, Type: TypeWatchOnly}

	_, err := SignMessage(w, iks, []byte("test"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "watch-only")
}

func TestSignMessageKeystoreNotAvailable(t *testing.T) {
	iks := NewInMemoryKeystore()
	w := &Wallet{Name: "w", Address: testSignerAddr, Type: TypeSigning, KeyRef: "w3cli.missing"}

	_, err := SignMessage(w, iks, []byte("test"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retrieving key")
}

// ---------------------------------------------------------------------------
// VerifyMessage — invalid signature length
// ---------------------------------------------------------------------------

func TestVerifyMessageInvalidSigLength(t *testing.T) {
	_, err := VerifyMessage([]byte("test"), []byte("tooshort"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature length")
}

// ---------------------------------------------------------------------------
// eip191Hash — deterministic
// ---------------------------------------------------------------------------

func TestEIP191HashDeterministic(t *testing.T) {
	msg := []byte("deterministic test")
	h1 := eip191Hash(msg)
	h2 := eip191Hash(msg)
	assert.Equal(t, hex.EncodeToString(h1), hex.EncodeToString(h2))
}

func TestEIP191HashDifferentMessages(t *testing.T) {
	h1 := eip191Hash([]byte("message A"))
	h2 := eip191Hash([]byte("message B"))
	assert.NotEqual(t, hex.EncodeToString(h1), hex.EncodeToString(h2))
}
