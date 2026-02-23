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

var walletUnlockAll bool

var walletUnlockCmd = &cobra.Command{
	Use:   "unlock [name]",
	Short: "Cache wallet key(s) for the session (skips future keychain prompts)",
	Long: `Retrieve private keys from the OS keychain once and cache them in a
restricted session file so all future commands run without any prompt.

  # Interactive — pick a wallet from a list
  w3cli wallet unlock

  # Unlock a specific wallet by name
  w3cli wallet unlock alice

  # Unlock every signing wallet at once
  w3cli wallet unlock --all

Note: the OS may prompt once per wallet during unlock:
  macOS        — Keychain Access GUI dialog
  Ubuntu (GUI) — GNOME Keyring password popup
  Ubuntu (SSH) — terminal passphrase for the file backend
  Windows      — silent (Credential Manager handles it)

After unlock, every send / mint / studio write runs with zero prompts.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := newWalletManager()
		ks := wallet.DefaultKeystore()

		// Collect all signing wallets upfront (needed for picker + --all).
		var signingWallets []string
		for _, w := range mgr.List() {
			if w.Type == wallet.TypeSigning {
				signingWallets = append(signingWallets, w.Name)
			}
		}
		if len(signingWallets) == 0 {
			fmt.Println(ui.Info("No signing wallets found."))
			fmt.Println(ui.Hint("Add one with: w3cli wallet add <name> --key <private-key>"))
			return nil
		}

		var names []string
		switch {
		case walletUnlockAll:
			names = signingWallets

		case len(args) > 0:
			names = []string{args[0]}

		default:
			// Interactive picker — navigate and select one wallet.
			items := make([]ui.PickerItem, len(signingWallets))
			for i, name := range signingWallets {
				w, _ := mgr.Get(name)
				sub := ""
				if w != nil {
					sub = ui.TruncateAddr(w.Address)
					if wallet.GetSessionKeyCached(name) {
						sub += "  " + ui.Meta("[cached]")
					}
				}
				items[i] = ui.PickerItem{
					Label:    name,
					SubLabel: sub,
					Value:    name,
				}
			}

			picked, err := ui.PickItem("Unlock Wallet  ·  select to cache key", items)
			if err != nil {
				return err
			}
			if picked == "" {
				fmt.Println(ui.Meta("Cancelled."))
				return nil
			}
			names = []string{picked}
		}

		// Inform user about the upcoming OS prompt (before it fires).
		fmt.Println(ui.Info("Your OS keychain may prompt once per wallet being unlocked."))
		fmt.Println()

		// Load the session file once upfront so we can batch-check what's
		// already cached without N separate file reads in the loop.
		existingSession := wallet.LoadSessionSnapshot()

		var unlocked, skipped int
		newKeys := make(map[string]string) // collected for a single bulk write
		for _, name := range names {
			ref := "w3cli." + name
			if _, ok := existingSession[ref]; ok {
				fmt.Println(ui.Meta(fmt.Sprintf("  %-20s already cached", name)))
				skipped++
				continue
			}
			hexKey, err := ks.Retrieve(ref) // OS prompt fires here if needed
			if err != nil {
				fmt.Println(ui.Err(fmt.Sprintf("  %-20s %v", name, err)))
				continue
			}
			newKeys[ref] = hexKey
			fmt.Println(ui.Success(fmt.Sprintf("  %-20s unlocked", name)))
			unlocked++
		}

		// Single file write for all newly unlocked keys.
		if len(newKeys) > 0 {
			wallet.BulkPutSessionKeys(newKeys)
		}

		fmt.Println()
		if unlocked > 0 {
			fmt.Println(ui.Success(fmt.Sprintf(
				"%d wallet(s) cached. Zero prompts until 'w3cli wallet lock'.", unlocked)))
		}
		if skipped > 0 {
			fmt.Println(ui.Meta(fmt.Sprintf("  %d already cached, skipped.", skipped)))
		}
		return nil
	},
}

var walletLockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Clear the session cache (re-enables keychain prompts)",
	Long:  `Delete the session file written by 'w3cli wallet unlock'. The next transaction will prompt the OS keychain again.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !wallet.SessionActive() {
			fmt.Println(ui.Meta("No active session — nothing to clear."))
			return nil
		}
		if err := wallet.ClearSession(); err != nil {
			return fmt.Errorf("clearing session: %w", err)
		}
		fmt.Println(ui.Success("Session cleared. Keychain will be used on next access."))
		return nil
	},
}

func init() {
	walletAddCmd.Flags().StringVar(&walletKeyFlag, "key", "", "private key for signing wallet (stored in OS keychain)")
	walletUnlockCmd.Flags().BoolVar(&walletUnlockAll, "all", false, "unlock all signing wallets")
	walletCmd.AddCommand(walletAddCmd, walletListCmd, walletRemoveCmd, walletUseCmd,
		walletGenerateCmd, walletExportCmd, walletUnlockCmd, walletLockCmd)
}

// warnIfNoSession prints a one-line hint when no session file is active.
// Call this at the start of any command that will sign a transaction so the
// user understands why the OS keychain dialog is about to appear.
func warnIfNoSession() {
	if !wallet.SessionActive() {
		fmt.Println(ui.Info(
			"No session active — keychain may prompt for each tx.\n" +
				"  Run 'w3cli wallet unlock --all' once to cache all keys and skip future prompts.",
		))
		fmt.Println()
	}
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

// loadSigningWallet loads a wallet by name and verifies it can sign transactions.
// Returns the wallet, the manager (for further lookups), and any error.
// Used by every write-transaction command to eliminate boilerplate.
func loadSigningWallet(walletName string) (*wallet.Wallet, *wallet.Manager, error) {
	mgr := newWalletManager()
	w, err := mgr.Get(walletName)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"wallet %q not found — run `w3cli wallet list` or set a default with `w3cli wallet use <name>`",
			walletName,
		)
	}
	if w.Type != wallet.TypeSigning {
		return nil, nil, fmt.Errorf(
			"wallet %q is watch-only and cannot sign transactions\n  To add a signing wallet: w3cli wallet add <name> --key <private-key>",
			walletName,
		)
	}
	return w, mgr, nil
}
