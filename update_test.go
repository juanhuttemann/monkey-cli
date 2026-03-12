package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	goos, goarch, err := detectPlatform()
	if err != nil {
		t.Fatalf("detectPlatform() returned error: %v", err)
	}
	validOS := map[string]bool{"Linux": true, "Darwin": true}
	if !validOS[goos] {
		t.Errorf("detectPlatform() OS = %q, want Linux or Darwin", goos)
	}
	validArch := map[string]bool{"x86_64": true, "arm64": true}
	if !validArch[goarch] {
		t.Errorf("detectPlatform() arch = %q, want x86_64 or arm64", goarch)
	}
}

func TestFetchLatestVersion_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.2.3"})
	}))
	defer server.Close()

	version, err := fetchLatestVersion(server.URL)
	if err != nil {
		t.Fatalf("fetchLatestVersion() returned error: %v", err)
	}
	if version != "v1.2.3" {
		t.Errorf("fetchLatestVersion() = %q, want %q", version, "v1.2.3")
	}
}

func TestFetchLatestVersion_EmptyTagName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": ""})
	}))
	defer server.Close()

	_, err := fetchLatestVersion(server.URL)
	if err == nil {
		t.Fatal("fetchLatestVersion() should return error when tag_name is empty")
	}
}

func TestFetchLatestVersion_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{invalid`))
	}))
	defer server.Close()

	_, err := fetchLatestVersion(server.URL)
	if err == nil {
		t.Fatal("fetchLatestVersion() should return error on invalid JSON")
	}
}

func TestFetchLatestVersion_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	_, err := fetchLatestVersion(server.URL)
	if err == nil {
		t.Fatal("fetchLatestVersion() should return error on non-200 status")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should mention status code 403, got: %v", err)
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"v0.2.0", "v0.3.0", true},
		{"v0.2.0", "v0.2.0", false},
		{"v0.3.0", "v0.2.0", false},
		{"0.2.0", "v0.3.0", true},
		{"v0.2.0", "0.3.0", true},
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.1", "v1.0.0", false},
		{"v2.0.0", "v1.9.9", false},
	}
	for _, tt := range tests {
		got := isNewerVersion(tt.current, tt.latest)
		if got != tt.want {
			t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}

// makeTarGz creates an in-memory tar.gz archive containing a single file with the given content.
func makeTarGz(t *testing.T, filename, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := []byte(content)
	hdr := &tar.Header{
		Name: filename,
		Mode: 0755,
		Size: int64(len(body)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDownloadAndInstall_Success(t *testing.T) {
	binaryContent := "fake-binary-content"
	archiveBytes := makeTarGz(t, "monkey", binaryContent)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, "monkey")

	err := downloadAndInstall(server.URL, destPath)
	if err != nil {
		t.Fatalf("downloadAndInstall() returned error: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("could not read installed binary: %v", err)
	}
	if string(data) != binaryContent {
		t.Errorf("installed binary content = %q, want %q", string(data), binaryContent)
	}

	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("installed binary should be executable")
	}
}

func TestDownloadAndInstall_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := downloadAndInstall(server.URL, filepath.Join(t.TempDir(), "monkey"))
	if err == nil {
		t.Fatal("downloadAndInstall() should return error on HTTP error")
	}
}

func TestRunUpdate_AlreadyLatest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v" + Version})
	}))
	defer server.Close()

	msg, err := runUpdate(server.URL, "")
	if err != nil {
		t.Fatalf("runUpdate() returned error: %v", err)
	}
	if !strings.Contains(msg, "already") {
		t.Errorf("runUpdate() message = %q, should indicate already up to date", msg)
	}
}

func TestRunUpdate_FetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := runUpdate(server.URL, "")
	if err == nil {
		t.Fatal("runUpdate should return error when fetchLatestVersion fails")
	}
}

func TestDownloadAndInstall_NotGzip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not a gzip archive"))
	}))
	defer server.Close()

	err := downloadAndInstall(server.URL, filepath.Join(t.TempDir(), "monkey"))
	if err == nil {
		t.Fatal("downloadAndInstall should return error for non-gzip response")
	}
}

func TestDownloadAndInstall_BinaryNotFoundInArchive(t *testing.T) {
	// Archive contains "other" but not "monkey"
	archiveBytes := makeTarGz(t, "other-binary", "content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	err := downloadAndInstall(server.URL, filepath.Join(t.TempDir(), "monkey"))
	if err == nil {
		t.Fatal("downloadAndInstall should return error when binary not in archive")
	}
	if !strings.Contains(err.Error(), AppName) {
		t.Errorf("error should mention binary name, got: %v", err)
	}
}

func TestRunUpdate_DownloadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "download") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v99.99.99"})
	}))
	defer server.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, "monkey")

	_, err := runUpdate(server.URL, destPath)
	if err == nil {
		t.Fatal("runUpdate should return error when download fails")
	}
}

func TestDownloadAndInstall_NetworkError(t *testing.T) {
	// Use a server that is already closed to simulate network error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	err := downloadAndInstall(url, filepath.Join(t.TempDir(), "monkey"))
	if err == nil {
		t.Fatal("downloadAndInstall should return error on network failure")
	}
}

func TestParseVersion_EmptyString(t *testing.T) {
	v := parseVersion("")
	if v != [3]int{0, 0, 0} {
		t.Errorf("parseVersion('') = %v, want [0,0,0]", v)
	}
}

func TestParseVersion_SingleComponent(t *testing.T) {
	v := parseVersion("5")
	if v[0] != 5 {
		t.Errorf("parseVersion('5')[0] = %d, want 5", v[0])
	}
}

func TestRunUpdate_NewVersionAvailable(t *testing.T) {
	binaryContent := "new-monkey-binary"
	archiveBytes := makeTarGz(t, "monkey", binaryContent)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "releases/latest") || r.URL.Path == "/" {
			_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v99.99.99"})
			return
		}
		// download request
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	dir := t.TempDir()
	destPath := filepath.Join(dir, "monkey")

	msg, err := runUpdate(server.URL, destPath)
	if err != nil {
		t.Fatalf("runUpdate() returned error: %v", err)
	}
	if !strings.Contains(msg, "v99.99.99") {
		t.Errorf("runUpdate() message = %q, should contain new version", msg)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("installed binary not found: %v", err)
	}
	if string(data) != binaryContent {
		t.Errorf("installed binary = %q, want %q", string(data), binaryContent)
	}
}
