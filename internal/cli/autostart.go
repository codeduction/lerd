package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/geodro/lerd/internal/config"
	lerdSystemd "github.com/geodro/lerd/internal/systemd"
	"github.com/spf13/cobra"
)

// NewAutostartCmd returns the autostart command with enable/disable subcommands.
func NewAutostartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "autostart",
		Short: "Manage autostart on login",
	}
	cmd.AddCommand(newAutostartEnableCmd())
	cmd.AddCommand(newAutostartDisableCmd())
	return cmd
}

func newAutostartEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable lerd autostart on login",
		RunE: func(_ *cobra.Command, _ []string) error {
			content, err := lerdSystemd.GetUnit("lerd-autostart")
			if err != nil {
				return err
			}
			if err := lerdSystemd.WriteService("lerd-autostart", content); err != nil {
				return fmt.Errorf("writing autostart service: %w", err)
			}
			if err := lerdSystemd.EnableService("lerd-autostart"); err != nil {
				return fmt.Errorf("enabling autostart service: %w", err)
			}
			fmt.Println("Autostart enabled — lerd will start automatically on login.")
			return nil
		},
	}
}

func newAutostartDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable lerd autostart on login",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := lerdSystemd.DisableService("lerd-autostart"); err != nil {
				return fmt.Errorf("disabling autostart service: %w", err)
			}
			// Remove the unit file
			unitPath := filepath.Join(config.SystemdUserDir(), "lerd-autostart.service")
			if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing autostart service file: %w", err)
			}
			fmt.Println("Autostart disabled — lerd will not start automatically on login.")
			return nil
		},
	}
}
