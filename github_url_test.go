package main

import "testing"

func TestBuildReadmeRawCandidates(t *testing.T) {
	t.Parallel()

	got := buildReadmeRawCandidates("owner/repo")
	want := []readmeCandidate{
		{
			URL:      "https://raw.githubusercontent.com/owner/repo/main/readme.md",
			Branch:   "main",
			FilePath: "readme.md",
		},
		{
			URL:      "https://raw.githubusercontent.com/owner/repo/main/README.md",
			Branch:   "main",
			FilePath: "README.md",
		},
		{
			URL:      "https://raw.githubusercontent.com/owner/repo/master/readme.md",
			Branch:   "master",
			FilePath: "readme.md",
		},
		{
			URL:      "https://raw.githubusercontent.com/owner/repo/master/README.md",
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

func TestApplyGitHubProxy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "github repo url",
			url:  "https://github.com/owner/repo",
			want: "https://ghproxy.net/https://github.com/owner/repo",
		},
		{
			name: "github api url",
			url:  "https://api.github.com/repos/owner/repo",
			want: "https://ghproxy.net/https://api.github.com/repos/owner/repo",
		},
		{
			name: "raw url",
			url:  "https://raw.githubusercontent.com/owner/repo/main/README.md",
			want: "https://ghproxy.net/https://raw.githubusercontent.com/owner/repo/main/README.md",
		},
		{
			name: "avatar url keeps direct",
			url:  "https://avatars.githubusercontent.com/owner",
			want: "https://avatars.githubusercontent.com/owner",
		},
		{
			name: "already proxied url",
			url:  "https://ghproxy.net/https://github.com/owner/repo",
			want: "https://ghproxy.net/https://github.com/owner/repo",
		},
		{
			name: "non github url",
			url:  "https://example.com/file.zip",
			want: "https://example.com/file.zip",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := applyGitHubProxy(tc.url)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
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
