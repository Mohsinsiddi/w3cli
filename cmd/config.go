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
	Long: `Manage w3cli configuration stored at ~/.w3cli/config.json.

Use subcommands to set defaults for network, wallet, network mode, RPCs,
and explorer API keys. These defaults are used when no CLI flags are provided.

The --testnet / --mainnet global flags override the persisted network_mode
for a single invocation without changing the config file.`,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		network := cfg.DefaultNetwork
		if network == "" {
			network = "(not set)"
		}
		wallet := cfg.DefaultWallet
		if wallet == "" {
			wallet = "(not set)"
		}
		mode := cfg.NetworkMode
		if mode == "" {
			mode = "mainnet"
		}
		algo := cfg.RPCAlgorithm
		if algo == "" {
			algo = "fastest"
		}
		apiKeyStatus := "(not set)"
		if cfg.ExplorerAPIKey != "" {
			apiKeyStatus = "configured ✓"
		}

		fmt.Println(ui.KeyValueBlock("Current Configuration", [][2]string{
			{"Default Network", network},
			{"Network Mode", mode},
			{"Default Wallet", wallet},
			{"RPC Algorithm", algo},
			{"Explorer API Key", apiKeyStatus},
			{"Config Directory", cfg.Dir()},
		}))

		if verbose {
			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}
			fmt.Printf("\n%s\n%s\n", ui.Meta("Raw JSON:"), string(data))
		}
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
		fmt.Println(ui.Hint("This wallet will be used when --wallet is not specified."))
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
		fmt.Println(ui.Hint("This network will be used when --network is not specified."))
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
		fmt.Println(ui.Hint("Run `w3cli rpc benchmark " + chainName + "` to test endpoint performance."))
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
			fmt.Println(ui.Hint("This key will be used for all chains. Override per chain with --chain flag."))
		} else {
			fmt.Println(ui.Success(fmt.Sprintf("Explorer API key for %q saved.", explorerKeyChain)))
			fmt.Println(ui.Hint("This key takes priority over the global key for " + explorerKeyChain + "."))
		}
		return nil
	},
}

var configSetNetworkModeCmd = &cobra.Command{
	Use:   "set-network-mode <mainnet|testnet>",
	Short: "Set the default network mode (mainnet or testnet)",
	Long: `Persist the default network mode so all commands use it automatically.

Examples:
  w3cli config set-network-mode testnet   # default to testnet
  w3cli config set-network-mode mainnet   # switch back to mainnet

You can still override per-invocation with --testnet or --mainnet.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mode := args[0]
		switch mode {
		case "mainnet", "testnet":
		default:
			return fmt.Errorf("invalid mode %q — choose: mainnet, testnet", mode)
		}
		cfg.NetworkMode = mode
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("Default network mode set to %q", mode)))
		fmt.Println(ui.Hint("Override per-call with --testnet or --mainnet."))
		return nil
	},
}

var configSetKeyCmd = &cobra.Command{
	Use:   "set-key <provider> <key>",
	Short: "Set an API key for a provider (ankr, alchemy, moralis, etherscan)",
	Long: `Store a provider API key to unlock richer transaction history.

Valid providers:
  ankr       Free advanced API with higher rate limits (ankr.com/rpc/apps)
  alchemy    Full tx history for Ethereum, Base, Polygon, Arbitrum, Optimism (alchemy.com)
  moralis    Multi-chain tx history for 14+ chains (moralis.io)
  etherscan  Etherscan V2 unified key covering all supported EVM chains (etherscan.io/apis)

Examples:
  w3cli config set-key ankr MYANKRKEY
  w3cli config set-key alchemy MYALCHEMYKEY
  w3cli config set-key moralis MYMORALISKEY
  w3cli config set-key etherscan MYETHERSCANKEY`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, key := args[0], args[1]

		validProviders := map[string]string{
			"ankr":      "Ankr Advanced API — higher rate limits (get key at ankr.com/rpc/apps)",
			"alchemy":   "Alchemy — full history for Ethereum, Base, Polygon, Arbitrum, Optimism (alchemy.com)",
			"moralis":   "Moralis Deep Index — 14+ chains including BNB, Fantom, Avalanche (moralis.io)",
			"etherscan": "Etherscan V2 — unified key covering all supported EVM chains (etherscan.io/apis)",
		}

		hint, ok := validProviders[provider]
		if !ok {
			return fmt.Errorf("unknown provider %q — valid choices: ankr, alchemy, moralis, etherscan", provider)
		}

		cfg.SetProviderKey(provider, key)
		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Println(ui.Success(fmt.Sprintf("API key for %q saved.", provider)))
		fmt.Println(ui.Hint(hint))
		return nil
	},
}

func init() {
	configSetExplorerKeyCmd.Flags().StringVar(&explorerKeyChain, "chain", "", "set key for a specific chain only")
	configCmd.AddCommand(
		configListCmd,
		configSetDefaultWalletCmd,
		configSetDefaultNetworkCmd,
		configSetNetworkModeCmd,
		configSetRPCCmd,
		configSetExplorerKeyCmd,
		configSetKeyCmd,
	)
}
