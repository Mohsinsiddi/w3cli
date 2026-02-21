package wallet

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// SignMessage signs a message using EIP-191 (personal_sign).
// The message is prefixed with "\x19Ethereum Signed Message:\n<len>" before hashing.
// Returns a 65-byte signature (R || S || V).
func SignMessage(w *Wallet, ks KeystoreBackend, message []byte) ([]byte, error) {
	if w.Type != TypeSigning {
		return nil, fmt.Errorf("wallet %q is watch-only and cannot sign", w.Name)
	}

	hexKey, err := ks.Retrieve(w.KeyRef)
	if err != nil {
		return nil, fmt.Errorf("retrieving key: %w", err)
	}

	privKey, err := crypto.HexToECDSA(stripHexPrefix(hexKey))
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	hash := eip191Hash(message)
	sig, err := crypto.Sign(hash, privKey)
	if err != nil {
		return nil, fmt.Errorf("signing message: %w", err)
	}

	// Adjust V from 0/1 to 27/28 for Ethereum compatibility.
	sig[64] += 27

	return sig, nil
}

// VerifyMessage recovers the signer address from an EIP-191 signature.
// Returns the recovered address.
func VerifyMessage(message, sig []byte) (common.Address, error) {
	if len(sig) != 65 {
		return common.Address{}, fmt.Errorf("invalid signature length: expected 65 bytes, got %d", len(sig))
	}

	// Adjust V from 27/28 back to 0/1 for ecrecover.
	recoverSig := make([]byte, 65)
	copy(recoverSig, sig)
	recoverSig[64] -= 27

	hash := eip191Hash(message)
	pubKey, err := crypto.SigToPub(hash, recoverSig)
	if err != nil {
		return common.Address{}, fmt.Errorf("recovering signer: %w", err)
	}

	return crypto.PubkeyToAddress(*pubKey), nil
}

// eip191Hash returns the Keccak-256 hash of the EIP-191 prefixed message.
func eip191Hash(message []byte) []byte {
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))
	data := append([]byte(prefix), message...)
	return crypto.Keccak256(data)
}
