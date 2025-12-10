package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/animus-coder/animus-coder/internal/config"
	"github.com/animus-coder/animus-coder/internal/daemon"
	"github.com/animus-coder/animus-coder/internal/logging"
	"github.com/animus-coder/animus-coder/internal/version"
)

func main() {
	var cfgPath string

	root := &cobra.Command{
		Use:     "mycodexd",
		Short:   "MyCodex daemon service",
		Version: version.Full(),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			logger, err := logging.NewLogger(cfg.Logging.Level, cfg.Logging.Format)
			if err != nil {
				return err
			}
			defer logger.Sync() //nolint:errcheck // best-effort

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			server, err := daemon.NewServer(cfg, logger)
			if err != nil {
				return err
			}
			return server.Run(ctx)
		},
	}

	root.Flags().StringVar(&cfgPath, "config", "", "Path to config file (default: configs/config.yaml)")

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
