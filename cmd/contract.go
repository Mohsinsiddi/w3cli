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
	contractBuiltin  string // --builtin <id>  e.g. "w3token", "erc20"
	contractFetchABI bool
	contractNetwork  string
)

var contractCmd = &cobra.Command{
	Use:   "contract",
	Short: "Interact with smart contracts",
}

// ── contract add ──────────────────────────────────────────────────────────────

var contractAddCmd = &cobra.Command{
	Use:   "add <name> <address>",
	Short: "Register a contract",
	Long: `Register a contract in the local registry so you can call it by name.

ABI source (pick one):
  --abi <file>        Raw ABI JSON array or Hardhat/Foundry artifact
  --builtin <id>      Use a bundled ABI (see: w3cli contract builtins)
  --fetch             Auto-fetch ABI from explorer (requires API key)

Examples:
  w3cli contract add myUSDC 0xA0b8...  --builtin erc20 --network ethereum
  w3cli contract add myNFT  0x1234...  --abi ./out/MyNFT.sol/MyNFT.json
  w3cli contract add myToken 0xABCD... --builtin w3token --network base`,
	Args: cobra.ExactArgs(2),
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
		var kind, builtinID, abiSource string
		var err error

		switch {
		case contractABIFile != "":
			// Supports both raw ABI arrays and Hardhat/Foundry artifacts.
			abi, err = contract.LoadFromArtifact(contractABIFile)
			if err != nil {
				return err
			}
			kind = "imported"
			abiSource = contractABIFile

		case contractBuiltin != "":
			b, ok := contract.GetBuiltin(contractBuiltin)
			if !ok {
				return fmt.Errorf("unknown built-in %q — run `w3cli contract builtins` to see all", contractBuiltin)
			}
			abi = b.ABI
			kind = "builtin"
			builtinID = contractBuiltin

		case contractFetchABI:
			fmt.Println(ui.Info("Fetching ABI from explorer..."))
			return fmt.Errorf("--fetch requires an explorer API key\n  Set one with: w3cli config set-explorer-key <key>\n  Or provide a local ABI file: --abi <file.json>")
		}

		reg.Add(&contract.Entry{
			Name:      name,
			Network:   network,
			Address:   address,
			ABI:       abi,
			Kind:      kind,
			BuiltinID: builtinID,
			ABISource: abiSource,
		})

		if err := reg.Save(); err != nil {
			return err
		}

		fmt.Println(ui.Success(fmt.Sprintf("Contract %q registered on %s at %s", name, network, ui.Addr(address))))
		if builtinID != "" {
			fmt.Println(ui.Hint(fmt.Sprintf("Using built-in ABI: %s (%d functions)", builtinID, countFunctions(abi))))
		}
		fmt.Println(ui.Hint("Explore it with: w3cli contract studio " + name))
		return nil
	},
}

// ── contract import ───────────────────────────────────────────────────────────

var contractImportCmd = &cobra.Command{
	Use:   "import <name> <address> --abi <path>",
	Short: "Import a contract from a Hardhat/Foundry artifact or raw ABI file",
	Long: `Import a contract from a local JSON file into the registry.

Supported formats:
  • Raw ABI array:       [{"type":"function",...}, ...]
  • Hardhat artifact:    {"abi":[...],"bytecode":"0x...","contractName":"..."}
  • Foundry artifact:    {"abi":[...],"bytecode":{"object":"0x..."},...}

Examples:
  w3cli contract import MyToken 0xABCD... --abi ./artifacts/MyToken.json
  w3cli contract import Vault   0x1234... --abi ./out/Vault.sol/Vault.json --network base`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, address := args[0], args[1]
		if contractABIFile == "" {
			return fmt.Errorf("--abi <path> is required for contract import")
		}
		network := contractNetwork
		if network == "" {
			network = cfg.DefaultNetwork
		}

		abi, err := contract.LoadFromArtifact(contractABIFile)
		if err != nil {
			return err
		}

		reg := newContractRegistry()
		if err := reg.Load(); err != nil {
			return err
		}

		reg.Add(&contract.Entry{
			Name:      name,
			Network:   network,
			Address:   address,
			ABI:       abi,
			Kind:      "imported",
			ABISource: contractABIFile,
		})

		if err := reg.Save(); err != nil {
			return err
		}

		fmt.Println(ui.Success(fmt.Sprintf(
			"Imported %q on %s at %s (%d ABI entries from %s)",
			name, network, ui.Addr(address), len(abi), contractABIFile)))
		fmt.Println(ui.Hint("Explore it with: w3cli contract studio " + name))
		return nil
	},
}

// ── contract builtins ─────────────────────────────────────────────────────────

