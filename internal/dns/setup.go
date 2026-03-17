package dns

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/geodro/lerd/internal/config"
)

const nmDnsConf = `[main]
dns=dnsmasq
`

const nmDnsmasqConf = `server=/test/127.0.0.1#5300
`

const resolvedDropin = `[Resolve]
DNS=127.0.0.1:5300
Domains=~test
`

// nmDispatcherScript is installed at /etc/NetworkManager/dispatcher.d/99-lerd-dns.
// On Ubuntu, NetworkManager manages systemd-resolved via DBUS and overrides global
// resolved.conf drop-ins. Per-interface DNS set via resolvectl is respected.
const nmDispatcherScript = `#!/bin/sh
# Lerd DNS: forward .test queries to local dnsmasq on port 5300
IFACE="$1"
ACTION="$2"
if [ "$ACTION" = "up" ] || [ "$ACTION" = "dhcp4-change" ] || [ "$ACTION" = "dhcp6-change" ]; then
    resolvectl dns "$IFACE" 127.0.0.1:5300 2>/dev/null || true
    resolvectl domain "$IFACE" ~test 2>/dev/null || true
fi
`

// isFileContent returns true if the file at path already contains exactly content.
func isFileContent(path string, content []byte) bool {
	existing, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return string(existing) == string(content)
}

// isSystemdResolvedActive returns true if systemd-resolved is the active DNS resolver.
func isSystemdResolvedActive() bool {
	cmd := exec.Command("systemctl", "is-active", "--quiet", "systemd-resolved")
	if err := cmd.Run(); err != nil {
		return false
	}
	// Also check that /etc/resolv.conf points to the stub resolver
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "127.0.0.53") || strings.Contains(string(data), "systemd-resolved")
}

// isNetworkManagerActive returns true if NetworkManager is running.
func isNetworkManagerActive() bool {
	cmd := exec.Command("systemctl", "is-active", "--quiet", "NetworkManager")
	return cmd.Run() == nil
}

// defaultInterface returns the name of the default network interface (e.g. "enp1s0").
func defaultInterface() string {
	out, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return ""
	}
	// "default via 192.168.1.1 dev enp1s0 ..."
	parts := strings.Fields(string(out))
	for i, p := range parts {
		if p == "dev" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// readUpstreamDNS returns upstream DNS server IPs from the running system.
// On systemd-resolved systems it reads /run/systemd/resolve/resolv.conf (the real
// upstream list, not the stub 127.0.0.53). Falls back to /etc/resolv.conf.
func readUpstreamDNS() []string {
	for _, path := range []string{"/run/systemd/resolve/resolv.conf", "/etc/resolv.conf"} {
		servers := parseNameservers(path)
		if len(servers) > 0 {
			return servers
		}
	}
	return nil
}

func parseNameservers(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var servers []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "nameserver ") {
			continue
		}
		ip := strings.TrimSpace(strings.TrimPrefix(line, "nameserver "))
		// Skip loopback / stub resolver addresses
		if ip == "" || ip == "127.0.0.1" || ip == "127.0.0.53" || ip == "::1" {
			continue
		}
		servers = append(servers, ip)
	}
	return servers
}

// Setup writes DNS configuration for .test resolution and restarts the resolver.
// On systemd-resolved + NetworkManager systems (Ubuntu etc.) it uses an NM dispatcher script.
// On pure systemd-resolved systems it uses a resolved drop-in.
// On NetworkManager-only systems it uses NM's embedded dnsmasq.
func Setup() error {
	if err := WriteDnsmasqConfig(config.DnsmasqDir()); err != nil {
		return fmt.Errorf("writing lerd dnsmasq config: %w", err)
	}

	if isSystemdResolvedActive() {
		if isNetworkManagerActive() {
			return setupNMWithResolved()
		}
		return setupSystemdResolved()
	}
	return setupNetworkManager()
}

