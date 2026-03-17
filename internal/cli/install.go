package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/geodro/lerd/internal/certs"
	"github.com/geodro/lerd/internal/config"
	"github.com/geodro/lerd/internal/dns"
	"github.com/geodro/lerd/internal/nginx"
	"github.com/geodro/lerd/internal/podman"
	lerdSystemd "github.com/geodro/lerd/internal/systemd"
	"github.com/spf13/cobra"
)

// NewInstallCmd returns the install command.
func NewInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Run one-time Lerd setup",
		RunE:  runInstall,
	}
}

func runInstall(_ *cobra.Command, _ []string) error {
	fmt.Println("==> Installing Lerd")

	// 1. Create XDG directories
	step("Creating directories")
	dirs := []string{
		config.ConfigDir(),
		config.DataDir(),
		config.BinDir(),
		config.NginxDir(),
		config.NginxConfD(),
		config.CertsDir(),
		filepath.Join(config.CertsDir(), "sites"),
		config.DnsmasqDir(),
		config.QuadletDir(),
		config.SystemdUserDir(),
		config.DataSubDir("mysql"),
		config.DataSubDir("redis"),
		config.DataSubDir("postgres"),
		config.DataSubDir("meilisearch"),
		config.DataSubDir("minio"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating %s: %w", d, err)
		}
	}
	ok()

	// 2. Podman network
	step("Creating lerd podman network")
	if err := podman.EnsureNetwork("lerd"); err != nil {
		return fmt.Errorf("podman network: %w", err)
	}
	ok()

	// 3. Download binaries
	if err := downloadBinaries(); err != nil {
		return err
	}

	// 4. mkcert -install
	step("Installing mkcert CA")
	if err := certs.InstallCA(); err != nil {
		return fmt.Errorf("mkcert CA: %w", err)
	}
	ok()

	// 5. DNS config
	step("Writing DNS configuration")
	if err := dns.Setup(); err != nil {
		return fmt.Errorf("dns config: %w", err)
	}
	ok()

	// 6. Nginx config and quadlet
	step("Writing nginx configuration")
	if err := nginx.EnsureNginxConfig(); err != nil {
		return fmt.Errorf("nginx config: %w", err)
	}
	ok()

	step("Writing nginx quadlet")
	nginxQuadlet, err := podman.GetQuadletTemplate("lerd-nginx.container")
	if err != nil {
		return err
	}
	if err := podman.WriteQuadlet("lerd-nginx", nginxQuadlet); err != nil {
		return fmt.Errorf("nginx quadlet: %w", err)
	}
	ok()

	// Write DNS quadlet
	step("Writing DNS quadlet")
	dnsQuadlet, err := podman.GetQuadletTemplate("lerd-dns.container")
	if err != nil {
		return err
	}
	if err := podman.WriteQuadlet("lerd-dns", dnsQuadlet); err != nil {
		return fmt.Errorf("dns quadlet: %w", err)
	}
	ok()

	// 7. daemon-reload and start services
	step("Reloading systemd daemon")
	if err := podman.DaemonReload(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	ok()

	step("Starting lerd-dns")
	if err := podman.StartUnit("lerd-dns"); err != nil {
		fmt.Printf(" [WARN: %v]\n", err)
	} else {
		ok()
	}

	step("Starting lerd-nginx")
	if err := podman.StartUnit("lerd-nginx"); err != nil {
		fmt.Printf(" [WARN: %v]\n", err)
	} else {
		ok()
	}

	// 8. Watcher service
	step("Writing watcher service")
	watcherContent, err := lerdSystemd.GetUnit("lerd-watcher")
	if err != nil {
		return err
	}
	if err := lerdSystemd.WriteService("lerd-watcher", watcherContent); err != nil {
		return fmt.Errorf("watcher service: %w", err)
	}
	if err := lerdSystemd.EnableService("lerd-watcher"); err != nil {
		fmt.Printf(" [WARN: %v]\n", err)
	} else {
		ok()
	}

	// 9. Shell shims
	step("Adding shell PATH configuration")
	if err := addShellShims(); err != nil {
		fmt.Printf(" [WARN: %v]\n", err)
	} else {
		ok()
	}

	fmt.Println("\nLerd installation complete!")
	return nil
}

func step(msg string) {
	fmt.Printf("  --> %s ... ", msg)
}

func ok() {
	fmt.Println("OK")
}

func downloadBinaries() error {
	arch := runtime.GOARCH
	binDir := config.BinDir()

	// composer
	step("Downloading composer")
	composerPath := filepath.Join(binDir, "composer")
	if _, err := os.Stat(composerPath); os.IsNotExist(err) {
		if err := downloadFile("https://getcomposer.org/composer-stable.phar", composerPath, 0755); err != nil {
			return fmt.Errorf("composer download: %w", err)
		}
	}
	ok()

	// fnm
	step("Downloading fnm")
	fnmPath := filepath.Join(binDir, "fnm")
	if _, err := os.Stat(fnmPath); os.IsNotExist(err) {
		fnmZip := filepath.Join(binDir, "fnm-linux.zip")
		if err := downloadFile(
			"https://github.com/Schniz/fnm/releases/latest/download/fnm-linux.zip",
			fnmZip, 0644,
		); err != nil {
			return fmt.Errorf("fnm download: %w", err)
		}
		extractCmd := exec.Command("unzip", "-o", fnmZip, "fnm", "-d", binDir)
		if out, err := extractCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("fnm extract: %w\n%s", err, out)
		}
		os.Remove(fnmZip)
		os.Chmod(fnmPath, 0755) //nolint:errcheck
	}
	ok()

	// mkcert
	step("Downloading mkcert")
	mkcertPath := certs.MkcertPath()
	if _, err := os.Stat(mkcertPath); os.IsNotExist(err) {
		mkcertArch := "amd64"
		if arch == "arm64" {
			mkcertArch = "arm64"
		}
		mkcertURL := fmt.Sprintf(
			"https://github.com/FiloSottile/mkcert/releases/latest/download/mkcert-v1.4.4-linux-%s",
			mkcertArch,
		)
		if err := downloadFile(mkcertURL, mkcertPath, 0755); err != nil {
			return fmt.Errorf("mkcert download: %w", err)
		}
	}
	ok()

	return nil
}

