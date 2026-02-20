package cmd

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"time"

	chainpkg "github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/contract"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/Mohsinsiddi/w3cli/internal/wallet"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/spf13/cobra"
)

// tokenAmountFunctions is the set of ERC-20 functions whose uint256 value/amount
// params represent token quantities (and should be scaled by token decimals).
var tokenAmountFunctions = map[string]bool{
	"transfer": true, "approve": true, "transferFrom": true,
	"burn": true, "burnFrom": true, "mint": true,
	"increaseAllowance": true, "decreaseAllowance": true,
}

var (
	contractABIFile    string
	contractBuiltin    string // --builtin <id>
	contractFetchABI   bool
	contractNetwork    string
	contractStudioWallet string
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
			t.AddRow(ui.Row{
				ui.Val(b.ID),
				b.Name,
				fmt.Sprintf("%d", countFunctions(b.ABI)),
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
			fmt.Println(ui.Hint("Add one: w3cli contract add <name> <address> --builtin erc20"))
			return nil
		}

		t := ui.NewTable([]ui.Column{
			{Title: "Name", Width: 16},
			{Title: "Network", Width: 14},
			{Title: "Address", Width: 44},
			{Title: "Kind", Width: 10},
			{Title: "Functions", Width: 10},
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
			return fmt.Errorf("unknown chain %q — run `w3cli network list`", network)
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

		fmt.Printf("\n%s  %s\n\n",
			ui.StyleTitle.Render(fmt.Sprintf("%s.%s() · %s (%s)", name, funcName, network, cfg.NetworkMode)),
			ui.Meta("→ result"))
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
			return fmt.Errorf("provide a contract name or --all")
		}
		fmt.Println(ui.Info("ABI sync via explorer requires an API key."))
		fmt.Println(ui.Hint("Use `w3cli sync run` for manifest-based sync."))
		return nil
	},
}

// ── contract studio ───────────────────────────────────────────────────────────

var contractStudioCmd = &cobra.Command{
	Use:   "studio [contract]",
	Short: "Interactive contract explorer — navigate, call, and send transactions",
	Long: `Launch the interactive contract studio.

Navigate functions with ↑↓ (or j/k), press Enter to select.
For read functions: results are displayed immediately.
For write functions: inputs are collected, a preview is shown, and you confirm before broadcasting.

Specify a contract name to open it directly, or omit to pick from the registered list.

Examples:
  w3cli contract studio MTK --network ethereum --testnet
  w3cli contract studio USDC --network base --wallet my-signer`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		network := contractNetwork
		if network == "" {
			network = cfg.DefaultNetwork
		}

		reg := newContractRegistry()
		if err := reg.Load(); err != nil {
			return err
		}

		// ── Contract selection ────────────────────────────────────────────
		var contractName string
		if len(args) == 1 {
			contractName = args[0]
		} else {
			entries := reg.All()
			if len(entries) == 0 {
				return fmt.Errorf("no contracts registered\n  Add one: w3cli contract add <name> <address> --builtin erc20")
			}
			fmt.Println(ui.StyleTitle.Render("  Registered Contracts") + "\n")
			t := ui.NewTable([]ui.Column{
				{Title: "#", Width: 4},
				{Title: "Name", Width: 16},
				{Title: "Network", Width: 14},
				{Title: "Address", Width: 44},
			})
			names := make([]string, len(entries))
			for i, e := range entries {
				names[i] = e.Name
				t.AddRow(ui.Row{
					fmt.Sprintf("%d", i+1),
					ui.Val(e.Name),
					ui.ChainName(e.Network),
					ui.Addr(e.Address),
				})
			}
			fmt.Println(t.Render())
			contractName = ui.PromptInput("Contract name")
			if contractName == "" {
				return nil
			}
		}

		entry, err := reg.Get(contractName, network)
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

		// ── Resolve wallet (needed for write functions) ───────────────────
		walletName := contractStudioWallet
		if walletName == "" {
			walletName = cfg.DefaultWallet
		}
		store := wallet.NewJSONStore(filepath.Join(cfg.Dir(), "wallets.json"))
		mgr := wallet.NewManager(wallet.WithStore(store))
		signerWallet, _ := mgr.Get(walletName) // nil if not found; checked on write

		// ── Build studio entries ──────────────────────────────────────────
		kind := entry.Kind
		if entry.BuiltinID != "" {
			kind = "builtin:" + entry.BuiltinID
		}

		// Fetch token decimals so amount params show correct examples.
		client := chainpkg.NewEVMClient(rpcURL)
		tokenDecimals := fetchTokenDecimals(client, entry.Address)

		studioEntries := abiToStudioEntries(entry.ABI, tokenDecimals)
		funcCount := countFunctions(entry.ABI)
		eventCount := 0
		for _, e := range entry.ABI {
			if e.Type == "event" {
				eventCount++
			}
		}

		// ── Main studio loop ──────────────────────────────────────────────
		for {
			model := ui.StudioModel{
				ContractName: contractName,
				Address:      entry.Address,
				Network:      network,
				Mode:         cfg.NetworkMode,
				Kind:         kind,
				FuncCount:    funcCount,
				EventCount:   eventCount,
				Entries:      studioEntries,
			}

			selected, err := ui.RunStudio(model)
			if err != nil {
				return err
			}
			if selected == nil {
				// User pressed q — exit
				fmt.Println(ui.Meta("Exiting contract studio."))
				return nil
			}

			// ── Collect inputs ────────────────────────────────────────────
			fmt.Println()
			fmt.Printf("%s\n\n",
				ui.StyleTitle.Render(fmt.Sprintf("  %s  ›  %s", contractName, selected.Sig)))
			if selected.Description != "" {
				fmt.Println(ui.Meta("  " + selected.Description))
				fmt.Println()
			}

			inputs, err := collectStudioInputs(selected.Inputs)
			if err != nil {
				return err
			}

			// ── Execute ───────────────────────────────────────────────────
			if !selected.IsWrite {
				// ── Read call ─────────────────────────────────────────────
				studioExecuteRead(entry, selected, inputs, rpcURL, contractName, network, cfg.NetworkMode)
			} else {
				// ── Write tx ──────────────────────────────────────────────
				if signerWallet == nil {
					fmt.Println(ui.Err(fmt.Sprintf("wallet %q not found — use --wallet <name>", walletName)))
				} else if signerWallet.Type != wallet.TypeSigning {
					fmt.Println(ui.Err(fmt.Sprintf("wallet %q is watch-only — use a signing wallet", walletName)))
				} else {
					studioExecuteWrite(entry, selected, inputs, client, c, signerWallet, contractName, cfg.NetworkMode)
				}
			}

			// ── Loop prompt ───────────────────────────────────────────────
			fmt.Println()
			if !ui.Confirm("Call another function?") {
				fmt.Println(ui.Meta("Exiting contract studio."))
				return nil
			}
		}
	},
}

