package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// normalizeSignature
// ---------------------------------------------------------------------------

func TestNormalizeSignature_AlreadyCanonical(t *testing.T) {
	assert.Equal(t, "transfer(address,uint256)", normalizeSignature("transfer(address,uint256)"))
}

func TestNormalizeSignature_WithNames(t *testing.T) {
	assert.Equal(t, "transfer(address,uint256)", normalizeSignature("transfer(address to, uint256 amount)"))
}

func TestNormalizeSignature_NoParams(t *testing.T) {
	assert.Equal(t, "name()", normalizeSignature("name()"))
}

func TestNormalizeSignature_SingleParam(t *testing.T) {
	assert.Equal(t, "balanceOf(address)", normalizeSignature("balanceOf(address account)"))
}

func TestNormalizeSignature_ThreeParams(t *testing.T) {
	assert.Equal(t, "transferFrom(address,address,uint256)", normalizeSignature("transferFrom(address from, address to, uint256 amount)"))
}

func TestNormalizeSignature_NoParens(t *testing.T) {
	// Edge case: no parentheses.
	assert.Equal(t, "noop", normalizeSignature("noop"))
}

func TestNormalizeSignature_ExtraSpaces(t *testing.T) {
	assert.Equal(t, "approve(address,uint256)", normalizeSignature("approve(  address  spender ,  uint256  amount  )"))
}

// ---------------------------------------------------------------------------
// computeEventTopic
// ---------------------------------------------------------------------------

func TestComputeEventTopic_Transfer(t *testing.T) {
	topic := computeEventTopic("Transfer(address,address,uint256)")
	assert.Equal(t, "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", topic)
}

func TestComputeEventTopic_Approval(t *testing.T) {
	topic := computeEventTopic("Approval(address,address,uint256)")
	assert.Equal(t, "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925", topic)
}

func TestComputeEventTopic_Deterministic(t *testing.T) {
	t1 := computeEventTopic("Transfer(address,address,uint256)")
	t2 := computeEventTopic("Transfer(address,address,uint256)")
	assert.Equal(t, t1, t2)
}
