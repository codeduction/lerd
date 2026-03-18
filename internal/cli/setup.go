package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

// setupStep describes one bootstrap action.
type setupStep struct {
	label   string
	enabled bool // default selection
	run     func() error
}

// NewSetupCmd returns the setup command.
func NewSetupCmd() *cobra.Command {
	var allSteps bool
	var skipOpen bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Bootstrap a Laravel project (composer, npm, env, migrate, build, open)",
		Long: `Runs a series of standard project setup steps with an interactive
step-selector so you can toggle which steps to execute before they run.

Steps (smart defaults based on project state):
  1. composer install        — skipped if vendor/ already exists
  2. npm ci                  — skipped if node_modules/ already exists
  3. lerd env                — configure .env with lerd service settings
  4. lerd mcp:inject         — inject MCP config (off by default)
  5. php artisan migrate     — run database migrations
  6. php artisan db:seed     — seed the database (off by default)
  7. npm run build           — build front-end assets (if package.json exists)
  8. lerd secure             — enable HTTPS via mkcert (off by default)
  9. lerd open               — open site in browser

Site registration (lerd link) always runs first, before the step selector.

Use --all to skip the selector and run everything (useful in CI).`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSetup(allSteps, skipOpen)
		},
	}

	cmd.Flags().BoolVarP(&allSteps, "all", "a", false, "Select all steps without prompting (for CI/automation)")
	cmd.Flags().BoolVar(&skipOpen, "skip-open", false, "Do not open the site in the browser at the end")
	return cmd
}

func runSetup(allSteps, skipOpen bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Always refresh site registration first so PHP/Node versions are correct
	// before any subsequent step (especially lerd secure) reads from the registry.
	fmt.Println("→ Registering site...")
	if err := runLink(nil, ""); err != nil {
		fmt.Printf("  [WARN] lerd link: %v\n", err)
	}

	_, vendorMissing := os.Stat(cwd + "/vendor")
	_, nodeModulesMissing := os.Stat(cwd + "/node_modules")
	_, pkgJSONErr := os.Stat(cwd + "/package.json")
	hasPackageJSON := pkgJSONErr == nil

	steps := []setupStep{
		{
			label:   "composer install",
			enabled: os.IsNotExist(vendorMissing),
			run: func() error {
				cmd := exec.Command("composer", "install")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				return cmd.Run()
			},
		},
		{
			label:   "npm ci",
			enabled: os.IsNotExist(nodeModulesMissing) && hasPackageJSON,
			run: func() error {
				return runWithFnm("npm", []string{"ci"})
			},
		},
		{
			label:   "lerd env",
			enabled: true,
			run: func() error {
				return runEnv(nil, nil)
			},
		},
		{
			label:   "lerd mcp:inject",
			enabled: false,
			run: func() error {
				return runMCPInject("")
			},
		},
		{
			label:   "php artisan migrate",
			enabled: true,
			run: func() error {
				return artisanIn(cwd, "migrate")
			},
		},
		{
			label:   "php artisan db:seed",
			enabled: false,
			run: func() error {
				return artisanIn(cwd, "db:seed")
			},
		},
		{
			label:   "npm run build",
			enabled: hasPackageJSON,
			run: func() error {
				return runWithFnm("npm", []string{"run", "build"})
			},
		},
	}

	steps = append(steps, setupStep{
		label:   "lerd secure",
		enabled: false,
		run: func() error {
			return runSecure(nil, nil)
		},
	})

	if !skipOpen {
		steps = append(steps, setupStep{
			label:   "lerd open",
			enabled: true,
			run: func() error {
				return runOpen(nil, nil)
			},
		})
	}

	// Determine which steps to run.
	var selected []string
	if allSteps {
		for _, s := range steps {
			selected = append(selected, s.label)
		}
	} else {
		options := make([]string, len(steps))
		defaults := []string{}
		for i, s := range steps {
			options[i] = s.label
			if s.enabled {
				defaults = append(defaults, s.label)
			}
		}

		prompt := &survey.MultiSelect{
			Message: "Select setup steps to run:",
			Options: options,
			Default: defaults,
		}
		if err := survey.AskOne(prompt, &selected); err != nil {
			return err
		}
	}

	if len(selected) == 0 {
		fmt.Println("No steps selected. Nothing to do.")
		return nil
	}

	// Build a set for O(1) lookup.
	selectedSet := make(map[string]bool, len(selected))
	for _, s := range selected {
		selectedSet[s] = true
	}

	// Execute steps in order.
	for _, s := range steps {
		if !selectedSet[s.label] {
			continue
		}
		fmt.Printf("\n→ Running: %s\n", s.label)
		if err := s.run(); err != nil {
			fmt.Printf("✗ %s failed: %v\n", s.label, err)
			if !promptContinue() {
				return fmt.Errorf("setup aborted after %q failed", s.label)
			}
		}
	}

	fmt.Println("\nSetup complete.")
	return nil
}

// promptContinue asks the user whether to continue after a step failure.
// Returns true if the user wants to continue.
func promptContinue() bool {
	fmt.Print("  Continue with remaining steps? [y/N]: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}
