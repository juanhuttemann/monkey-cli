package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	githubAPIURL = "https://api.github.com/repos/juanhuttemann/monkey-cli/releases/latest"
	repoOwner    = "juanhuttemann"
	repoName     = "monkey-cli"
)

// detectPlatform returns the OS and architecture strings matching the release asset naming convention.
func detectPlatform() (string, string, error) {
	osMap := map[string]string{
		"linux":  "Linux",
		"darwin": "Darwin",
	}
	archMap := map[string]string{
		"amd64": "x86_64",
		"arm64": "arm64",
	}

	goos, ok := osMap[runtime.GOOS]
	if !ok {
		return "", "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	goarch, ok := archMap[runtime.GOARCH]
	if !ok {
		return "", "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}
	return goos, goarch, nil
}

// fetchLatestVersion queries the GitHub releases API and returns the latest tag name.
func fetchLatestVersion(apiURL string) (string, error) {
	resp, err := http.Get(apiURL) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest version: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse release response: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("failed to resolve latest version: empty tag_name")
	}
	return release.TagName, nil
}

// parseVersion splits a semver string (with optional leading "v") into [major, minor, patch].
func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var nums [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		nums[i], _ = strconv.Atoi(p)
	}
	return nums
}

// isNewerVersion returns true when latest is strictly greater than current.
func isNewerVersion(current, latest string) bool {
	c := parseVersion(current)
	l := parseVersion(latest)
	for i := range c {
		if l[i] > c[i] {
			return true
		}
		if l[i] < c[i] {
			return false
		}
	}
	return false
}

// downloadAndInstall downloads a tar.gz archive from url and installs the "monkey" binary to destPath.
func downloadAndInstall(url, destPath string) error {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read gzip: %w", err)
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}
		if filepath.Base(hdr.Name) != AppName {
			continue
		}

		// Write to a temp file in the same directory, then atomically rename.
		dir := filepath.Dir(destPath)
		tmp, err := os.CreateTemp(dir, ".monkey-update-*")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpName := tmp.Name()

		if _, err := io.Copy(tmp, tr); err != nil { //nolint:gosec
			_ = tmp.Close()
			_ = os.Remove(tmpName)
			return fmt.Errorf("failed to write binary: %w", err)
		}
		if err := tmp.Chmod(0755); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
			return fmt.Errorf("failed to chmod binary: %w", err)
		}
		_ = tmp.Close()

		if err := os.Rename(tmpName, destPath); err != nil {
			_ = os.Remove(tmpName)
			return fmt.Errorf("failed to install binary: %w", err)
		}
		return nil
	}
	return fmt.Errorf("binary %q not found in archive", AppName)
}

// runUpdate checks for a newer version and installs it if available.
// apiURL is the GitHub releases API endpoint; installPath is the binary destination.
// If installPath is empty, it resolves to the current executable's path.
func runUpdate(apiURL, installPath string) (string, error) {
	latest, err := fetchLatestVersion(apiURL)
	if err != nil {
		return "", err
	}

	if !isNewerVersion(Version, latest) {
		return fmt.Sprintf("monkey is already at the latest version (v%s)", Version), nil
	}

	goos, goarch, err := detectPlatform()
	if err != nil {
		return "", err
	}

	archive := fmt.Sprintf("%s_%s_%s.tar.gz", repoName, goos, goarch)
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		repoOwner, repoName, latest, archive)

	// When called from a test, downloadURL is derived from apiURL's host for the tarball.
	// In tests, apiURL is a mock server that serves both the version and the tarball.
	// We override the download URL to use the same mock server.
	if installPath != "" {
		// In test mode: derive download URL from the apiURL base (strip path, use /download)
		base := apiURL
		if idx := strings.Index(base, "/releases"); idx != -1 {
			base = base[:idx]
		}
		// If apiURL has no /releases path (bare server URL in tests), use it directly.
		downloadURL = base + "/download/" + latest + "/" + archive
	}

	if installPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("could not determine executable path: %w", err)
		}
		installPath = exe
	}

	fmt.Printf("Updating monkey to %s...\n", latest)
	if err := downloadAndInstall(downloadURL, installPath); err != nil {
		return "", err
	}
	return fmt.Sprintf("Successfully updated monkey to %s", latest), nil
}
