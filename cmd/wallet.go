package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/Mohsinsiddi/w3cli/internal/wallet"
	"github.com/spf13/cobra"
)

var walletKeyFlag string

var walletCmd = &cobra.Command{
	Use:   "wallet",
	Short: "Manage wallets",
}

var walletAddCmd = &cobra.Command{
	Use:   "add <name> [address]",
	Short: "Add a wallet",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr := newWalletManager()

		if walletKeyFlag != "" {
			// Signing wallet.
			if err := mgr.AddWithKey(name, walletKeyFlag); err != nil {
				return err
			}
			w, _ := mgr.Get(name)
			fmt.Println(ui.Success(fmt.Sprintf("Signing wallet %q added: %s", name, ui.Addr(w.Address))))
		} else {
			if len(args) < 2 {
				return fmt.Errorf("address required for watch-only wallet (or use --key for signing wallet)")
			}
			address := args[1]
			if err := mgr.Add(name, &wallet.Wallet{
				Name:    name,
				Address: address,
				Type:    wallet.TypeWatchOnly,
			}); err != nil {
				return err
			}
			fmt.Println(ui.Success(fmt.Sprintf("Watch-only wallet %q added: %s", name, ui.Addr(address))))
		}
		return nil
	},
}

var walletListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all wallets",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := newWalletManager()
		wallets := mgr.List()

		if len(wallets) == 0 {
			fmt.Println(ui.Meta("No wallets configured. Run `w3cli wallet add <name> <address>`."))
			return nil
		}

		t := ui.NewTable([]ui.Column{
			{Title: "Name", Width: 16},
			{Title: "Address", Width: 44},
			{Title: "Type", Width: 12},
			{Title: "Default", Width: 8},
		})

		for _, w := range wallets {
			def := ""
			if w.IsDefault {
				def = ui.Success("✓")
			}
			t.AddRow(ui.Row{
				ui.Val(w.Name),
				ui.Addr(w.Address),
				ui.Meta(w.Type),
				def,
			})
		}
		fmt.Println(t.Render())
		return nil
	},
}

var walletRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a wallet",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !ui.ConfirmDanger(fmt.Sprintf("Remove wallet %q?", name)) {
			fmt.Println(ui.Meta("Cancelled."))
			return nil
		}
		mgr := newWalletManager()
		if err := mgr.Remove(name); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("Wallet %q removed.", name)))
		return nil
	},
}

var walletUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the default wallet",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr := newWalletManager()
		if err := mgr.SetDefault(name); err != nil {
			return err
		}
		cfg.DefaultWallet = name
		cfg.Save() //nolint:errcheck
		fmt.Println(ui.Success(fmt.Sprintf("Default wallet set to %q.", name)))
		return nil
	},
}

func init() {
	walletAddCmd.Flags().StringVar(&walletKeyFlag, "key", "", "private key for signing wallet (stored in OS keychain)")
	walletCmd.AddCommand(walletAddCmd, walletListCmd, walletRemoveCmd, walletUseCmd)
}

// newWalletManager creates a Manager backed by the config-dir JSON store.
func newWalletManager() *wallet.Manager {
	store := wallet.NewJSONStore(filepath.Join(cfg.Dir(), "wallets.json"))
	return wallet.NewManager(wallet.WithStore(store))
}
