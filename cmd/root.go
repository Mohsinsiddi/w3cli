package cmd

import (
	"fmt"
	"os"

	"github.com/Mohsinsiddi/w3cli/internal/config"
	"github.com/spf13/cobra"
)

// Version is the current release. Overridable via build ldflags:
//
//	go build -ldflags "-X github.com/Mohsinsiddi/w3cli/cmd.Version=1.2.3" .
var Version = "1.0.0"

var (
	cfgDir  string
	cfg     *config.Config
	verbose bool
	testnet bool
	mainnet bool
)

// rootCmd is the top-level command.
var rootCmd = &cobra.Command{
	Use:   "w3cli",
	Short: "The Web3 Power CLI",
	Long: `w3cli — Beautiful, blazing-fast terminal tool for Web3 developers.

  Check balances, query transactions, interact with smart contracts,
  sign and send transactions, and monitor wallets — across 26 chains.

Global flags --testnet and --mainnet override the configured network mode
for a single invocation. Without either flag the persisted mode is used
(default: mainnet). Persist with: w3cli config set-network-mode <mode>`,
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load config (skip for commands that don't need it).
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			return nil
		}
		var err error
		cfg, err = config.Load(cfgDir)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if testnet {
			cfg.NetworkMode = "testnet"
		}
		if mainnet {
			cfg.NetworkMode = "mainnet"
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// CHAIN_CONFIG_DIR env var overrides --config flag.
	if envDir := os.Getenv("CHAIN_CONFIG_DIR"); envDir != "" {
		cfgDir = envDir
	}

	rootCmd.PersistentFlags().StringVar(&cfgDir, "config", cfgDir, "config directory (default: ~/.w3cli)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&testnet, "testnet", false, "use testnet instead of mainnet")
	rootCmd.PersistentFlags().BoolVar(&mainnet, "mainnet", false, "use mainnet instead of testnet")
	rootCmd.MarkFlagsMutuallyExclusive("testnet", "mainnet")

	// Register all sub-commands.
	rootCmd.AddCommand(
		initCmd,
		networkCmd,
		walletCmd,
		balanceCmd,
		allBalCmd,
		allGasCmd,
		blockCmd,
		txsCmd,
		txCmd,
		sendCmd,
		tokenCmd,
		rpcCmd,
		contractCmd,
		configCmd,
		syncCmd,
		watchCmd,
		faucetCmd,
	)
}