// setupNMWithResolved handles Ubuntu-style: NM manages systemd-resolved via DBUS.
// NM overrides global DNS set in resolved.conf drop-ins, so we use an NM dispatcher
// script that applies per-interface DNS via resolvectl on each "up" event, then
// applies it immediately to the current default interface.
func setupNMWithResolved() error {
	dispatcherScript := "/etc/NetworkManager/dispatcher.d/99-lerd-dns"

	if !isFileContent(dispatcherScript, []byte(nmDispatcherScript)) {
		fmt.Println("  [sudo required] Configuring NetworkManager dispatcher for .test DNS resolution")

		if err := sudoWriteFile(dispatcherScript, []byte(nmDispatcherScript)); err != nil {
			return fmt.Errorf("writing NM dispatcher script: %w", err)
		}

		chmodCmd := exec.Command("sudo", "chmod", "755", dispatcherScript)
		chmodCmd.Stdin = os.Stdin
		chmodCmd.Stdout = os.Stdout
		chmodCmd.Stderr = os.Stderr
		if err := chmodCmd.Run(); err != nil {
			return fmt.Errorf("chmod dispatcher script: %w", err)
		}
	}

	// Remove stale resolved drop-in if present (it doesn't work with NM)
	dropin := "/etc/systemd/resolved.conf.d/lerd.conf"
	if _, err := os.Stat(dropin); err == nil {
		rmCmd := exec.Command("sudo", "rm", "-f", dropin)
		rmCmd.Stdin = os.Stdin
		rmCmd.Stdout = os.Stdout
		rmCmd.Stderr = os.Stderr
		rmCmd.Run() //nolint:errcheck
	}

	// Apply immediately to the current default interface
	iface := defaultInterface()
	if iface == "" {
		return nil
	}

	dnsCmd := exec.Command("sudo", "resolvectl", "dns", iface, "127.0.0.1:5300")
	dnsCmd.Stdin = os.Stdin
	dnsCmd.Stdout = os.Stdout
	dnsCmd.Stderr = os.Stderr
	if err := dnsCmd.Run(); err != nil {
		return fmt.Errorf("applying DNS to %s: %w", iface, err)
	}

	domainCmd := exec.Command("sudo", "resolvectl", "domain", iface, "~test")
	domainCmd.Stdin = os.Stdin
	domainCmd.Stdout = os.Stdout
	domainCmd.Stderr = os.Stderr
	if err := domainCmd.Run(); err != nil {
		return fmt.Errorf("applying domain routing to %s: %w", iface, err)
	}

	return nil
}

// setupSystemdResolved configures systemd-resolved to forward .test to port 5300.
// Used only when systemd-resolved is active without NetworkManager managing it.
func setupSystemdResolved() error {
	dropin := "/etc/systemd/resolved.conf.d/lerd.conf"

	if isFileContent(dropin, []byte(resolvedDropin)) {
		return nil
	}

	fmt.Println("  [sudo required] Configuring systemd-resolved for .test DNS resolution")

	if err := sudoWriteFile(dropin, []byte(resolvedDropin)); err != nil {
		return fmt.Errorf("writing resolved drop-in: %w", err)
	}

	cmd := exec.Command("sudo", "systemctl", "restart", "systemd-resolved")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restarting systemd-resolved: %w", err)
	}
	return nil
}

// setupNetworkManager configures NetworkManager's embedded dnsmasq.
func setupNetworkManager() error {
	nmConfFile := "/etc/NetworkManager/conf.d/lerd.conf"
	nmDnsmasqFile := "/etc/NetworkManager/dnsmasq.d/lerd.conf"

	if isFileContent(nmConfFile, []byte(nmDnsConf)) && isFileContent(nmDnsmasqFile, []byte(nmDnsmasqConf)) {
		return nil
	}

	fmt.Println("  [sudo required] Configuring NetworkManager for .test DNS resolution")

	if err := sudoWriteFile(nmConfFile, []byte(nmDnsConf)); err != nil {
		return fmt.Errorf("writing NetworkManager conf: %w", err)
	}

	if err := sudoWriteFile(nmDnsmasqFile, []byte(nmDnsmasqConf)); err != nil {
		return fmt.Errorf("writing NetworkManager dnsmasq conf: %w", err)
	}

	cmd := exec.Command("sudo", "systemctl", "restart", "NetworkManager")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restarting NetworkManager: %w", err)
	}
	return nil
}

// WriteDnsmasqConfig writes the lerd dnsmasq config to the given directory.
// Upstream DNS servers are detected from the running system so they are never hardcoded.
func WriteDnsmasqConfig(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("# Lerd DNS configuration\n")
	sb.WriteString("port=5300\n")
	sb.WriteString("no-resolv\n")
	sb.WriteString("address=/.test/127.0.0.1\n")
	for _, ip := range readUpstreamDNS() {
		fmt.Fprintf(&sb, "server=%s\n", ip)
	}

	return os.WriteFile(filepath.Join(dir, "lerd.conf"), []byte(sb.String()), 0644)
}

// sudoWriteFile writes content to a system path by writing to a temp file
// then using sudo cp, so sudo can prompt for a password on the terminal.
func sudoWriteFile(path string, content []byte) error {
	tmp, err := os.CreateTemp("", "lerd-sudo-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	dir := filepath.Dir(path)
	mkdirCmd := exec.Command("sudo", "mkdir", "-p", dir)
	mkdirCmd.Stdin = os.Stdin
	mkdirCmd.Stdout = os.Stdout
	mkdirCmd.Stderr = os.Stderr
	if err := mkdirCmd.Run(); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	cpCmd := exec.Command("sudo", "cp", tmp.Name(), path)
	cpCmd.Stdin = os.Stdin
	cpCmd.Stdout = os.Stdout
	cpCmd.Stderr = os.Stderr
	if err := cpCmd.Run(); err != nil {
		return fmt.Errorf("cp to %s: %w", path, err)
	}
	return nil
}
