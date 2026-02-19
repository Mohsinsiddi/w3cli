package cmd

import (
	"fmt"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	blockNetwork string
)

var blockCmd = &cobra.Command{
	Use:   "block [chain]",
	Short: "Show latest block details for a chain",
	Long: `Fetch and display the latest block header for a chain.

Defaults to the configured default network. Override with --network.

Examples:
  w3cli block
  w3cli block ethereum
  w3cli block base --testnet
  w3cli block arbitrum --mainnet`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chainName := cfg.DefaultNetwork
		if blockNetwork != "" {
			chainName = blockNetwork
		}
		if len(args) == 1 {
			chainName = args[0]
		}
		if chainName == "" {
			chainName = "ethereum"
		}

		mode := cfg.NetworkMode

		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q — run `w3cli network list` to see all chains", chainName)
		}

		spin := ui.NewSpinner(fmt.Sprintf(
			"Fetching latest block on %s (%s)…", ui.ChainName(chainName), mode,
		))
		spin.Start()

		rpcURL, err := pickBestRPC(c, mode)
		if err != nil {
			spin.Stop()
			return err
		}

		block, err := chain.NewEVMClient(rpcURL).GetLatestBlockInfo()
		spin.Stop()
		if err != nil {
			return fmt.Errorf("fetching block: %w", err)
		}

		// Timestamp
		ts := "—"
		if block.Timestamp > 0 {
			t := time.Unix(int64(block.Timestamp), 0)
			ts = t.UTC().Format("2006-01-02 15:04:05 UTC") + "  (" + block.Age() + ")"
		}

		// Gas
		gasStr := fmt.Sprintf(
			"%s / %s  (%s)",
			commaSep(block.GasUsed),
			commaSep(block.GasLimit),
			block.GasUsedPct(),
		)

		// Base fee
		baseFeeStr := "—  (legacy / pre-EIP-1559)"
		if block.BaseFee != nil {
			baseFeeStr = fmt.Sprintf("%.4f Gwei", chain.WeiToGwei(block.BaseFee))
		}

		// Miner
		miner := block.Miner
		if miner == "" {
			miner = "—"
		}

		title := fmt.Sprintf("🧱 Latest Block  ·  %s · %s", chainName, mode)
		pairs := [][2]string{
			{"Block", fmt.Sprintf("#%s", commaSep(block.Number))},
			{"Hash", block.Hash},
			{"Timestamp", ts},
			{"Transactions", fmt.Sprintf("%d", block.TxCount)},
			{"Gas Used / Limit", gasStr},
			{"Base Fee", baseFeeStr},
			{"Miner / Validator", ui.TruncateAddr(miner)},
		}

		fmt.Println(ui.KeyValueBlock(title, pairs))
		return nil
	},
}

// commaSep formats a uint64 with comma thousands separators.
func commaSep(n uint64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	result := make([]byte, 0, len(s)+len(s)/3)
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(ch))
	}
	return string(result)
}

func init() {
	blockCmd.Flags().StringVar(&blockNetwork, "network", "", "chain to query")
}
