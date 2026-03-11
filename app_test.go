package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppsConfigFromFileSuccess(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `{
  "market_name": "ossam",
  "last_updated": "20260309",
  "apps": {
    "DevTools": [
      {
        "name": "fzf",
        "repo": "junegunn/fzf",
        "photo": "",
        "match": ".*\\.zip"
      },
      {
        "name": "placeholder",
        "repo": "invalidrepoformat",
        "photo": "",
        "match": ".*\\.zip"
      },
      {
        "name": "customphoto",
        "repo": "example/demo",
        "photo": "https://example.com/icon.png",
        "match": ".*\\.zip"
      }
    ]
  }
}`)

	cfg, err := loadAppsConfigFromFile(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	apps := cfg.Apps["DevTools"]
	if len(apps) != 3 {
		t.Fatalf("expected 3 apps, got: %d", len(apps))
	}

	if apps[0].Photo != "https://github.com/junegunn.png" {
		t.Fatalf("expected owner avatar fallback, got: %s", apps[0].Photo)
	}

	if apps[1].Photo != defaultAppPlaceholder {
		t.Fatalf("expected default placeholder fallback, got: %s", apps[1].Photo)
	}

	if apps[2].Photo != "https://example.com/icon.png" {
		t.Fatalf("expected explicit photo to be kept, got: %s", apps[2].Photo)
	}
}

func TestLoadAppsConfigFromFileMissingFile(t *testing.T) {
	t.Parallel()

	_, err := loadAppsConfigFromFile(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected an error for missing config file")
	}

	if !strings.Contains(err.Error(), "config file not found") {
		t.Fatalf("expected missing file error, got: %v", err)
	}
}

func TestLoadAppsConfigFromFileInvalidJSON(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `{"market_name":`)

	_, err := loadAppsConfigFromFile(path)
	if err == nil {
		t.Fatal("expected JSON parsing error")
	}

	if !strings.Contains(err.Error(), "invalid JSON format") {
		t.Fatalf("expected invalid JSON error, got: %v", err)
	}
}

func TestLoadAppsConfigFromFileInvalidFields(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `{
  "market_name": "",
  "last_updated": "20260309",
  "apps": {}
}`)

	_, err := loadAppsConfigFromFile(path)
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(err.Error(), "invalid config fields") {
		t.Fatalf("expected invalid fields error, got: %v", err)
	}
}

func TestResolveDialogDefaultDirectory(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	existingDir := filepath.Join(baseDir, "downloads")
	if err := os.Mkdir(existingDir, 0o755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	filePath := filepath.Join(baseDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty path",
			input: "",
			want:  "",
		},
		{
			name:  "existing directory",
			input: existingDir,
			want:  existingDir,
		},
		{
			name:  "missing directory",
			input: filepath.Join(baseDir, "missing"),
			want:  "",
		},
		{
			name:  "file path",
			input: filePath,
			want:  "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolveDialogDefaultDirectory(tc.input)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "appsconfig.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	return path
}
