package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var faucetOpen bool

var faucetCmd = &cobra.Command{
	Use:   "faucet [chain]",
	Short: "Show testnet faucet links for all chains or a specific chain",
	Long: `Display official testnet faucet links for supported chains.

Without a chain argument, a table of all 26 chains is shown.
With a chain argument, the faucet URL for that chain is displayed
and optionally opened in the browser.

Examples:
  w3cli faucet                  # list all testnet faucets
  w3cli faucet base             # show Base Sepolia faucet
  w3cli faucet ethereum --open  # open Sepolia faucet in browser
  w3cli faucet solana           # show Solana Devnet faucet`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reg := chain.NewRegistry()

		// Single chain: print URL and optionally open in browser.
		if len(args) == 1 {
			return showChainFaucet(reg, args[0])
		}

		// No args: show full table.
		return listAllFaucets(reg)
	},
}

func showChainFaucet(reg *chain.Registry, name string) error {
	c, err := reg.GetByName(name)
	if err != nil {
		return fmt.Errorf("unknown chain %q — run `w3cli network list` to see all chains", name)
	}

	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.ChainName(c.Name), ui.Meta("testnet: "+c.TestnetName))
	fmt.Println()

	if c.FaucetURL == "" {
		fmt.Println(ui.Warn("No dedicated faucet for this chain. Bridge assets from its parent network."))
		return nil
	}

	fmt.Printf("  %s  %s\n", ui.Meta("Faucet  :"), ui.Addr(c.FaucetURL))
	fmt.Printf("  %s  %s\n", ui.Meta("Currency:"), c.NativeCurrency)
	fmt.Printf("  %s  %s\n\n", ui.Meta("Explorer:"), ui.Addr(c.TestnetExplorer))

	if faucetOpen {
		fmt.Println(ui.Meta("  Opening in browser…"))
		openBrowserFaucet(c.FaucetURL)
	} else {
		fmt.Println(ui.Hint("  Tip: add --open to launch in your browser."))
	}

	return nil
}

func listAllFaucets(reg *chain.Registry) error {
	t := ui.NewTable([]ui.Column{
		{Title: "Chain", Width: 16},
		{Title: "Testnet", Width: 18},
		{Title: "Currency", Width: 10},
		{Title: "Faucet URL", Width: 52},
	})

	for _, c := range reg.All() {
		faucet := c.FaucetURL
		if faucet == "" {
			faucet = ui.Meta("(bridge from parent chain)")
		}
		t.AddRow(ui.Row{
			ui.ChainName(c.Name),
			c.TestnetName,
			c.NativeCurrency,
			faucet,
		})
	}

	fmt.Println(t.Render())
	fmt.Println(ui.Info("Tip: run `w3cli faucet <chain> --open` to launch a faucet in your browser."))
	return nil
}

func openBrowserFaucet(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start() //nolint:errcheck
}

func init() {
	faucetCmd.Flags().BoolVar(&faucetOpen, "open", false, "open the faucet URL in your default browser")
}
