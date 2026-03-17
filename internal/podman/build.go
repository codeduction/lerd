package podman

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/geodro/lerd/internal/config"
)

// ContainerfileHash returns the SHA-256 hash of the embedded PHP-FPM Containerfile.
// This is used to detect when images need to be rebuilt after a lerd update.
func ContainerfileHash() (string, error) {
	tmpl, err := GetQuadletTemplate("lerd-php-fpm.Containerfile")
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(tmpl))
	return fmt.Sprintf("%x", sum), nil
}

// NeedsFPMRebuild returns true if the stored Containerfile hash differs from the
// current embedded Containerfile, meaning images should be rebuilt.
func NeedsFPMRebuild() bool {
	current, err := ContainerfileHash()
	if err != nil {
		return false
	}
	stored, err := os.ReadFile(config.PHPImageHashFile())
	if err != nil {
		// No stored hash yet — treat as needing rebuild only if images exist
		return false
	}
	return strings.TrimSpace(string(stored)) != current
}

// StoreFPMHash writes the current Containerfile hash to disk.
func StoreFPMHash() error {
	hash, err := ContainerfileHash()
	if err != nil {
		return err
	}
	return os.WriteFile(config.PHPImageHashFile(), []byte(hash), 0644)
}

// BuildFPMImage builds the lerd PHP-FPM image for the given version if it doesn't exist.
// Prints build output to stdout so the user can see progress.
func BuildFPMImage(version string) error {
	return buildFPMImage(version, false)
}

// RebuildFPMImage force-removes and rebuilds the PHP-FPM image for the given version.
func RebuildFPMImage(version string) error {
	return buildFPMImage(version, true)
}

func buildFPMImage(version string, force bool) error {
	short := strings.ReplaceAll(version, ".", "")
	imageName := "lerd-php" + short + "-fpm:local"

	if !force {
		// Skip if image already exists
		checkCmd := exec.Command("podman", "image", "exists", imageName)
		if checkCmd.Run() == nil {
			return nil
		}
	} else {
		// Remove existing image so we get a clean rebuild
		rmCmd := exec.Command("podman", "rmi", "-f", imageName)
		_ = rmCmd.Run() // ignore error if image didn't exist
	}

	fmt.Printf("\n  Building PHP %s image (may take a few minutes)...\n", version)

	containerfileTmpl, err := GetQuadletTemplate("lerd-php-fpm.Containerfile")
	if err != nil {
		return err
	}
	containerfile := strings.ReplaceAll(containerfileTmpl, "{{.Version}}", version)

	tmp, err := os.MkdirTemp("", "lerd-php-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	cfPath := tmp + "/Containerfile"
	if err := os.WriteFile(cfPath, []byte(containerfile), 0644); err != nil {
		return err
	}

	cmd := exec.Command("podman", "build", "-t", imageName, "-f", cfPath, tmp)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("building PHP %s image: %w", version, err)
	}

	fmt.Printf("  PHP %s image built successfully.\n", version)
	return nil
}
