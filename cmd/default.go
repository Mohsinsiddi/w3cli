package cmd

import (
	"fmt"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/Mohsinsiddi/w3cli/internal/wallet"
	"github.com/spf13/cobra"
)

var (
	defaultSetWallet  string
	defaultSetNetwork string
	defaultSetMode    string
)

var defaultCmd = &cobra.Command{
	Use:   "default",
	Short: "Show and quickly update all active defaults",
	Long: `Display every switchable default in one place and optionally update them inline.

  # View all defaults
  w3cli default

  # Update defaults without running separate config commands
  w3cli default --wallet alice
  w3cli default --network base
  w3cli default --mode testnet
  w3cli default --wallet deployer --network ethereum --mode mainnet`,
	RunE: func(cmd *cobra.Command, args []string) error {
		changed := false

		// ── Apply any inline updates ──────────────────────────────────────────
		if defaultSetWallet != "" {
			mgr := newWalletManager()
			if _, err := mgr.Get(defaultSetWallet); err != nil {
				return fmt.Errorf("wallet %q not found — run `w3cli wallet list` to see all wallets", defaultSetWallet)
			}
			cfg.DefaultWallet = defaultSetWallet
			changed = true
			fmt.Println(ui.Success(fmt.Sprintf("Default wallet → %s", defaultSetWallet)))
		}

		if defaultSetNetwork != "" {
			cfg.DefaultNetwork = defaultSetNetwork
			changed = true
			fmt.Println(ui.Success(fmt.Sprintf("Default network → %s", defaultSetNetwork)))
		}

		if defaultSetMode != "" {
			switch defaultSetMode {
			case "mainnet", "testnet":
			default:
				return fmt.Errorf("invalid mode %q — choose: mainnet, testnet", defaultSetMode)
			}
			cfg.NetworkMode = defaultSetMode
			changed = true
			fmt.Println(ui.Success(fmt.Sprintf("Network mode → %s", defaultSetMode)))
		}

		if changed {
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			fmt.Println()
		}

		// ── Resolve wallet display ────────────────────────────────────────────
		walletName := cfg.DefaultWallet
		walletLine := ui.Meta("(not set)")
		if walletName != "" {
			mgr := newWalletManager()
			if w, err := mgr.Get(walletName); err == nil {
				typeLabel := walletTypeLabel(w.Type)
				walletLine = fmt.Sprintf("%s  %s  %s",
					ui.Val(walletName),
					ui.Meta("("+typeLabel+")"),
					ui.Addr(ui.TruncateAddr(w.Address)),
				)
			} else {
				walletLine = ui.Warn(walletName + " (wallet not found)")
			}
		}

		// ── Resolve network display ───────────────────────────────────────────
		networkLine := ui.Meta("(not set)")
		if cfg.DefaultNetwork != "" {
			networkLine = ui.ChainName(cfg.DefaultNetwork)
		}

		// ── Mode display ──────────────────────────────────────────────────────
		mode := cfg.NetworkMode
		if mode == "" {
			mode = "mainnet"
		}
		modeColor := ui.StyleSuccess
		if mode == "testnet" {
			modeColor = ui.StyleWarning
		}
		modeLine := modeColor.Render(mode) + ui.Meta("  (override per call: --mainnet / --testnet)")

		// ── Session display ───────────────────────────────────────────────────
		sessionLine := ui.Err("not unlocked — keychain will prompt on each write tx")
		if wallet.SessionActive() {
			snap := wallet.LoadSessionSnapshot()
			sessionLine = ui.Success(fmt.Sprintf("%d wallet(s) cached · zero keychain prompts", len(snap)))
		}

		// ── Custom RPCs ───────────────────────────────────────────────────────
		rpcCount := 0
		for _, urls := range cfg.CustomRPCs {
			rpcCount += len(urls)
		}
		rpcLine := ui.Val(cfg.RPCAlgorithm)
		if rpcLine == "" {
			rpcLine = ui.Val("fastest")
		}
		if rpcCount > 0 {
			rpcLine += ui.Meta(fmt.Sprintf("  · %d custom override(s)", rpcCount))
		}

		// ── Explorer key ──────────────────────────────────────────────────────
		explorerLine := ui.Meta("(not set)  — `w3cli config set-explorer-key <key>`")
		if cfg.ExplorerAPIKey != "" {
			k := cfg.ExplorerAPIKey
			masked := k[:min(6, len(k))] + "…"
			explorerLine = ui.Success("configured " + masked)
		}

		// ── Provider keys ─────────────────────────────────────────────────────
		providers := []string{"ankr", "alchemy", "moralis", "etherscan"}
		var provSet, provMissing []string
		for _, p := range providers {
			if cfg.GetProviderKey(p) != "" {
				provSet = append(provSet, ui.StyleSuccess.Render("✓ "+p))
			} else {
				provMissing = append(provMissing, ui.StyleMeta.Render("✗ "+p))
			}
		}
		providerLine := strings.Join(append(provSet, provMissing...), "  ")

		// ── Print ─────────────────────────────────────────────────────────────
		fmt.Println(ui.KeyValueBlock("Active Defaults", [][2]string{
			{"Wallet", walletLine},
			{"Network", networkLine},
			{"Mode", modeLine},
			{"Session", sessionLine},
		}))

		fmt.Println(ui.KeyValueBlock("Connectivity", [][2]string{
			{"RPC Algorithm", rpcLine},
			{"Explorer Key", explorerLine},
			{"Provider Keys", providerLine},
		}))

		fmt.Println(ui.KeyValueBlock("Quick Update  (flags apply immediately)", [][2]string{
			{"Wallet", "w3cli default --wallet <name>"},
			{"Network", "w3cli default --network <chain>"},
			{"Mode", "w3cli default --mode mainnet|testnet"},
			{"Unlock session", "w3cli wallet unlock --all"},
			{"Lock session", "w3cli wallet lock"},
			{"RPC override", "w3cli config set-rpc <chain> <url>"},
			{"Explorer key", "w3cli config set-explorer-key <key>"},
			{"Provider key", "w3cli config set-key <provider> <key>"},
		}))

		return nil
	},
}

func init() {
	defaultCmd.Flags().StringVar(&defaultSetWallet, "wallet", "", "set the default wallet")
	defaultCmd.Flags().StringVar(&defaultSetNetwork, "network", "", "set the default network chain")
	defaultCmd.Flags().StringVar(&defaultSetMode, "mode", "", "set the network mode (mainnet|testnet)")
}

// min is a Go 1.21+ builtin but defined here for older toolchains.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
