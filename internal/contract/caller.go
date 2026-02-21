package contract

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
)

// Caller calls read-only (view/pure) contract functions.
type Caller struct {
	client *chain.EVMClient
	abi    []ABIEntry
}

// NewCaller creates a Caller using a raw ABI JSON byte slice.
func NewCaller(rpcURL string, abiJSON []byte) *Caller {
	abi, _ := parseABI(abiJSON)
	return &Caller{
		client: chain.NewEVMClient(rpcURL),
		abi:    abi,
	}
}

// NewCallerFromEntries creates a Caller from already-parsed ABI entries.
func NewCallerFromEntries(rpcURL string, abi []ABIEntry) *Caller {
	return &Caller{
		client: chain.NewEVMClient(rpcURL),
		abi:    abi,
	}
}

// Call calls a read function on a contract and returns decoded results as strings.
func (c *Caller) Call(contractAddr, funcName string, args ...string) ([]string, error) {
	fn := c.findFunction(funcName)
	if fn == nil {
		return nil, fmt.Errorf("function %q not found in ABI", funcName)
	}

	if !fn.IsReadFunction() {
		return nil, fmt.Errorf("function %q is not a read function (stateMutability: %s)", funcName, fn.StateMutability)
	}

	calldata, err := encodeCall(fn, args)
	if err != nil {
		return nil, fmt.Errorf("encoding call: %w", err)
	}

	result, err := c.client.CallContract(contractAddr, calldata)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	decoded, err := decodeResult(fn, result)
	if err != nil {
		return nil, fmt.Errorf("decoding result: %w", err)
	}

	return decoded, nil
}

// findFunction finds an ABI function entry by name.
func (c *Caller) findFunction(name string) *ABIEntry {
	for i := range c.abi {
		if c.abi[i].Type == "function" && c.abi[i].Name == name {
			return &c.abi[i]
		}
	}
	return nil
}

// --- ABI encoding (simplified, for common types) ---

// encodeCall builds calldata: 4-byte selector + encoded args.
func encodeCall(fn *ABIEntry, args []string) (string, error) {
	selector := functionSelector(fn)

	var encoded strings.Builder
	encoded.WriteString(selector)

	for i, param := range fn.Inputs {
		var argStr string
		if i < len(args) {
			argStr = args[i]
		}
		enc, err := encodeParam(param.Type, argStr)
		if err != nil {
			return "", fmt.Errorf("encoding param %s: %w", param.Name, err)
		}
		encoded.WriteString(enc)
	}

	return encoded.String(), nil
}

// functionSelector returns the 4-byte selector for a function.
// Delegates to ABIEntry.Selector() which is defined in registry.go.
func functionSelector(fn *ABIEntry) string {
	return fn.Selector()
}

// EncodeCalldata builds ABI calldata for a function and returns both the
// 0x-prefixed hex string (for eth_estimateGas) and the raw bytes (for tx Data).
func EncodeCalldata(fn ABIEntry, args []string) (hexStr string, raw []byte, err error) {
	hexStr, err = encodeCall(&fn, args)
	if err != nil {
		return
	}
	raw, err = hex.DecodeString(strings.TrimPrefix(hexStr, "0x"))
	if err != nil {
		err = fmt.Errorf("encoding calldata: %w", err)
	}
	return
}

// encodeParam encodes a single ABI parameter value as a 32-byte hex word.
func encodeParam(typ, val string) (string, error) {
	val = strings.TrimPrefix(val, "0x")

	switch {
	case typ == "address":
		// Pad to 32 bytes (left-padded with zeros).
		padded := fmt.Sprintf("%064s", val)
		return padded, nil

	case strings.HasPrefix(typ, "uint") || strings.HasPrefix(typ, "int"):
		n := new(big.Int)
		if _, ok := n.SetString(val, 0); !ok {
			return "", fmt.Errorf("invalid integer: %s", val)
		}
		return fmt.Sprintf("%064x", n), nil

	case typ == "bool":
		if val == "true" || val == "1" {
			return fmt.Sprintf("%064d", 1), nil
		}
		return fmt.Sprintf("%064d", 0), nil

	case typ == "bytes32":
		padded := fmt.Sprintf("%-64s", val)
		return padded[:64], nil

	default:
		// For string/bytes/arrays, use zero for now.
		return fmt.Sprintf("%064d", 0), nil
	}
}

