package cmd

import (
	"fmt"
	"math/big"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	simFrom    string
	simTo      string
	simData    string
	simValue   string
	simNetwork string
)

var simulateCmd = &cobra.Command{
	Use:   "simulate",
	Short: "Simulate a transaction via eth_call (dry-run)",
	Long: `Simulate a transaction without broadcasting it.

Uses eth_call to check if a transaction would succeed or revert,
and reports the gas estimate.

Examples:
  w3cli simulate --from 0x... --to 0x... --data 0xa9059cbb...
  w3cli simulate --from 0x... --to 0x... --value 1.0 --network ethereum`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if simFrom == "" {
			addr, _, err := resolveWalletAndChain("", simNetwork)
			if err != nil {
				return fmt.Errorf("--from is required or set a default wallet")
			}
			simFrom = addr
		}
		if simTo == "" {
			return fmt.Errorf("--to is required — provide the target address")
		}

		chainName := simNetwork
		if chainName == "" {
			chainName = cfg.DefaultNetwork
		}

		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q", chainName)
		}

		rpcURL, err := pickBestRPC(c, cfg.NetworkMode)
		if err != nil {
			return err
		}

		client := chain.NewEVMClient(rpcURL)

		// Parse value.
		var valueWei *big.Int
		if simValue != "" {
			valueWei, err = ethToWei(simValue)
			if err != nil {
				return fmt.Errorf("invalid value %q: %w", simValue, err)
			}
		}

		spin := ui.NewSpinner(fmt.Sprintf("Simulating on %s...", c.DisplayName))
		spin.Start()

		ok, result, err := client.SimulateCall(simFrom, simTo, simData, valueWei)
		if err != nil {
			spin.Stop()
			return fmt.Errorf("simulation error: %w", err)
		}

		// Also try to estimate gas.
		gasEstimate := uint64(0)
		if ok {
			if gas, gasErr := client.EstimateGas(simFrom, simTo, simData, valueWei); gasErr == nil {
				gasEstimate = gas
			}
		}
		spin.Stop()

		pairs := [][2]string{
			{"From", ui.Addr(simFrom)},
			{"To", ui.Addr(simTo)},
		}
		if simData != "" {
			method := chain.DecodeMethod(simData)
			pairs = append(pairs, [2]string{"Method", method})
		}
		if simValue != "" {
			pairs = append(pairs, [2]string{"Value", simValue + " ETH"})
		}
		pairs = append(pairs, [2]string{"Network", fmt.Sprintf("%s (%s)", c.DisplayName, cfg.NetworkMode)})

		if ok {
			pairs = append(pairs, [2]string{"Status", ui.Success("transaction would succeed")})
			if gasEstimate > 0 {
				pairs = append(pairs, [2]string{"Gas Estimate", fmt.Sprintf("%d", gasEstimate)})
			}
			if result != "" && result != "0x" {
				pairs = append(pairs, [2]string{"Return Data", result})
			}
		} else {
			pairs = append(pairs, [2]string{"Status", ui.Err("transaction would REVERT")})
			if result != "" {
				pairs = append(pairs, [2]string{"Revert Reason", result})
			}
		}

		fmt.Println(ui.KeyValueBlock("Simulation Result", pairs))
		return nil
	},
}

func init() {
	simulateCmd.Flags().StringVar(&simFrom, "from", "", "sender address or wallet name")
	simulateCmd.Flags().StringVar(&simTo, "to", "", "target address (required)")
	simulateCmd.Flags().StringVar(&simData, "data", "", "calldata (hex)")
	simulateCmd.Flags().StringVar(&simValue, "value", "", "ETH value to send")
	simulateCmd.Flags().StringVar(&simNetwork, "network", "", "chain (default: config)")
}
