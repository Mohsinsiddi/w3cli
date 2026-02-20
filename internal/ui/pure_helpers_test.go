package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// padR
// ---------------------------------------------------------------------------

func TestPadRShort(t *testing.T) {
	result := padR("hi", 10)
	assert.Equal(t, 10, len(result))
	assert.True(t, strings.HasPrefix(result, "hi"))
}

func TestPadRExact(t *testing.T) {
	result := padR("hello", 5)
	assert.Equal(t, "hello", result)
}

func TestPadRLonger(t *testing.T) {
	// When string is already longer, return as-is.
	result := padR("toolongstring", 5)
	assert.Equal(t, "toolongstring", result)
}

func TestPadREmpty(t *testing.T) {
	result := padR("", 4)
	assert.Equal(t, "    ", result)
}

func TestPadRZeroWidth(t *testing.T) {
	result := padR("x", 0)
	assert.Equal(t, "x", result)
}

// ---------------------------------------------------------------------------
// trimErr
// ---------------------------------------------------------------------------

func TestTrimErrShortString(t *testing.T) {
	result := trimErr("short error")
	assert.Equal(t, "short error", result)
}

func TestTrimErrLongStringTruncated(t *testing.T) {
	long := strings.Repeat("x", 50)
	result := trimErr(long)
	assert.True(t, len(result) <= 34, "trimErr result length should be truncated")
	assert.Contains(t, result, "…")
}

func TestTrimErrExactly30(t *testing.T) {
	s := strings.Repeat("a", 30)
	result := trimErr(s)
	assert.Equal(t, s, result, "30 chars is exact limit — no truncation")
}

func TestTrimErrDialTCP(t *testing.T) {
	s := "some prefix: dial tcp 127.0.0.1:8545: connection refused"
	result := trimErr(s)
	assert.True(t, strings.HasPrefix(result, "dial tcp"), "should trim to 'dial tcp' prefix")
}

func TestTrimErrContextDeadline(t *testing.T) {
	s := "error fetching: context deadline exceeded (Client.Timeout exceeded while awaiting headers)"
	result := trimErr(s)
	assert.True(t, strings.HasPrefix(result, "context deadline"))
}

func TestTrimErrNoMatch(t *testing.T) {
	s := "RPC error: method not found"
	result := trimErr(s)
	// No matching prefix — string returned with truncation if needed.
	assert.Equal(t, s, result)
}

// ---------------------------------------------------------------------------
// mainUSDFloat
// ---------------------------------------------------------------------------

func TestMainUSDFloatWithDollarSign(t *testing.T) {
	row := AllBalRow{MainUSD: "$1234.56"}
	assert.InDelta(t, 1234.56, row.mainUSDFloat(), 0.001)
}

func TestMainUSDFloatEmpty(t *testing.T) {
	row := AllBalRow{MainUSD: ""}
	assert.Equal(t, float64(0), row.mainUSDFloat())
}

func TestMainUSDFloatDash(t *testing.T) {
	row := AllBalRow{MainUSD: "—"}
	assert.Equal(t, float64(0), row.mainUSDFloat())
}

func TestMainUSDFloatSmall(t *testing.T) {
	row := AllBalRow{MainUSD: "$0.01"}
	assert.InDelta(t, 0.01, row.mainUSDFloat(), 0.0001)
}

func TestMainUSDFloatZeroString(t *testing.T) {
	row := AllBalRow{MainUSD: "$0.00"}
	assert.Equal(t, float64(0), row.mainUSDFloat())
}

// ---------------------------------------------------------------------------
// formatGwei
// ---------------------------------------------------------------------------

func TestFormatGweiZero(t *testing.T) {
	assert.Equal(t, "0", formatGwei(0))
}

func TestFormatGweiSubMilli(t *testing.T) {
	result := formatGwei(0.0000001)
	assert.Contains(t, result, ".")
	assert.Len(t, strings.Split(result, ".")[1], 6)
}

func TestFormatGweiSubOne(t *testing.T) {
	result := formatGwei(0.5)
	// Should have 4 decimal places.
	assert.Equal(t, "0.5000", result)
}

func TestFormatGweiUnderHundred(t *testing.T) {
	result := formatGwei(12.5)
	assert.Equal(t, "12.50", result)
}

func TestFormatGweiHundredOrMore(t *testing.T) {
	result := formatGwei(150.0)
	assert.Equal(t, "150", result)
}

func TestFormatGweiLarge(t *testing.T) {
	result := formatGwei(1000.0)
	assert.Equal(t, "1000", result)
}

// ---------------------------------------------------------------------------
// studioParamSig
// ---------------------------------------------------------------------------

func TestStudioParamSigEmpty(t *testing.T) {
	result := studioParamSig(nil)
	assert.Equal(t, "", result)
}

func TestStudioParamSigSingleWithName(t *testing.T) {
	params := []StudioParam{{Type: "address", Name: "to"}}
	assert.Equal(t, "address to", studioParamSig(params))
}

func TestStudioParamSigSingleNoName(t *testing.T) {
	params := []StudioParam{{Type: "uint256"}}
	assert.Equal(t, "uint256", studioParamSig(params))
}

func TestStudioParamSigMultiple(t *testing.T) {
	params := []StudioParam{
		{Type: "address", Name: "to"},
		{Type: "uint256", Name: "amount"},
	}
	result := studioParamSig(params)
	assert.Equal(t, "address to, uint256 amount", result)
}

func TestStudioParamSigMixedNameNoName(t *testing.T) {
	params := []StudioParam{
		{Type: "address", Name: "recipient"},
		{Type: "bytes"},
	}
	result := studioParamSig(params)
	assert.Equal(t, "address recipient, bytes", result)
}

// ---------------------------------------------------------------------------
// DangerBox
// ---------------------------------------------------------------------------

func TestDangerBoxNotEmpty(t *testing.T) {
	result := DangerBox("WARNING: your private key")
	assert.NotEmpty(t, result)
}

func TestDangerBoxContainsContent(t *testing.T) {
	result := DangerBox("my secret content")
	assert.Contains(t, result, "my secret content")
}

func TestDangerBoxEmptyContent(t *testing.T) {
	// Should not panic on empty input.
	assert.NotPanics(t, func() { DangerBox("") })
}
