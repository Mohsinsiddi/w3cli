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
	Use:   "balance",
	Short: "Check wallet balance",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve wallet and network.
		walletAddr, chainName, err := resolveWalletAndChain(balanceWallet, balanceNetwork)
		if err != nil {
			return err
		}

		if balanceLive {
			return runLiveDashboard(walletAddr, chainName)
		}

		return fetchAndPrintBalance(walletAddr, chainName)
	},
}

func fetchAndPrintBalance(address, chainName string) error {
	reg := chain.NewRegistry()
	c, err := reg.GetByName(chainName)
	if err != nil {
		return fmt.Errorf("unknown chain %q", chainName)
	}

	spin := ui.NewSpinner(fmt.Sprintf("Fetching balance on %s...", ui.ChainName(chainName)))
	spin.Start()

	// Pick best RPC.
	rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
	if err != nil {
		spin.Stop()
		return err
	}

	var priceFetcher = price.NewFetcher(cfg.PriceCurrency)

	if balanceToken != "" {
		// ERC-20 token balance.
		client := chain.NewEVMClient(rpcURL)
		bal, err := client.GetTokenBalance(balanceToken, address, 18) // TODO: fetch decimals
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
				{"Network", c.DisplayName + " (" + cfg.NetworkMode + ")"},
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
			fmt.Sprintf("Balance on Solana"),
			[][2]string{
				{"Address", ui.Addr(address)},
				{"Balance", bal.ETH + " SOL"},
			},
		))

	case chain.ChainTypeSUI:
		client := chain.NewSUIClient(rpcURL)
		bal, err := client.GetBalance(address)
		spin.Stop()
		if err != nil {
			return err
		}
		fmt.Println(ui.KeyValueBlock(
			"Balance on SUI",
			[][2]string{
				{"Address", ui.Addr(address)},
				{"Balance", bal.ETH + " SUI"},
			},
		))
	}

	return nil
}

func runLiveDashboard(address, chainName string) error {
	reg := chain.NewRegistry()
	c, _ := reg.GetByName(chainName)

	fetcher := func() ([]ui.BalanceEntry, error) {
		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
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
		return "", fmt.Errorf("no RPCs configured for %s (%s)", c.Name, mode)
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
// walletFlag may be a wallet name (looked up in config) or a raw hex address.
func resolveWalletAndChain(walletFlag, networkFlag string) (string, string, error) {
	chainName := networkFlag
	if chainName == "" {
		chainName = cfg.DefaultNetwork
	}

	mgr := newWalletManager()

	if walletFlag == "" {
		// No flag — use default wallet.
		w := mgr.Default()
		if w == nil {
			return "", "", fmt.Errorf("no wallet specified — use --wallet or set a default with `w3cli wallet use <name>`")
		}
		return w.Address, chainName, nil
	}

	// If it starts with 0x and is long enough, treat as raw address.
	if len(walletFlag) >= 40 && (walletFlag[:2] == "0x" || walletFlag[:2] == "0X") {
		return walletFlag, chainName, nil
	}

	// Otherwise treat it as a wallet name.
	w, err := mgr.Get(walletFlag)
	if err != nil {
		return "", "", fmt.Errorf("wallet %q not found — run `w3cli wallet list` to see available wallets", walletFlag)
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