var contractBuiltinsCmd = &cobra.Command{
	Use:   "builtins",
	Short: "List all bundled built-in contract ABIs",
	Long: `List all contract ABIs that are bundled into w3cli.

These can be used with:  w3cli contract add <name> <address> --builtin <id>

To add your own built-in, create internal/contract/<name>_abi.go
and call RegisterBuiltin() from init().`,
	RunE: func(cmd *cobra.Command, args []string) error {
		builtins := contract.AllBuiltins()
		if len(builtins) == 0 {
			fmt.Println(ui.Info("No built-ins registered."))
			return nil
		}

		fmt.Printf("%s\n\n", ui.StyleTitle.Render("Built-in Contract ABIs"))

		t := ui.NewTable([]ui.Column{
			{Title: "ID", Width: 12},
			{Title: "Name", Width: 36},
			{Title: "Functions", Width: 10},
			{Title: "Description", Width: 54},
		})

		for _, b := range builtins {
			funcs := countFunctions(b.ABI)
			t.AddRow(ui.Row{
				ui.Val(b.ID),
				b.Name,
				fmt.Sprintf("%d", funcs),
				ui.Meta(b.Description),
			})
		}
		fmt.Println(t.Render())
		fmt.Println(ui.Hint("Use: w3cli contract add <name> <addr> --builtin <id>"))
		return nil
	},
}

// ── contract list ─────────────────────────────────────────────────────────────

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
			fmt.Println(ui.Info("No contracts registered yet."))
			fmt.Println(ui.Hint("Add one with: w3cli contract add <name> <address> --abi <file.json>"))
			fmt.Println(ui.Hint("Or use a built-in: w3cli contract add <name> <address> --builtin erc20"))
			return nil
		}

		t := ui.NewTable([]ui.Column{
			{Title: "Name", Width: 16},
			{Title: "Network", Width: 14},
			{Title: "Address", Width: 44},
			{Title: "Kind", Width: 10},
			{Title: "ABI Funcs", Width: 10},
		})

		for _, e := range entries {
			kind := e.Kind
			if kind == "" {
				kind = "custom"
			}
			t.AddRow(ui.Row{
				ui.Val(e.Name),
				ui.ChainName(e.Network),
				ui.Addr(e.Address),
				kind,
				fmt.Sprintf("%d", countFunctions(e.ABI)),
			})
		}
		fmt.Println(t.Render())
		fmt.Println(ui.Meta(fmt.Sprintf("%d contract(s) registered", len(entries))))
		return nil
	},
}

// ── contract call ─────────────────────────────────────────────────────────────

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
			return fmt.Errorf("unknown chain %q — run `w3cli network list` to see all chains", network)
		}

		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
		if err != nil {
			return err
		}

		spin := ui.NewSpinner(fmt.Sprintf("Calling %s.%s() on %s (%s)...", name, funcName, network, cfg.NetworkMode))
		spin.Start()

		caller := contract.NewCallerFromEntries(rpcURL, entry.ABI)
		results, err := caller.Call(entry.Address, funcName, funcArgs...)
		spin.Stop()
		if err != nil {
			return err
		}

		fmt.Printf("\n%s  %s\n\n", ui.StyleTitle.Render(fmt.Sprintf("%s.%s() · %s (%s)", name, funcName, network, cfg.NetworkMode)), ui.Meta("→ result"))
		for i, r := range results {
			fmt.Printf("  [%d]  %s\n", i, ui.Val(r))
		}
		fmt.Println()
		return nil
	},
}

// ── contract sync ─────────────────────────────────────────────────────────────

var contractSyncCmd = &cobra.Command{
	Use:   "sync [contract]",
	Short: "Re-fetch ABI for a contract (or all)",
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		if !all && len(args) == 0 {
			return fmt.Errorf("provide a contract name or --all\n  Example: w3cli contract sync myToken\n  Or sync all: w3cli contract sync --all")
		}
		fmt.Println(ui.Info("ABI sync via explorer requires an API key."))
		fmt.Println(ui.Hint("Use `w3cli sync run` for manifest-based sync, or `w3cli config set-explorer-key <key>` to enable explorer sync."))
		return nil
	},
}

