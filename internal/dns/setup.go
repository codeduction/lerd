package dns

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/geodro/lerd/internal/config"
)

const dnsmasqConf = `# Lerd DNS configuration
address=/.test/127.0.0.1
port=53
`

const nmDnsConf = `[main]
dns=dnsmasq
`

const nmDnsmasqConf = `server=/test/127.0.0.1#5300
`

// Setup writes NetworkManager dnsmasq configuration and restarts NetworkManager.
func Setup() error {
	if err := WriteDnsmasqConfig(config.DnsmasqDir()); err != nil {
		return fmt.Errorf("writing lerd dnsmasq config: %w", err)
	}

	fmt.Println("  [sudo required] Configuring NetworkManager for .test DNS resolution")

	// Write /etc/NetworkManager/conf.d/lerd.conf
	nmConfDir := "/etc/NetworkManager/conf.d"
	nmConfFile := filepath.Join(nmConfDir, "lerd.conf")
	if err := sudoWriteFile(nmConfFile, []byte(nmDnsConf)); err != nil {
		return fmt.Errorf("writing NetworkManager conf: %w", err)
	}

	// Write /etc/NetworkManager/dnsmasq.d/lerd.conf
	nmDnsmasqDir := "/etc/NetworkManager/dnsmasq.d"
	nmDnsmasqFile := filepath.Join(nmDnsmasqDir, "lerd.conf")
	if err := sudoWriteFile(nmDnsmasqFile, []byte(nmDnsmasqConf)); err != nil {
		return fmt.Errorf("writing NetworkManager dnsmasq conf: %w", err)
	}

	// Restart NetworkManager
	cmd := exec.Command("sudo", "systemctl", "restart", "NetworkManager")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restarting NetworkManager: %w\n%s", err, out)
	}
	return nil
}

// WriteDnsmasqConfig writes the lerd dnsmasq config to the given directory.
func WriteDnsmasqConfig(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "lerd.conf"), []byte(dnsmasqConf), 0644)
}

// sudoWriteFile writes content to a system path using sudo tee.
func sudoWriteFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	mkdirCmd := exec.Command("sudo", "mkdir", "-p", dir)
	if out, err := mkdirCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mkdir %s: %w\n%s", dir, err, out)
	}

	teeCmd := exec.Command("sudo", "tee", path)
	teeCmd.Stdin = bytes.NewReader(content)
	out, err := teeCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tee %s: %w\n%s", path, err, out)
	}
	return nil
}
