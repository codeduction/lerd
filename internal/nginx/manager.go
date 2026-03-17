package nginx

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/geodro/lerd/internal/config"
	"github.com/geodro/lerd/internal/podman"
)

type nginxConfData struct {
	Resolver string
}

// VhostData is the data passed to vhost templates.
type VhostData struct {
	Domain         string
	Path           string
	PHPVersion     string
	PHPVersionShort string
}

// phpShort converts "8.4" → "84".
func phpShort(version string) string {
	return strings.ReplaceAll(version, ".", "")
}

// GenerateVhost renders the HTTP vhost template and writes it to conf.d.
func GenerateVhost(site config.Site, phpVersion string) error {
	tmplData, err := GetTemplate("vhost.conf.tmpl")
	if err != nil {
		return err
	}

	tmpl, err := template.New("vhost").Parse(string(tmplData))
	if err != nil {
		return err
	}

	data := VhostData{
		Domain:          site.Domain,
		Path:            site.Path,
		PHPVersion:      phpVersion,
		PHPVersionShort: phpShort(phpVersion),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	if err := os.MkdirAll(config.NginxConfD(), 0755); err != nil {
		return err
	}
	confPath := filepath.Join(config.NginxConfD(), site.Domain+".conf")
	return os.WriteFile(confPath, buf.Bytes(), 0644)
}

// GenerateSSLVhost renders the SSL vhost template and writes it to conf.d.
func GenerateSSLVhost(site config.Site, phpVersion string) error {
	tmplData, err := GetTemplate("vhost-ssl.conf.tmpl")
	if err != nil {
		return err
	}

	tmpl, err := template.New("vhost-ssl").Parse(string(tmplData))
	if err != nil {
		return err
	}

	data := VhostData{
		Domain:          site.Domain,
		Path:            site.Path,
		PHPVersion:      phpVersion,
		PHPVersionShort: phpShort(phpVersion),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	if err := os.MkdirAll(config.NginxConfD(), 0755); err != nil {
		return err
	}
	confPath := filepath.Join(config.NginxConfD(), site.Domain+"-ssl.conf")
	return os.WriteFile(confPath, buf.Bytes(), 0644)
}

// RemoveVhost deletes the vhost config files for the given domain.
func RemoveVhost(domain string) error {
	confD := config.NginxConfD()
	for _, suffix := range []string{".conf", "-ssl.conf"} {
		path := filepath.Join(confD, domain+suffix)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// proxyVhostData is the template data for vhost-proxy.conf.tmpl.
type proxyVhostData struct {
	Domain       string
	UpstreamHost string
	UpstreamPort int
	Resolver     string
}

// GenerateProxyVhost renders vhost-proxy.conf.tmpl and writes conf.d/{domain}.conf.
func GenerateProxyVhost(domain, upstreamHost string, upstreamPort int) error {
	tmplData, err := GetTemplate("vhost-proxy.conf.tmpl")
	if err != nil {
		return err
	}

	tmpl, err := template.New("vhost-proxy").Parse(string(tmplData))
	if err != nil {
		return err
	}

	resolver := podman.NetworkGateway("lerd")
	data := proxyVhostData{
		Domain:       domain,
		UpstreamHost: upstreamHost,
		UpstreamPort: upstreamPort,
		Resolver:     resolver,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	if err := os.MkdirAll(config.NginxConfD(), 0755); err != nil {
		return err
	}
	confPath := filepath.Join(config.NginxConfD(), domain+".conf")
	return os.WriteFile(confPath, buf.Bytes(), 0644)
}

// Reload signals nginx to reload its configuration.
func Reload() error {
	_, err := podman.Run("exec", "lerd-nginx", "nginx", "-s", "reload")
	return err
}

// EnsureNginxConfig copies the base nginx.conf to the data dir if it is missing.
func EnsureNginxConfig() error {
	nginxDir := config.NginxDir()
	if err := os.MkdirAll(nginxDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(config.NginxConfD(), 0755); err != nil {
		return err
	}

	destPath := filepath.Join(nginxDir, "nginx.conf")
	tmplData, err := GetTemplate("nginx.conf")
	if err != nil {
		return fmt.Errorf("failed to read embedded nginx.conf: %w", err)
	}
	tmpl, err := template.New("nginx.conf").Parse(string(tmplData))
	if err != nil {
		return fmt.Errorf("parsing nginx.conf template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nginxConfData{
		Resolver: podman.NetworkGateway("lerd"),
	}); err != nil {
		return fmt.Errorf("rendering nginx.conf: %w", err)
	}
	return os.WriteFile(destPath, buf.Bytes(), 0644)
}
