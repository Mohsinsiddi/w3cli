package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n\n", ui.StyleTitle.Render("Current Configuration"))
		fmt.Println(string(data))
		fmt.Println(ui.Meta("Config directory: " + cfg.Dir()))
		return nil
	},
}

var configSetDefaultWalletCmd = &cobra.Command{
	Use:   "set-default-wallet <name>",
	Short: "Set the default wallet",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.DefaultWallet = args[0]
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("Default wallet set to %q", args[0])))
		return nil
	},
}

var configSetDefaultNetworkCmd = &cobra.Command{
	Use:   "set-default-network <chain>",
	Short: "Set the default network",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.DefaultNetwork = args[0]
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("Default network set to %q", args[0])))
		return nil
	},
}

var configSetRPCCmd = &cobra.Command{
	Use:   "set-rpc <chain> <url>",
	Short: "Add or override the RPC for a chain",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		chainName, url := args[0], args[1]
		if err := cfg.AddRPC(chainName, url); err != nil {
			// Already exists — not fatal.
			fmt.Println(ui.Warn(err.Error()))
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("RPC for %s set to %s", chainName, url)))
		return nil
	},
}

var explorerKeyChain string

var configSetExplorerKeyCmd = &cobra.Command{
	Use:   "set-explorer-key <key>",
	Short: "Set a block explorer API key (Etherscan, BlockScout Pro, etc.)",
	Long: `Store an explorer API key to unlock higher rate limits.

Without --chain, the key is stored as a global fallback that works for all chains
(e.g. an Etherscan V2 key covers all EVM chains).

With --chain, the key is stored for that specific chain only and takes priority
over the global key.

Examples:
  w3cli config set-explorer-key MYKEY123
  w3cli config set-explorer-key --chain base MYBASEKEY456`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		cfg.SetExplorerAPIKey(explorerKeyChain, key)
		if err := cfg.Save(); err != nil {
			return err
		}
		if explorerKeyChain == "" {
			fmt.Println(ui.Success("Global explorer API key saved."))
		} else {
			fmt.Println(ui.Success(fmt.Sprintf("Explorer API key for %q saved.", explorerKeyChain)))
		}
		return nil
	},
}

func init() {
	configSetExplorerKeyCmd.Flags().StringVar(&explorerKeyChain, "chain", "", "set key for a specific chain only")
	configCmd.AddCommand(
		configListCmd,
		configSetDefaultWalletCmd,
		configSetDefaultNetworkCmd,
		configSetRPCCmd,
		configSetExplorerKeyCmd,
	)
}