// downloadFile downloads a URL to a local file with progress.
func downloadFile(url, dest string, mode os.FileMode) error {
	fmt.Printf("\n      Downloading %s\n      ", url)

	resp, err := http.Get(url) //nolint:gosec,noctx
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()

	written, err := io.Copy(f, &progressReader{r: resp.Body, total: resp.ContentLength})
	if err != nil {
		return err
	}
	fmt.Printf(" (%d bytes)\n      ", written)

	return os.Chmod(dest, mode)
}

type progressReader struct {
	r       io.Reader
	total   int64
	written int64
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.written += int64(n)
	if p.total > 0 {
		pct := int(float64(p.written) / float64(p.total) * 50)
		bar := ""
		for i := 0; i < 50; i++ {
			if i < pct {
				bar += "="
			} else {
				bar += " "
			}
		}
		fmt.Printf("\r      [%s] %d%%", bar, pct*2)
	}
	return n, err
}

func addShellShims() error {
	home, _ := os.UserHomeDir()
	binDir := config.BinDir()
	shell := os.Getenv("SHELL")

	switch {
	case isShell(shell, "fish"):
		fishConfigDir := filepath.Join(home, ".config", "fish", "conf.d")
		if err := os.MkdirAll(fishConfigDir, 0755); err != nil {
			return err
		}
		fishConf := filepath.Join(fishConfigDir, "lerd.fish")
		content := fmt.Sprintf("set -gx PATH %s $PATH\n", binDir)
		return os.WriteFile(fishConf, []byte(content), 0644)
	case isShell(shell, "zsh"):
		return appendShellRC(filepath.Join(home, ".zshrc"), binDir)
	default:
		return appendShellRC(filepath.Join(home, ".bashrc"), binDir)
	}
}

func appendShellRC(rcFile, binDir string) error {
	line := fmt.Sprintf("\n# Lerd\nexport PATH=\"%s:$PATH\"\n", binDir)
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

func isShell(shell, name string) bool {
	return len(shell) > 0 && filepath.Base(shell) == name
}
