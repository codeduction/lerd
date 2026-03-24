package cli

import (
	"fmt"

	"github.com/geodro/lerd/internal/config"
	gitpkg "github.com/geodro/lerd/internal/git"
	"github.com/spf13/cobra"
)

// NewSitesCmd returns the sites command.
func NewSitesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sites",
		Short: "List all registered sites",
		RunE:  runSites,
	}
}

func runSites(_ *cobra.Command, _ []string) error {
	reg, err := config.LoadSites()
	if err != nil {
		return err
	}

	if len(reg.Sites) == 0 {
		fmt.Println("No sites registered. Use 'lerd park' or 'lerd link' to add sites.")
		return nil
	}

	// Print header
	fmt.Printf("%-25s %-35s %-8s %-8s %-5s %-10s %s\n",
		"Name", "Domain", "PHP", "Node", "TLS", "Framework", "Path")
	fmt.Printf("%-25s %-35s %-8s %-8s %-5s %-10s %s\n",
		"─────────────────────────",
		"───────────────────────────────────",
		"────────",
		"────────",
		"─────",
		"──────────",
		"──────────────────────────────",
	)

	for _, s := range reg.Sites {
		tls := "No"
		if s.Secured {
			tls = "Yes"
		}
		fwName := s.Framework
		if fwName == "" {
			fwName, _ = config.DetectFramework(s.Path)
		}
		fwLabel := ""
		if fw, ok := config.GetFramework(fwName); ok {
			fwLabel = fw.Label
		}
		fmt.Printf("%-25s %-35s %-8s %-8s %-5s %-10s %s\n",
			truncate(s.Name, 25),
			truncate(s.Domain, 35),
			s.PHPVersion,
			s.NodeVersion,
			tls,
			fwLabel,
			s.Path,
		)
		if gitpkg.IsMainRepo(s.Path) {
			worktrees, _ := gitpkg.DetectWorktrees(s.Path, s.Domain)
			for _, wt := range worktrees {
				fmt.Printf("  %-23s %-35s %-8s %-8s %-5s %-10s %s\n",
					"↳ "+truncate(wt.Branch, 21),
					truncate(wt.Domain, 35),
					s.PHPVersion,
					s.NodeVersion,
					"—",
					"",
					wt.Path,
				)
			}
		}
	}

	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
