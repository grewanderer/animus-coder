package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/animus-coder/animus-coder/internal/config"
	"github.com/animus-coder/animus-coder/internal/version"
)

// Options holds global CLI options.
type Options struct {
	ConfigPath string
}

// NewRootCmd constructs the base CLI command tree.
func NewRootCmd() *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:           "mycodex",
		Short:         "MyCodex CLI – local/remote agent driver",
		Version:       version.Full(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVar(&opts.ConfigPath, "config", "", "Path to config file (default: configs/config.yaml)")

	cmd.AddCommand(NewDoctorCmd(opts))
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewRunCmd(opts))

	return cmd
}

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// loadConfig wraps config loading with shared options.
func loadConfig(opts *Options) (*config.Config, error) {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}
