package cmd

import (
	"fmt"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var codeNetwork string

var codeCmd = &cobra.Command{
	Use:   "code <address>",
	Short: "Check if an address is a contract (has bytecode) or an EOA",
	Long: `Query the bytecode at an address to determine if it's a smart contract
or an externally-owned account (EOA).

Shows the bytecode size and a preview of the first bytes.

Examples:
  w3cli code 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48   # USDC (contract)
  w3cli code 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045   # vitalik (EOA)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		address := args[0]

		chainName := codeNetwork
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

		spin := ui.NewSpinner(fmt.Sprintf("Querying bytecode on %s...", c.DisplayName))
		spin.Start()
		code, err := client.GetCode(address)
		spin.Stop()
		if err != nil {
			return fmt.Errorf("querying code: %w", err)
		}

		clean := strings.TrimPrefix(code, "0x")
		isContract := len(clean) > 0 && clean != "0"
		byteLen := len(clean) / 2

		pairs := [][2]string{
			{"Address", ui.Addr(address)},
			{"Network", fmt.Sprintf("%s (%s)", c.DisplayName, cfg.NetworkMode)},
		}

		if isContract {
			pairs = append(pairs, [2]string{"Type", ui.Val("Contract")})
			pairs = append(pairs, [2]string{"Bytecode Size", ui.Val(fmt.Sprintf("%d bytes", byteLen))})

			// Show first 32 bytes as preview.
			preview := clean
			if len(preview) > 64 {
				preview = preview[:64] + "..."
			}
			pairs = append(pairs, [2]string{"Preview", "0x" + preview})
		} else {
			pairs = append(pairs, [2]string{"Type", ui.Val("EOA (no code)")})
		}

		fmt.Println(ui.KeyValueBlock("Address Type", pairs))
		return nil
	},
}

func init() {
	codeCmd.Flags().StringVar(&codeNetwork, "network", "", "chain (default: config)")
}
