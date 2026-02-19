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
			fmt.Println(ui.Hint(fmt.Sprintf("Set as default with: w3cli wallet use %s", name)))
		} else {
			if len(args) < 2 {
				return fmt.Errorf("address required for watch-only wallet\n  Usage: w3cli wallet add <name> <address>\n  Or for signing: w3cli wallet add <name> --key <private-key>")
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
			fmt.Println(ui.Hint(fmt.Sprintf("Set as default with: w3cli wallet use %s", name)))
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
			fmt.Println(ui.Info("No wallets configured yet."))
			fmt.Println(ui.Hint("Add one with: w3cli wallet add myWallet 0xYourAddress"))
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
				def = ui.StyleSuccess.Render("✓")
			}
			t.AddRow(ui.Row{
				ui.Val(w.Name),
				ui.Addr(w.Address),
				ui.Meta(walletTypeLabel(w.Type)),
				def,
			})
		}
		fmt.Println(t.Render())
		fmt.Println(ui.Meta(fmt.Sprintf("%d wallet(s) configured", len(wallets))))
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
		fmt.Println(ui.Hint("This wallet will be used for all commands when --wallet is not specified."))
		return nil
	},
}

var walletGenerateCmd = &cobra.Command{
	Use:   "generate <name>",
	Short: "Generate a new EVM wallet",
	Long: `Generate a brand-new EVM keypair and store the private key in the OS keychain.

The private key is displayed ONCE immediately after creation.
Copy it and store it in a password manager — if you lose it, the wallet is gone forever.

Re-export later with: w3cli wallet export <name>`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr := newWalletManager()
		w, hexKey, err := mgr.Generate(name)
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Printf("  %s  %s\n", ui.Meta("Wallet :"), ui.Val(w.Name))
		fmt.Printf("  %s  %s\n\n", ui.Meta("Address:"), ui.Addr(w.Address))

		box := ui.DangerBox(
			ui.Warn("SAVE YOUR PRIVATE KEY — shown only once. Never share it.") + "\n\n" +
				ui.Val(hexKey) + "\n\n" +
				ui.Hint("Store in a password manager. Lose it → wallet gone forever."),
		)
		fmt.Println(box)
		fmt.Println(ui.Hint("  Re-export anytime: w3cli wallet export " + name))
		fmt.Println()
		return nil
	},
}

var walletExportCmd = &cobra.Command{
	Use:   "export <name>",
	Short: "Re-export the private key of a signing wallet",
	Long: `Retrieve and display the stored private key for a signing wallet.

You must type the wallet name exactly to confirm before the key is shown.
The key is retrieved from the OS keychain — it never leaves your machine.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		fmt.Println()
		fmt.Println(ui.Warn("  You are about to reveal a private key. Keep it secret."))
		fmt.Println()
		input := ui.PromptInput(fmt.Sprintf("  Type wallet name %q to confirm", name))
		if input != name {
			fmt.Println()
			fmt.Println(ui.Err("  Name mismatch — export cancelled."))
			return nil
		}

		mgr := newWalletManager()
		hexKey, err := mgr.ExportKey(name)
		if err != nil {
			return err
		}

		fmt.Println()
		box := ui.DangerBox(
			ui.Warn("PRIVATE KEY — do not share this with anyone.") + "\n\n" +
				ui.Val(hexKey),
		)
		fmt.Println(box)
		fmt.Println()
		return nil
	},
}

func init() {
	walletAddCmd.Flags().StringVar(&walletKeyFlag, "key", "", "private key for signing wallet (stored in OS keychain)")
	walletCmd.AddCommand(walletAddCmd, walletListCmd, walletRemoveCmd, walletUseCmd, walletGenerateCmd, walletExportCmd)
}

// walletTypeLabel converts an internal wallet type to a user-friendly label.
func walletTypeLabel(t string) string {
	switch t {
	case wallet.TypeSigning:
		return "read-write"
	default:
		return t // "watch-only" is already user-friendly
	}
}

// newWalletManager creates a Manager backed by the config-dir JSON store.
func newWalletManager() *wallet.Manager {
	store := wallet.NewJSONStore(filepath.Join(cfg.Dir(), "wallets.json"))
	return wallet.NewManager(wallet.WithStore(store))
}
