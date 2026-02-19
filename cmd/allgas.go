package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var allGasCmd = &cobra.Command{
	Use:   "allgas",
	Short: "Scan gas prices across all 24 EVM chains in real-time",
	Long: `Fetch current gas prices across all 24 EVM chains simultaneously.

All chains are queried concurrently and the table updates live as each chain
responds. Results are sorted cheapest-first once all chains have responded.

Chains that support EIP-1559 show both the base fee and chain type.
Legacy chains show eth_gasPrice.

Network mode is taken from your config (mainnet by default).

Keyboard controls:
  r   retry all failed chains
  o   open address on Debank in browser
  q   quit

Examples:
  w3cli allgas
  w3cli allgas --testnet`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAllGas(cfg.NetworkMode)
	},
}

func runAllGas(mode string) error {
	reg := chain.NewRegistry()

	var chains []chain.Chain
	for _, c := range reg.All() {
		if c.Type == chain.ChainTypeEVM {
			chains = append(chains, c)
		}
	}

	rows := make([]ui.AllGasRow, len(chains))
	rowIndex := make(map[string]int, len(chains))
	chainsByName := make(map[string]chain.Chain, len(chains))
	for i, c := range chains {
		rows[i] = ui.AllGasRow{
			ChainName:   c.Name,
			DisplayName: c.DisplayName,
			Status:      ui.AGStatusFetching,
		}
		rowIndex[c.Name] = i
		chainsByName[c.Name] = c
	}

	// FetchFn for retry support.
	fetchFn := func(chainName string) tea.Cmd {
		c := chainsByName[chainName]
		return func() tea.Msg {
			return ui.AllGasResultMsg(fetchChainGas(c, mode))
		}
	}

	m := ui.AllGasModel{
		Mode:     mode,
		Rows:     rows,
		RowIndex: rowIndex,
		Total:    len(chains),
		FetchFn:  fetchFn,
	}

	prog := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	// Fire one goroutine per chain.
	for _, c := range chains {
		c := c
		go func() { prog.Send(ui.AllGasResultMsg(fetchChainGas(c, mode))) }()
	}

	_, err := prog.Run()
	return err
}

// fetchChainGas fetches gas info for one chain and returns the result.
func fetchChainGas(c chain.Chain, mode string) ui.AllGasResult {
	start := time.Now()

	rpcURL, err := pickBestRPC(&c, mode)
	if err != nil {
		return ui.AllGasResult{ChainName: c.Name, Latency: time.Since(start), Err: err}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	type result struct {
		info *chain.GasInfo
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		info, e := chain.NewEVMClient(rpcURL).GetGasInfo()
		ch <- result{info, e}
	}()

	select {
	case <-ctx.Done():
		return ui.AllGasResult{
			ChainName: c.Name,
			Latency:   time.Since(start),
			Err:       fmt.Errorf("timeout"),
		}
	case r := <-ch:
		if r.err != nil {
			return ui.AllGasResult{ChainName: c.Name, Latency: time.Since(start), Err: r.err}
		}
		gwei, isEIP1559 := r.info.GasPriceDisplay()
		return ui.AllGasResult{
			ChainName:   c.Name,
			GasGwei:     gwei,
			BaseFeeGwei: r.info.BaseFeeGwei,
			IsEIP1559:   isEIP1559,
			Latency:     time.Since(start),
		}
	}
}