// ── studio: input collection ─────────────────────────────────────────────────

func collectStudioInputs(params []ui.StudioParam) ([]string, error) {
	if len(params) == 0 {
		return nil, nil
	}

	inputs := make([]string, len(params))
	for i, p := range params {
		typeLabel := p.Type
		if p.IsTokenAmount {
			typeLabel = fmt.Sprintf("token amount  [decimals: %d]", p.Decimals)
		}

		fmt.Printf("%s\n", ui.StyleHeader.Render(fmt.Sprintf(
			"  ── Parameter %d / %d: %s (%s) ", i+1, len(params), p.Name, typeLabel)))
		if p.Example != "" {
			fmt.Printf("  %s %s\n", ui.Meta("Example:"), ui.StyleMeta.Render(p.Example))
		}

		for {
			val := ui.PromptInput(fmt.Sprintf("  %s", p.Name))

			if p.IsTokenAmount {
				// Accept human-readable decimal input (e.g. "1.5") and scale.
				scaled, errMsg := scaleTokenInput(val, p.Decimals)
				if errMsg != "" {
					fmt.Println(ui.Err("  " + errMsg))
					continue
				}
				rawStr := scaled.String()
				fmt.Printf("  %s\n", ui.Success(fmt.Sprintf("✓  %s = %s  (raw: %s)",
					p.Name, val, rawStr)))
				inputs[i] = rawStr
			} else {
				if errMsg := validateABIInput(p.Type, val); errMsg != "" {
					fmt.Println(ui.Err("  " + errMsg))
					continue
				}
				inputs[i] = val
				fmt.Printf("  %s\n", ui.Success(fmt.Sprintf("✓  %s = %s", p.Name, val)))
			}
			break
		}
		fmt.Println()
	}
	return inputs, nil
}

