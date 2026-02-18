package cmd

import (
	"fmt"
	"strings"
	"time"

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
	Long: `List recent transactions for a wallet with an interactive TUI.

Uses the configured network mode (mainnet/testnet) by default.
Override per-call with --testnet or --mainnet.

Examples:
  w3cli txs 0xABC... --network base --last 10
  w3cli txs --network ethereum --testnet --last 5`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 && txsWallet == "" {
			txsWallet = args[0]
		}

		networkMode := cfg.NetworkMode
		address, chainName, err := resolveWalletAndChain(txsWallet, txsNetwork)
		if err != nil {
			return err
		}

		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q — run `w3cli network list` to see all chains", chainName)
		}

		spin := ui.NewSpinner(fmt.Sprintf("Fetching last %d transactions on %s (%s)...", txsLast, ui.ChainName(chainName), networkMode))
		spin.Start()

		var txs []*chain.Transaction
		dataSource := "explorer"

		explorerAPI := c.ExplorerAPIURL(networkMode)
		apiKey := cfg.GetExplorerAPIKey(chainName)
		if explorerAPI != "" {
			txs, err = chain.GetTransactionsFromExplorer(explorerAPI, address, txsLast, apiKey)
			if err != nil {
				spin.Stop()
				fmt.Println(ui.Warn(fmt.Sprintf("Explorer unavailable: %v", err)))
				spin = ui.NewSpinner("Falling back to RPC block scanning (slower, limited history)...")
				spin.Start()
				dataSource = "rpc"
				err = nil
			}
		} else {
			dataSource = "rpc"
		}

		if txs == nil {
			rpcURL, rpcErr := pickBestRPC(c, networkMode)
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
			fmt.Println(ui.Info("No recent transactions found."))
			if dataSource == "rpc" {
				fmt.Println(ui.Hint("RPC block scan only checks the last 20 blocks. Older transactions may not appear."))
			}
			return nil
		}

		if dataSource == "rpc" {
			fmt.Println(ui.Info(fmt.Sprintf("Found %d transaction(s) via RPC block scan.", len(txs))))
			fmt.Println(ui.Hint("Block scan only checks the last 20 blocks. For full history, ensure the explorer API is reachable."))
		} else {
			fmt.Println(ui.Info(fmt.Sprintf("Found %d transaction(s) via BlockScout explorer.", len(txs))))
		}

		// Collect unique To addresses for contract name lookup.
		var contractNames map[string]string
		if explorerAPI != "" {
			spin2 := ui.NewSpinner("Resolving contract names...")
			spin2.Start()
			toAddrs := make([]string, 0, len(txs))
			for _, tx := range txs {
				if tx.IsContract && tx.To != "" {
					toAddrs = append(toAddrs, tx.To)
				}
			}
			contractNames = chain.FetchContractNames(explorerAPI, toAddrs, apiKey)
			spin2.Stop()
		}

		// Build table + per-row data for interactivity.
		t := ui.NewTable([]ui.Column{
			{Title: "Hash", Width: 14},
			{Title: "St", Width: 2},
			{Title: "Method", Width: 14},
			{Title: "To / Contract", Width: 20},
			{Title: "Value (ETH)", Width: 20},
			{Title: "Age", Width: 10},
		})

		explorer := c.Explorer(networkMode)
		now := uint64(time.Now().Unix())

		txRowData := make([]ui.TxRow, 0, len(txs))

		for _, tx := range txs {
			// Status icon.
			status := ui.StyleSuccess.Render("v")
			if !tx.Success {
				status = ui.StyleError.Render("x")
			}

			// To label: contract name, or truncated address.
			toLabel := ui.TruncateAddr(tx.To)
			if name, ok := contractNames[strings.ToLower(tx.To)]; ok {
				toLabel = name
				if len(toLabel) > 20 {
					toLabel = toLabel[:18] + ".."
				}
			}

			// Value.
			valueStr := "0"
			if tx.ValueETH != "" && tx.ValueETH != "0.000000000000000000" {
				v := tx.ValueETH
				if len(v) > 20 {
					v = v[:18] + ".."
				}
				valueStr = v
			}

			// Relative age.
			age := ""
			if tx.Timestamp > 0 && now > tx.Timestamp {
				diff := now - tx.Timestamp
				switch {
				case diff < 60:
					age = fmt.Sprintf("%ds ago", diff)
				case diff < 3600:
					age = fmt.Sprintf("%dm ago", diff/60)
				case diff < 86400:
					age = fmt.Sprintf("%dh ago", diff/3600)
				default:
					age = fmt.Sprintf("%dd ago", diff/86400)
				}
			}

			t.AddRow(ui.Row{
				ui.TruncateAddr(tx.Hash),
				status,
				tx.FunctionName,
				toLabel,
				valueStr,
				age,
			})

			explorerURL := ""
			if explorer != "" && tx.Hash != "" {
				explorerURL = explorer + "/tx/" + tx.Hash
			}
			txRowData = append(txRowData, ui.TxRow{
				FullHash:    tx.Hash,
				ExplorerURL: explorerURL,
			})
		}

		shortAddr := address
		if len(shortAddr) > 10 {
			shortAddr = shortAddr[:8] + ".."
		}
		title := fmt.Sprintf("%s  %s",
			ui.StyleTitle.Render("Recent Transactions"),
			ui.Meta(fmt.Sprintf("(%s · %s · %s)", chainName, networkMode, shortAddr)))

		return ui.RunTxList(title, t, txRowData)
	},
}

func init() {
	txsCmd.Flags().StringVar(&txsWallet, "wallet", "", "wallet name or address")
	txsCmd.Flags().StringVar(&txsNetwork, "network", "", "chain to query")
	txsCmd.Flags().IntVar(&txsLast, "last", 10, "number of transactions to show")
}
