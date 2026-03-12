package main

import (
	"regexp"
	"testing"
)

func TestBuildPlatformDownloadsKeepsMatchedPlatformAssets(t *testing.T) {
	t.Parallel()

	pattern := regexp.MustCompile(`(?i:.*\.zip$)`)
	assets := []githubAsset{
		{
			Name:               "tool-windows-amd64.zip",
			BrowserDownloadURL: "https://github.com/owner/repo/releases/download/v1/tool-windows-amd64.zip",
		},
		{
			Name:               "tool-linux-amd64.zip",
			BrowserDownloadURL: "https://github.com/owner/repo/releases/download/v1/tool-linux-amd64.zip",
		},
		{
			Name:               "tool-macos-amd64.zip",
			BrowserDownloadURL: "https://github.com/owner/repo/releases/download/v1/tool-macos-amd64.zip",
		},
	}

	downloads := buildPlatformDownloads(assets, "https://api.github.com/repos/owner/repo/zipball/v1", pattern, archAMD64)
	for _, platform := range []string{platformWindows, platformLinux, platformMacOS} {
		item := downloads[platform]
		if !item.Available {
			t.Fatalf("expected %s to be available", platform)
		}
		if item.AssetName == "source-code.zip" {
			t.Fatalf("expected %s to keep matched platform asset, got fallback asset", platform)
		}
		if item.DownloadURL == "https://ghproxy.net/https://api.github.com/repos/owner/repo/zipball/v1" {
			t.Fatalf("expected %s to keep platform asset URL, got source zip URL", platform)
		}
	}
}

func TestBuildPlatformDownloadsFallsBackPerMissingPlatform(t *testing.T) {
	t.Parallel()

	pattern := regexp.MustCompile(`(?i:.*\.zip$)`)
	assets := []githubAsset{
		{
			Name:               "tool-windows-amd64.zip",
			BrowserDownloadURL: "https://github.com/owner/repo/releases/download/v1/tool-windows-amd64.zip",
		},
	}

	sourceURL := "https://api.github.com/repos/owner/repo/zipball/v1"
	downloads := buildPlatformDownloads(assets, sourceURL, pattern, archAMD64)

	if !downloads[platformWindows].Available || downloads[platformWindows].AssetName == "source-code.zip" {
		t.Fatalf("expected windows to use matched asset, got %+v", downloads[platformWindows])
	}

	for _, platform := range []string{platformLinux, platformMacOS} {
		item := downloads[platform]
		if !item.Available {
			t.Fatalf("expected %s to fallback to source code zip", platform)
		}
		if item.AssetName != "source-code.zip" {
			t.Fatalf("expected %s fallback asset name source-code.zip, got %s", platform, item.AssetName)
		}
		if item.DownloadURL != "https://ghproxy.net/"+sourceURL {
			t.Fatalf("expected %s fallback URL through ghproxy, got %s", platform, item.DownloadURL)
		}
	}
}

func TestBuildPlatformDownloadsFallsBackForAllPlatformsWhenNoMatches(t *testing.T) {
	t.Parallel()

	pattern := regexp.MustCompile(`(?i:.*\.zip$)`)
	sourceURL := "https://api.github.com/repos/owner/repo/zipball/v1"
	downloads := buildPlatformDownloads(nil, sourceURL, pattern, archAMD64)

	for _, platform := range []string{platformWindows, platformLinux, platformMacOS} {
		item := downloads[platform]
		if !item.Available {
			t.Fatalf("expected %s to fallback to source code zip", platform)
		}
		if item.AssetName != "source-code.zip" {
			t.Fatalf("expected %s fallback asset name source-code.zip, got %s", platform, item.AssetName)
		}
		if item.DownloadURL != "https://ghproxy.net/"+sourceURL {
			t.Fatalf("expected %s fallback URL through ghproxy, got %s", platform, item.DownloadURL)
		}
	}
}

func TestBuildPlatformDownloadsStaysUnavailableWithoutSourceZip(t *testing.T) {
	t.Parallel()

	pattern := regexp.MustCompile(`(?i:.*\.zip$)`)
	downloads := buildPlatformDownloads(nil, "", pattern, archAMD64)

	for _, platform := range []string{platformWindows, platformLinux, platformMacOS} {
		item := downloads[platform]
		if item.Available {
			t.Fatalf("expected %s unavailable without platform assets and source zip, got %+v", platform, item)
		}
	}
}
