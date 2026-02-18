package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/price"
	"github.com/Mohsinsiddi/w3cli/internal/rpc"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	balanceWallet  string
	balanceNetwork string
	balanceToken   string
	balanceLive    bool
)

var balanceCmd = &cobra.Command{
	Use:   "balance [wallet-name-or-address]",
	Short: "Check wallet balance",
	Long: `Check native or ERC-20 token balance for a wallet.

Uses the configured network mode (mainnet/testnet) by default.
Override per-call with --testnet or --mainnet.

Examples:
  w3cli balance 0xABC...                        # default chain + mode
  w3cli balance --network base --testnet         # Base Sepolia
  w3cli balance --network ethereum --mainnet     # Ethereum mainnet
  w3cli balance --token 0xUSDC... --live         # ERC-20 live dashboard`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Allow positional arg as shorthand for --wallet.
		if len(args) == 1 && balanceWallet == "" {
			balanceWallet = args[0]
		}

		walletAddr, chainName, err := resolveWalletAndChain(balanceWallet, balanceNetwork)
		if err != nil {
			return err
		}

		if balanceLive {
			return runLiveDashboard(walletAddr, chainName, cfg.NetworkMode)
		}

		return fetchAndPrintBalance(walletAddr, chainName, cfg.NetworkMode)
	},
}

func fetchAndPrintBalance(address, chainName, networkMode string) error {
	reg := chain.NewRegistry()
	c, err := reg.GetByName(chainName)
	if err != nil {
		return fmt.Errorf("unknown chain %q — run `w3cli network list` to see all chains", chainName)
	}

	spin := ui.NewSpinner(fmt.Sprintf("Fetching balance on %s (%s)...", ui.ChainName(chainName), networkMode))
	spin.Start()

	rpcURL, err := pickBestRPC(c, networkMode)
	if err != nil {
		spin.Stop()
		return err
	}

	var priceFetcher = price.NewFetcher(cfg.PriceCurrency)

	if balanceToken != "" {
		client := chain.NewEVMClient(rpcURL)
		bal, err := client.GetTokenBalance(balanceToken, address, 18)
		spin.Stop()
		if err != nil {
			return err
		}
		fmt.Println(ui.KeyValueBlock(
			fmt.Sprintf("Token Balance on %s", c.DisplayName),
			[][2]string{
				{"Address", ui.Addr(address)},
				{"Token", ui.Addr(balanceToken)},
				{"Balance", bal.Formatted},
			},
		))
		return nil
	}

	switch c.Type {
	case chain.ChainTypeEVM:
		client := chain.NewEVMClient(rpcURL)
		bal, err := client.GetBalance(address)
		spin.Stop()
		if err != nil {
			return err
		}
		usdPrice, _ := priceFetcher.GetPrice(chainName)
		usdValue := fmt.Sprintf("—")
		if usdPrice > 0 {
			ethFloat := parseFloat(bal.ETH)
			usdValue = fmt.Sprintf("$%.2f", ethFloat*usdPrice)
		}
		fmt.Println(ui.KeyValueBlock(
			fmt.Sprintf("Balance on %s", c.DisplayName),
			[][2]string{
				{"Address", ui.Addr(address)},
				{"Network", c.DisplayName + " (" + networkMode + ")"},
				{"Balance", bal.ETH + " " + c.NativeCurrency},
				{"USD Value", usdValue},
			},
		))

	case chain.ChainTypeSolana:
		client := chain.NewSolanaClient(rpcURL)
		bal, err := client.GetBalance(address)
		spin.Stop()
		if err != nil {
			return err
		}
		fmt.Println(ui.KeyValueBlock(
			fmt.Sprintf("Balance on Solana (%s)", networkMode),
			[][2]string{
				{"Address", ui.Addr(address)},
				{"Network", "Solana (" + networkMode + ")"},
				{"Balance", bal.ETH + " SOL"},
				{"USD Value", "—"},
			},
		))
		fmt.Println(ui.Hint("USD pricing for Solana coming soon."))

	case chain.ChainTypeSUI:
		client := chain.NewSUIClient(rpcURL)
		bal, err := client.GetBalance(address)
		spin.Stop()
		if err != nil {
			return err
		}
		fmt.Println(ui.KeyValueBlock(
			fmt.Sprintf("Balance on SUI (%s)", networkMode),
			[][2]string{
				{"Address", ui.Addr(address)},
				{"Network", "SUI (" + networkMode + ")"},
				{"Balance", bal.ETH + " SUI"},
				{"USD Value", "—"},
			},
		))
		fmt.Println(ui.Hint("USD pricing for SUI coming soon."))
	}

	return nil
}

