package main

import "testing"

func TestBuildReadmeRawCandidates(t *testing.T) {
	t.Parallel()

	got := buildReadmeRawCandidates("owner/repo")
	want := []readmeCandidate{
		{
			URL:      "https://ghproxy.net/https://raw.githubusercontent.com/owner/repo/main/readme.md",
			Branch:   "main",
			FilePath: "readme.md",
		},
		{
			URL:      "https://ghproxy.net/https://raw.githubusercontent.com/owner/repo/main/README.md",
			Branch:   "main",
			FilePath: "README.md",
		},
		{
			URL:      "https://ghproxy.net/https://raw.githubusercontent.com/owner/repo/master/readme.md",
			Branch:   "master",
			FilePath: "readme.md",
		},
		{
			URL:      "https://ghproxy.net/https://raw.githubusercontent.com/owner/repo/master/README.md",
			Branch:   "master",
			FilePath: "README.md",
		},
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d candidates, got %d", len(want), len(got))
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("expected candidate %d to be %+v, got %+v", idx, want[idx], got[idx])
		}
	}
}

func TestResolveAppPhotoKeepsCustomURLAndSkipsProxyForDefault(t *testing.T) {
	t.Parallel()

	defaultAvatar := resolveAppPhoto("junegunn/fzf", "")
	if defaultAvatar != "https://avatars.githubusercontent.com/junegunn" {
		t.Fatalf("expected default avatar without proxy, got %q", defaultAvatar)
	}

	defaultPlaceholder := resolveAppPhoto("invalid", "")
	if defaultPlaceholder != defaultAppPlaceholder {
		t.Fatalf("expected default placeholder without proxy, got %q", defaultPlaceholder)
	}

	custom := resolveAppPhoto("owner/repo", "https://github.com/custom.png")
	if custom != "https://github.com/custom.png" {
		t.Fatalf("expected custom photo to keep original URL, got %q", custom)
	}
}
