package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func writeFixtureFile(t *testing.T, filePath, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("create parent dir failed: %v", err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture failed: %v", err)
	}
}

func TestNormalizeMatchPatternCompatibility(t *testing.T) {
	pattern := `.*\\.(zip|tar\\.gz)$`
	regex, err := compileCaseInsensitiveRegex(pattern)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	if !regex.MatchString("tool.zip") {
		t.Fatal("expected .zip to match normalized pattern")
	}
	if !regex.MatchString("tool.tar.gz") {
		t.Fatal("expected .tar.gz to match normalized pattern")
	}
}

func TestLoadAppsConfigFromFilesNormalizesLegacyEscapes(t *testing.T) {
	tempDir := t.TempDir()
	rulesPath := filepath.Join(tempDir, "config", "rules.json")
	categoryPath := filepath.Join(tempDir, "config", "categories", "network.json")

	writeFixtureFile(t, rulesPath, `{
  "market_name": "ossam",
  "last_updated": "20260312",
  "default_match": ".*\\\\.(zip|tar\\\\.gz)$",
  "platform_keywords": {
    "windows": ["windows", "win", "win32", "win64"],
    "linux": ["linux", "gnu", "musl"],
    "macos": ["mac", "macos", "darwin", "osx"]
  },
  "categories": [
    {"name": "Network", "file": "categories/network.json"}
  ]
}`)

	writeFixtureFile(t, categoryPath, `{
  "apps": [
    {
      "name": "test-app",
      "repo": "owner/repo",
      "photo": "",
      "summary": "demo"
    },
    {
      "name": "special-app",
      "repo": "owner/special",
      "photo": "",
      "match": "special\\\\.zip$",
      "summary": "demo"
    }
  ]
}`)

	cfg, err := loadAppsConfigFromFiles(rulesPath)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	apps := cfg.Apps["Network"]
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(apps))
	}

	if strings.Contains(apps[0].Match, `\\`) {
		t.Fatalf("default match should be normalized, got %q", apps[0].Match)
	}
	if strings.Contains(apps[1].Match, `\\`) {
		t.Fatalf("custom match should be normalized, got %q", apps[1].Match)
	}

	r1, _ := regexp.Compile("(?i:" + apps[0].Match + ")")
	if !r1.MatchString("artifact.zip") || !r1.MatchString("artifact.tar.gz") {
		t.Fatalf("normalized default match should match expected assets, got %q", apps[0].Match)
	}
	r2, _ := regexp.Compile("(?i:" + apps[1].Match + ")")
	if !r2.MatchString("special.zip") {
		t.Fatalf("normalized custom match should match expected assets, got %q", apps[1].Match)
	}
}

func TestResolvePlatformKeywordsFromRules(t *testing.T) {
	keywords, err := resolvePlatformKeywords(map[string][]string{
		platformWindows: {"windows", "win", "win64"},
		platformLinux:   {"linux", "gnu", "musl"},
		platformMacOS:   {"mac", "macos", "darwin", "osx"},
	}, nil)
	if err != nil {
		t.Fatalf("expected keywords to resolve, got error: %v", err)
	}

	if len(keywords[platformWindows]) == 0 || len(keywords[platformLinux]) == 0 || len(keywords[platformMacOS]) == 0 {
		t.Fatalf("resolved keywords should contain all platforms: %+v", keywords)
	}
}

func TestResolvePlatformKeywordsFallbackFromPlatformMatch(t *testing.T) {
	keywords, err := resolvePlatformKeywords(nil, map[string]string{
		platformWindows: "(windows|win|win64)",
		platformLinux:   "(linux|gnu|musl)",
		platformMacOS:   "(mac|macos|darwin|osx)",
	})
	if err != nil {
		t.Fatalf("expected fallback keywords to resolve, got error: %v", err)
	}

	if !containsKeyword(keywords[platformWindows], "win64") {
		t.Fatalf("expected win64 keyword in fallback keywords: %+v", keywords[platformWindows])
	}
	if !containsKeyword(keywords[platformMacOS], "darwin") {
		t.Fatalf("expected darwin keyword in fallback keywords: %+v", keywords[platformMacOS])
	}
}

func TestMatchesPlatformWithAliasesAndAmbiguousAsset(t *testing.T) {
	keywords := map[string][]string{
		platformWindows: {"windows", "win", "win32", "win64"},
		platformLinux:   {"linux", "gnu", "musl"},
		platformMacOS:   {"mac", "macos", "darwin", "osx"},
	}

	if !matchesPlatform("tool-win64-amd64.zip", platformWindows, keywords) {
		t.Fatal("expected win64 asset to match windows")
	}
	if !matchesPlatform("tool-darwin-arm64.tar.gz", platformMacOS, keywords) {
		t.Fatal("expected darwin asset to match macOS")
	}
	if !matchesPlatform("tool-linux-musl-amd64.tar.gz", platformLinux, keywords) {
		t.Fatal("expected linux asset to match linux")
	}

	// Ambiguous assets should be skipped for all platforms.
	if matchesPlatform("tool-windows-linux-amd64.zip", platformWindows, keywords) {
		t.Fatal("ambiguous asset should not match windows")
	}
	if matchesPlatform("tool-windows-linux-amd64.zip", platformLinux, keywords) {
		t.Fatal("ambiguous asset should not match linux")
	}
}

func TestBuildPlatformDownloadsSourceFallbackWhenNoPlatformKeyword(t *testing.T) {
	basePattern, err := compileCaseInsensitiveRegex(`.*\.zip$`)
	if err != nil {
		t.Fatalf("compile base pattern failed: %v", err)
	}

	downloads := buildPlatformDownloads(
		[]githubAsset{
			{Name: "tool-v1.0.0.zip", BrowserDownloadURL: "https://example.com/tool-v1.0.0.zip"},
		},
		"https://github.com/owner/repo/archive/refs/tags/v1.0.0.zip",
		basePattern,
		archAMD64,
		map[string][]string{
			platformWindows: {"windows", "win", "win64"},
			platformLinux:   {"linux", "gnu", "musl"},
			platformMacOS:   {"mac", "macos", "darwin", "osx"},
		},
	)

	for _, platform := range supportedPlatforms {
		item := downloads[platform]
		if !item.Available {
			t.Fatalf("%s should fallback to source package", platform)
		}
		if item.AssetName != "source-code.zip" {
			t.Fatalf("%s fallback should be source-code.zip, got %q", platform, item.AssetName)
		}
	}
}

func containsKeyword(keywords []string, target string) bool {
	for _, keyword := range keywords {
		if keyword == target {
			return true
		}
	}
	return false
}
