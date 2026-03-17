package cli

import (
	"github.com/geodro/lerd/internal/ui"
	"github.com/geodro/lerd/internal/version"
	"github.com/spf13/cobra"
)

func NewServeUICmd() *cobra.Command {
	return &cobra.Command{
		Use:    "serve-ui",
		Short:  "Start the Lerd UI dashboard server",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return ui.Start(version.Version)
		},
	}
}