// ── Constructor encoding (for contract deploy) ──────────────────────────────

// EncodeConstructorArgs ABI-encodes constructor arguments and returns the raw
// bytes (no 4-byte selector — constructors don't have one). The result is
// appended directly to the deployment bytecode.
func EncodeConstructorArgs(params []ABIParam, args []string) ([]byte, error) {
	if len(params) != len(args) {
		return nil, fmt.Errorf("constructor expects %d args, got %d", len(params), len(args))
	}
	if len(params) == 0 {
		return nil, nil
	}

	// Check for unsupported types (v1 limitation).
	for _, p := range params {
		if strings.Contains(p.Type, "[") || strings.Contains(p.Type, "tuple") {
			return nil, fmt.Errorf("constructor param %q has type %q — array and tuple types are not yet supported", p.Name, p.Type)
		}
	}

	// Head section: 32 bytes per param (static value or offset for dynamic).
	// Tail section: variable-length data for dynamic types.
	nParams := len(params)
	head := make([]byte, 0, nParams*32)
	tail := make([]byte, 0)

	for i, p := range params {
		if isDynamicType(p.Type) {
			// Write offset (from start of encoding) to tail data.
			offset := uint64(nParams*32) + uint64(len(tail))
			head = appendUint256Big(head, new(big.Int).SetUint64(offset))
			// Encode dynamic data into tail.
			dynData, err := encodeDynamicParam(p.Type, args[i])
			if err != nil {
				return nil, fmt.Errorf("encoding constructor param %q (%s): %w", p.Name, p.Type, err)
			}
			tail = append(tail, dynData...)
		} else {
			// Static: encode directly into head.
			enc, err := encodeStaticParam(p.Type, args[i])
			if err != nil {
				return nil, fmt.Errorf("encoding constructor param %q (%s): %w", p.Name, p.Type, err)
			}
			head = append(head, enc...)
		}
	}

	return append(head, tail...), nil
}

// isDynamicType returns true for ABI types that use head/tail encoding.
func isDynamicType(typ string) bool {
	return typ == "string" || typ == "bytes"
}

// encodeStaticParam encodes a single static ABI value as exactly 32 bytes.
func encodeStaticParam(typ, val string) ([]byte, error) {
	switch {
	case typ == "address":
		v := strings.TrimPrefix(strings.TrimSpace(val), "0x")
		b, err := hex.DecodeString(v)
		if err != nil || len(b) != 20 {
			return nil, fmt.Errorf("invalid address %q", val)
		}
		word := make([]byte, 32)
		copy(word[12:], b)
		return word, nil

	case strings.HasPrefix(typ, "uint") || strings.HasPrefix(typ, "int"):
		n := new(big.Int)
		if _, ok := n.SetString(strings.TrimSpace(val), 0); !ok {
			return nil, fmt.Errorf("invalid integer %q", val)
		}
		return padInt256(n), nil

	case typ == "bool":
		v := strings.ToLower(strings.TrimSpace(val))
		word := make([]byte, 32)
		if v == "true" || v == "1" {
			word[31] = 1
		} else if v != "false" && v != "0" {
			return nil, fmt.Errorf("invalid bool %q", val)
		}
		return word, nil

	case typ == "bytes32":
		v := strings.TrimPrefix(strings.TrimSpace(val), "0x")
		b, err := hex.DecodeString(v)
		if err != nil {
			return nil, fmt.Errorf("invalid bytes32 %q", val)
		}
		word := make([]byte, 32)
		copy(word, b) // right-padded
		return word, nil

	default:
		return nil, fmt.Errorf("unsupported static type %q", typ)
	}
}

