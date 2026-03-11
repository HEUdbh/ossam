package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestGetRepoStarsFetchesAndFallsBackToCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "stars_cache.json")
	t.Setenv(repoStarsCacheEnv, cachePath)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/repos/owner/repo" {
			http.NotFound(writer, request)
			return
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"stargazers_count": 1234,
		})
	}))
	defer server.Close()

	app := NewApp()
	app.githubAPIBaseURL = server.URL
	app.apiClient = server.Client()

	stars, err := app.GetRepoStars([]string{"owner/repo", "owner/repo"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if stars["owner/repo"] != 1234 {
		t.Fatalf("expected stars 1234, got: %d", stars["owner/repo"])
	}

	cached, err := readRepoStarsCache(cachePath)
	if err != nil {
		t.Fatalf("expected cache read success, got: %v", err)
	}
	if cached.Repos["owner/repo"].Stars != 1234 {
		t.Fatalf("expected cached stars 1234, got: %d", cached.Repos["owner/repo"].Stars)
	}

	app.githubAPIBaseURL = "http://127.0.0.1:1"
	app.apiClient = &http.Client{Timeout: 200 * time.Millisecond}

	fallbackStars, err := app.GetRepoStars([]string{"owner/repo", "owner/missing"})
	if err != nil {
		t.Fatalf("expected no error when using cache fallback, got: %v", err)
	}
	if fallbackStars["owner/repo"] != 1234 {
		t.Fatalf("expected cached stars 1234, got: %d", fallbackStars["owner/repo"])
	}
	if _, exists := fallbackStars["owner/missing"]; exists {
		t.Fatalf("expected missing repo without cache to be omitted, got: %+v", fallbackStars)
	}
}

func TestGetRepoStarsSkipsInvalidRepo(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "stars_cache.json")
	t.Setenv(repoStarsCacheEnv, cachePath)

	app := NewApp()
	app.githubAPIBaseURL = "http://127.0.0.1:1"
	app.apiClient = &http.Client{Timeout: 200 * time.Millisecond}

	stars, err := app.GetRepoStars([]string{"", "invalid", "owner/repo"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(stars) != 0 {
		t.Fatalf("expected no stars because only valid repo request failed without cache, got: %+v", stars)
	}
}
