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
	wei, err := ethToWei(amountStr)
	if err != nil {
		return fmt.Errorf("invalid amount: %s", amountStr)
	}

	// Derive gwei from wei (exact integer division).
	weiPerGwei := new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)
	gweiWhole := new(big.Int).Div(wei, weiPerGwei)
	gweiRem := new(big.Int).Mod(wei, weiPerGwei)

	gweiStr := gweiWhole.String()
	if gweiRem.Sign() > 0 {
		frac := fmt.Sprintf("%09d", gweiRem.Uint64())
		frac = strings.TrimRight(frac, "0")
		gweiStr += "." + frac
	}

	fmt.Println(ui.KeyValueBlock("Unit Conversion", [][2]string{
		{"Input", ui.Val(amountStr + " ETH")},
		{"Gwei", ui.Val(gweiStr + " gwei")},
		{"Wei", ui.Val(wei.String() + " wei")},
		{"Hex", ui.Val("0x" + wei.Text(16))},
	}))
	return nil
}

func convertFromGwei(amountStr string) error {
	// Use string-based scaling: gwei → wei is ×10^9.
	parts := strings.SplitN(strings.TrimSpace(amountStr), ".", 2)
	wholePart := parts[0]
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}
	if wholePart == "" {
		wholePart = "0"
	}
	if len(fracPart) > 9 {
		fracPart = fracPart[:9]
	}
	for len(fracPart) < 9 {
		fracPart += "0"
	}
	weiStr := strings.TrimLeft(wholePart+fracPart, "0")
	if weiStr == "" {
		weiStr = "0"
	}
	wei, ok := new(big.Int).SetString(weiStr, 10)
	if !ok {
		return fmt.Errorf("invalid amount: %s", amountStr)
	}

	// Derive ETH from wei (exact).
	weiPerETH := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	ethF := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetInt(weiPerETH))

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
