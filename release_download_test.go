package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSelectLatestReleasePrefersStable(t *testing.T) {
	t.Parallel()

	releases := []githubRelease{
		{
			TagName:    "v2.0.0-rc1",
			Prerelease: true,
		},
		{
			TagName: "v1.9.0",
		},
	}

	selected, err := selectLatestRelease(releases)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if selected.TagName != "v1.9.0" {
		t.Fatalf("expected stable release to be selected, got: %s", selected.TagName)
	}
}

func TestSelectLatestReleaseFallsBackToPrerelease(t *testing.T) {
	t.Parallel()

	releases := []githubRelease{
		{
			TagName:    "v2.0.0-rc1",
			Prerelease: true,
		},
		{
			TagName:    "v1.0.0-rc1",
			Prerelease: true,
		},
	}

	selected, err := selectLatestRelease(releases)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if selected.TagName != "v2.0.0-rc1" {
		t.Fatalf("expected first non-draft release to be selected, got: %s", selected.TagName)
	}
}

func TestBuildPlatformDownloadsMatchesPlatformAndArch(t *testing.T) {
	t.Parallel()

	pattern, err := compileCaseInsensitiveRegex(`tool-v1\.0\.0-.*`)
	if err != nil {
		t.Fatalf("failed to compile regex: %v", err)
	}

	assets := []githubAsset{
		{Name: "tool-v1.0.0-windows-amd64.zip", BrowserDownloadURL: "https://example.com/windows-amd64.zip"},
		{Name: "tool-v1.0.0-windows-386.zip", BrowserDownloadURL: "https://example.com/windows-386.zip"},
		{Name: "tool-v1.0.0-linux-arm64.tar.gz", BrowserDownloadURL: "https://example.com/linux-arm64.tar.gz"},
		{Name: "tool-v1.0.0-linux-amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux-amd64.tar.gz"},
		{Name: "tool-v1.0.0-darwin-x86_64.zip", BrowserDownloadURL: "https://example.com/macos-amd64.zip"},
		{Name: "tool-v1.0.0-darwin-arm64.zip", BrowserDownloadURL: "https://example.com/macos-arm64.zip"},
	}

	downloads := buildPlatformDownloads(assets, pattern, archARM64)

	windows := downloads[platformWindows]
	if !windows.Available {
		t.Fatal("expected windows download to be available")
	}
	if windows.Arch != archAMD64 {
		t.Fatalf("expected windows fallback arch amd64, got: %s", windows.Arch)
	}

	linux := downloads[platformLinux]
	if !linux.Available {
		t.Fatal("expected linux download to be available")
	}
	if linux.Arch != archARM64 {
		t.Fatalf("expected linux arch arm64, got: %s", linux.Arch)
	}

	macos := downloads[platformMacOS]
	if !macos.Available {
		t.Fatal("expected macos download to be available")
	}
	if macos.Arch != archARM64 {
		t.Fatalf("expected macos arch arm64, got: %s", macos.Arch)
	}
}

func TestBuildPlatformDownloadsMarksUnavailableWhenMissing(t *testing.T) {
	t.Parallel()

	pattern, err := compileCaseInsensitiveRegex(`tool-v1\.0\.0-.*`)
	if err != nil {
		t.Fatalf("failed to compile regex: %v", err)
	}

	assets := []githubAsset{
		{Name: "tool-v1.0.0-windows-amd64.zip", BrowserDownloadURL: "https://example.com/windows-amd64.zip"},
		{Name: "tool-v1.0.0-darwin-arm64.zip", BrowserDownloadURL: "https://example.com/macos-arm64.zip"},
	}

	downloads := buildPlatformDownloads(assets, pattern, archARM64)
	linux := downloads[platformLinux]
	if linux.Available {
		t.Fatal("expected linux download to be unavailable when no matching asset exists")
	}
}

func TestSelectAssetForPlatformAppliesProxyForGitHubDownloadURL(t *testing.T) {
	t.Parallel()

	pattern, err := compileCaseInsensitiveRegex(`tool-v1\.0\.0-.*`)
	if err != nil {
		t.Fatalf("failed to compile regex: %v", err)
	}

	assets := []githubAsset{
		{
			Name:               "tool-v1.0.0-windows-amd64.zip",
			BrowserDownloadURL: "https://github.com/owner/repo/releases/download/v1.0.0/tool-v1.0.0-windows-amd64.zip",
		},
	}

	selected := selectAssetForPlatform(assets, pattern, platformWindows, archAMD64)
	if !selected.Available {
		t.Fatal("expected selected asset to be available")
	}

	want := "https://ghproxy.net/https://github.com/owner/repo/releases/download/v1.0.0/tool-v1.0.0-windows-amd64.zip"
	if selected.DownloadURL != want {
		t.Fatalf("expected proxied download URL %q, got %q", want, selected.DownloadURL)
	}
}

func TestStartDownloadLifecycleSuccess(t *testing.T) {
	payload := strings.Repeat("ossam-test-", 4096)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Length", "45056")
		writer.Header().Set("ETag", "unit-test-etag")
		writer.Header().Set("Accept-Ranges", "bytes")
		_, _ = writer.Write([]byte(payload))
	}))
	defer server.Close()

	app := NewApp()
	downloadDir := t.TempDir()

	started, err := app.StartDownload(StartDownloadRequest{
		DownloadURL: server.URL + "/tool.zip",
		FileName:    "tool.zip",
		Platform:    platformWindows,
		DownloadDir: downloadDir,
	})
	if err != nil {
		t.Fatalf("expected start download success, got error: %v", err)
	}

	terminal := waitForTerminalTaskState(t, app, started.TaskID)
	if terminal.Status != taskStatusCompleted {
		t.Fatalf("expected completed status, got: %s (%s)", terminal.Status, terminal.Error)
	}
	if terminal.Progress != 100 {
		t.Fatalf("expected progress 100, got: %d", terminal.Progress)
	}

	downloadedContent, err := os.ReadFile(filepath.Join(downloadDir, "tool.zip"))
	if err != nil {
		t.Fatalf("expected output file to exist, got error: %v", err)
	}

	if string(downloadedContent) != payload {
		t.Fatalf("downloaded file content mismatch")
	}
}

func TestStartDownloadLifecycleFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "test error", http.StatusInternalServerError)
	}))
	defer server.Close()

	app := NewApp()
	started, err := app.StartDownload(StartDownloadRequest{
		DownloadURL: server.URL + "/tool.zip",
		FileName:    "tool.zip",
		Platform:    platformWindows,
		DownloadDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("expected start download success, got error: %v", err)
	}

	terminal := waitForTerminalTaskState(t, app, started.TaskID)
	if terminal.Status != taskStatusFailed {
		t.Fatalf("expected failed status, got: %s", terminal.Status)
	}
	if !strings.Contains(terminal.Error, "status 500") {
		t.Fatalf("expected error to mention status 500, got: %s", terminal.Error)
	}
}

func waitForTerminalTaskState(t *testing.T, app *App, taskID string) *DownloadTaskSnapshot {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := app.GetDownloadTask(taskID)
		if err != nil {
			t.Fatalf("failed to read task snapshot: %v", err)
		}
		if snapshot.Status == taskStatusCompleted || snapshot.Status == taskStatusFailed {
			return snapshot
		}

		time.Sleep(40 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for terminal task state for task id: %s", taskID)
	return nil
}
