// check-balances: queries native balance for a set of wallets across all EVM
// chains (mainnet + testnet) in parallel and prints a summary table.
//
// Run from the module root:
//
//	go run ./scripts/check-balances
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
)

// ── config ────────────────────────────────────────────────────────────────────

var wallets = []string{
	"0x802D8097eC1D49808F3c2c866020442891adde57",
	"0x315a352720E52EaDCB62f5e0879D5Fea82B959A4",
	"0x5d1D0b1d5790B1c88cC1e94366D3B242991DC05d",
}

const rpcTimeout = 12 * time.Second

// ── types ─────────────────────────────────────────────────────────────────────

type result struct {
	chain   string
	mode    string
	wallet  string // short form
	balance string
	symbol  string
	err     string
	sortKey string // chain + mode for stable ordering
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	reg := chain.NewRegistry()

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []result
	)

	modes := []string{"mainnet", "testnet"}

	for _, c := range reg.All() {
		if c.Type != chain.ChainTypeEVM {
			continue // skip Solana / SUI (different address format)
		}

		for _, mode := range modes {
			rpcs := c.RPCs(mode)
			if len(rpcs) == 0 {
				continue
			}
			rpcURL := rpcs[0] // use first built-in RPC

			for _, wallet := range wallets {
				wg.Add(1)
				go func(c chain.Chain, mode, rpcURL, wallet string) {
					defer wg.Done()

					ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
					defer cancel()

					client := chain.NewEVMClient(rpcURL)

					// Quick ping first — skip chains that don't respond.
					_, _, pingErr := client.Ping(ctx)

					r := result{
						chain:   c.Name,
						mode:    mode,
						wallet:  shortAddr(wallet),
						symbol:  c.NativeCurrency,
						sortKey: c.Name + "|" + mode,
					}

					if pingErr != nil {
						r.balance = "—"
						r.err = "unreachable"
					} else {
						bal, err := client.GetBalance(wallet)
						if err != nil {
							r.balance = "—"
							r.err = shortErr(err)
						} else {
							r.balance = trimZeros(bal.ETH)
						}
					}

					mu.Lock()
					results = append(results, r)
					mu.Unlock()
				}(c, mode, rpcURL, wallet)
			}
		}
	}

	wg.Wait()

	printTable(results)
}

// ── output ────────────────────────────────────────────────────────────────────

func printTable(results []result) {
	// Sort by chain name → mode (mainnet first) → wallet.
	sort.Slice(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if a.chain != b.chain {
			return a.chain < b.chain
		}
		if a.mode != b.mode {
			return a.mode < b.mode // mainnet < testnet alphabetically
		}
		return a.wallet < b.wallet
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "CHAIN\tMODE\tWALLET\tBALANCE\tSYMBOL\tNOTE")
	fmt.Fprintln(w, strings.Repeat("-", 10)+"\t"+
		strings.Repeat("-", 8)+"\t"+
		strings.Repeat("-", 14)+"\t"+
		strings.Repeat("-", 24)+"\t"+
		strings.Repeat("-", 6)+"\t"+
		strings.Repeat("-", 12))

	lastChain := ""
	for _, r := range results {
		if r.chain != lastChain {
			if lastChain != "" {
				fmt.Fprintln(w, "\t\t\t\t\t") // blank separator between chains
			}
			lastChain = r.chain
		}
		note := r.err
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.chain, r.mode, r.wallet, r.balance, r.symbol, note)
	}
	w.Flush()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func shortAddr(addr string) string {
	if len(addr) < 10 {
		return addr
	}
	return addr[:6] + "…" + addr[len(addr)-4:]
}

func shortErr(err error) string {
	s := err.Error()
	if len(s) > 30 {
		return s[:30] + "…"
	}
	return s
}

// trimZeros removes trailing zeros after decimal: "0.050000000000000000" → "0.05"
func trimZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}
