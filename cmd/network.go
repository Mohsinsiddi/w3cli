package cmd

import (
	"fmt"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Manage networks",
}

var networkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all 26 supported chains",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg := chain.NewRegistry()
		t := ui.NewTable([]ui.Column{
			{Title: "#", Width: 3},
			{Title: "Name", Width: 16},
			{Title: "Display", Width: 20},
			{Title: "Chain ID", Width: 10},
			{Title: "Type", Width: 8},
			{Title: "Currency", Width: 10},
			{Title: "Testnet", Width: 16},
		})

		for i, c := range reg.All() {
			chainID := fmt.Sprintf("%d", c.ChainID)
			if c.ChainID == 0 {
				chainID = "—"
			}
			t.AddRow(ui.Row{
				fmt.Sprintf("%d", i+1),
				ui.ChainName(c.Name),
				c.DisplayName,
				chainID,
				string(c.Type),
				c.NativeCurrency,
				c.TestnetName,
			})
		}

		fmt.Println(t.Render())
		fmt.Printf("%s\n", ui.Meta(fmt.Sprintf("%d chains total", len(reg.All()))))
		return nil
	},
}

var networkUseCmd = &cobra.Command{
	Use:   "use <chain>",
	Short: "Set the default network",
	Long: `Set the default chain and persist it to config.

When combined with --testnet or --mainnet the network mode is also persisted.

Examples:
  w3cli network use base              # set default chain, keep current mode
  w3cli network use base --testnet    # set default chain + persist testnet mode
  w3cli network use polygon --mainnet # set default chain + persist mainnet mode`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reg := chain.NewRegistry()
		chainName := args[0]

		if _, err := reg.GetByName(chainName); err != nil {
			return fmt.Errorf("unknown chain %q — run `w3cli network list` to see all chains", chainName)
		}

		cfg.DefaultNetwork = chainName

		if err := cfg.Save(); err != nil {
			return err
		}

		mode := cfg.NetworkMode
		fmt.Println(ui.Success(fmt.Sprintf("Default network set to %s (%s)", ui.ChainName(chainName), mode)))
		return nil
	},
}

func init() {
	networkCmd.AddCommand(networkListCmd, networkUseCmd)
}
