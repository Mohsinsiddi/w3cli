package cmd

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
)

var keccakCmd = &cobra.Command{
	Use:   "keccak <input>",
	Short: "Compute Keccak-256 hash of text or hex input",
	Long: `Compute the Keccak-256 hash of the given input.

If the input starts with 0x, it's treated as raw hex bytes.
Otherwise, it's treated as a UTF-8 string.

Also shows the 4-byte selector (first 4 bytes) for quick function
selector lookups.

Examples:
  w3cli keccak "transfer(address,uint256)"    # → function selector
  w3cli keccak "Hello, world!"                # → hash of string
  w3cli keccak 0xdeadbeef                     # → hash of raw bytes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]

		var data []byte
		inputType := "text"

		if strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X") {
			// Treat as raw hex bytes.
			raw, err := hex.DecodeString(strings.TrimPrefix(strings.TrimPrefix(input, "0x"), "0X"))
			if err != nil {
				return fmt.Errorf("invalid hex input: %w", err)
			}
			data = raw
			inputType = "hex"
		} else {
			data = []byte(input)
		}

		h := sha3.NewLegacyKeccak256()
		h.Write(data)
		hash := h.Sum(nil)

		hashHex := "0x" + hex.EncodeToString(hash)
		selector := "0x" + hex.EncodeToString(hash[:4])

		pairs := [][2]string{
			{"Input", input},
			{"Type", inputType},
			{"Keccak-256", ui.Val(hashHex)},
			{"Selector (4 bytes)", selector},
		}

		fmt.Println(ui.KeyValueBlock("Keccak-256 Hash", pairs))
		return nil
	},
}

func init() {
	// No flags — positional arg only.
}
