package cmd

import (
	"fmt"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var decodeCmd = &cobra.Command{
	Use:   "decode <calldata>",
	Short: "Decode EVM calldata into a human-readable method name",
	Long: `Decode raw calldata (hex) into a human-readable function name.

Uses a built-in database of common selectors. No RPC call needed.

Examples:
  w3cli decode 0xa9059cbb000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa960450000000000000000000000000000000000000000000000000de0b6b3a7640000
  w3cli decode 0x095ea7b3`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		calldata := args[0]

		// Validate hex input.
		clean := strings.TrimPrefix(calldata, "0x")
		if len(clean) == 0 {
			return fmt.Errorf("empty calldata — provide a hex string starting with 0x")
		}

		methodName := chain.DecodeMethod(calldata)

		// Extract selector (first 4 bytes = 8 hex chars).
		selector := ""
		if len(clean) >= 8 {
			selector = "0x" + strings.ToLower(clean[:8])
		}

		// Extract raw args (everything after selector).
		rawArgs := ""
		if len(clean) > 8 {
			rawArgs = clean[8:]
		}

		pairs := [][2]string{
			{"Method", ui.Val(methodName)},
		}
		if selector != "" {
			pairs = append(pairs, [2]string{"Selector", selector})
		}
		if rawArgs != "" {
			// Show args split into 32-byte words.
			words := splitHexWords(rawArgs)
			for i, w := range words {
				label := fmt.Sprintf("Arg[%d]", i)
				pairs = append(pairs, [2]string{label, "0x" + w})
			}
		}

		fmt.Println(ui.KeyValueBlock("Decoded Calldata", pairs))
		return nil
	},
}

// splitHexWords splits a hex string into 64-char (32-byte) words.
func splitHexWords(hex string) []string {
	var words []string
	for i := 0; i+64 <= len(hex); i += 64 {
		words = append(words, hex[i:i+64])
	}
	// Trailing partial word.
	if remainder := len(hex) % 64; remainder > 0 && len(hex) > 64 {
		words = append(words, hex[len(hex)-remainder:])
	} else if len(hex) < 64 && len(hex) > 0 {
		words = append(words, hex)
	}
	return words
}

func init() {
	// No flags — positional arg only.
}
