package cmd

import (
	"fmt"
	"strings"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ens"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var ensNetwork string

var ensCmd = &cobra.Command{
	Use:   "ens <name-or-address>",
	Short: "Resolve ENS names to addresses and vice versa",
	Long: `Resolve ENS names to Ethereum addresses or perform reverse lookups.

Auto-detects direction: if the input starts with 0x, it does a reverse
lookup. Otherwise, it resolves the name to an address.

ENS resolution always uses Ethereum mainnet (the ENS registry lives there).

Examples:
  w3cli ens vitalik.eth
  w3cli ens 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]

		// ENS always resolves on Ethereum mainnet.
		chainName := ensNetwork
		if chainName == "" {
			chainName = "ethereum"
		}

		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q", chainName)
		}

		// Force mainnet for ENS resolution.
		rpcURL, err := pickBestRPC(c, "mainnet")
		if err != nil {
			return err
		}

		client := chain.NewEVMClient(rpcURL)

		isAddress := strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X")

		if isAddress {
			// Reverse lookup: address → name.
			spin := ui.NewSpinner("Looking up reverse ENS record...")
			spin.Start()
			name, err := ens.ReverseLookup(input, client)
			spin.Stop()
			if err != nil {
				return fmt.Errorf("reverse lookup failed: %w", err)
			}

			// Also verify forward resolution matches.
			spin = ui.NewSpinner("Verifying forward resolution...")
			spin.Start()
			fwdAddr, fwdErr := ens.Resolve(name, client)
			spin.Stop()

			pairs := [][2]string{
				{"Address", ui.Addr(input)},
				{"ENS Name", ui.Val(name)},
			}
			if fwdErr == nil {
				if strings.EqualFold(fwdAddr, input) {
					pairs = append(pairs, [2]string{"Forward Check", ui.Success("matches")})
				} else {
					pairs = append(pairs, [2]string{"Forward Check", ui.Warn("forward resolves to " + fwdAddr)})
				}
			}

			fmt.Println(ui.KeyValueBlock("ENS Reverse Lookup", pairs))
		} else {
			// Forward resolution: name → address.
			spin := ui.NewSpinner(fmt.Sprintf("Resolving %s...", input))
			spin.Start()
			address, err := ens.Resolve(input, client)
			spin.Stop()
			if err != nil {
				return fmt.Errorf("resolution failed: %w", err)
			}

			// Try reverse lookup too.
			spin = ui.NewSpinner("Checking reverse record...")
			spin.Start()
			reverseName, revErr := ens.ReverseLookup(address, client)
			spin.Stop()

			pairs := [][2]string{
				{"ENS Name", ui.Val(input)},
				{"Address", ui.Addr(address)},
			}
			if revErr == nil && reverseName != "" {
				if strings.EqualFold(reverseName, input) {
					pairs = append(pairs, [2]string{"Reverse Check", ui.Success("matches")})
				} else {
					pairs = append(pairs, [2]string{"Reverse Check", ui.Warn("reverse resolves to " + reverseName)})
				}
			}

			fmt.Println(ui.KeyValueBlock("ENS Resolution", pairs))
		}

		return nil
	},
}

func init() {
	ensCmd.Flags().StringVar(&ensNetwork, "network", "", "chain for RPC (default: ethereum)")
}
