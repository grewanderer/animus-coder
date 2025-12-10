package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewDoctorCmd returns a health-check command validating config and environment.
func NewDoctorCmd(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Validate configuration and environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(opts)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Config OK. Providers: %d, models: %d\n", len(cfg.Providers), len(cfg.Models))
			fmt.Fprintf(out, "Sandbox enabled: %v, metrics: %v\n", cfg.Sandbox.Enabled, cfg.Server.MetricsEnabled)
			return nil
		},
	}
}
