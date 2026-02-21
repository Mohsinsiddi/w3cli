package cmd

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
)

var selectorCmd = &cobra.Command{
	Use:   "selector <signature-or-selector>",
	Short: "Compute or look up a 4-byte function selector",
	Long: `Compute a 4-byte function selector from a canonical signature,
or look up a known selector in the built-in database.

Examples:
  w3cli selector "transfer(address,uint256)"     # → 0xa9059cbb
  w3cli selector "approve(address,uint256)"       # → 0x095ea7b3
  w3cli selector "balanceOf(address)"             # → 0x70a08231
  w3cli selector 0xa9059cbb                       # → transfer`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]

		// If input starts with 0x, it's a selector to look up.
		if strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X") {
			methodName := chain.DecodeMethod(input)
			pairs := [][2]string{
				{"Selector", input},
				{"Method", ui.Val(methodName)},
			}
			fmt.Println(ui.KeyValueBlock("Selector Lookup", pairs))
			return nil
		}

		// Otherwise, compute selector from signature.
		sig := normalizeSignature(input)

		h := sha3.NewLegacyKeccak256()
		h.Write([]byte(sig))
		hash := h.Sum(nil)
		selector := "0x" + hex.EncodeToString(hash[:4])

		pairs := [][2]string{
			{"Signature", sig},
			{"Selector", ui.Val(selector)},
			{"Full Hash", "0x" + hex.EncodeToString(hash)},
		}

		fmt.Println(ui.KeyValueBlock("Function Selector", pairs))
		return nil
	},
}

// normalizeSignature removes parameter names, keeping only types.
// "transfer(address to, uint256 amount)" → "transfer(address,uint256)"
func normalizeSignature(sig string) string {
	parenIdx := strings.Index(sig, "(")
	if parenIdx < 0 {
		return sig
	}

	name := sig[:parenIdx]
	paramStr := sig[parenIdx+1 : len(sig)-1]

	if paramStr == "" {
		return name + "()"
	}

	params := strings.Split(paramStr, ",")
	var types []string
	for _, p := range params {
		p = strings.TrimSpace(p)
		// Take only the first word (the type), skip the name.
		parts := strings.Fields(p)
		if len(parts) > 0 {
			types = append(types, parts[0])
		}
	}

	return name + "(" + strings.Join(types, ",") + ")"
}

func init() {
	// No flags — positional arg only.
}
