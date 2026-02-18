package cmd

import (
	"fmt"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	txNetwork string
)

var txCmd = &cobra.Command{
	Use:   "tx <hash>",
	Short: "Show transaction details",
	Long: `Show details for a single transaction by hash.

Uses the configured network mode (mainnet/testnet) by default.
Override per-call with --testnet or --mainnet.

Examples:
  w3cli tx 0xHASH --network base
  w3cli tx 0xHASH --network ethereum --testnet`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hash := args[0]
		chainName := txNetwork
		if chainName == "" {
			chainName = cfg.DefaultNetwork
		}

		networkMode := cfg.NetworkMode
		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q — run `w3cli network list` to see all chains", chainName)
		}

		spin := ui.NewSpinner(fmt.Sprintf("Fetching transaction on %s (%s)...", ui.ChainName(chainName), networkMode))
		spin.Start()

		rpcURL, err := pickBestRPC(c, networkMode)
		if err != nil {
			spin.Stop()
			return err
		}

		client := chain.NewEVMClient(rpcURL)
		tx, err := client.GetTransactionByHash(hash)
		spin.Stop()
		if err != nil {
			return err
		}

		explorer := c.Explorer(networkMode)

		fmt.Println(ui.KeyValueBlock(
			fmt.Sprintf("Transaction Details · %s (%s)", c.DisplayName, networkMode),
			[][2]string{
				{"Hash", ui.Addr(tx.Hash)},
				{"From", ui.Addr(tx.From)},
				{"To", ui.Addr(tx.To)},
				{"Value", tx.ValueETH + " " + c.NativeCurrency},
				{"Gas Limit", fmt.Sprintf("%d", tx.Gas)},
				{"Block", fmt.Sprintf("%d", tx.BlockNum)},
				{"Nonce", fmt.Sprintf("%d", tx.Nonce)},
				{"Explorer", explorer + "/tx/" + tx.Hash},
			},
		))
		return nil
	},
}

func init() {
	txCmd.Flags().StringVar(&txNetwork, "network", "", "chain (default: config)")
}
