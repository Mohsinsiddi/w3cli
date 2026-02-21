package cmd

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/contract"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
)

var encodeCmd = &cobra.Command{
	Use:   "encode <signature> [args...]",
	Short: "Encode calldata from a function signature and arguments",
	Long: `Build ABI-encoded calldata from a function signature and arguments.

This is the reverse of the decode command. Useful for building calldata
for multisigs, timelocks, or manual eth_call/eth_sendTransaction.

Examples:
  w3cli encode "transfer(address,uint256)" 0xRecipient 1000000000000000000
  w3cli encode "approve(address,uint256)" 0xSpender 115792089237316195423570985008687907853269984665640564039457584007913129639935
  w3cli encode "balanceOf(address)" 0xAddress`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sig := args[0]
		funcArgs := args[1:]

		// Parse signature into function name and param types.
		parenIdx := strings.Index(sig, "(")
		if parenIdx < 0 || !strings.HasSuffix(sig, ")") {
			return fmt.Errorf("invalid signature %q — expected format: name(type1,type2)", sig)
		}

		name := sig[:parenIdx]
		typeStr := sig[parenIdx+1 : len(sig)-1]

		// Compute selector.
		canonical := normalizeSignature(sig)
		h := sha3.NewLegacyKeccak256()
		h.Write([]byte(canonical))
		selectorBytes := h.Sum(nil)[:4]
		selector := "0x" + hex.EncodeToString(selectorBytes)

		// Build ABI params.
		var inputs []contract.ABIParam
		if typeStr != "" {
			for _, t := range strings.Split(typeStr, ",") {
				t = strings.TrimSpace(t)
				// Take only the type (first word), ignore names.
				parts := strings.Fields(t)
				if len(parts) > 0 {
					inputs = append(inputs, contract.ABIParam{Name: "", Type: parts[0]})
				}
			}
		}

		entry := contract.ABIEntry{
			Name:            name,
			Type:            "function",
			Inputs:          inputs,
			StateMutability: "nonpayable",
		}

		calldataHex, calldataRaw, err := contract.EncodeCalldata(entry, funcArgs)
		if err != nil {
			return fmt.Errorf("encoding failed: %w", err)
		}

		pairs := [][2]string{
			{"Signature", canonical},
			{"Selector", selector},
		}

		for i, arg := range funcArgs {
			typ := ""
			if i < len(inputs) {
				typ = inputs[i].Type
			}
			pairs = append(pairs, [2]string{fmt.Sprintf("Arg[%d] (%s)", i, typ), arg})
		}

		pairs = append(pairs, [2]string{"Calldata", ui.Val(calldataHex)})
		pairs = append(pairs, [2]string{"Bytes", fmt.Sprintf("%d", len(calldataRaw))})

		fmt.Println(ui.KeyValueBlock("Encoded Calldata", pairs))
		return nil
	},
}

func init() {
	// No flags — positional args only.
}
