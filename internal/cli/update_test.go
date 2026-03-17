package cli

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// ── stripV ───────────────────────────────────────────────────────────────────

func TestStripV(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"v1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
		{"v0.1.0", "0.1.0"},
		{"", ""},
		{"v", ""},
	}
	for _, c := range cases {
		got := stripV(c.in)
		if got != c.want {
			t.Errorf("stripV(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ── fetchLatestVersion ───────────────────────────────────────────────────────

func TestFetchLatestVersion_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.2.3"})
	}))
	defer srv.Close()

	orig := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = orig }()

	got, err := fetchLatestVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "v1.2.3" {
		t.Errorf("got %q, want v1.2.3", got)
	}
}

func TestFetchLatestVersion_withoutVPrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "0.9.0"})
	}))
	defer srv.Close()

	orig := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = orig }()

	got, err := fetchLatestVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "0.9.0" {
		t.Errorf("got %q, want 0.9.0", got)
	}
}

func TestFetchLatestVersion_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	orig := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = orig }()

	_, err := fetchLatestVersion()
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
}

func TestFetchLatestVersion_emptyTagName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"tag_name": ""})
	}))
	defer srv.Close()

	orig := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = orig }()

	_, err := fetchLatestVersion()
	if err == nil {
		t.Fatal("expected error for empty tag_name, got nil")
	}
}

func TestFetchLatestVersion_serverError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	orig := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = orig }()

	_, err := fetchLatestVersion()
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

// ── downloadReleaseBinary ────────────────────────────────────────────────────

// makeFakeTarGz creates a .tar.gz archive in dir containing a file named "lerd"
// with the given content.
func makeFakeTarGz(t *testing.T, dir, content string) string {
	t.Helper()
	archivePath := filepath.Join(dir, "lerd_0.1.0_linux_amd64.tar.gz")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	data := []byte(content)
	tw.WriteHeader(&tar.Header{Name: "lerd", Mode: 0755, Size: int64(len(data))})
	tw.Write(data)
	return archivePath
}

func TestDownloadReleaseBinary_success(t *testing.T) {
	// Build a fake tar.gz to serve
	tmp := t.TempDir()
	makeFakeTarGz(t, tmp, "#!/bin/sh\necho lerd")

	archiveBytes, err := os.ReadFile(filepath.Join(tmp, "lerd_0.1.0_linux_amd64.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(archiveBytes)
	}))
	defer srv.Close()

	orig := githubDownloadBase
	githubDownloadBase = srv.URL
	defer func() { githubDownloadBase = orig }()

	binary, cleanup, err := downloadReleaseBinary("v0.1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if _, err := os.Stat(binary); err != nil {
		t.Errorf("binary not found at %s: %v", binary, err)
	}
}

func TestDownloadReleaseBinary_serverError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	orig := githubDownloadBase
	githubDownloadBase = srv.URL
	defer func() { githubDownloadBase = orig }()

	_, cleanup, err := downloadReleaseBinary("v0.1.0")
	cleanup()
	if err == nil {
		t.Fatal("expected error for 404 download, got nil")
	}
}

// ── copyFile ─────────────────────────────────────────────────────────────────

func TestCopyFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")

	if err := os.WriteFile(src, []byte("hello lerd"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst, 0755); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello lerd" {
		t.Errorf("got %q, want %q", got, "hello lerd")
	}

	info, _ := os.Stat(dst)
	if info.Mode() != 0755 {
		t.Errorf("mode = %v, want 0755", info.Mode())
	}
}

func TestCopyFile_missingSource(t *testing.T) {
	tmp := t.TempDir()
	err := copyFile(filepath.Join(tmp, "nope"), filepath.Join(tmp, "dst"), 0644)
	if err == nil {
		t.Fatal("expected error for missing source, got nil")
	}
}

// ── runUpdate (integration-style) ────────────────────────────────────────────

func TestRunUpdate_alreadyLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.0.0"})
	}))
	defer srv.Close()

	orig := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = orig }()

	// Should return nil without downloading anything
	err := runUpdate("1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
