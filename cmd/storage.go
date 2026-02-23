package cmd

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var storageNetwork string

var storageCmd = &cobra.Command{
	Use:   "storage <address> <slot>",
	Short: "Read a raw storage slot from a contract",
	Long: `Read the raw 32-byte value at a specific storage slot of a contract.

Storage slots can be specified as decimal numbers or hex (0x-prefixed).
The output shows both hex and decimal interpretations, plus an address
interpretation (useful for proxy implementation slots).

Common slots:
  0    — First state variable
  0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc
         — EIP-1967 implementation slot (proxy contracts)
  0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103
         — EIP-1967 admin slot

Examples:
  w3cli storage 0xContract 0
  w3cli storage 0xProxy 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		address := args[0]
		slot := args[1]

		chainName := storageNetwork
		if chainName == "" {
			chainName = cfg.DefaultNetwork
		}

		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q", chainName)
		}

		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
		if err != nil {
			return err
		}

		client := chain.NewEVMClient(rpcURL)

		// Normalize slot to hex.
		slotHex := slot
		if !strings.HasPrefix(slot, "0x") {
			n, ok := new(big.Int).SetString(slot, 10)
			if !ok {
				return fmt.Errorf("invalid slot %q — use decimal or 0x-prefixed hex", slot)
			}
			slotHex = "0x" + fmt.Sprintf("%064x", n)
		}

		spin := ui.NewSpinner(fmt.Sprintf("Reading storage on %s...", c.DisplayName))
		spin.Start()
		value, err := client.GetStorageAt(address, slotHex)
		spin.Stop()
		if err != nil {
			return fmt.Errorf("reading storage: %w", err)
		}

		// Parse value for display.
		cleanValue := strings.TrimPrefix(value, "0x")
		decValue := new(big.Int)
		decValue.SetString(cleanValue, 16)

		// Check if value looks like an address (12 leading zero bytes + 20 bytes).
		addrStr := ""
		if len(cleanValue) == 64 && strings.HasPrefix(cleanValue, "000000000000000000000000") {
			addr := cleanValue[24:]
			allZero := true
			for _, c := range addr {
				if c != '0' {
					allZero = false
					break
				}
			}
			if !allZero {
				addrStr = "0x" + addr
			}
		}

		pairs := [][2]string{
			{"Contract", ui.Addr(address)},
			{"Slot", slot},
			{"Network", fmt.Sprintf("%s (%s)", c.DisplayName, cfg.NetworkMode)},
			{"Raw (hex)", ui.Val(value)},
			{"Decimal", decValue.String()},
		}

		if addrStr != "" {
			pairs = append(pairs, [2]string{"As Address", ui.Addr(addrStr)})
		}

		// Check if it's all zeros.
		if decValue.Sign() == 0 {
			pairs = append(pairs, [2]string{"Note", ui.Meta("slot is empty (zero)")})
		}

		fmt.Println(ui.KeyValueBlock("Storage Read", pairs))
		return nil
	},
}

func init() {
	storageCmd.Flags().StringVar(&storageNetwork, "network", "", "chain (default: config)")
}
