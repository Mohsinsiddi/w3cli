package cmd

import (
	"fmt"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	txsNetwork string
	txsLast    int
	txsWallet  string
)

var txsCmd = &cobra.Command{
	Use:   "txs",
	Short: "List recent transactions",
	RunE: func(cmd *cobra.Command, args []string) error {
		address, chainName, err := resolveWalletAndChain(txsWallet, txsNetwork)
		if err != nil {
			return err
		}

		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q", chainName)
		}

		spin := ui.NewSpinner(fmt.Sprintf("Fetching last %d transactions on %s...", txsLast, ui.ChainName(chainName)))
		spin.Start()

		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
		if err != nil {
			spin.Stop()
			return err
		}

		client := chain.NewEVMClient(rpcURL)
		txs, err := client.GetRecentTransactions(address, txsLast)
		spin.Stop()
		if err != nil {
			return err
		}

		if len(txs) == 0 {
			fmt.Println(ui.Meta("No recent transactions found."))
			return nil
		}

		t := ui.NewTable([]ui.Column{
			{Title: "Hash", Width: 14},
			{Title: "From", Width: 14},
			{Title: "To", Width: 14},
			{Title: "Value (ETH)", Width: 22},
			{Title: "Block", Width: 10},
		})

		explorer := c.Explorer(cfg.NetworkMode)

		for _, tx := range txs {
			valueStr := "0"
			if tx.ValueETH != "" {
				valueStr = tx.ValueETH
			}
			t.AddRow(ui.Row{
				ui.TruncateAddr(tx.Hash),
				ui.TruncateAddr(tx.From),
				ui.TruncateAddr(tx.To),
				valueStr,
				fmt.Sprintf("%d", tx.BlockNum),
			})
		}

		fmt.Printf("%s  %s\n\n", ui.StyleTitle.Render("Recent Transactions"), ui.Meta(fmt.Sprintf("(%s, %s)", chainName, cfg.NetworkMode)))
		fmt.Println(t.Render())
		fmt.Println(ui.Meta(fmt.Sprintf("Explorer: %s/address/%s", explorer, address)))
		return nil
	},
}

func init() {
	txsCmd.Flags().StringVar(&txsWallet, "wallet", "", "wallet name or address")
	txsCmd.Flags().StringVar(&txsNetwork, "network", "", "chain to query")
	txsCmd.Flags().IntVar(&txsLast, "last", 10, "number of transactions to show")
}
