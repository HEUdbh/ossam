package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetchReadmeUsesProxyRawURLAndReturnsDetails(t *testing.T) {
	t.Parallel()

	expected := []string{
		"https://ghproxy.net/https://raw.githubusercontent.com/owner/repo/main/readme.md",
		"https://ghproxy.net/https://raw.githubusercontent.com/owner/repo/main/README.md",
	}
	var requested []string

	app := NewApp()
	app.apiClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requested = append(requested, req.URL.String())
			if len(requested) == 1 {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("# hello")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	readme, err := app.fetchReadme("owner/repo")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if readme.Content != "# hello" {
		t.Fatalf("expected readme body to match, got %q", readme.Content)
	}
	if readme.SourceURL != expected[1] {
		t.Fatalf("expected readme source URL %q, got %q", expected[1], readme.SourceURL)
	}
	if readme.Branch != "main" {
		t.Fatalf("expected readme branch to be main, got %q", readme.Branch)
	}
	if readme.FilePath != "README.md" {
		t.Fatalf("expected readme file path to be README.md, got %q", readme.FilePath)
	}
	if len(requested) != len(expected) {
		t.Fatalf("expected %d requests, got %d (%v)", len(expected), len(requested), requested)
	}
	for idx := range expected {
		if requested[idx] != expected[idx] {
			t.Fatalf("expected request %d URL %q, got %q", idx, expected[idx], requested[idx])
		}
	}
}

func TestFetchReadmeFallsBackToMasterWhenMainMissing(t *testing.T) {
	t.Parallel()

	expected := []string{
		"https://ghproxy.net/https://raw.githubusercontent.com/owner/repo/main/readme.md",
		"https://ghproxy.net/https://raw.githubusercontent.com/owner/repo/main/README.md",
		"https://ghproxy.net/https://raw.githubusercontent.com/owner/repo/master/readme.md",
	}
	var requested []string

	app := NewApp()
	app.apiClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requested = append(requested, req.URL.String())
			if len(requested) < 3 {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("master readme")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	readme, err := app.fetchReadme("owner/repo")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if readme.Content != "master readme" {
		t.Fatalf("expected fallback readme body, got %q", readme.Content)
	}
	if readme.SourceURL != expected[2] {
		t.Fatalf("expected readme source URL %q, got %q", expected[2], readme.SourceURL)
	}
	if readme.Branch != "master" {
		t.Fatalf("expected readme branch to be master, got %q", readme.Branch)
	}
	if readme.FilePath != "readme.md" {
		t.Fatalf("expected readme file path to be readme.md, got %q", readme.FilePath)
	}
	if len(requested) != len(expected) {
		t.Fatalf("expected %d requests, got %d (%v)", len(expected), len(requested), requested)
	}
	for idx := range expected {
		if requested[idx] != expected[idx] {
			t.Fatalf("expected request %d URL %q, got %q", idx, expected[idx], requested[idx])
		}
	}
}

func TestFetchReadmeReturnsEmptyWhenAllCandidatesMissing(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.apiClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	readme, err := app.fetchReadme("owner/repo")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if readme.Content != "" {
		t.Fatalf("expected empty readme when all candidates missing, got %q", readme.Content)
	}
	if readme.SourceURL != "" || readme.Branch != "" || readme.FilePath != "" {
		t.Fatalf("expected empty readme details when all candidates missing, got %+v", readme)
	}
}
