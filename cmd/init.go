package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/Mohsinsiddi/w3cli/internal/wallet"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard",
	Long:  "Launch the interactive setup wizard to configure w3cli.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.Banner())

		result, err := ui.RunWizard()
		if err != nil {
			return err
		}

		// Apply wizard results to config.
		if result.DefaultNetwork != "" {
			cfg.DefaultNetwork = result.DefaultNetwork
		}
		if result.NetworkMode != "" {
			cfg.NetworkMode = result.NetworkMode
		}
		if result.RPCAlgorithm != "" {
			cfg.RPCAlgorithm = result.RPCAlgorithm
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		// Add wallet if provided.
		if result.WalletAddress != "" {
			store := wallet.NewJSONStore(filepath.Join(cfg.Dir(), "wallets.json"))
			mgr := wallet.NewManager(wallet.WithStore(store))
			if err := mgr.Add(result.WalletName, &wallet.Wallet{
				Name:      result.WalletName,
				Address:   result.WalletAddress,
				Type:      wallet.TypeWatchOnly,
				IsDefault: true,
			}); err != nil {
				fmt.Println(ui.Warn(fmt.Sprintf("Could not add wallet: %v", err)))
			}
		}

		fmt.Println(ui.Success("w3cli configured! Run `w3cli --help` to explore commands."))
		return nil
	},
}