// scaleTokenInput parses a human decimal (e.g. "1.5") and returns
// the raw uint256 value scaled by 10^decimals.
func scaleTokenInput(val string, decimals int) (*big.Int, string) {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil, "value cannot be empty"
	}
	f, ok := new(big.Float).SetPrec(256).SetString(val)
	if !ok || f.Sign() < 0 {
		return nil, fmt.Sprintf("invalid amount %q — enter a non-negative number (e.g. 1.5)", val)
	}
	scale := new(big.Float).SetInt(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	scaled := new(big.Float).Mul(f, scale)
	result, accuracy := scaled.Int(nil)
	_ = accuracy
	if result.Sign() < 0 {
		return nil, "amount must be non-negative"
	}
	return result, ""
}

// validateABIInput returns an error message string (empty = valid).
func validateABIInput(typ, val string) string {
	if val == "" {
		return "value cannot be empty"
	}
	switch {
	case typ == "address":
		v := strings.TrimSpace(val)
		if !strings.HasPrefix(v, "0x") && !strings.HasPrefix(v, "0X") {
			return "address must start with 0x"
		}
		if len(v) != 42 {
			return fmt.Sprintf("address must be 42 characters (got %d)", len(v))
		}
		if _, err := hex.DecodeString(v[2:]); err != nil {
			return "address contains invalid hex characters"
		}

	case typ == "bool":
		v := strings.ToLower(strings.TrimSpace(val))
		if v != "true" && v != "false" && v != "1" && v != "0" {
			return "must be: true, false, 1, or 0"
		}

	case strings.HasPrefix(typ, "uint"):
		n, ok := new(big.Int).SetString(strings.TrimSpace(val), 10)
		if !ok {
			return "must be a non-negative decimal integer (e.g. 1000000)"
		}
		if n.Sign() < 0 {
			return fmt.Sprintf("%s cannot be negative", typ)
		}

	case strings.HasPrefix(typ, "int"):
		if _, ok := new(big.Int).SetString(strings.TrimSpace(val), 10); !ok {
			return "must be a decimal integer (e.g. -100 or 100)"
		}

	case typ == "bytes32":
		v := strings.TrimPrefix(strings.TrimSpace(val), "0x")
		if len(v) > 64 {
			return "bytes32 must be at most 64 hex characters"
		}
		if _, err := hex.DecodeString(v); err != nil {
			return "must be a hex string (e.g. 0xdeadbeef...)"
		}

	case typ == "bytes":
		v := strings.TrimPrefix(strings.TrimSpace(val), "0x")
		if _, err := hex.DecodeString(v); err != nil {
			return "must be a hex string (e.g. 0xdeadbeef)"
		}
	}
	return "" // valid
}

// ── studio: read execution ────────────────────────────────────────────────────

