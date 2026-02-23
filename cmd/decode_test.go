package cmd

import (
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// DecodeMethod — known selectors
// ---------------------------------------------------------------------------

func TestDecodeMethod_Transfer(t *testing.T) {
	calldata := "0xa9059cbb000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa960450000000000000000000000000000000000000000000000000de0b6b3a7640000"
	assert.Equal(t, "transfer", chain.DecodeMethod(calldata))
}

func TestDecodeMethod_Approve(t *testing.T) {
	calldata := "0x095ea7b3000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa960450000000000000000000000000000000000000000000000000de0b6b3a7640000"
	assert.Equal(t, "approve", chain.DecodeMethod(calldata))
}

func TestDecodeMethod_BalanceOf(t *testing.T) {
	calldata := "0x70a08231000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266"
	assert.Equal(t, "balanceOf", chain.DecodeMethod(calldata))
}

func TestDecodeMethod_TransferFrom(t *testing.T) {
	calldata := "0x23b872dd"
	assert.Equal(t, "transferFrom", chain.DecodeMethod(calldata))
}

func TestDecodeMethod_Decimals(t *testing.T) {
	assert.Equal(t, "decimals", chain.DecodeMethod("0x313ce567"))
}

func TestDecodeMethod_Name(t *testing.T) {
	assert.Equal(t, "name", chain.DecodeMethod("0x06fdde03"))
}

func TestDecodeMethod_Symbol(t *testing.T) {
	assert.Equal(t, "symbol", chain.DecodeMethod("0x95d89b41"))
}

func TestDecodeMethod_TotalSupply(t *testing.T) {
	assert.Equal(t, "totalSupply", chain.DecodeMethod("0x18160ddd"))
}

// ---------------------------------------------------------------------------
// DecodeMethod — unknown selectors
// ---------------------------------------------------------------------------

func TestDecodeMethod_Unknown(t *testing.T) {
	result := chain.DecodeMethod("0xdeadbeef0000")
	// Should return first 10 chars of "0xdeadbeef" (the selector).
	assert.Equal(t, "0xdeadbeef", result)
}

// ---------------------------------------------------------------------------
// DecodeMethod — edge cases
// ---------------------------------------------------------------------------

func TestDecodeMethod_Empty(t *testing.T) {
	assert.Equal(t, "transfer", chain.DecodeMethod(""))
}

func TestDecodeMethod_PlainTransfer(t *testing.T) {
	assert.Equal(t, "transfer", chain.DecodeMethod("0x"))
}

func TestDecodeMethod_ShortInput(t *testing.T) {
	// Less than 4 bytes.
	assert.Equal(t, "call", chain.DecodeMethod("0xabcd"))
}

func TestDecodeMethod_JustSelector(t *testing.T) {
	// Exactly 4 bytes (8 hex chars), no args.
	assert.Equal(t, "approve", chain.DecodeMethod("0x095ea7b3"))
}

// ---------------------------------------------------------------------------
// DecodeMethod — swap methods
// ---------------------------------------------------------------------------

func TestDecodeMethod_SwapExactETHForTokens(t *testing.T) {
	assert.Equal(t, "swapExactETHForTokens", chain.DecodeMethod("0x7ff36ab5"))
}

func TestDecodeMethod_Multicall(t *testing.T) {
	assert.Equal(t, "multicall", chain.DecodeMethod("0xac9650d8"))
}

func TestDecodeMethod_Deposit(t *testing.T) {
	assert.Equal(t, "deposit", chain.DecodeMethod("0xd0e30db0"))
}

func TestDecodeMethod_Withdraw(t *testing.T) {
	assert.Equal(t, "withdraw", chain.DecodeMethod("0x2e1a7d4d"))
}
