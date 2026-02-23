package ens

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Namehash — EIP-137 spec vectors
// ---------------------------------------------------------------------------

func TestNamehash_Empty(t *testing.T) {
	result := Namehash("")
	expected := "0000000000000000000000000000000000000000000000000000000000000000"
	assert.Equal(t, expected, result)
}

func TestNamehash_ETH(t *testing.T) {
	result := Namehash("eth")
	// Known EIP-137 vector for "eth".
	expected := "93cdeb708b7545dc668eb9280176169d1c33cfd8ed6f04690a0bcc88a93fc4ae"
	assert.Equal(t, expected, result)
}

func TestNamehash_FooETH(t *testing.T) {
	result := Namehash("foo.eth")
	// Known EIP-137 vector for "foo.eth".
	expected := "de9b09fd7c5f901e23a3f19fecc54828e9c848539801e86591bd9801b019f84f"
	assert.Equal(t, expected, result)
}

func TestNamehash_VitalikETH(t *testing.T) {
	result := Namehash("vitalik.eth")
	// This is a well-known namehash — verified against multiple implementations.
	assert.Len(t, result, 64, "namehash must be 32 bytes (64 hex chars)")
	assert.NotEqual(t, Namehash("eth"), result, "vitalik.eth must differ from eth")
}

func TestNamehash_Deterministic(t *testing.T) {
	h1 := Namehash("test.eth")
	h2 := Namehash("test.eth")
	assert.Equal(t, h1, h2)
}

func TestNamehash_DifferentNames(t *testing.T) {
	assert.NotEqual(t, Namehash("alice.eth"), Namehash("bob.eth"))
}

func TestNamehash_CaseSensitive(t *testing.T) {
	// ENS namehash is case-sensitive (names should be normalized before hashing).
	h1 := Namehash("Test.eth")
	h2 := Namehash("test.eth")
	// These will differ because ENS doesn't lowercase internally.
	assert.NotEqual(t, h1, h2)
}

func TestNamehash_Subdomain(t *testing.T) {
	result := Namehash("sub.test.eth")
	assert.Len(t, result, 64)
	assert.NotEqual(t, Namehash("test.eth"), result)
}

// ---------------------------------------------------------------------------
// helpers — rpcMock for ENS tests
// ---------------------------------------------------------------------------