func studioExecuteRead(
	entry *contract.Entry,
	fn *ui.StudioEntry,
	inputs []string,
	rpcURL string,
	contractName, network, mode string,
) {
	spin := ui.NewSpinner(fmt.Sprintf("Calling %s()...", fn.Name))
	spin.Start()

	caller := contract.NewCallerFromEntries(rpcURL, entry.ABI)
	results, err := caller.Call(entry.Address, fn.Name, inputs...)
	spin.Stop()

	if err != nil {
		fmt.Println(ui.Err(err.Error()))
		return
	}

	pairs := make([][2]string, 0, len(results)+2)
	pairs = append(pairs, [2]string{"Contract", ui.Addr(entry.Address)})
	pairs = append(pairs, [2]string{"Function", fn.Sig})
	for i, r := range results {
		label := fmt.Sprintf("Result [%d]", i)
		if i < len(fn.OutputTypes) {
			label = fn.OutputTypes[i]
		}
		pairs = append(pairs, [2]string{label, ui.Val(r)})
	}

	fmt.Println()
	fmt.Println(ui.KeyValueBlock(fmt.Sprintf("%s() · %s (%s)", fn.Name, network, mode), pairs))
}

// ── studio: write tx execution ────────────────────────────────────────────────

func studioExecuteWrite(
	entry *contract.Entry,
	fn *ui.StudioEntry,
	inputs []string,
	client *chainpkg.EVMClient,
	c *chainpkg.Chain,
	w *wallet.Wallet,
	contractName, mode string,
) {
	// Find the matching ABIEntry for EncodeCalldata
	var abiEntry contract.ABIEntry
	for _, e := range entry.ABI {
		if e.Type == "function" && e.Name == fn.Name {
			abiEntry = e
			break
		}
	}

	calldataHex, calldataRaw, err := contract.EncodeCalldata(abiEntry, inputs)
	if err != nil {
		fmt.Println(ui.Err("encoding calldata: " + err.Error()))
		return
	}

	gasPrice, err := client.GasPrice()
	if err != nil {
		fmt.Println(ui.Err("getting gas price: " + err.Error()))
		return
	}

	gasLimit, err := client.EstimateGas(w.Address, entry.Address, calldataHex, nil)
	if err != nil {
		gasLimit = 200_000 // safe fallback
	}

	chainID, err := client.ChainID()
	if err != nil {
		fmt.Println(ui.Err("getting chainID: " + err.Error()))
		return
	}

	nonce, err := client.GetNonce(w.Address)
	if err != nil {
		fmt.Println(ui.Err("getting nonce: " + err.Error()))
		return
	}

	// ── Preview ───────────────────────────────────────────────────────────
	pairs := [][2]string{
		{"Contract", ui.Addr(entry.Address)},
		{"Function", fn.Sig},
		{"Selector", fn.Selector},
		{"From", ui.Addr(w.Address)},
	}
	for i, p := range fn.Inputs {
		lbl := p.Name
		if lbl == "" {
			lbl = fmt.Sprintf("arg%d", i)
		}
		if i < len(inputs) {
			pairs = append(pairs, [2]string{lbl, inputs[i]})
		}
	}
	pairs = append(pairs,
		[2]string{"Gas Limit", fmt.Sprintf("%d", gasLimit)},
		[2]string{"Gas Price", fmt.Sprintf("%d Gwei", toGwei(gasPrice))},
		[2]string{"Network", fmt.Sprintf("%s (%s)", c.DisplayName, mode)},
	)

	fmt.Println()
	fmt.Println(ui.KeyValueBlock(fmt.Sprintf("Transaction Preview · %s()", fn.Name), pairs))

	if !ui.Confirm("Broadcast this transaction?") {
		fmt.Println(ui.Meta("Cancelled."))
		return
	}

	// ── Sign + broadcast ──────────────────────────────────────────────────
	contractAddr, err := parseAddress(entry.Address)
	if err != nil {
		fmt.Println(ui.Err(err.Error()))
		return
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(chainID),
		Nonce:     nonce,
		GasTipCap: gasPrice,
		GasFeeCap: new(big.Int).Mul(gasPrice, big.NewInt(2)),
		Gas:       gasLimit,
		To:        &contractAddr,
		Value:     big.NewInt(0),
		Data:      calldataRaw,
	})

	ks := wallet.DefaultKeystore()
	signer := wallet.NewSigner(w, ks)

	spin := ui.NewSpinner(fmt.Sprintf("Broadcasting %s()...", fn.Name))
	spin.Start()
	raw, err := signer.SignTx(tx, big.NewInt(chainID))
	if err != nil {
		spin.Stop()
		fmt.Println(ui.Err("signing: " + err.Error()))
		return
	}
	hash, err := client.SendRawTransaction("0x" + hex.EncodeToString(raw))
	spin.Stop()
	if err != nil {
		fmt.Println(ui.Err("broadcast: " + err.Error()))
		return
	}

	// ── Wait for receipt ──────────────────────────────────────────────────
	spin = ui.NewSpinner(fmt.Sprintf("Waiting for %s() to be mined...", fn.Name))
	spin.Start()
	receipt, err := client.WaitForReceipt(hash, 3*time.Minute)
	spin.Stop()

	explorer := c.Explorer(mode)
	if err != nil {
		fmt.Println(ui.Err(fmt.Sprintf("tx %s failed: %s", hash, err)))
		fmt.Printf("  Explorer: %s\n", explorer+"/tx/"+hash)
		return
	}

	fmt.Println()
	fmt.Println(ui.KeyValueBlock(fmt.Sprintf("%s() Confirmed ✓", fn.Name), [][2]string{
		{"Tx Hash", ui.Addr(hash)},
		{"Block", fmt.Sprintf("%d", receipt.BlockNumber)},
		{"Gas Used", fmt.Sprintf("%d", receipt.GasUsed)},
		{"Status", ui.Success("success")},
		{"Explorer", explorer + "/tx/" + hash},
	}))
	ui.OpenURL(explorer + "/tx/" + hash)
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
	contractStudioCmd.Flags().StringVar(&contractStudioWallet, "wallet", "", "signing wallet for write functions (default: config)")

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

