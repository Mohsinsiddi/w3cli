package ens

import (
	"fmt"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"golang.org/x/crypto/sha3"
)

// ENS Registry address — same on Ethereum mainnet and Sepolia.
const registryAddr = "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e"

// Resolve resolves an ENS name to an address.
// It queries the ENS registry for the resolver, then calls addr(bytes32) on it.
func Resolve(name string, client *chain.EVMClient) (string, error) {
	node := Namehash(name)

	// Step 1: Get resolver address from registry.
	// resolver(bytes32) = 0x0178b8bf
	resolverCalldata := "0x0178b8bf" + node
	resolverResult, err := client.CallContract(registryAddr, resolverCalldata)
	if err != nil {
		return "", fmt.Errorf("querying ENS registry: %w", err)
	}

	resolverAddr := parseAddress(resolverResult)
	if resolverAddr == "" || resolverAddr == "0x0000000000000000000000000000000000000000" {
		return "", fmt.Errorf("no resolver set for %q", name)
	}

	// Step 2: Call addr(bytes32) on the resolver.
	// addr(bytes32) = 0x3b3b57de
	addrCalldata := "0x3b3b57de" + node
	addrResult, err := client.CallContract(resolverAddr, addrCalldata)
	if err != nil {
		return "", fmt.Errorf("querying ENS resolver: %w", err)
	}

	resolved := parseAddress(addrResult)
	if resolved == "" || resolved == "0x0000000000000000000000000000000000000000" {
		return "", fmt.Errorf("no address record for %q", name)
	}

	return resolved, nil
}

// ReverseLookup resolves an address to an ENS name.
// It uses the addr.reverse registry.
func ReverseLookup(address string, client *chain.EVMClient) (string, error) {
	clean := strings.ToLower(strings.TrimPrefix(address, "0x"))
	reverseName := clean + ".addr.reverse"
	node := Namehash(reverseName)

	// Step 1: Get resolver from registry.
	resolverCalldata := "0x0178b8bf" + node
	resolverResult, err := client.CallContract(registryAddr, resolverCalldata)
	if err != nil {
		return "", fmt.Errorf("querying reverse registry: %w", err)
	}

	resolverAddr := parseAddress(resolverResult)
	if resolverAddr == "" || resolverAddr == "0x0000000000000000000000000000000000000000" {
		return "", fmt.Errorf("no reverse record for %s", address)
	}

	// Step 2: Call name(bytes32) on the resolver.
	// name(bytes32) = 0x691f3431
	nameCalldata := "0x691f3431" + node
	nameResult, err := client.CallContract(resolverAddr, nameCalldata)
	if err != nil {
		return "", fmt.Errorf("querying reverse resolver: %w", err)
	}

	name := decodeString(nameResult)
	if name == "" {
		return "", fmt.Errorf("no reverse name for %s", address)
	}

	return name, nil
}

// Namehash implements EIP-137 namehash algorithm.
// namehash("") = 0x00...00
// namehash("eth") = keccak256(namehash("") + keccak256("eth"))
func Namehash(name string) string {
	node := make([]byte, 32) // starts as 32 zero bytes

	if name == "" {
		return fmt.Sprintf("%064x", node)
	}

	labels := strings.Split(name, ".")
	// Process labels right-to-left.
	for i := len(labels) - 1; i >= 0; i-- {
		labelHash := keccak256([]byte(labels[i]))
		combined := append(node, labelHash...)
		node = keccak256(combined)
	}

	return fmt.Sprintf("%064x", node)
}

// keccak256 returns the Keccak-256 hash of data.
func keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

// parseAddress extracts a 20-byte address from a 32-byte ABI-encoded word.
func parseAddress(hexResult string) string {
	clean := strings.TrimPrefix(hexResult, "0x")
	if len(clean) < 64 {
		return ""
	}
	// Address is in the last 40 characters of the 64-char word.
	addr := clean[24:64]
	// Check if it's all zeros.
	allZero := true
	for _, c := range addr {
		if c != '0' {
			allZero = false
			break
		}
	}
	if allZero {
		return "0x0000000000000000000000000000000000000000"
	}
	return "0x" + addr
}

// decodeString decodes an ABI-encoded string return value.
func decodeString(hexResult string) string {
	clean := strings.TrimPrefix(hexResult, "0x")
	if len(clean) < 128 { // offset (64) + length (64) minimum
		return ""
	}

	// First word is the offset to the string data (should be 0x20 = 32).
	// Second word is the length.
	// Remaining bytes are the string content.
	lengthHex := clean[64:128]
	length := 0
	for _, c := range lengthHex {
		length = length*16 + hexDigit(byte(c))
	}
	if length == 0 {
		return ""
	}

	dataStart := 128
	dataEnd := dataStart + length*2
	if dataEnd > len(clean) {
		dataEnd = len(clean)
	}

	result := make([]byte, 0, length)
	for i := dataStart; i+1 < dataEnd; i += 2 {
		b := byte(hexDigit(clean[i])<<4 | hexDigit(clean[i+1]))
		result = append(result, b)
	}
	return string(result)
}

func hexDigit(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	default:
		return 0
	}
}
