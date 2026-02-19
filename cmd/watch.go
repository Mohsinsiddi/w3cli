package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	watchNetwork string
	watchWallet  string
)

var watchCmd = &cobra.Command{
	Use:   "watch [address]",
	Short: "Stream live transactions for an address",
	Long: `Watch an address for incoming and outgoing transactions in real-time.

Polls the chain every 3 seconds for new blocks and streams any matching
transactions into a live TUI table. No WebSocket required — works with
all public HTTP RPCs.

Direction legend:
  ←  incoming (to your address)
  →  outgoing (from your address)

Keyboard controls:
  ↑↓ / j k   navigate rows
  o           open selected tx in explorer
  c           copy selected tx hash
  q           quit

Examples:
  w3cli watch 0xabc...
  w3cli watch 0xabc... --network base
  w3cli watch --network ethereum --testnet`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		address, chainName, err := resolveWalletAndChain(watchWallet, watchNetwork)
		if len(args) == 1 {
			address = args[0]
			if watchNetwork == "" {
				chainName = cfg.DefaultNetwork
				if chainName == "" {
					chainName = "ethereum"
				}
			}
		}
		if err != nil && address == "" {
			return err
		}

		mode := cfg.NetworkMode
		return runWatch(address, chainName, mode)
	},
}

func runWatch(address, chainName, mode string) error {
	reg := chain.NewRegistry()
	c, err := reg.GetByName(chainName)
	if err != nil {
		return fmt.Errorf("unknown chain %q — run `w3cli network list` to see all chains", chainName)
	}

	rpcURL, err := pickBestRPC(c, mode)
	if err != nil {
		return err
	}

	client := chain.NewEVMClient(rpcURL)
	explorer := c.Explorer(mode)

	m := ui.WatchModel{
		Address: address,
		Chain:   chainName,
		Mode:    mode,
	}

	prog := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	// Background goroutine: poll for new blocks every 3 seconds.
	go func() {
		// Anchor to current block so we don't replay history.
		startBlock, err := client.GetBlockNumber()
		if err != nil {
			prog.Send(ui.WatchStatusMsg{ErrMsg: "could not get starting block: " + err.Error()})
			return
		}
		lastBlock := startBlock
		prog.Send(ui.WatchStatusMsg{BlockNum: lastBlock, Fetching: false})

		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			latest, err := client.GetBlockNumber()
			if err != nil {
				prog.Send(ui.WatchStatusMsg{BlockNum: lastBlock, ErrMsg: trimWatchErr(err.Error())})
				continue
			}

			// Fetch all new blocks since lastBlock.
			for blk := lastBlock + 1; blk <= latest; blk++ {
				prog.Send(ui.WatchStatusMsg{BlockNum: blk, Fetching: true})

				txs, err := client.GetBlockTransactions(blk)
				if err != nil {
					continue
				}

				addrLower := strings.ToLower(address)
				for _, tx := range txs {
					isFrom := strings.EqualFold(tx.From, addrLower)
					isTo := strings.EqualFold(tx.To, addrLower)
					if !isFrom && !isTo {
						continue
					}

					direction := "→"
					counterpart := tx.To
					if isTo && !isFrom {
						direction = "←"
						counterpart = tx.From
					}

					val := parseFloat(tx.ValueETH)
					valStr := fmt.Sprintf("%.4f", val)

					explorerURL := ""
					if explorer != "" && tx.Hash != "" {
						explorerURL = explorer + "/tx/" + tx.Hash
					}

					prog.Send(ui.WatchTxMsg{
						Hash:        tx.Hash,
						Direction:   direction,
						Counterpart: ui.TruncateAddr(counterpart),
						ValueStr:    valStr,
						Currency:    c.NativeCurrency,
						BlockNum:    blk,
						TxRow: ui.TxRow{
							FullHash:    tx.Hash,
							ExplorerURL: explorerURL,
						},
					})
				}
			}

			lastBlock = latest
			prog.Send(ui.WatchStatusMsg{BlockNum: latest, Fetching: false})
		}
	}()

	_, err = prog.Run()
	return err
}

func trimWatchErr(s string) string {
	if len(s) > 40 {
		return s[:40] + "…"
	}
	return s
}

func init() {
	watchCmd.Flags().StringVar(&watchWallet, "wallet", "", "wallet name or address")
	watchCmd.Flags().StringVar(&watchNetwork, "network", "", "chain to watch")
}
