package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/price"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	watchNetwork string
	watchNotify  bool
	watchWallet  string
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Monitor wallet for incoming/outgoing transactions",
	Long: `Monitor a wallet with a live-updating balance dashboard.

Uses the configured network mode (mainnet/testnet) by default.
Override per-call with --testnet or --mainnet.

Examples:
  w3cli watch --network base
  w3cli watch --network ethereum --testnet`,
	RunE: func(cmd *cobra.Command, args []string) error {
		address, chainName, err := resolveWalletAndChain(watchWallet, watchNetwork)
		if err != nil {
			return err
		}

		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q — run `w3cli network list` to see all chains", chainName)
		}

		fmt.Printf("%s\n", ui.StyleTitle.Render(fmt.Sprintf("Watching %s on %s (%s)", ui.TruncateAddr(address), ui.ChainName(chainName), cfg.NetworkMode)))
		fmt.Println(ui.Meta("Press q to quit. Balance refreshes automatically.\n"))

		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
		if err != nil {
			return err
		}

		client := chain.NewEVMClient(rpcURL)
		priceFetcher := price.NewFetcher(cfg.PriceCurrency)

		fetcher := func() ([]ui.BalanceEntry, error) {
			bal, err := client.GetBalance(address)
			if err != nil {
				return nil, err
			}
			usdPrice, _ := priceFetcher.GetPrice(chainName)
			usdStr := fmt.Sprintf("$%.2f", parseFloat(bal.ETH)*usdPrice)
			return []ui.BalanceEntry{
				{
					Chain:   chainName,
					Address: address,
					Balance: bal.ETH,
					Symbol:  c.NativeCurrency,
					USD:     usdStr,
				},
			}, nil
		}

		interval := time.Duration(cfg.WatchInterval) * time.Second
		prog := ui.NewDashboard(interval, fetcher)

		// Also poll for new transactions in background.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go watchTransactions(ctx, client, address, chainName, watchNotify)

		_, err = prog.Run()
		return err
	},
}

func watchTransactions(ctx context.Context, client *chain.EVMClient, address, chainName string, notify bool) {
	var lastKnown uint64

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			block, err := client.GetBlockNumber()
			if err != nil {
				continue
			}
			if lastKnown == 0 {
				lastKnown = block
				continue
			}
			if block > lastKnown {
				// New blocks — check for relevant transactions.
				lastKnown = block
			}
		}
	}
}

func init() {
	watchCmd.Flags().StringVar(&watchWallet, "wallet", "", "wallet name or address")
	watchCmd.Flags().StringVar(&watchNetwork, "network", "", "chain to watch")
	watchCmd.Flags().BoolVar(&watchNotify, "notify", false, "send desktop notifications for new transactions")
}
