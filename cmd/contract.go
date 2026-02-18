package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	chainreg "github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/contract"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	contractABIFile  string
	contractFetchABI bool
	contractNetwork  string
)

var contractCmd = &cobra.Command{
	Use:   "contract",
	Short: "Interact with smart contracts",
}

var contractAddCmd = &cobra.Command{
	Use:   "add <name> <address>",
	Short: "Register a contract",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, address := args[0], args[1]
		network := contractNetwork
		if network == "" {
			network = cfg.DefaultNetwork
		}

		reg := newContractRegistry()
		if err := reg.Load(); err != nil {
			return err
		}

		var abi []contract.ABIEntry
		var err error

		if contractABIFile != "" {
			abi, err = contract.LoadFromFile(contractABIFile)
			if err != nil {
				return err
			}
		} else if contractFetchABI {
			fmt.Println(ui.Meta("Fetching ABI from explorer..."))
			// TODO: use explorer API URL from chain registry.
			return fmt.Errorf("--fetch requires an explorer API key — use --abi to provide a local ABI file")
		}

		reg.Add(&contract.Entry{
			Name:    name,
			Network: network,
			Address: address,
			ABI:     abi,
		})

		if err := reg.Save(); err != nil {
			return err
		}

		fmt.Println(ui.Success(fmt.Sprintf("Contract %q registered on %s at %s", name, network, ui.Addr(address))))
		return nil
	},
}

var contractListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered contracts",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg := newContractRegistry()
		if err := reg.Load(); err != nil {
			return err
		}

		entries := reg.All()
		if len(entries) == 0 {
			fmt.Println(ui.Meta("No contracts registered. Use `w3cli contract add`."))
			return nil
		}

		t := ui.NewTable([]ui.Column{
			{Title: "Name", Width: 16},
			{Title: "Network", Width: 14},
			{Title: "Address", Width: 44},
			{Title: "ABI Funcs", Width: 10},
		})

		for _, e := range entries {
			funcCount := fmt.Sprintf("%d", len(e.ABI))
			t.AddRow(ui.Row{
				ui.Val(e.Name),
				ui.ChainName(e.Network),
				ui.Addr(e.Address),
				funcCount,
			})
		}
		fmt.Println(t.Render())
		return nil
	},
}

var contractCallCmd = &cobra.Command{
	Use:   "call <contract> <function> [args...]",
	Short: "Call a read-only contract function",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		funcName := args[1]
		funcArgs := args[2:]

		network := contractNetwork
		if network == "" {
			network = cfg.DefaultNetwork
		}

		reg := newContractRegistry()
		if err := reg.Load(); err != nil {
			return err
		}

		entry, err := reg.Get(name, network)
		if err != nil {
			return err
		}

		chainReg := newChainRegistry()
		c, err := chainReg.GetByName(network)
		if err != nil {
			return fmt.Errorf("unknown chain %q", network)
		}

		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
		if err != nil {
			return err
		}

		spin := ui.NewSpinner(fmt.Sprintf("Calling %s.%s()...", name, funcName))
		spin.Start()

		caller := contract.NewCallerFromEntries(rpcURL, entry.ABI)
		results, err := caller.Call(entry.Address, funcName, funcArgs...)
		spin.Stop()
		if err != nil {
			return err
		}

		fmt.Printf("\n%s  %s\n\n", ui.StyleTitle.Render(name+"."+funcName+"()"), ui.Meta("→ result"))
		for i, r := range results {
			fmt.Printf("  [%d]  %s\n", i, ui.Val(r))
		}
		fmt.Println()
		return nil
	},
}

var contractSyncCmd = &cobra.Command{
	Use:   "sync [contract]",
	Short: "Re-fetch ABI for a contract (or all)",
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		if !all && len(args) == 0 {
			return fmt.Errorf("provide a contract name or --all")
		}
		fmt.Println(ui.Meta("ABI sync via explorer requires an API key — use `w3cli sync run` for manifest sync."))
		return nil
	},
}

var contractStudioCmd = &cobra.Command{
	Use:   "studio <contract>",
	Short: "Interactive TUI contract explorer",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		network := contractNetwork
		if network == "" {
			network = cfg.DefaultNetwork
		}

		reg := newContractRegistry()
		if err := reg.Load(); err != nil {
			return err
		}

		entry, err := reg.Get(name, network)
		if err != nil {
			return err
		}

		fmt.Printf("%s\n\n", ui.StyleTitle.Render(fmt.Sprintf("Contract Studio: %s on %s", name, network)))
		fmt.Printf("  %s %s\n\n", ui.Meta("Address:"), ui.Addr(entry.Address))

		// Show read functions.
		fmt.Println(ui.StyleHeader.Render("Read Functions:"))
		for _, fn := range entry.ABI {
			if fn.IsReadFunction() {
				params := make([]string, len(fn.Inputs))
				for i, p := range fn.Inputs {
					params[i] = p.Type + " " + p.Name
				}
				fmt.Printf("  %s(%s)\n", ui.Val(fn.Name), ui.Meta(strings.Join(params, ", ")))
			}
		}

		fmt.Println()
		fmt.Println(ui.StyleHeader.Render("Write Functions:"))
		for _, fn := range entry.ABI {
			if fn.IsWriteFunction() {
				params := make([]string, len(fn.Inputs))
				for i, p := range fn.Inputs {
					params[i] = p.Type + " " + p.Name
				}
				fmt.Printf("  %s(%s)\n", ui.Warn(fn.Name), ui.Meta(strings.Join(params, ", ")))
			}
		}

		fmt.Println()
		fmt.Println(ui.Meta("Use `w3cli contract call <name> <function> [args...]` to call a function."))
		return nil
	},
}

func init() {
	contractAddCmd.Flags().StringVar(&contractABIFile, "abi", "", "path to ABI JSON file")
	contractAddCmd.Flags().BoolVar(&contractFetchABI, "fetch", false, "auto-fetch ABI from explorer")
	contractAddCmd.Flags().StringVar(&contractNetwork, "network", "", "chain (default: config)")
	contractCallCmd.Flags().StringVar(&contractNetwork, "network", "", "chain (default: config)")
	contractStudioCmd.Flags().StringVar(&contractNetwork, "network", "", "chain (default: config)")
	contractSyncCmd.Flags().Bool("all", false, "sync all contracts")

	contractCmd.AddCommand(contractAddCmd, contractListCmd, contractCallCmd, contractSyncCmd, contractStudioCmd)
}

func newContractRegistry() *contract.Registry {
	return contract.NewRegistry(filepath.Join(cfg.Dir(), "contracts.json"))
}

func newChainRegistry() *chainreg.Registry {
	return chainreg.NewRegistry()
}
