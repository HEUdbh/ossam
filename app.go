package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type AppsConfig struct {
	MarketName  string               `json:"market_name"`
	LastUpdated string               `json:"last_updated"`
	Apps        map[string][]AppInfo `json:"apps"`
}

type AppInfo struct {
	Name  string `json:"name"`
	Repo  string `json:"repo"`
	Photo string `json:"photo"`
	Match string `json:"match"`
}

const (
	appsConfigPath        = "appsconfig.json"
	defaultAppPlaceholder = "https://github.githubassets.com/favicons/favicon.png"
)

// GetAppsConfig loads app catalog data from the local appsconfig.json file.
func (a *App) GetAppsConfig() (*AppsConfig, error) {
	return loadAppsConfigFromFile(appsConfigPath)
}

func loadAppsConfigFromFile(path string) (*AppsConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg AppsConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON format in appsconfig.json: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	cfg.applyDisplayDefaults()

	return &cfg, nil
}

func (c *AppsConfig) validate() error {
	if strings.TrimSpace(c.MarketName) == "" {
		return errors.New("invalid config fields: market_name is required")
	}

	if strings.TrimSpace(c.LastUpdated) == "" {
		return errors.New("invalid config fields: last_updated is required")
	}

	if c.Apps == nil {
		return errors.New("invalid config fields: apps is required")
	}

	for category, apps := range c.Apps {
		if strings.TrimSpace(category) == "" {
			return errors.New("invalid config fields: category name is required")
		}

		for idx, app := range apps {
			if strings.TrimSpace(app.Name) == "" {
				return fmt.Errorf("invalid config fields: apps[%s][%d].name is required", category, idx)
			}
			if strings.TrimSpace(app.Repo) == "" {
				return fmt.Errorf("invalid config fields: apps[%s][%d].repo is required", category, idx)
			}
			if strings.TrimSpace(app.Match) == "" {
				return fmt.Errorf("invalid config fields: apps[%s][%d].match is required", category, idx)
			}
		}
	}

	return nil
}

func (c *AppsConfig) applyDisplayDefaults() {
	for category, apps := range c.Apps {
		for idx := range apps {
			apps[idx].Photo = resolveAppPhoto(apps[idx].Repo, apps[idx].Photo)
		}
		c.Apps[category] = apps
	}
}

func resolveAppPhoto(repo, photo string) string {
	if strings.TrimSpace(photo) != "" {
		return photo
	}

	parts := strings.Split(strings.TrimSpace(repo), "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return fmt.Sprintf("https://github.com/%s.png", parts[0])
	}

	return defaultAppPlaceholder
}

// SelectDownloadDirectory opens a system directory picker for selecting download location.
func (a *App) SelectDownloadDirectory(defaultDir string) (string, error) {
	selected, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "选择下载目录",
		DefaultDirectory: resolveDialogDefaultDirectory(defaultDir),
	})
	if err != nil {
		return "", fmt.Errorf("failed to open download directory dialog: %w", err)
	}

	return strings.TrimSpace(selected), nil
}

func resolveDialogDefaultDirectory(path string) string {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return ""
	}

	info, err := os.Stat(cleanPath)
	if err != nil || !info.IsDir() {
		return ""
	}

	return cleanPath
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
