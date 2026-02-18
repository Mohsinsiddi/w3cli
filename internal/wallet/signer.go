package wallet

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Signer signs EVM transactions for a signing wallet.
type Signer struct {
	wallet  *Wallet
	ks      *Keystore
}

// NewSigner creates a signer for the given wallet.
func NewSigner(w *Wallet, ks *Keystore) *Signer {
	return &Signer{wallet: w, ks: ks}
}

// SignTx signs an EVM transaction and returns the raw signed bytes.
func (s *Signer) SignTx(tx *types.Transaction, chainID *big.Int) ([]byte, error) {
	if s.wallet.Type != TypeSigning {
		return nil, fmt.Errorf("wallet %q is watch-only and cannot sign", s.wallet.Name)
	}

	hexKey, err := s.ks.Retrieve(s.wallet.KeyRef)
	if err != nil {
		return nil, fmt.Errorf("retrieving key: %w", err)
	}

	privKey, err := crypto.HexToECDSA(stripHexPrefix(hexKey))
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	signer := types.NewLondonSigner(chainID)
	signed, err := types.SignTx(tx, signer, privKey)
	if err != nil {
		return nil, fmt.Errorf("signing transaction: %w", err)
	}

	raw, err := signed.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshaling signed tx: %w", err)
	}

	return raw, nil
}

// Address returns the wallet's address.
func (s *Signer) Address() string {
	return s.wallet.Address
}
