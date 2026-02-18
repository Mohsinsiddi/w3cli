package contract

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Sender sends write transactions to contracts.
type Sender struct {
	client  *chain.EVMClient
	abi     []ABIEntry
	signer  *wallet.Signer
	chainID *big.Int
}

// NewSender creates a Sender.
func NewSender(rpcURL string, abi []ABIEntry, signer *wallet.Signer, chainID *big.Int) *Sender {
	return &Sender{
		client:  chain.NewEVMClient(rpcURL),
		abi:     abi,
		signer:  signer,
		chainID: chainID,
	}
}

// Send calls a write function and broadcasts the transaction.
// Returns the transaction hash.
func (s *Sender) Send(contractAddr, funcName string, args ...string) (string, error) {
	// Find function in ABI.
	var fn *ABIEntry
	for i := range s.abi {
		if s.abi[i].Type == "function" && s.abi[i].Name == funcName {
			fn = &s.abi[i]
			break
		}
	}
	if fn == nil {
		return "", fmt.Errorf("function %q not found in ABI", funcName)
	}
	if !fn.IsWriteFunction() {
		return "", fmt.Errorf("function %q is not a write function", funcName)
	}

	calldata, err := encodeCall(fn, args)
	if err != nil {
		return "", fmt.Errorf("encoding call: %w", err)
	}

	from := s.signer.Address()

	// Estimate gas.
	gas, err := s.client.EstimateGas(from, contractAddr, calldata, nil)
	if err != nil {
		gas = 100000 // fallback
	}

	// Get gas price.
	gasPrice, err := s.client.GasPrice()
	if err != nil {
		return "", fmt.Errorf("getting gas price: %w", err)
	}

	// Get nonce.
	nonce, err := s.client.GetNonce(from)
	if err != nil {
		return "", fmt.Errorf("getting nonce: %w", err)
	}

	// Decode calldata hex to bytes.
	calldataBytes := hexToBytes(calldata)
	toAddr := common.HexToAddress(contractAddr)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   s.chainID,
		Nonce:     nonce,
		GasTipCap: gasPrice,
		GasFeeCap: new(big.Int).Mul(gasPrice, big.NewInt(2)),
		Gas:       gas,
		To:        &toAddr,
		Value:     big.NewInt(0),
		Data:      calldataBytes,
	})

	raw, err := s.signer.SignTx(tx, s.chainID)
	if err != nil {
		return "", fmt.Errorf("signing transaction: %w", err)
	}

	hash, err := s.client.SendRawTransaction("0x" + bytesToHex(raw))
	if err != nil {
		return "", fmt.Errorf("broadcasting transaction: %w", err)
	}

	return hash, nil
}

// hexToBytes converts a hex string (with or without 0x) to bytes.
func hexToBytes(s string) []byte {
	s = strings.TrimPrefix(s, "0x")
	if len(s)%2 != 0 {
		s = "0" + s
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		var byt byte
		fmt.Sscanf(s[i:i+2], "%02x", &byt)
		b[i/2] = byt
	}
	return b
}

func bytesToHex(b []byte) string {
	return fmt.Sprintf("%x", b)
}
