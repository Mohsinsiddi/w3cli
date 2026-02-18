package contract

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"golang.org/x/crypto/sha3"
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

// functionSelector computes the 4-byte selector for a function.
func functionSelector(fn *ABIEntry) string {
	sig := fn.Name + "("
	types := make([]string, len(fn.Inputs))
	for i, p := range fn.Inputs {
		types[i] = p.Type
	}
	sig += strings.Join(types, ",") + ")"

	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(sig))
	return "0x" + hex.EncodeToString(h.Sum(nil)[:4])
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