// ── package-level helpers ─────────────────────────────────────────────────────

func newContractRegistry() *contract.Registry {
	return contract.NewRegistry(filepath.Join(cfg.Dir(), "contracts.json"))
}

func newChainRegistry() *chainpkg.Registry {
	return chainpkg.NewRegistry()
}

func countFunctions(abi []contract.ABIEntry) int {
	n := 0
	for _, e := range abi {
		if e.Type == "function" {
			n++
		}
	}
	return n
}

// formatParams and formatOutputs used by the old static studio — kept for compatibility.
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

// abiToStudioEntries converts contract ABI entries to ui.StudioEntry values.
// decimals is the token decimals (used to annotate amount params); pass -1 if unknown.
func abiToStudioEntries(abi []contract.ABIEntry, decimals int) []ui.StudioEntry {
	entries := make([]ui.StudioEntry, 0, len(abi))
	for _, e := range abi {
		if e.Type != "function" && e.Type != "event" {
			continue
		}

		isTokenFunc := tokenAmountFunctions[e.Name]

		params := make([]ui.StudioParam, len(e.Inputs))
		for i, p := range e.Inputs {
			isAmt := isTokenFunc && p.Type == "uint256" &&
				(p.Name == "value" || p.Name == "amount" || p.Name == "wad")

			example := abiTypeExample(p.Type)
			if isAmt && decimals >= 0 {
				one := new(big.Float).SetPrec(256).SetFloat64(1.0)
				example = fmt.Sprintf("1  or  0.5  (human units, scaled ×10^%d = %s raw per unit)",
					decimals, scaledOneStr(decimals))
				_ = one
			}

			params[i] = ui.StudioParam{
				Name:          p.Name,
				Type:          p.Type,
				Example:       example,
				IsTokenAmount: isAmt && decimals >= 0,
				Decimals:      decimals,
			}
		}

		outTypes := make([]string, len(e.Outputs))
		for i, o := range e.Outputs {
			outTypes[i] = o.Type
		}

		// Build canonical signature
		inputTypes := make([]string, len(e.Inputs))
		for i, p := range e.Inputs {
			inputTypes[i] = p.Type
		}
		sig := e.Name + "(" + strings.Join(inputTypes, ",") + ")"

		entries = append(entries, ui.StudioEntry{
			Name:        e.Name,
			Selector:    e.Selector(),
			Sig:         sig,
			IsWrite:     e.IsWriteFunction(),
			IsEvent:     e.Type == "event",
			Inputs:      params,
			OutputTypes: outTypes,
			Description: abiKnownDescription(e.Name),
		})
	}
	return entries
}