// ── contract studio ───────────────────────────────────────────────────────────

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

		// ── Header ───────────────────────────────────────────────────────────
		fmt.Printf("%s\n\n", ui.StyleTitle.Render(fmt.Sprintf("Contract Studio: %s on %s (%s)", name, network, cfg.NetworkMode)))
		fmt.Printf("  %s %s\n", ui.Meta("Address:"), ui.Addr(entry.Address))

		if entry.Kind != "" {
			badge := entry.Kind
			if entry.BuiltinID != "" {
				badge = fmt.Sprintf("builtin:%s", entry.BuiltinID)
			}
			fmt.Printf("  %s %s\n", ui.Meta("Kind:   "), ui.Val(badge))
		}
		if entry.Deployer != "" {
			fmt.Printf("  %s %s\n", ui.Meta("Deployer:"), ui.Addr(entry.Deployer))
		}
		if entry.TxHash != "" {
			fmt.Printf("  %s %s\n", ui.Meta("Deploy Tx:"), ui.Addr(entry.TxHash))
		}
		if entry.DeployedAt != "" {
			fmt.Printf("  %s %s\n", ui.Meta("Deployed: "), ui.Meta(entry.DeployedAt))
		}
		fmt.Println()

		// ── Read Functions ────────────────────────────────────────────────────
		fmt.Println(ui.StyleHeader.Render("Read Functions:"))
		hasRead := false
		for _, fn := range entry.ABI {
			if fn.IsReadFunction() {
				hasRead = true
				fmt.Printf("  %s  %s(%s)  →  %s\n",
					ui.Meta(fn.Selector()),
					ui.Val(fn.Name),
					ui.Meta(formatParams(fn.Inputs)),
					ui.Meta(formatOutputs(fn.Outputs)),
				)
			}
		}
		if !hasRead {
			fmt.Println(ui.Meta("  (none)"))
		}

		fmt.Println()
		fmt.Println(ui.StyleHeader.Render("Write Functions:"))
		hasWrite := false
		for _, fn := range entry.ABI {
			if fn.IsWriteFunction() {
				hasWrite = true
				fmt.Printf("  %s  %s(%s)\n",
					ui.Meta(fn.Selector()),
					ui.Warn(fn.Name),
					ui.Meta(formatParams(fn.Inputs)),
				)
			}
		}
		if !hasWrite {
			fmt.Println(ui.Meta("  (none)"))
		}

		// ── Events ────────────────────────────────────────────────────────────
		var events []contract.ABIEntry
		for _, fn := range entry.ABI {
			if fn.Type == "event" {
				events = append(events, fn)
			}
		}
		if len(events) > 0 {
			fmt.Println()
			fmt.Println(ui.StyleHeader.Render("Events:"))
			for _, ev := range events {
				fmt.Printf("  %s(%s)\n",
					ui.Info(ev.Name),
					ui.Meta(formatParams(ev.Inputs)),
				)
			}
		}

		fmt.Println()
		fmt.Println(ui.Hint("Use `w3cli contract call " + name + " <function> [args...]` to call a function."))
		return nil
	},
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	// add
	contractAddCmd.Flags().StringVar(&contractABIFile, "abi", "", "path to ABI JSON file or Hardhat/Foundry artifact")
	contractAddCmd.Flags().StringVar(&contractBuiltin, "builtin", "", "use a bundled ABI (see: w3cli contract builtins)")
	contractAddCmd.Flags().BoolVar(&contractFetchABI, "fetch", false, "auto-fetch ABI from explorer")
	contractAddCmd.Flags().StringVar(&contractNetwork, "network", "", "chain (default: config)")

	// import
	contractImportCmd.Flags().StringVar(&contractABIFile, "abi", "", "path to ABI JSON file or artifact (required)")
	contractImportCmd.Flags().StringVar(&contractNetwork, "network", "", "chain (default: config)")

	// call
	contractCallCmd.Flags().StringVar(&contractNetwork, "network", "", "chain (default: config)")

	// studio
	contractStudioCmd.Flags().StringVar(&contractNetwork, "network", "", "chain (default: config)")

	// sync
	contractSyncCmd.Flags().Bool("all", false, "sync all contracts")

	contractCmd.AddCommand(
		contractAddCmd,
		contractImportCmd,
		contractBuiltinsCmd,
		contractListCmd,
		contractCallCmd,
		contractSyncCmd,
		contractStudioCmd,
	)
}

// ── package helpers ───────────────────────────────────────────────────────────

func newContractRegistry() *contract.Registry {
	return contract.NewRegistry(filepath.Join(cfg.Dir(), "contracts.json"))
}

func newChainRegistry() *chainreg.Registry {
	return chainreg.NewRegistry()
}

// formatParams returns a comma-separated string of "type name" pairs.
func formatParams(params []contract.ABIParam) string {
	parts := make([]string, len(params))
	for i, p := range params {
		if p.Name != "" {
			parts[i] = p.Type + " " + p.Name
		} else {
			parts[i] = p.Type
		}
	}
	return strings.Join(parts, ", ")
}

// formatOutputs returns a comma-separated list of output types.
func formatOutputs(params []contract.ABIParam) string {
	if len(params) == 0 {
		return ""
	}
	types := make([]string, len(params))
	for i, p := range params {
		types[i] = p.Type
	}
	return strings.Join(types, ", ")
}

// countFunctions returns the number of "function" type entries in an ABI.
func countFunctions(abi []contract.ABIEntry) int {
	n := 0
	for _, e := range abi {
		if e.Type == "function" {
			n++
		}
	}
	return n
}
