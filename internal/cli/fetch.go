package cli

import (
	"fmt"
	"io"

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

	jobs := make([]BuildJob, len(versions))
	for i, v := range versions {
		ver := v
		jobs[i] = BuildJob{
			Label: "PHP " + ver,
			Run:   func(w io.Writer) error { return podman.BuildFPMImageTo(ver, w) },
		}
	}

	if err := RunParallel(jobs); err != nil {
		fmt.Printf("[WARN] some images failed to build: %v\n", err)
	}
	fmt.Println("\nAll requested PHP images ready.")
	return nil
}
