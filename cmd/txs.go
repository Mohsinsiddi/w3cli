package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/providers"
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

		chainReg := chain.NewRegistry()
		c, err := chainReg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q — run `w3cli network list` to see all chains", chainName)
		}

		spin := ui.NewSpinner(fmt.Sprintf("Fetching last %d transactions on %s (%s)...", txsLast, ui.ChainName(chainName), networkMode))
		spin.Start()

		// Build provider registry via factory (etherscan → alchemy → moralis → blockscout → ankr → rpc).
		rpcURL, rpcErr := pickBestRPC(c, networkMode)
		if rpcErr != nil {
			spin.Stop()
			return rpcErr
		}
		explorerAPI := c.ExplorerAPIURL(networkMode)
		apiKey := cfg.GetExplorerAPIKey(chainName)

		provReg := providers.BuildRegistry(chainName, c, networkMode, rpcURL, cfg)
		result, _ := provReg.GetTransactions(address, txsLast)

		spin.Stop()

		// Print any non-fatal provider warnings.
		for _, w := range result.Warnings {
			fmt.Println(ui.Warn(w))
		}

		txs := result.Txs
		if len(txs) == 0 {
			fmt.Println(ui.Info("No recent transactions found."))
			// Give chain-specific guidance for chains without a free keyless explorer.
			noExplorer := c.ExplorerAPIURL(networkMode) == ""
			ankrUnconfigured := cfg.GetProviderKey("ankr") == ""
			if noExplorer && ankrUnconfigured {
				fmt.Println(ui.Hint(fmt.Sprintf(
					"For full %s history, add a free Ankr key (ankr.com/rpc/apps):\n  w3cli config set-key ankr <key>",
					chainName,
				)))
			} else {
				fmt.Println(ui.Hint("RPC block scan checks the last 200 blocks. Older transactions may not appear."))
			}
			return nil
		}

		switch result.Source {
		case "rpc":
			fmt.Println(ui.Info(fmt.Sprintf("Found %d transaction(s) via RPC block scan.", len(txs))))
			fmt.Println(ui.Hint("Block scan checks the last 200 blocks. For full history add a provider API key."))
		default:
			fmt.Println(ui.Info(fmt.Sprintf("Found %d transaction(s) via %s.", len(txs), result.Source)))
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
			{Title: "HASH", Width: 14},
			{Title: "ST", Width: 2},
			{Title: "METHOD", Width: 14},
			{Title: "TO / CONTRACT", Width: 20},
			{Title: "VALUE", Width: 16},
			{Title: "AGE", Width: 10},
		})

		explorer := c.Explorer(networkMode)
		now := uint64(time.Now().Unix())

		txRowData := make([]ui.TxRow, 0, len(txs))

		for _, tx := range txs {
			// Status icon.
			status := ui.StyleSuccess.Render("✓")
			if !tx.Success {
				status = ui.StyleError.Render("✗")
			}

			// To label: contract name, or truncated address.
			toLabel := ui.TruncateAddr(tx.To)
			if name, ok := contractNames[strings.ToLower(tx.To)]; ok {
				toLabel = name
				if len(toLabel) > 20 {
					toLabel = toLabel[:18] + ".."
				}
			}

			// Value — show 4 decimal places, trim trailing zeros.
			valueStr := "0.0000 " + c.NativeCurrency
			if tx.ValueETH != "" && tx.ValueETH != "0.000000000000000000" {
				if f := parseFloat(tx.ValueETH); f > 0 {
					valueStr = fmt.Sprintf("%.4f %s", f, c.NativeCurrency)
				}
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

		title := ui.StyleTitle.Render(
			fmt.Sprintf("📋 Recent Transactions  ·  %s  ·  %s · %s",
				ui.TruncateAddr(address), chainName, networkMode),
		)

		return ui.RunTxList(title, t, txRowData)
	},
}

func init() {
	txsCmd.Flags().StringVar(&txsWallet, "wallet", "", "wallet name or address")
	txsCmd.Flags().StringVar(&txsNetwork, "network", "", "chain to query")
	txsCmd.Flags().IntVar(&txsLast, "last", 10, "number of transactions to show")
}
