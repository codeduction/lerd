package git

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Worktree represents a git worktree checkout for a registered site.
type Worktree struct {
	Name   string // subdirectory name under .git/worktrees/
	Branch string // sanitized branch (subdomain-safe)
	Path   string // absolute path to checkout dir
	Domain string // "<sanitized-branch>.<siteDomain>"
}

// IsMainRepo returns true if sitePath/.git is a directory (not a file).
// A file means the repo itself is a worktree, not the main checkout.
func IsMainRepo(sitePath string) bool {
	info, err := os.Stat(filepath.Join(sitePath, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DetectWorktrees returns the list of active worktrees for the given site.
func DetectWorktrees(sitePath, siteDomain string) ([]Worktree, error) {
	if !IsMainRepo(sitePath) {
		return nil, nil
	}

	worktreesDir := filepath.Join(sitePath, ".git", "worktrees")
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []Worktree
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		wtDir := filepath.Join(worktreesDir, name)

		branch := readBranch(wtDir)
		path := readCheckoutPath(wtDir)
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			continue // checkout dir gone
		}

		sanitized := SanitizeBranch(branch)
		result = append(result, Worktree{
			Name:   name,
			Branch: sanitized,
			Path:   path,
			Domain: sanitized + "." + siteDomain,
		})
	}
	return result, nil
}

// readBranch reads the branch name from .git/worktrees/<name>/HEAD.
func readBranch(wtDir string) string {
	data, err := os.ReadFile(filepath.Join(wtDir, "HEAD"))
	if err != nil {
		return "detached"
	}
	line := strings.TrimSpace(string(data))
	const prefix = "ref: refs/heads/"
	if strings.HasPrefix(line, prefix) {
		return strings.TrimPrefix(line, prefix)
	}
	// detached HEAD — use first 7 chars of SHA
	if len(line) >= 7 {
		return "detached-" + line[:7]
	}
	return "detached"
}

// readCheckoutPath reads the checkout directory path from .git/worktrees/<name>/gitdir.
func readCheckoutPath(wtDir string) string {
	data, err := os.ReadFile(filepath.Join(wtDir, "gitdir"))
	if err != nil {
		return ""
	}
	// gitdir contains the path to the .git file inside the checkout, e.g. /path/to/checkout/.git
	gitFile := strings.TrimSpace(string(data))
	return filepath.Dir(gitFile)
}

// EnsureWorktreeDeps sets up a worktree checkout with the dependencies it needs:
//   - vendor/ and node_modules/ are symlinked from the main repo
//   - .env is copied from the main repo with APP_URL rewritten to http(s)://<worktreeDomain>
func EnsureWorktreeDeps(mainRepoPath, worktreePath, worktreeDomain string, secured bool) {
	for _, dir := range []string{"vendor", "node_modules"} {
		dst := filepath.Join(worktreePath, dir)
		if _, err := os.Lstat(dst); err == nil {
			continue // already exists or is already a symlink
		}
		src := filepath.Join(mainRepoPath, dir)
		if _, err := os.Stat(src); err != nil {
			continue // main repo doesn't have it either
		}
		_ = os.Symlink(src, dst)
	}

	// .env: copy from main repo and set APP_URL to the worktree domain.
	worktreeEnv := filepath.Join(worktreePath, ".env")
	if _, err := os.Lstat(worktreeEnv); err == nil {
		return // already exists
	}
	mainEnv := filepath.Join(mainRepoPath, ".env")
	if err := copyFile(mainEnv, worktreeEnv); err != nil {
		return
	}
	scheme := "http"
	if secured {
		scheme = "https"
	}
	_ = rewriteAppURL(worktreeEnv, scheme+"://"+worktreeDomain)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// rewriteAppURL replaces APP_URL in the given .env file.
func rewriteAppURL(envPath, appURL string) error {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "APP_URL=") || strings.HasPrefix(line, "APP_URL =") {
			lines[i] = "APP_URL=" + appURL
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, "APP_URL="+appURL)
	}
	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0644)
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9-]`)
var multiHyphen = regexp.MustCompile(`-{2,}`)

// SanitizeBranch converts a branch name to a subdomain-safe slug.
func SanitizeBranch(branch string) string {
	s := strings.ToLower(branch)
	// Replace common separators with hyphens
	s = strings.NewReplacer("/", "-", "_", "-", ".", "-").Replace(s)
	// Strip anything not alphanumeric or hyphen
	s = nonSlugChars.ReplaceAllString(s, "")
	// Collapse consecutive hyphens
	s = multiHyphen.ReplaceAllString(s, "-")
	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")
	// Truncate to 50 chars
	if len(s) > 50 {
		s = strings.TrimRight(s[:50], "-")
	}
	if s == "" {
		return "branch"
	}
	return s
}