// abiTypeExample returns a human-friendly example value for an ABI type.
func abiTypeExample(typ string) string {
	switch {
	case typ == "address":
		return "0xAbCd1234...EF56  (42 hex chars, 0x prefix)"
	case typ == "uint256":
		return "1000000000000000000  (= 1.0 with 18 decimals)  or  100  (for 6-decimal tokens)"
	case typ == "uint8":
		return "18  (typical token decimals)"
	case typ == "bool":
		return "true  or  false"
	case typ == "string":
		return "hello world"
	case typ == "bytes32":
		return "0x0000000000000000000000000000000000000000000000000000000000000000"
	case typ == "bytes":
		return "0xdeadbeef"
	case strings.HasPrefix(typ, "uint"):
		return "1000000"
	case strings.HasPrefix(typ, "int"):
		return "100  or  -100"
	default:
		return ""
	}
}

// fetchTokenDecimals calls decimals() on the contract and returns the result.
// Returns -1 if the call fails or the contract has no decimals() function.
func fetchTokenDecimals(client *chainpkg.EVMClient, contractAddr string) int {
	raw, err := client.CallContract(contractAddr, "0x313ce567")
	if err != nil || len(raw) < 66 {
		return -1
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(raw, "0x"), 16)
	if !ok {
		return -1
	}
	return int(n.Int64())
}

// scaledOneStr returns the string representation of 10^decimals (= 1 token in raw units).
func scaledOneStr(decimals int) string {
	v := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	return v.String()
}

// abiKnownDescription maps well-known function names to human descriptions.
func abiKnownDescription(name string) string {
	descriptions := map[string]string{
		"name":               "Returns the token name set at deployment (e.g. \"MyToken\").",
		"symbol":             "Returns the token ticker symbol (e.g. \"MTK\").",
		"decimals":           "Returns the number of decimal places (typically 18 for ERC-20, 6 for USDC).",
		"totalSupply":        "Returns total tokens in circulation as a raw integer (divide by 10^decimals for display).",
		"balanceOf":          "Returns the token balance of a given address as a raw integer.",
		"allowance":          "Returns how many tokens the `spender` is approved to spend on behalf of `owner`.",
		"transfer":           "Transfer tokens from your wallet to `to`. Emits a Transfer event.",
		"approve":            "Approve `spender` to transfer up to `value` tokens on your behalf. Emits Approval.",
		"transferFrom":       "Transfer tokens from `from` to `to` using an existing allowance. Emits Transfer.",
		"burn":               "Permanently destroy `value` tokens from your own wallet. Irreversible.",
		"burnFrom":           "Burn `value` tokens from `account` using your allowance. Irreversible.",
		"owner":              "Returns the current contract owner address (Ownable).",
		"transferOwnership":  "Transfer contract ownership to `newOwner`. Only the current owner can call this.",
		"renounceOwnership":  "Permanently renounce ownership. The contract will have no owner. Irreversible!",
		"mint":               "Mint `amount` new tokens to `to`. Only the contract owner can call this.",
		"Transfer":           "Emitted on every token transfer including mints (from=0x0) and burns (to=0x0).",
		"Approval":           "Emitted when an allowance is set via approve() or increaseAllowance().",
		"OwnershipTransferred": "Emitted when ownership is transferred or renounced.",
	}
	if d, ok := descriptions[name]; ok {
		return d
	}
	return ""
}