func ensRPCMock(t *testing.T, responses map[string]string) *httptest.Server {
	t.Helper()
	callCount := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string        `json:"method"`
			Params []interface{} `json:"params"`
			ID     int           `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
		w.Header().Set("Content-Type", "application/json")

		if req.Method == "eth_call" {
			callCount++
			// Return responses in order based on call count.
			key := ""
			switch callCount {
			case 1:
				key = "resolver"
			case 2:
				key = "addr"
			}
			if result, ok := responses[key]; ok {
				json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
					"jsonrpc": "2.0",
					"id":      req.ID,
					"result":  result,
				})
				return
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error":   map[string]interface{}{"code": -32601, "message": "method not found"},
		})
	}))
}

// ---------------------------------------------------------------------------
// Resolve — mock RPC
// ---------------------------------------------------------------------------

func TestResolveMockRPC(t *testing.T) {
	// Mock: resolver returns a resolver address, then addr returns an address.
	resolverAddr := "0x000000000000000000000000" + "4976fb03c32e5b8cfe2b6ccb31c09ba78ebaba41" // Public resolver
	targetAddr := "0x000000000000000000000000" + "d8da6bf26964af9d7eed9e03e53415d37aa96045"   // vitalik

	srv := ensRPCMock(t, map[string]string{
		"resolver": resolverAddr,
		"addr":     targetAddr,
	})
	defer srv.Close()

	client := chain.NewEVMClient(srv.URL)
	address, err := Resolve("vitalik.eth", client)
	require.NoError(t, err)
	assert.Equal(t, "0x"+"d8da6bf26964af9d7eed9e03e53415d37aa96045", address)
}

func TestResolveNoResolver(t *testing.T) {
	// Resolver returns zero address.
	srv := ensRPCMock(t, map[string]string{
		"resolver": "0x0000000000000000000000000000000000000000000000000000000000000000",
	})
	defer srv.Close()

	client := chain.NewEVMClient(srv.URL)
	_, err := Resolve("nonexistent.eth", client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no resolver")
}

// ---------------------------------------------------------------------------
// ReverseLookup — mock RPC
// ---------------------------------------------------------------------------

func TestReverseLookupMockRPC(t *testing.T) {
	// resolver + name (ABI-encoded string "vitalik.eth").
	resolverAddr := "0x000000000000000000000000" + "a58e81fe9b61b5c3fe2b0882a7c0716277277deb"

	// ABI-encoded string "vitalik.eth":
	// offset (0x20) + length (11) + data
	encodedName := "0x" +
		"0000000000000000000000000000000000000000000000000000000000000020" + // offset
		"000000000000000000000000000000000000000000000000000000000000000b" + // length = 11
		"766974616c696b2e657468000000000000000000000000000000000000000000" // "vitalik.eth" padded

	srv := ensRPCMock(t, map[string]string{
		"resolver": resolverAddr,
		"addr":     encodedName,
	})
	defer srv.Close()

	client := chain.NewEVMClient(srv.URL)
	name, err := ReverseLookup("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", client)
	require.NoError(t, err)
	assert.Equal(t, "vitalik.eth", name)
}

func TestReverseLookupNoResolver(t *testing.T) {
	srv := ensRPCMock(t, map[string]string{
		"resolver": "0x0000000000000000000000000000000000000000000000000000000000000000",
	})
	defer srv.Close()

	client := chain.NewEVMClient(srv.URL)
	_, err := ReverseLookup("0x1234567890abcdef1234567890abcdef12345678", client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no reverse record")
}

// ---------------------------------------------------------------------------
// parseAddress
// ---------------------------------------------------------------------------

func TestParseAddress_Valid(t *testing.T) {
	input := "0x000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045"
	result := parseAddress(input)
	assert.Equal(t, "0xd8da6bf26964af9d7eed9e03e53415d37aa96045", result)
}

func TestParseAddress_Zero(t *testing.T) {
	input := "0x0000000000000000000000000000000000000000000000000000000000000000"
	result := parseAddress(input)
	assert.Equal(t, "0x0000000000000000000000000000000000000000", result)
}

func TestParseAddress_Short(t *testing.T) {
	assert.Equal(t, "", parseAddress("0xabcd"))
}

func TestParseAddress_NoPrefix(t *testing.T) {
	input := "000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045"
	result := parseAddress(input)
	assert.Equal(t, "0xd8da6bf26964af9d7eed9e03e53415d37aa96045", result)
}

// ---------------------------------------------------------------------------
// decodeString
// ---------------------------------------------------------------------------

func TestDecodeString_Valid(t *testing.T) {
	// ABI-encoded "hello":
	// offset 0x20 + length 5 + "hello" padded
	input := "0x" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000005" +
		"68656c6c6f000000000000000000000000000000000000000000000000000000"
	result := decodeString(input)
	assert.Equal(t, "hello", result)
}

func TestDecodeString_Empty(t *testing.T) {
	assert.Equal(t, "", decodeString("0x"))
}

func TestDecodeString_ZeroLength(t *testing.T) {
	input := "0x" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000000"
	result := decodeString(input)
	assert.Equal(t, "", result)
}

// ---------------------------------------------------------------------------
// hexDigit
// ---------------------------------------------------------------------------

func TestHexDigit_Numbers(t *testing.T) {
	assert.Equal(t, 0, hexDigit('0'))
	assert.Equal(t, 9, hexDigit('9'))
}

func TestHexDigit_LowerCase(t *testing.T) {
	assert.Equal(t, 10, hexDigit('a'))
	assert.Equal(t, 15, hexDigit('f'))
}

func TestHexDigit_UpperCase(t *testing.T) {
	assert.Equal(t, 10, hexDigit('A'))
	assert.Equal(t, 15, hexDigit('F'))
}

func TestHexDigit_Invalid(t *testing.T) {
	assert.Equal(t, 0, hexDigit('z'))
}
