package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Mohsinsiddi/w3cli/internal/contract"
	csync "github.com/Mohsinsiddi/w3cli/internal/sync"
	"github.com/Mohsinsiddi/w3cli/internal/ui"
	"github.com/spf13/cobra"
)

var syncWatch bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync contracts from a remote manifest",
}

var syncSetSourceCmd = &cobra.Command{
	Use:   "set-source <url>",
	Short: "Set the remote deployments manifest URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		reg := contract.NewRegistry(filepath.Join(cfg.Dir(), "contracts.json"))
		syncer := csync.New(cfg, reg)
		if err := syncer.SetSource(url); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("Sync source set to: %s", url)))
		return nil
	},
}

var syncRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Fetch latest contracts from the manifest",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg := contract.NewRegistry(filepath.Join(cfg.Dir(), "contracts.json"))
		if err := reg.Load(); err != nil {
			return err
		}

		syncer := csync.New(cfg, reg)

		if syncWatch {
			fmt.Println(ui.Meta("Watching for changes every 30s. Press Ctrl+C to stop."))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			return syncer.Watch(ctx, 30*time.Second)
		}

		spin := ui.NewSpinner("Syncing contracts...")
		spin.Start()
		err := syncer.Run(context.Background())
		spin.Stop()
		if err != nil {
			return err
		}
		fmt.Println(ui.Success("Contracts synced successfully!"))
		return nil
	},
}

func init() {
	syncRunCmd.Flags().BoolVar(&syncWatch, "watch", false, "poll every 30s for changes")
	syncCmd.AddCommand(syncSetSourceCmd, syncRunCmd)
}
