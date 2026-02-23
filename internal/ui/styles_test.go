package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfoContainsPrefixAndMessage(t *testing.T) {
	result := Info("test message")
	assert.Contains(t, result, "ℹ")
	assert.Contains(t, result, "test message")
}

func TestInfoEmptyMessage(t *testing.T) {
	result := Info("")
	assert.Contains(t, result, "ℹ")
}

func TestHintContainsPrefixAndMessage(t *testing.T) {
	result := Hint("try this command")
	assert.Contains(t, result, "💡")
	assert.Contains(t, result, "try this command")
}

func TestHintEmptyMessage(t *testing.T) {
	result := Hint("")
	assert.Contains(t, result, "💡")
}

func TestSuccessContainsPrefixAndMessage(t *testing.T) {
	result := Success("done")
	assert.Contains(t, result, "✓")
	assert.Contains(t, result, "done")
}

func TestWarnContainsPrefixAndMessage(t *testing.T) {
	result := Warn("careful")
	assert.Contains(t, result, "⚠")
	assert.Contains(t, result, "careful")
}

func TestErrContainsPrefixAndMessage(t *testing.T) {
	result := Err("failed")
	assert.Contains(t, result, "✗")
	assert.Contains(t, result, "failed")
}

func TestAddrContainsAddress(t *testing.T) {
	result := Addr("0xABCDEF")
	assert.Contains(t, result, "0xABCDEF")
}

func TestValContainsValue(t *testing.T) {
	result := Val("1.5 ETH")
	assert.Contains(t, result, "1.5 ETH")
}

func TestMetaContainsText(t *testing.T) {
	result := Meta("some metadata")
	assert.Contains(t, result, "some metadata")
}

func TestChainNameContainsName(t *testing.T) {
	result := ChainName("ethereum")
	assert.Contains(t, result, "ethereum")
}

func TestTruncateAddrShortAddress(t *testing.T) {
	assert.Equal(t, "0x1234", TruncateAddr("0x1234"))
}

func TestTruncateAddrExactBoundary(t *testing.T) {
	assert.Equal(t, "0x12345678", TruncateAddr("0x12345678"))
}

func TestTruncateAddrLongAddress(t *testing.T) {
	addr := "0x1234567890abcdef1234567890abcdef12345678"
	result := TruncateAddr(addr)
	assert.Equal(t, "0x1234…5678", result)
	assert.Less(t, len(result), len(addr))
}

func TestTruncateAddrEmptyString(t *testing.T) {
	assert.Equal(t, "", TruncateAddr(""))
}

func TestInfoDifferentFromHint(t *testing.T) {
	info := Info("message")
	hint := Hint("message")
	assert.NotEqual(t, info, hint, "Info and Hint should produce different output for the same message")
}

func TestAllFormattersReturnNonEmpty(t *testing.T) {
	formatters := map[string]func(string) string{
		"Success":   Success,
		"Warn":      Warn,
		"Err":       Err,
		"Info":      Info,
		"Hint":      Hint,
		"Addr":      Addr,
		"Val":       Val,
		"Meta":      Meta,
		"ChainName": ChainName,
	}
	for name, fn := range formatters {
		t.Run(name, func(t *testing.T) {
			result := fn("test")
			assert.NotEmpty(t, result, "%s should return non-empty string", name)
			assert.Contains(t, result, "test", "%s should contain the input message", name)
		})
	}
}
