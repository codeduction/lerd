package php

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/geodro/lerd/internal/config"
	"gopkg.in/yaml.v3"
)

// DetectVersion detects the PHP version for the given directory.
// It checks, in order:
//  1. .lerd.yaml php_version field (explicit lerd override)
//  2. composer.json require.php semver (project requirement)
//  3. .php-version file (generic tooling hint, lowest priority)
//  4. global config default
func DetectVersion(dir string) (string, error) {
	// 1. .lerd.yaml — explicit lerd override takes top priority
	lerdYaml := filepath.Join(dir, ".lerd.yaml")
	if data, err := os.ReadFile(lerdYaml); err == nil {
		var lerdCfg struct {
			PHPVersion string `yaml:"php_version"`
		}
		if yaml.Unmarshal(data, &lerdCfg) == nil && lerdCfg.PHPVersion != "" {
			return lerdCfg.PHPVersion, nil
		}
	}

	// 2. composer.json require.php — authoritative project requirement
	composerFile := filepath.Join(dir, "composer.json")
	if data, err := os.ReadFile(composerFile); err == nil {
		var composer struct {
			Require map[string]string `json:"require"`
		}
		if json.Unmarshal(data, &composer) == nil {
			if phpConstraint, ok := composer.Require["php"]; ok {
				if v := parseComposerPHP(phpConstraint); v != "" {
					return v, nil
				}
			}
		}
	}

	// 3. .php-version file — generic tooling hint
	phpVersionFile := filepath.Join(dir, ".php-version")
	if data, err := os.ReadFile(phpVersionFile); err == nil {
		v := strings.TrimSpace(string(data))
		if v != "" {
			return v, nil
		}
	}

	// 4. global config default
	cfg, err := config.LoadGlobal()
	if err != nil {
		return "8.4", nil
	}
	return cfg.PHP.DefaultVersion, nil
}

// parseComposerPHP extracts a simple major.minor version from a composer PHP constraint.
// e.g. "^8.2" → "8.2", ">=8.1" → "8.1", "~8.3.0" → "8.3"
func parseComposerPHP(constraint string) string {
	// Strip operators and whitespace
	re := regexp.MustCompile(`(\d+\.\d+)`)
	matches := re.FindStringSubmatch(constraint)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
