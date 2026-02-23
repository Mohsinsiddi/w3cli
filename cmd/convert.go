package cmd

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var convertCmd = &cobra.Command{
	Use:   "convert <amount> [unit]",
	Short: "Convert between ETH, Gwei, Wei, and hex/decimal",
	Long: `Convert between Ethereum denomination units and hex/decimal formats.

Units: eth, gwei, wei, hex, decimal
If no unit is given and the value starts with 0x, it's treated as hex.

Examples:
  w3cli convert 1.5 eth          # → gwei + wei
  w3cli convert 50 gwei          # → eth + wei
  w3cli convert 1000000000 wei   # → eth + gwei
  w3cli convert 0xff             # → 255 (decimal)
  w3cli convert 255 hex          # → 0xff`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		amount := args[0]
		unit := ""
		if len(args) > 1 {
			unit = strings.ToLower(args[1])
		}

		// Auto-detect hex input.
		if unit == "" && strings.HasPrefix(strings.ToLower(amount), "0x") {
			unit = "hex_input"
		}

		switch unit {
		case "eth":
			return convertFromETH(amount)
		case "gwei":
			return convertFromGwei(amount)
		case "wei":
			return convertFromWei(amount)
		case "hex":
			return convertDecToHex(amount)
		case "hex_input":
			return convertHexToDec(amount)
		case "decimal", "dec":
			return convertDecToHex(amount)
		case "":
			// Default: try as wei.
			return convertFromWei(amount)
		default:
			return fmt.Errorf("unknown unit %q — use eth, gwei, wei, hex, or decimal", unit)
		}
	},
}

func convertFromETH(amountStr string) error {
	f, ok := new(big.Float).SetString(amountStr)
	if !ok {
		return fmt.Errorf("invalid amount: %s", amountStr)
	}

	weiPerETH := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	gweiPerETH := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil))

	weiF := new(big.Float).Mul(f, weiPerETH)
	gweiF := new(big.Float).Mul(f, gweiPerETH)

	wei, _ := weiF.Int(nil)
	gwei := gweiF.Text('f', 0)

	fmt.Println(ui.KeyValueBlock("Unit Conversion", [][2]string{
		{"Input", ui.Val(amountStr + " ETH")},
		{"Gwei", ui.Val(gwei + " gwei")},
		{"Wei", ui.Val(wei.String() + " wei")},
		{"Hex", ui.Val("0x" + wei.Text(16))},
	}))
	return nil
}

func convertFromGwei(amountStr string) error {
	f, ok := new(big.Float).SetString(amountStr)
	if !ok {
		return fmt.Errorf("invalid amount: %s", amountStr)
	}

	gweiPerETH := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil))
	weiPerGwei := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil))

	ethF := new(big.Float).Quo(f, gweiPerETH)
	weiF := new(big.Float).Mul(f, weiPerGwei)

	wei, _ := weiF.Int(nil)

	fmt.Println(ui.KeyValueBlock("Unit Conversion", [][2]string{
		{"Input", ui.Val(amountStr + " gwei")},
		{"ETH", ui.Val(ethF.Text('f', 18) + " ETH")},
		{"Wei", ui.Val(wei.String() + " wei")},
		{"Hex", ui.Val("0x" + wei.Text(16))},
	}))
	return nil
}

func convertFromWei(amountStr string) error {
	wei, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return fmt.Errorf("invalid wei amount: %s", amountStr)
	}

	weiPerETH := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	weiPerGwei := new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)

	ethF := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetInt(weiPerETH))
	gweiF := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetInt(weiPerGwei))

	fmt.Println(ui.KeyValueBlock("Unit Conversion", [][2]string{
		{"Input", ui.Val(amountStr + " wei")},
		{"ETH", ui.Val(ethF.Text('f', 18) + " ETH")},
		{"Gwei", ui.Val(gweiF.Text('f', 9) + " gwei")},
		{"Hex", ui.Val("0x" + wei.Text(16))},
	}))
	return nil
}

func convertHexToDec(amountStr string) error {
	clean := strings.TrimPrefix(strings.TrimPrefix(amountStr, "0x"), "0X")
	n, ok := new(big.Int).SetString(clean, 16)
	if !ok {
		return fmt.Errorf("invalid hex value: %s", amountStr)
	}

	fmt.Println(ui.KeyValueBlock("Hex → Decimal", [][2]string{
		{"Hex", ui.Val(amountStr)},
		{"Decimal", ui.Val(n.String())},
	}))
	return nil
}

func convertDecToHex(amountStr string) error {
	n, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return fmt.Errorf("invalid decimal value: %s", amountStr)
	}

	fmt.Println(ui.KeyValueBlock("Decimal → Hex", [][2]string{
		{"Decimal", ui.Val(amountStr)},
		{"Hex", ui.Val("0x" + n.Text(16))},
	}))
	return nil
}

func init() {
	// No flags needed — pure positional args.
}