// encodeDynamicParam encodes a dynamic ABI value (string or bytes) as
// length-prefixed data padded to 32-byte boundaries.
func encodeDynamicParam(typ, val string) ([]byte, error) {
	switch typ {
	case "string":
		return encodeBytesData([]byte(val)), nil
	case "bytes":
		v := strings.TrimPrefix(strings.TrimSpace(val), "0x")
		b, err := hex.DecodeString(v)
		if err != nil {
			return nil, fmt.Errorf("invalid bytes hex %q", val)
		}
		return encodeBytesData(b), nil
	default:
		return nil, fmt.Errorf("unsupported dynamic type %q", typ)
	}
}

// encodeBytesData returns ABI-encoded bytes/string data:
// 32-byte length prefix + data padded to next 32-byte boundary.
func encodeBytesData(data []byte) []byte {
	length := len(data)
	padded := roundUp32Bytes(length)
	out := make([]byte, 32+padded)
	// Length prefix.
	big.NewInt(int64(length)).FillBytes(out[:32])
	// Data.
	copy(out[32:], data)
	return out
}

// appendUint256Big appends a big.Int as a 32-byte big-endian word.
func appendUint256Big(buf []byte, n *big.Int) []byte {
	word := make([]byte, 32)
	n.FillBytes(word)
	return append(buf, word...)
}

// padInt256 returns a big.Int as a 32-byte word (two's complement for negatives).
func padInt256(n *big.Int) []byte {
	word := make([]byte, 32)
	if n.Sign() >= 0 {
		n.FillBytes(word)
	} else {
		// Two's complement: 2^256 + n
		mod := new(big.Int).Lsh(big.NewInt(1), 256)
		pos := new(big.Int).Add(mod, n)
		pos.FillBytes(word)
	}
	return word
}

// roundUp32Bytes rounds n up to the next multiple of 32.
func roundUp32Bytes(n int) int {
	if n%32 == 0 {
		return n
	}
	return n + (32 - n%32)
}

// decodeResult decodes the raw hex result into string values.
func decodeResult(fn *ABIEntry, hexData string) ([]string, error) {
	data, err := hex.DecodeString(strings.TrimPrefix(hexData, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decoding hex result: %w", err)
	}

	if len(fn.Outputs) == 0 {
		return nil, nil
	}

	var results []string
	offset := 0

	for _, out := range fn.Outputs {
		if offset+32 > len(data) {
			results = append(results, "")
			continue
		}

		word := data[offset : offset+32]
		offset += 32

		val, err := decodeWord(out.Type, word, data)
		if err != nil {
			results = append(results, "")
			continue
		}
		results = append(results, val)
	}

	return results, nil
}

func decodeWord(typ string, word []byte, fullData []byte) (string, error) {
	switch {
	case typ == "address":
		return "0x" + hex.EncodeToString(word[12:]), nil

	case strings.HasPrefix(typ, "uint") || strings.HasPrefix(typ, "int"):
		n := new(big.Int).SetBytes(word)
		return n.String(), nil

	case typ == "bool":
		if word[31] == 1 {
			return "true", nil
		}
		return "false", nil

	case typ == "string":
		// String uses an offset + length encoding.
		offsetVal := new(big.Int).SetBytes(word).Uint64()
		if int(offsetVal)+32 > len(fullData) {
			return "", nil
		}
		length := new(big.Int).SetBytes(fullData[offsetVal : offsetVal+32]).Uint64()
		start := offsetVal + 32
		if start+length > uint64(len(fullData)) {
			return "", nil
		}
		return string(fullData[start : start+length]), nil

	default:
		return "0x" + hex.EncodeToString(word), nil
	}
}
