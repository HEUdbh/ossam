package main

import "testing"

func TestApplyGitHubProxy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "github repo url",
			in:   "https://github.com/owner/repo",
			want: "https://ghproxy.net/https://github.com/owner/repo",
		},
		{
			name: "github api url",
			in:   "https://api.github.com/repos/owner/repo",
			want: "https://ghproxy.net/https://api.github.com/repos/owner/repo",
		},
		{
			name: "githubusercontent url",
			in:   "https://raw.githubusercontent.com/owner/repo/main/README.md",
			want: "https://ghproxy.net/https://raw.githubusercontent.com/owner/repo/main/README.md",
		},
		{
			name: "already proxied",
			in:   "https://ghproxy.net/https://github.com/owner/repo",
			want: "https://ghproxy.net/https://github.com/owner/repo",
		},
		{
			name: "non github url",
			in:   "https://example.com/file.zip",
			want: "https://example.com/file.zip",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := applyGitHubProxy(tc.in)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

