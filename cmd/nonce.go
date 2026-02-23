package cmd

import (
	"fmt"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	nonceWallet  string
	nonceNetwork string
)

var nonceCmd = &cobra.Command{
	Use:   "nonce",
	Short: "Show confirmed and pending nonce for a wallet",
	Long: `Query the confirmed and pending transaction count (nonce) for a wallet.

If confirmed and pending nonces differ, it means transactions are pending
or stuck in the mempool.

Examples:
  w3cli nonce
  w3cli nonce --wallet myWallet --network ethereum`,
	RunE: func(cmd *cobra.Command, args []string) error {
		address, chainName, err := resolveWalletAndChain(nonceWallet, nonceNetwork)
		if err != nil {
			return err
		}

		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q — run `w3cli network list`", chainName)
		}

		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
		if err != nil {
			return err
		}

		client := chain.NewEVMClient(rpcURL)

		spin := ui.NewSpinner(fmt.Sprintf("Querying nonce on %s...", c.DisplayName))
		spin.Start()

		confirmed, err := client.GetNonce(address)
		if err != nil {
			spin.Stop()
			return fmt.Errorf("getting confirmed nonce: %w", err)
		}

		pending, err := client.GetPendingNonce(address)
		if err != nil {
			spin.Stop()
			return fmt.Errorf("getting pending nonce: %w", err)
		}
		spin.Stop()

		pairs := [][2]string{
			{"Wallet", ui.Addr(address)},
			{"Network", fmt.Sprintf("%s (%s)", c.DisplayName, cfg.NetworkMode)},
			{"Confirmed Nonce", ui.Val(fmt.Sprintf("%d", confirmed))},
			{"Pending Nonce", ui.Val(fmt.Sprintf("%d", pending))},
		}

		if pending > confirmed {
			pairs = append(pairs, [2]string{"Status", ui.Warn(fmt.Sprintf("%d pending tx(s) in mempool", pending-confirmed))})
		} else {
			pairs = append(pairs, [2]string{"Status", ui.Success("no pending transactions")})
		}

		fmt.Println(ui.KeyValueBlock("Nonce", pairs))
		return nil
	},
}

func init() {
	nonceCmd.Flags().StringVar(&nonceWallet, "wallet", "", "wallet name or address")
	nonceCmd.Flags().StringVar(&nonceNetwork, "network", "", "chain to query (default: config)")
}
