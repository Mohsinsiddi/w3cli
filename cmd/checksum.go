package cmd

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
)

var checksumCmd = &cobra.Command{
	Use:   "checksum <address>",
	Short: "Validate or convert an address to EIP-55 checksum format",
	Long: `Convert any Ethereum address to its EIP-55 checksummed form and
validate if the input was already correctly checksummed.

Examples:
  w3cli checksum 0xd8da6bf26964af9d7eed9e03e53415d37aa96045
  w3cli checksum 0xD8DA6BF26964AF9D7EED9E03E53415D37AA96045`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]

		clean := strings.TrimPrefix(strings.TrimPrefix(input, "0x"), "0X")
		if len(clean) != 40 {
			return fmt.Errorf("invalid address length: expected 40 hex chars, got %d", len(clean))
		}

		// Validate hex.
		if _, err := hex.DecodeString(clean); err != nil {
			return fmt.Errorf("invalid hex address: %w", err)
		}

		checksummed := toChecksumAddress(clean)
		wasValid := input == checksummed

		pairs := [][2]string{
			{"Input", input},
			{"Checksummed", ui.Addr(checksummed)},
		}

		if wasValid {
			pairs = append(pairs, [2]string{"Valid", ui.Success("address is correctly checksummed")})
		} else if strings.EqualFold(input, checksummed) {
			pairs = append(pairs, [2]string{"Valid", ui.Warn("valid address but not checksummed")})
		} else {
			pairs = append(pairs, [2]string{"Valid", ui.Err("checksum mismatch")})
		}

		fmt.Println(ui.KeyValueBlock("EIP-55 Checksum", pairs))
		return nil
	},
}

// toChecksumAddress implements EIP-55 mixed-case checksum encoding.
func toChecksumAddress(addr string) string {
	lower := strings.ToLower(addr)

	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(lower))
	hash := hex.EncodeToString(h.Sum(nil))

	var result strings.Builder
	result.WriteString("0x")
	for i, c := range lower {
		if c >= '0' && c <= '9' {
			result.WriteByte(byte(c))
		} else {
			// If the corresponding nibble in the hash is >= 8, uppercase it.
			if hash[i] >= '8' {
				result.WriteByte(byte(c - 32)) // to uppercase
			} else {
				result.WriteByte(byte(c))
			}
		}
	}
	return result.String()
}

func init() {
	// No flags — positional arg only.
}
