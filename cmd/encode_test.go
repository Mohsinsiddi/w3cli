package cmd

import (
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/contract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeTransferCalldata(t *testing.T) {
	entry := contract.ABIEntry{
		Name: "transfer",
		Type: "function",
		Inputs: []contract.ABIParam{
			{Name: "to", Type: "address"},
			{Name: "value", Type: "uint256"},
		},
		StateMutability: "nonpayable",
	}

	hexStr, raw, err := contract.EncodeCalldata(entry, []string{
		"0xd8da6bf26964af9d7eed9e03e53415d37aa96045",
		"1000000000000000000",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, hexStr)
	assert.True(t, len(raw) > 4, "calldata must include selector + args")

	// Check selector is transfer.
	assert.Equal(t, "0xa9059cbb", hexStr[:10])
}

func TestEncodeApproveCalldata(t *testing.T) {
	entry := contract.ABIEntry{
		Name: "approve",
		Type: "function",
		Inputs: []contract.ABIParam{
			{Name: "spender", Type: "address"},
			{Name: "value", Type: "uint256"},
		},
		StateMutability: "nonpayable",
	}

	hexStr, _, err := contract.EncodeCalldata(entry, []string{
		"0xd8da6bf26964af9d7eed9e03e53415d37aa96045",
		"115792089237316195423570985008687907853269984665640564039457584007913129639935",
	})
	require.NoError(t, err)
	assert.Equal(t, "0x095ea7b3", hexStr[:10])
}

func TestEncodeBalanceOfCalldata(t *testing.T) {
	entry := contract.ABIEntry{
		Name: "balanceOf",
		Type: "function",
		Inputs: []contract.ABIParam{
			{Name: "account", Type: "address"},
		},
		StateMutability: "view",
	}

	hexStr, raw, err := contract.EncodeCalldata(entry, []string{
		"0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
	})
	require.NoError(t, err)
	assert.Equal(t, "0x70a08231", hexStr[:10])
	assert.Equal(t, 36, len(raw), "balanceOf calldata should be 4+32 bytes")
}

func TestEncodeNoArgs(t *testing.T) {
	entry := contract.ABIEntry{
		Name:            "name",
		Type:            "function",
		Inputs:          nil,
		StateMutability: "view",
	}

	hexStr, raw, err := contract.EncodeCalldata(entry, nil)
	require.NoError(t, err)
	assert.Equal(t, 4, len(raw), "name() calldata should be just 4 bytes")
	assert.Equal(t, "0x06fdde03", hexStr[:10])
}
