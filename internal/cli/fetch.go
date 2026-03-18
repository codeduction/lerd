package cli

import (
	"fmt"

	"github.com/geodro/lerd/internal/podman"
	"github.com/spf13/cobra"
)

// SupportedPHPVersions lists the PHP versions lerd can build FPM images for.
var SupportedPHPVersions = []string{"8.1", "8.2", "8.3", "8.4", "8.5"}

// NewFetchCmd returns the fetch command.
func NewFetchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch [version...]",
		Short: "Pre-build PHP FPM images so first use isn't slow",
		Long:  "Builds PHP-FPM container images for the given versions (or all supported versions if none specified).\nSkips any version whose image already exists.",
		RunE:  runFetch,
	}
}

func runFetch(_ *cobra.Command, args []string) error {
	versions := args
	if len(versions) == 0 {
		versions = SupportedPHPVersions
	}

	for _, v := range versions {
		fmt.Printf("==> Fetching PHP %s FPM image\n", v)
		if err := podman.BuildFPMImage(v); err != nil {
			fmt.Printf("  [WARN] PHP %s: %v\n", v, err)
		}
	}

	fmt.Println("\nAll requested PHP images ready.")
	return nil
}