func runLiveDashboard(address, chainName, networkMode string) error {
	reg := chain.NewRegistry()
	c, _ := reg.GetByName(chainName)

	fetcher := func() ([]ui.BalanceEntry, error) {
		rpcURL, err := pickBestRPC(c, networkMode)
		if err != nil {
			return nil, err
		}
		client := chain.NewEVMClient(rpcURL)
		bal, err := client.GetBalance(address)
		if err != nil {
			return nil, err
		}
		p := price.NewFetcher(cfg.PriceCurrency)
		usdPrice, _ := p.GetPrice(chainName)
		usdStr := fmt.Sprintf("$%.2f", parseFloat(bal.ETH)*usdPrice)

		return []ui.BalanceEntry{
			{
				Chain:   chainName,
				Address: address,
				Balance: bal.ETH,
				Symbol:  c.NativeCurrency,
				USD:     usdStr,
			},
		}, nil
	}

	prog := ui.NewDashboard(time.Duration(cfg.WatchInterval)*time.Second, fetcher)
	_, err := prog.Run()
	return err
}

// pickBestRPC selects the best RPC for a chain using the configured algorithm.
func pickBestRPC(c *chain.Chain, mode string) (string, error) {
	rpcs := c.RPCs(mode)
	if len(rpcs) == 0 {
		return "", fmt.Errorf("no RPCs configured for %s (%s) — try adding one with `w3cli rpc add %s <url>`", c.Name, mode, c.Name)
	}
	// Merge custom RPCs.
	if custom := cfg.GetRPCs(c.Name); len(custom) > 0 {
		rpcs = append(custom, rpcs...)
	}
	if len(rpcs) == 1 {
		return rpcs[0], nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return rpc.BestEVM(ctx, rpcs, rpc.Algorithm(cfg.RPCAlgorithm))
}

// resolveWalletAndChain returns the effective wallet address and chain name.
func resolveWalletAndChain(walletFlag, networkFlag string) (string, string, error) {
	chainName := networkFlag
	if chainName == "" {
		chainName = cfg.DefaultNetwork
	}

	mgr := newWalletManager()

	if walletFlag == "" {
		w := mgr.Default()
		if w == nil {
			return "", "", fmt.Errorf("no wallet specified — use --wallet <address> or set a default:\n  w3cli wallet add myWallet 0x...\n  w3cli wallet use myWallet")
		}
		return w.Address, chainName, nil
	}

	if len(walletFlag) >= 40 && (walletFlag[:2] == "0x" || walletFlag[:2] == "0X") {
		return walletFlag, chainName, nil
	}

	w, err := mgr.Get(walletFlag)
	if err != nil {
		return "", "", fmt.Errorf("wallet %q not found — run `w3cli wallet list` to see available wallets, or pass an address directly", walletFlag)
	}
	return w.Address, chainName, nil
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func init() {
	balanceCmd.Flags().StringVar(&balanceWallet, "wallet", "", "wallet name or address")
	balanceCmd.Flags().StringVar(&balanceNetwork, "network", "", "chain to query (default: config)")
	balanceCmd.Flags().StringVar(&balanceToken, "token", "", "ERC-20 token contract address")
	balanceCmd.Flags().BoolVar(&balanceLive, "live", false, "live refresh mode")
}
