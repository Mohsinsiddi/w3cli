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
	Use:   "txs [wallet-name-or-address]",
	Short: "List recent transactions",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 && txsWallet == "" {
			txsWallet = args[0]
		}
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

		var txs []*chain.Transaction

		// Try Etherscan-compatible explorer API first (has full history).
		explorerAPI := c.ExplorerAPIURL(cfg.NetworkMode)
		if explorerAPI != "" {
			txs, err = chain.GetTransactionsFromExplorer(explorerAPI, address, txsLast)
			if err != nil {
				// Fall back to block scan — log the reason as a dim note.
				spin.Stop()
				spin = ui.NewSpinner(fmt.Sprintf("Explorer unavailable (%v) — scanning recent blocks...", err))
				spin.Start()
				err = nil // reset for fallback path
			}
		}

		// Fallback: scan last N blocks via RPC (much less complete).
		if txs == nil {
			rpcURL, rpcErr := pickBestRPC(c, cfg.NetworkMode)
			if rpcErr != nil {
				spin.Stop()
				return rpcErr
			}
			client := chain.NewEVMClient(rpcURL)
			txs, err = client.GetRecentTransactions(address, txsLast)
		}

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
