package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/chain"
	"github.com/Mohsinsiddi/w3cli/internal/rpc"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var rpcCmd = &cobra.Command{
	Use:   "rpc",
	Short: "Manage RPC endpoints",
}

var rpcAddCmd = &cobra.Command{
	Use:   "add <chain> <url>",
	Short: "Add a custom RPC URL for a chain",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		chainName, url := args[0], args[1]
		reg := chain.NewRegistry()
		if _, err := reg.GetByName(chainName); err != nil {
			return fmt.Errorf("unknown chain %q", chainName)
		}
		if err := cfg.AddRPC(chainName, url); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("Added RPC for %s: %s", ui.ChainName(chainName), url)))
		return nil
	},
}

var rpcRemoveCmd = &cobra.Command{
	Use:   "remove <chain> <url>",
	Short: "Remove a custom RPC URL",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		chainName, url := args[0], args[1]
		if err := cfg.RemoveRPC(chainName, url); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("Removed RPC for %s: %s", chainName, url)))
		return nil
	},
}

var rpcListCmd = &cobra.Command{
	Use:   "list <chain>",
	Short: "List all RPCs for a chain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chainName := args[0]
		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q", chainName)
		}

		fmt.Printf("%s\n", ui.StyleTitle.Render(fmt.Sprintf("RPCs for %s", c.DisplayName)))

		fmt.Println(ui.StyleHeader.Render("Built-in RPCs:"))
		for _, r := range c.MainnetRPCs {
			fmt.Printf("  %s %s\n", ui.Meta("(mainnet)"), r)
		}
		for _, r := range c.TestnetRPCs {
			fmt.Printf("  %s %s\n", ui.Meta("(testnet)"), r)
		}

		custom := cfg.GetRPCs(chainName)
		if len(custom) > 0 {
			fmt.Println(ui.StyleHeader.Render("Custom RPCs:"))
			for _, r := range custom {
				fmt.Printf("  %s\n", r)
			}
		}
		return nil
	},
}

var rpcBenchmarkCmd = &cobra.Command{
	Use:   "benchmark <chain>",
	Short: "Benchmark all RPCs for a chain and pick the fastest",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chainName := args[0]
		reg := chain.NewRegistry()
		c, err := reg.GetByName(chainName)
		if err != nil {
			return fmt.Errorf("unknown chain %q", chainName)
		}

		rpcs := c.RPCs(cfg.NetworkMode)
		if custom := cfg.GetRPCs(chainName); len(custom) > 0 {
			rpcs = append(custom, rpcs...)
		}

		fmt.Printf("%s\n\n", ui.StyleTitle.Render(fmt.Sprintf("Benchmarking %s RPCs...", c.DisplayName)))

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		results := rpc.BenchmarkEVM(ctx, rpcs)

		t := ui.NewTable([]ui.Column{
			{Title: "RPC URL", Width: 40},
			{Title: "Latency", Width: 12},
			{Title: "Block #", Width: 12},
			{Title: "Status", Width: 10},
		})

		for _, r := range results {
			status := ui.Success("healthy")
			latency := fmt.Sprintf("%dms", r.Latency.Milliseconds())
			block := fmt.Sprintf("%d", r.BlockNumber)

			if r.Err != nil {
				status = ui.Err("down")
				latency = "—"
				block = "—"
			}

			t.AddRow(ui.Row{r.URL, latency, block, status})
		}

		fmt.Println(t.Render())
		return nil
	},
}

var rpcAlgorithmCmd = &cobra.Command{
	Use:   "algorithm set <fastest|round-robin|failover>",
	Short: "Set the RPC selection algorithm",
}

var rpcAlgorithmSetCmd = &cobra.Command{
	Use:   "set <algorithm>",
	Short: "Set the RPC selection algorithm",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		algo := args[0]
		switch algo {
		case "fastest", "round-robin", "failover":
		default:
			return fmt.Errorf("invalid algorithm %q — choose: fastest, round-robin, failover", algo)
		}
		cfg.RPCAlgorithm = algo
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("RPC algorithm set to %q", algo)))
		return nil
	},
}

func init() {
	rpcAlgorithmCmd.AddCommand(rpcAlgorithmSetCmd)
	rpcCmd.AddCommand(rpcAddCmd, rpcRemoveCmd, rpcListCmd, rpcBenchmarkCmd, rpcAlgorithmCmd)
}
