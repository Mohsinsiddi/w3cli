package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/price" // batch price pre-fetch
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var allBalBothFlag bool

var allBalCmd = &cobra.Command{
	Use:   "allbal [address]",
	Short: "Scan balance across all 24 EVM chains in real-time",
	Long: `Fetch native token balances for an address across all 24 EVM chains simultaneously.

All chains are queried concurrently. Each row updates live as its chain responds —
no waiting for the slowest chain before seeing any results.

Once every chain has responded the table auto-sorts by USD value (highest first)
and displays a total at the bottom.

Network mode is taken from your config (mainnet by default).
Override with global flags or --both for a side-by-side view.

Keyboard controls:
  r   retry all failed chains
  o   open address on Debank (multi-chain portfolio) in browser
  q   quit

Examples:
  w3cli allbal 0xf39Fd6...            # mainnet, all chains
  w3cli allbal --testnet              # testnet, uses default wallet
  w3cli allbal --both 0xf39Fd6...    # mainnet + testnet columns side by side`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var address string
		if len(args) == 1 {
			address = args[0]
		} else {
			mgr := newWalletManager()
			w := mgr.Default()
			if w == nil {
				return fmt.Errorf(
					"no address provided and no default wallet set\n" +
						"  Pass an address: w3cli allbal 0x...\n" +
						"  Or set a default: w3cli wallet use <name>",
				)
			}
			address = w.Address
		}

		mode := cfg.NetworkMode
		if allBalBothFlag {
			mode = "both"
		}

		return runAllBal(address, mode)
	},
}

func runAllBal(address, mode string) error {
	reg := chain.NewRegistry()

	// EVM-only — Solana/SUI use non-0x address formats.
	var chains []chain.Chain
	for _, c := range reg.All() {
		if c.Type == chain.ChainTypeEVM {
			chains = append(chains, c)
		}
	}

	// Build rows, reverse-index, and name→chain lookup.
	rows := make([]ui.AllBalRow, len(chains))
	rowIndex := make(map[string]int, len(chains))
	chainsByName := make(map[string]chain.Chain, len(chains))
	for i, c := range chains {
		rows[i] = ui.AllBalRow{
			ChainName:   c.Name,
			DisplayName: c.DisplayName,
			Currency:    c.NativeCurrency,
			MainStatus:  ui.ABStatusFetching,
			TestStatus:  ui.ABStatusFetching,
		}
		rowIndex[c.Name] = i
		chainsByName[c.Name] = c
	}

	// Total goroutines: 1 per chain in single mode, 2 in "both".
	total := len(chains)
	if mode == "both" {
		total *= 2
	}

	// Pre-fetch ALL prices in one batch CoinGecko call (avoids per-goroutine rate-limiting).
	chainNames := make([]string, len(chains))
	for i, c := range chains {
		chainNames[i] = c.Name
	}
	priceFetcher := price.NewFetcher(cfg.PriceCurrency)
	priceMap, _ := priceFetcher.GetPrices(chainNames) // ignore error — rows show "—" if unavailable

	// FetchFn wraps fetchChainBal as a tea.Cmd for retry support.
	fetchFn := func(chainName, netMode string) tea.Cmd {
		c := chainsByName[chainName]
		return func() tea.Msg {
			return ui.AllBalResultMsg(fetchChainBal(c, address, netMode, priceMap))
		}
	}

	m := ui.AllBalModel{
		Address:  address,
		Mode:     mode,
		Rows:     rows,
		RowIndex: rowIndex,
		Total:    total,
		FetchFn:  fetchFn,
	}

	prog := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	// Fire one goroutine per chain (two for "both") for the initial load.
	for _, c := range chains {
		c := c
		if mode == "both" {
			go func() { prog.Send(ui.AllBalResultMsg(fetchChainBal(c, address, "mainnet", priceMap))) }()
			go func() { prog.Send(ui.AllBalResultMsg(fetchChainBal(c, address, "testnet", priceMap))) }()
		} else {
			go func() { prog.Send(ui.AllBalResultMsg(fetchChainBal(c, address, mode, priceMap))) }()
		}
	}

	_, err := prog.Run()
	return err
}

// fetchChainBal fetches the balance for one chain/netMode and returns the result.
// It is a pure function — safe to call from both goroutines and tea.Cmd.
func fetchChainBal(c chain.Chain, address, netMode string, priceMap map[string]float64) ui.AllBalResult {
	start := time.Now()

	rpcURL, err := pickBestRPC(&c, netMode)
	if err != nil {
		return ui.AllBalResult{
			ChainName: c.Name,
			NetMode:   netMode,
			Latency:   time.Since(start),
			Err:       err,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	client := chain.NewEVMClient(rpcURL)

	// Wrap the blocking call so the context timeout can cancel it.
	type result struct {
		bal *chain.Balance
		err error
	}
	ch := make(chan result, 1)
	go func() {
		b, e := client.GetBalance(address)
		ch <- result{b, e}
	}()

	var bal *chain.Balance
	select {
	case <-ctx.Done():
		return ui.AllBalResult{
			ChainName: c.Name,
			NetMode:   netMode,
			Latency:   time.Since(start),
			Err:       fmt.Errorf("timeout"),
		}
	case r := <-ch:
		if r.err != nil {
			return ui.AllBalResult{
				ChainName: c.Name,
				NetMode:   netMode,
				Latency:   time.Since(start),
				Err:       r.err,
			}
		}
		bal = r.bal
	}

	latency := time.Since(start)

	usdStr := "—"
	if netMode == "mainnet" {
		// Testnet tokens have no real USD value — only show price for mainnet.
		if usdPrice, ok := priceMap[c.Name]; ok && usdPrice > 0 {
			usdStr = fmt.Sprintf("$%.2f", parseFloat(bal.ETH)*usdPrice)
		}
	}

	return ui.AllBalResult{
		ChainName: c.Name,
		NetMode:   netMode,
		Balance:   bal.ETH,
		USD:       usdStr,
		Latency:   latency,
	}
}

func init() {
	allBalCmd.Flags().BoolVar(&allBalBothFlag, "both", false, "show mainnet and testnet balances side by side")
}
