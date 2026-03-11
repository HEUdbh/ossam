package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchReleasesUsesDirectURL(t *testing.T) {
	t.Parallel()

	expectedURL := "https://api.github.com/repos/owner/repo/releases?per_page=30"
	var requestedURL string

	app := NewApp()
	app.apiClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestedURL = req.URL.String()
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	if _, err := app.fetchReleases("owner/repo"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if requestedURL != expectedURL {
		t.Fatalf("expected release URL %q, got %q", expectedURL, requestedURL)
	}
}

func TestFetchRepoStarsUsesProxyURL(t *testing.T) {
	t.Parallel()

	expectedURL := "https://ghproxy.net/https://api.github.com/repos/owner/repo"
	var requestedURL string

	app := NewApp()
	app.apiClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestedURL = req.URL.String()
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"stargazers_count": 42}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	stars, err := app.fetchRepoStars("owner/repo")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if stars != 42 {
		t.Fatalf("expected stars 42, got %d", stars)
	}

	if requestedURL != expectedURL {
		t.Fatalf("expected stars URL %q, got %q", expectedURL, requestedURL)
	}
}

func TestStartDownloadAppliesProxyForGitHubURL(t *testing.T) {
	expectedURL := "https://ghproxy.net/https://github.com/owner/repo/releases/download/v1/tool.zip"
	var requestedURL string

	app := NewApp()
	app.downloadClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestedURL = req.URL.String()
			return &http.Response{
				StatusCode:    http.StatusOK,
				Body:          io.NopCloser(strings.NewReader("ok")),
				Header:        make(http.Header),
				ContentLength: int64(len("ok")),
			}, nil
		}),
	}

	started, err := app.StartDownload(StartDownloadRequest{
		DownloadURL: "https://github.com/owner/repo/releases/download/v1/tool.zip",
		FileName:    "tool.zip",
		Platform:    platformWindows,
		DownloadDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if started.DownloadURL != expectedURL {
		t.Fatalf("expected task download URL %q, got %q", expectedURL, started.DownloadURL)
	}

	terminal := waitForTerminalTaskState(t, app, started.TaskID)
	if terminal.Status != taskStatusCompleted {
		t.Fatalf("expected completed status, got: %s (%s)", terminal.Status, terminal.Error)
	}

	if requestedURL != expectedURL {
		t.Fatalf("expected HTTP request URL %q, got %q", expectedURL, requestedURL)
	}
}

func TestStartDownloadKeepsNonGitHubURLDirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("ok"))
	}))
	defer server.Close()

	app := NewApp()
	app.downloadClient = server.Client()

	downloadURL := server.URL + "/tool.zip"
	started, err := app.StartDownload(StartDownloadRequest{
		DownloadURL: downloadURL,
		FileName:    "tool.zip",
		Platform:    platformWindows,
		DownloadDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if started.DownloadURL != downloadURL {
		t.Fatalf("expected non-github URL to stay direct %q, got %q", downloadURL, started.DownloadURL)
	}

	terminal := waitForTerminalTaskState(t, app, started.TaskID)
	if terminal.Status != taskStatusCompleted {
		t.Fatalf("expected completed status, got: %s (%s)", terminal.Status, terminal.Error)
	}
}
