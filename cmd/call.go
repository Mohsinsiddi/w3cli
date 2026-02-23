package cmd

import (
	"fmt"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/contract"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var callNetwork string

var callCmd = &cobra.Command{
	Use:   "call <address> <function> [args...]",
	Short: "Call a read-only contract function",
	Long: `Call a read-only (view/pure) function on a smart contract.

For known ERC-20 functions (name, symbol, decimals, totalSupply, balanceOf,
allowance), the ABI is built-in. For other functions, supply the full
signature like "foo(uint256,address)".

Examples:
  w3cli call 0xUSDC name
  w3cli call 0xUSDC symbol
  w3cli call 0xUSDC decimals
  w3cli call 0xUSDC balanceOf 0xYourAddress
  w3cli call 0xUSDC allowance 0xOwner 0xSpender
  w3cli call 0xUSDC totalSupply --network ethereum`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		contractAddr := args[0]
		funcName := args[1]
		funcArgs := args[2:]

		chainName := callNetwork
		if chainName == "" {
			chainName = cfg.DefaultNetwork
		}

		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q — run `w3cli network list`", chainName)
		}

		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
		if err != nil {
			return err
		}

		spin := ui.NewSpinner(fmt.Sprintf("Calling %s on %s...", funcName, c.DisplayName))
		spin.Start()

		// Try ERC-20 ABI first (built-in functions).
		caller := contract.NewCallerFromEntries(rpcURL, contract.GetBuiltinABI("erc20"))
		results, err := caller.Call(contractAddr, funcName, funcArgs...)
		if err != nil {
			// If function not found in ERC-20 ABI, try parsing a manual signature.
			if strings.Contains(err.Error(), "not found") && strings.Contains(funcName, "(") {
				results, err = callWithSignature(rpcURL, contractAddr, funcName, funcArgs)
				if err != nil {
					spin.Stop()
					return err
				}
			} else {
				spin.Stop()
				return fmt.Errorf("contract call failed: %w", err)
			}
		}
		spin.Stop()

		// Format output.
		pairs := [][2]string{
			{"Contract", ui.Addr(contractAddr)},
			{"Function", ui.Val(funcName)},
			{"Network", fmt.Sprintf("%s (%s)", c.DisplayName, cfg.NetworkMode)},
		}

		if len(results) == 1 {
			pairs = append(pairs, [2]string{"Result", ui.Val(results[0])})
		} else {
			for i, r := range results {
				pairs = append(pairs, [2]string{fmt.Sprintf("Result[%d]", i), ui.Val(r)})
			}
		}

		fmt.Println(ui.KeyValueBlock("Contract Call", pairs))
		return nil
	},
}

// callWithSignature handles user-supplied function signatures like "foo(uint256,address)".
func callWithSignature(rpcURL, contractAddr, sig string, args []string) ([]string, error) {
	// Parse "funcName(type1,type2,...)" into an ABIEntry.
	parenIdx := strings.Index(sig, "(")
	if parenIdx < 0 || !strings.HasSuffix(sig, ")") {
		return nil, fmt.Errorf("invalid function signature %q — expected format: name(type1,type2)", sig)
	}

	name := sig[:parenIdx]
	typeStr := sig[parenIdx+1 : len(sig)-1]

	var inputs []contract.ABIParam
	if typeStr != "" {
		for _, t := range strings.Split(typeStr, ",") {
			inputs = append(inputs, contract.ABIParam{Name: "", Type: strings.TrimSpace(t)})
		}
	}

	abi := []contract.ABIEntry{{
		Name:            name,
		Type:            "function",
		Inputs:          inputs,
		Outputs:         []contract.ABIParam{{Name: "", Type: "uint256"}}, // default to uint256 output
		StateMutability: "view",
	}}

	caller := contract.NewCallerFromEntries(rpcURL, abi)
	return caller.Call(contractAddr, name, args...)
}

func init() {
	callCmd.Flags().StringVar(&callNetwork, "network", "", "chain (default: config)")
}
