package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx               context.Context
	apiClient         *http.Client
	downloadClient    *http.Client
	githubAPIBaseURL  string
	releaseAPIBaseURL string
	cdnSettings       *cdnSettingsState
	downloadTasks     map[string]*downloadTaskState
	downloadTasksLock sync.RWMutex
	repoStarsLock     sync.Mutex
	taskCounter       uint64
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		apiClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		downloadClient:    &http.Client{},
		githubAPIBaseURL:  "https://api.github.com",
		releaseAPIBaseURL: "https://ossam.hqs.qzz.io",
		cdnSettings:       globalCDNSettings,
		downloadTasks:     make(map[string]*downloadTaskState),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func newCDNSettingsState() *cdnSettingsState {
	return &cdnSettingsState{
		settings: CDNSettings{
			Enabled:        true,
			SelectedSource: defaultCDNSource,
			BuiltinSources: cloneStringSlice(builtinCDNSources),
			CustomSources:  []string{},
		},
	}
}

func (s *cdnSettingsState) getSnapshot() CDNSettings {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return CDNSettings{
		Enabled:        s.settings.Enabled,
		SelectedSource: s.settings.SelectedSource,
		BuiltinSources: cloneStringSlice(s.settings.BuiltinSources),
		CustomSources:  cloneStringSlice(s.settings.CustomSources),
	}
}

func (s *cdnSettingsState) update(request SetCDNSettingsRequest) (CDNSettings, error) {
	builtin := cloneStringSlice(builtinCDNSources)
	normalizedCustom, err := normalizeCustomCDNSources(request.CustomSources, builtin)
	if err != nil {
		return CDNSettings{}, err
	}

	candidates := append(cloneStringSlice(builtin), normalizedCustom...)
	if len(candidates) == 0 {
		return CDNSettings{}, errors.New("invalid CDN sources: at least one source is required")
	}

	selected := strings.TrimSpace(request.SelectedSource)
	if selected != "" {
		selected, err = normalizeCDNSourceURL(selected)
		if err != nil {
			return CDNSettings{}, fmt.Errorf("invalid selected_source: %w", err)
		}
	}
	if selected == "" || !containsString(candidates, selected) {
		selected = candidates[0]
	}

	next := CDNSettings{
		Enabled:        request.Enabled,
		SelectedSource: selected,
		BuiltinSources: builtin,
		CustomSources:  normalizedCustom,
	}

	s.lock.Lock()
	s.settings = next
	s.lock.Unlock()

	return next, nil
}

func normalizeCustomCDNSources(sources []string, builtin []string) ([]string, error) {
	builtinSet := make(map[string]struct{}, len(builtin))
	for _, item := range builtin {
		builtinSet[item] = struct{}{}
	}

	normalized := make([]string, 0, len(sources))
	seen := make(map[string]struct{}, len(sources))
	for idx, source := range sources {
		normalizedSource, err := normalizeCDNSourceURL(source)
		if err != nil {
			return nil, fmt.Errorf("custom_sources[%d]: %w", idx, err)
		}
		if _, isBuiltin := builtinSet[normalizedSource]; isBuiltin {
			return nil, fmt.Errorf("custom_sources[%d]: built-in source %q cannot be modified or removed", idx, normalizedSource)
		}
		if _, exists := seen[normalizedSource]; exists {
			continue
		}
		seen[normalizedSource] = struct{}{}
		normalized = append(normalized, normalizedSource)
	}

	return normalized, nil
}

func normalizeCDNSourceURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("source URL is required")
	}

	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL format")
	}
	if strings.ToLower(parsed.Scheme) != "https" {
		return "", errors.New("only https URLs are supported")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", errors.New("host is required")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("query and fragment are not allowed")
	}

	normalized := strings.TrimRight(parsed.String(), "/") + "/"
	return normalized, nil
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type AppsConfig struct {
	MarketName  string               `json:"market_name"`
	LastUpdated string               `json:"last_updated"`
	Apps        map[string][]AppInfo `json:"apps"`
}

type AppInfo struct {
	Name    string `json:"name"`
	Repo    string `json:"repo"`
	Photo   string `json:"photo"`
	Match   string `json:"match"`
	Summary string `json:"summary"`
}

type rulesConfig struct {
	MarketName       string              `json:"market_name"`
	LastUpdated      string              `json:"last_updated"`
	DefaultMatch     string              `json:"default_match"`
	PlatformKeywords map[string][]string `json:"platform_keywords"`
	PlatformMatch    map[string]string   `json:"platform_match"`
	Categories       []categoryRule      `json:"categories"`
}

type categoryRule struct {
	Name string `json:"name"`
	File string `json:"file"`
}

type categoryAppsConfig struct {
	Apps []AppInfo `json:"apps"`
}

type PlatformDownload struct {
	Platform    string `json:"platform"`
	Available   bool   `json:"available"`
	AssetName   string `json:"asset_name"`
	DownloadURL string `json:"download_url"`
	Arch        string `json:"arch"`
}

type AppReleaseDetail struct {
	Repo               string                      `json:"repo"`
	Match              string                      `json:"match"`
	ReleaseTag         string                      `json:"release_tag"`
	ReleaseName        string                      `json:"release_name"`
	ReleaseBody        string                      `json:"release_body"`
	ReleasePublishedAt string                      `json:"release_published_at"`
	Readme             string                      `json:"readme"`
	ReadmeSourceURL    string                      `json:"readme_source_url"`
	ReadmeBranch       string                      `json:"readme_branch"`
	ReadmeFilePath     string                      `json:"readme_file_path"`
	Downloads          map[string]PlatformDownload `json:"downloads"`
}

type StartDownloadRequest struct {
	DownloadURL string `json:"download_url"`
	FileName    string `json:"file_name"`
	Platform    string `json:"platform"`
	DownloadDir string `json:"download_dir"`
}

type DownloadTaskSnapshot struct {
	TaskID          string `json:"task_id"`
	Status          string `json:"status"`
	Progress        int    `json:"progress"`
	DownloadedBytes int64  `json:"downloaded_bytes"`
	TotalBytes      int64  `json:"total_bytes"`
	FilePath        string `json:"file_path"`
	Error           string `json:"error"`
	DownloadURL     string `json:"download_url"`
	FileName        string `json:"file_name"`
	Platform        string `json:"platform"`
	StartedAt       string `json:"started_at"`
	UpdatedAt       string `json:"updated_at"`
	TempFile        string `json:"temp_file"`
	ETag            string `json:"etag"`
	AcceptRanges    string `json:"accept_ranges"`
	ResumeOffset    int64  `json:"resume_offset"`
}

type downloadTaskState struct {
	lock     sync.RWMutex
	snapshot DownloadTaskSnapshot
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	ZipballURL  string        `json:"zipball_url"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRepo struct {
	StargazersCount int `json:"stargazers_count"`
}

type repoStarsCache struct {
	Repos map[string]repoStarsCacheEntry `json:"repos"`
}

type repoStarsCacheEntry struct {
	Stars     int    `json:"stars"`
	FetchedAt string `json:"fetched_at"`
}

type matchedAsset struct {
	asset githubAsset
	arch  string
}

type readmeCandidate struct {
	URL      string
	Branch   string
	FilePath string
}

type readmeFetchResult struct {
	Content   string
	SourceURL string
	Branch    string
	FilePath  string
}

type CDNSettings struct {
	Enabled        bool     `json:"enabled"`
	SelectedSource string   `json:"selected_source"`
	BuiltinSources []string `json:"builtin_sources"`
	CustomSources  []string `json:"custom_sources"`
}

type SetCDNSettingsRequest struct {
	Enabled        bool     `json:"enabled"`
	SelectedSource string   `json:"selected_source"`
	CustomSources  []string `json:"custom_sources"`
}

type cdnSettingsState struct {
	lock     sync.RWMutex
	settings CDNSettings
}

const (
	rulesConfigPath       = "config/rules.json"
	defaultAppPlaceholder = "https://github.githubassets.com/favicons/favicon.png"
	defaultCDNSource      = "https://ghproxy.net/"
	alternateCDNSource    = "https://ghfast.top/"

	platformWindows = "windows"
	platformLinux   = "linux"
	platformMacOS   = "macos"

	archAMD64 = "amd64"
	archARM64 = "arm64"
	arch386   = "386"

	taskStatusStarted    = "started"
	taskStatusInProgress = "in_progress"
	taskStatusFailed     = "failed"
	taskStatusCompleted  = "completed"
	downloadErrorBodyMax = 2048

	repoStarsWorkers  = 8
	repoStarsCacheEnv = "OSSAM_STARS_CACHE_PATH"
)

var supportedPlatforms = []string{platformWindows, platformLinux, platformMacOS}
var builtinCDNSources = []string{defaultCDNSource, alternateCDNSource}
var globalCDNSettings = newCDNSettingsState()

var archMatchers = []struct {
	arch     string
	patterns []*regexp.Regexp
}{
	{
		arch: archARM64,
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(^|[^a-z0-9])(arm64|aarch64)([^a-z0-9]|$)`),
		},
	},
	{
		arch: archAMD64,
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(^|[^a-z0-9])(amd64|x86_64|x64)([^a-z0-9]|$)`),
			regexp.MustCompile(`(^|[^a-z0-9])(64bit|64-bit)([^a-z0-9]|$)`),
		},
	},
	{
		arch: arch386,
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(^|[^a-z0-9])(386|i386|i686|x86|ia32)([^a-z0-9]|$)`),
			regexp.MustCompile(`(^|[^a-z0-9])(32bit|32-bit)([^a-z0-9]|$)`),
		},
	},
}

// GetAppsConfig loads app catalog data from config/rules.json and category files.
func (a *App) GetAppsConfig() (*AppsConfig, error) {
	return loadAppsConfigFromFiles(rulesConfigPath)
}

// GetCDNSettings returns current CDN acceleration settings.
func (a *App) GetCDNSettings() (*CDNSettings, error) {
	if a.cdnSettings == nil {
		a.cdnSettings = globalCDNSettings
	}
	settings := a.cdnSettings.getSnapshot()
	return &settings, nil
}

// SetCDNSettings updates CDN acceleration settings.
func (a *App) SetCDNSettings(request SetCDNSettingsRequest) (*CDNSettings, error) {
	if a.cdnSettings == nil {
		a.cdnSettings = globalCDNSettings
	}
	settings, err := a.cdnSettings.update(request)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// GetRepoStars fetches stargazer counts for repos with cache fallback.
func (a *App) GetRepoStars(repos []string) (map[string]int, error) {
	normalizedRepos := normalizeUniqueRepos(repos)
	if len(normalizedRepos) == 0 {
		return map[string]int{}, nil
	}

	cachePath, err := resolveRepoStarsCachePath()
	if err != nil {
		return nil, err
	}

	a.repoStarsLock.Lock()
	defer a.repoStarsLock.Unlock()

	cacheData, _ := readRepoStarsCache(cachePath)
	if cacheData.Repos == nil {
		cacheData.Repos = make(map[string]repoStarsCacheEntry)
	}

	results := make(map[string]int, len(normalizedRepos))
	var resultsLock sync.Mutex

	workerCount := repoStarsWorkers
	if len(normalizedRepos) < workerCount {
		workerCount = len(normalizedRepos)
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	jobs := make(chan string)
	var workers sync.WaitGroup
	now := time.Now().UTC().Format(time.RFC3339)

	for idx := 0; idx < workerCount; idx++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for repo := range jobs {
				stars, fetchErr := a.fetchRepoStars(repo)
				resultsLock.Lock()
				if fetchErr == nil {
					results[repo] = stars
					cacheData.Repos[repo] = repoStarsCacheEntry{
						Stars:     stars,
						FetchedAt: now,
					}
				} else if entry, exists := cacheData.Repos[repo]; exists {
					results[repo] = entry.Stars
				}
				resultsLock.Unlock()
			}
		}()
	}

	for _, repo := range normalizedRepos {
		jobs <- repo
	}
	close(jobs)
	workers.Wait()

	if writeErr := writeRepoStarsCache(cachePath, cacheData); writeErr != nil {
		// Cache write failures should not block stars display.
	}

	return results, nil
}

func loadAppsConfigFromFiles(path string) (*AppsConfig, error) {
	rules, err := loadRulesConfigFromFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &AppsConfig{
		MarketName:  rules.MarketName,
		LastUpdated: rules.LastUpdated,
		Apps:        make(map[string][]AppInfo, len(rules.Categories)),
	}

	for _, category := range rules.Categories {
		categoryFilePath := resolveCategoryFilePath(path, category.File)
		apps, err := loadCategoryAppsFromFile(categoryFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load category %s: %w", category.Name, err)
		}

		resolvedApps := make([]AppInfo, len(apps))
		for idx, app := range apps {
			name := strings.TrimSpace(app.Name)
			repo := strings.TrimSpace(app.Repo)
			summary := strings.TrimSpace(app.Summary)
			if name == "" {
				return nil, fmt.Errorf("invalid config fields: %s.apps[%d].name is required", category.Name, idx)
			}
			if repo == "" {
				return nil, fmt.Errorf("invalid config fields: %s.apps[%d].repo is required", category.Name, idx)
			}
			if summary == "" {
				return nil, fmt.Errorf("invalid config fields: %s.apps[%d].summary is required", category.Name, idx)
			}

			match := strings.TrimSpace(app.Match)
			if match == "" {
				match = rules.DefaultMatch
			}
			match = normalizeMatchPattern(match)
			if _, err := compileCaseInsensitiveRegex(match); err != nil {
				return nil, fmt.Errorf("invalid config fields: %s.apps[%d].match: %w", category.Name, idx, err)
			}

			resolvedApps[idx] = AppInfo{
				Name:    name,
				Repo:    repo,
				Photo:   strings.TrimSpace(app.Photo),
				Match:   match,
				Summary: summary,
			}
		}

		cfg.Apps[category.Name] = resolvedApps
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	cfg.applyDisplayDefaults()

	return cfg, nil
}

func loadRulesConfigFromFile(path string) (*rulesConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("rules config file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read rules config file: %w", err)
	}

	var cfg rulesConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON format in rules config: %w", err)
	}

	if strings.TrimSpace(cfg.MarketName) == "" {
		return nil, errors.New("invalid rules config fields: market_name is required")
	}
	if strings.TrimSpace(cfg.LastUpdated) == "" {
		return nil, errors.New("invalid rules config fields: last_updated is required")
	}
	cfg.DefaultMatch = normalizeMatchPattern(cfg.DefaultMatch)
	if cfg.DefaultMatch == "" {
		return nil, errors.New("invalid rules config fields: default_match is required")
	}
	if _, err := compileCaseInsensitiveRegex(cfg.DefaultMatch); err != nil {
		return nil, fmt.Errorf("invalid rules config fields: default_match: %w", err)
	}
	resolvedKeywords, err := resolvePlatformKeywords(cfg.PlatformKeywords, cfg.PlatformMatch)
	if err != nil {
		return nil, fmt.Errorf("invalid rules config fields: platform keywords: %w", err)
	}
	cfg.PlatformKeywords = resolvedKeywords
	if len(cfg.Categories) == 0 {
		return nil, errors.New("invalid rules config fields: categories is required")
	}

	seenCategories := make(map[string]struct{}, len(cfg.Categories))
	for idx := range cfg.Categories {
		cfg.Categories[idx].Name = strings.TrimSpace(cfg.Categories[idx].Name)
		cfg.Categories[idx].File = strings.TrimSpace(cfg.Categories[idx].File)
		if cfg.Categories[idx].Name == "" {
			return nil, fmt.Errorf("invalid rules config fields: categories[%d].name is required", idx)
		}
		if cfg.Categories[idx].File == "" {
			return nil, fmt.Errorf("invalid rules config fields: categories[%d].file is required", idx)
		}
		if _, exists := seenCategories[cfg.Categories[idx].Name]; exists {
			return nil, fmt.Errorf("invalid rules config fields: duplicate category name %q", cfg.Categories[idx].Name)
		}
		seenCategories[cfg.Categories[idx].Name] = struct{}{}
	}

	return &cfg, nil
}

func loadCategoryAppsFromFile(path string) ([]AppInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("category file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read category file: %w", err)
	}

	var categoryCfg categoryAppsConfig
	if err := json.Unmarshal(content, &categoryCfg); err != nil {
		return nil, fmt.Errorf("invalid JSON format in category file: %w", err)
	}
	if categoryCfg.Apps == nil {
		return nil, errors.New("invalid category config fields: apps is required")
	}

	return categoryCfg.Apps, nil
}

func resolveCategoryFilePath(rulesPath, categoryPath string) string {
	trimmed := strings.TrimSpace(categoryPath)
	if trimmed == "" {
		return ""
	}
	if filepath.IsAbs(trimmed) {
		return trimmed
	}

	cleaned := filepath.Clean(trimmed)
	rulesDir := filepath.Dir(rulesPath)
	relativeCandidate := filepath.Join(rulesDir, cleaned)
	if _, err := os.Stat(relativeCandidate); err == nil {
		return relativeCandidate
	}

	return cleaned
}

func resolvePlatformKeywords(platformKeywords map[string][]string, platformMatch map[string]string) (map[string][]string, error) {
	resolved := make(map[string][]string, len(supportedPlatforms))

	for _, platform := range supportedPlatforms {
		rawKeywords := platformKeywords[platform]
		if len(rawKeywords) == 0 {
			rawKeywords = extractKeywordsFromPlatformMatch(platformMatch[platform])
		}
		normalized := normalizeKeywords(rawKeywords)
		if len(normalized) == 0 {
			return nil, fmt.Errorf("%s is required", platform)
		}
		resolved[platform] = normalized
	}

	return resolved, nil
}

func extractKeywordsFromPlatformMatch(pattern string) []string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}

	trimmed := strings.TrimPrefix(pattern, "(?:")
	if trimmed == pattern {
		trimmed = strings.TrimPrefix(pattern, "(")
	}
	trimmed = strings.TrimSuffix(trimmed, ")")
	trimmed = strings.TrimSpace(trimmed)

	parts := strings.Split(trimmed, "|")
	keywords := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		sanitized := make([]rune, 0, len(part))
		for _, r := range strings.ToLower(part) {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				sanitized = append(sanitized, r)
			}
		}
		if len(sanitized) > 0 {
			keywords = append(keywords, string(sanitized))
		}
	}

	return keywords
}

func normalizeKeywords(keywords []string) []string {
	if len(keywords) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(keywords))
	seen := make(map[string]struct{}, len(keywords))
	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword == "" {
			continue
		}

		sanitized := make([]rune, 0, len(keyword))
		for _, r := range keyword {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				sanitized = append(sanitized, r)
			}
		}
		if len(sanitized) == 0 {
			continue
		}
		normalizedKeyword := string(sanitized)
		if _, exists := seen[normalizedKeyword]; exists {
			continue
		}
		seen[normalizedKeyword] = struct{}{}
		normalized = append(normalized, normalizedKeyword)
	}

	return normalized
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
			if strings.TrimSpace(app.Summary) == "" {
				return fmt.Errorf("invalid config fields: apps[%s][%d].summary is required", category, idx)
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
		return strings.TrimSpace(photo)
	}

	parts := strings.Split(strings.TrimSpace(repo), "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return buildGitHubAvatarURL(parts[0])
	}

	return defaultAppPlaceholder
}

// GetAppReleaseDetail fetches latest release metadata, README markdown and matched assets for each platform.
func (a *App) GetAppReleaseDetail(repo, match string) (*AppReleaseDetail, error) {
	repo = strings.TrimSpace(repo)
	if !isValidRepo(repo) {
		return nil, errors.New("invalid repo format: expected owner/repo")
	}

	match = strings.TrimSpace(match)
	if match == "" {
		return nil, errors.New("invalid match: regular expression is required")
	}
	match = normalizeMatchPattern(match)

	pattern, err := compileCaseInsensitiveRegex(match)
	if err != nil {
		return nil, fmt.Errorf("invalid match regular expression: %w", err)
	}
	rules, err := loadRulesConfigFromFile(rulesConfigPath)
	if err != nil {
		return nil, err
	}
	platformKeywords, err := resolvePlatformKeywords(rules.PlatformKeywords, rules.PlatformMatch)
	if err != nil {
		return nil, fmt.Errorf("invalid platform rules: %w", err)
	}

	releases, err := a.fetchReleases(repo)
	if err != nil {
		return nil, err
	}

	release, err := selectLatestRelease(releases)
	if err != nil {
		return nil, err
	}

	readmeResult, readmeErr := a.fetchReadme(repo)
	if readmeErr != nil {
		// README failure should not block download capabilities.
		readmeResult = readmeFetchResult{}
	}

	detail := &AppReleaseDetail{
		Repo:            repo,
		Match:           match,
		ReleaseTag:      release.TagName,
		ReleaseName:     release.Name,
		ReleaseBody:     release.Body,
		Readme:          readmeResult.Content,
		ReadmeSourceURL: readmeResult.SourceURL,
		ReadmeBranch:    readmeResult.Branch,
		ReadmeFilePath:  readmeResult.FilePath,
		Downloads:       buildPlatformDownloads(release.Assets, resolveSourceCodeZipURL(repo, release), pattern, normalizeArch(goruntime.GOARCH), platformKeywords),
	}

	if !release.PublishedAt.IsZero() {
		detail.ReleasePublishedAt = release.PublishedAt.UTC().Format(time.RFC3339)
	}

	return detail, nil
}

// StartDownload starts a download task asynchronously and returns the task snapshot immediately.
func (a *App) StartDownload(request StartDownloadRequest) (*DownloadTaskSnapshot, error) {
	originalDownloadURL := strings.TrimSpace(request.DownloadURL)
	downloadURL := applyGitHubProxy(originalDownloadURL)
	if downloadURL == "" {
		return nil, errors.New("download_url is required")
	}

	parsedURL, err := url.ParseRequestURI(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("invalid download_url: %w", err)
	}

	downloadDir := strings.TrimSpace(request.DownloadDir)
	if downloadDir == "" {
		return nil, errors.New("download_dir is required")
	}

	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create download_dir: %w", err)
	}

	fileName := sanitizeFileName(request.FileName)
	if fileName == "" {
		fileName = sanitizeFileName(path.Base(parsedURL.Path))
	}
	if fileName == "" {
		fileName = "download.bin"
	}

	targetPath := makeUniqueFilePath(downloadDir, fileName)
	taskID := a.nextTaskID()
	now := time.Now().UTC().Format(time.RFC3339)

	task := DownloadTaskSnapshot{
		TaskID:       taskID,
		Status:       taskStatusStarted,
		Progress:     0,
		FilePath:     targetPath,
		DownloadURL:  downloadURL,
		FileName:     fileName,
		Platform:     strings.ToLower(strings.TrimSpace(request.Platform)),
		StartedAt:    now,
		UpdatedAt:    now,
		TempFile:     targetPath + ".part",
		ResumeOffset: 0,
	}

	a.setTask(taskID, task)
	log.Printf(
		"download task created task_id=%s platform=%s original_url=%q proxied_url=%q file=%q path=%q",
		taskID,
		task.Platform,
		originalDownloadURL,
		downloadURL,
		task.FileName,
		task.FilePath,
	)
	go a.runDownloadTask(taskID)

	return a.GetDownloadTask(taskID)
}

// GetDownloadTask returns current task snapshot for the given task id.
func (a *App) GetDownloadTask(taskID string) (*DownloadTaskSnapshot, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, errors.New("task_id is required")
	}

	state := a.getTask(taskID)
	if state == nil {
		return nil, fmt.Errorf("download task not found: %s", taskID)
	}

	state.lock.RLock()
	defer state.lock.RUnlock()

	snapshotCopy := state.snapshot
	return &snapshotCopy, nil
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

func (a *App) fetchReleases(repo string) ([]githubRelease, error) {
	repoPath := buildRepoPath(repo)
	if repoPath == "" {
		return nil, errors.New("invalid repo format: expected owner/repo")
	}

	endpoint := fmt.Sprintf(
		"%s/api/releases/%s?per_page=30",
		strings.TrimRight(a.releaseAPIBaseURL, "/"),
		repoPath,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := a.newReleaseProxyRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := a.apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases from proxy: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read release proxy response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("release proxy API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse release proxy response: %w", err)
	}

	return releases, nil
}

func (a *App) fetchReadme(repo string) (readmeFetchResult, error) {
	candidates := buildReadmeRawCandidates(repo)
	if len(candidates) == 0 {
		return readmeFetchResult{}, errors.New("invalid repo format: expected owner/repo")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	for _, candidate := range candidates {
		requestURL := applyGitHubProxy(candidate.URL)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return readmeFetchResult{}, err
		}
		req.Header.Set("User-Agent", "ossam-app")

		resp, err := a.apiClient.Do(req)
		if err != nil {
			return readmeFetchResult{}, fmt.Errorf("failed to fetch readme: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return readmeFetchResult{}, fmt.Errorf("failed to read readme response: %w", readErr)
		}

		if resp.StatusCode == http.StatusNotFound {
			continue
		}

		if resp.StatusCode >= http.StatusBadRequest {
			return readmeFetchResult{}, fmt.Errorf("github raw readme error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		return readmeFetchResult{
			Content:   string(body),
			SourceURL: requestURL,
			Branch:    candidate.Branch,
			FilePath:  candidate.FilePath,
		}, nil
	}

	return readmeFetchResult{}, nil
}

func (a *App) fetchRepoStars(repo string) (int, error) {
	endpoint := fmt.Sprintf(
		"%s/repos/%s",
		strings.TrimRight(a.githubAPIBaseURL, "/"),
		buildRepoPath(repo),
	)
	endpoint = applyGitHubProxy(endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := a.newGitHubRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return 0, err
	}

	resp, err := a.apiClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return 0, fmt.Errorf("github repo API error (%d)", resp.StatusCode)
		}
		return 0, fmt.Errorf("github repo API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var repoInfo githubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return 0, err
	}

	return repoInfo.StargazersCount, nil
}

func (a *App) newGitHubRequest(parent context.Context, method, endpoint string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(parent, method, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ossam-app")

	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req, nil
}

func (a *App) newReleaseProxyRequest(parent context.Context, endpoint string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(parent, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ossam-app")

	return req, nil
}

func normalizeUniqueRepos(repos []string) []string {
	unique := make(map[string]struct{}, len(repos))
	result := make([]string, 0, len(repos))
	for _, repo := range repos {
		normalized := strings.TrimSpace(repo)
		if !isValidRepo(normalized) {
			continue
		}
		if _, exists := unique[normalized]; exists {
			continue
		}
		unique[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func resolveRepoStarsCachePath() (string, error) {
	override := strings.TrimSpace(os.Getenv(repoStarsCacheEnv))
	if override != "" {
		return override, nil
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user cache directory: %w", err)
	}

	return filepath.Join(cacheDir, "ossam", "stars_cache.json"), nil
}

func readRepoStarsCache(path string) (repoStarsCache, error) {
	cache := repoStarsCache{Repos: make(map[string]repoStarsCacheEntry)}
	if strings.TrimSpace(path) == "" {
		return cache, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cache, nil
		}
		return cache, err
	}

	if len(content) == 0 {
		return cache, nil
	}

	if err := json.Unmarshal(content, &cache); err != nil {
		return repoStarsCache{Repos: make(map[string]repoStarsCacheEntry)}, nil
	}

	if cache.Repos == nil {
		cache.Repos = make(map[string]repoStarsCacheEntry)
	}
	return cache, nil
}

func writeRepoStarsCache(path string, cache repoStarsCache) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	if cache.Repos == nil {
		cache.Repos = make(map[string]repoStarsCacheEntry)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	content, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(path, content, 0o600)
}

func selectLatestRelease(releases []githubRelease) (githubRelease, error) {
	if len(releases) == 0 {
		return githubRelease{}, errors.New("no release found for this repository")
	}

	for _, release := range releases {
		if release.Draft {
			continue
		}
		if !release.Prerelease {
			return release, nil
		}
	}

	for _, release := range releases {
		if !release.Draft {
			return release, nil
		}
	}

	return githubRelease{}, errors.New("no usable release found for this repository")
}

func buildPlatformDownloads(assets []githubAsset, sourceZipURL string, basePattern *regexp.Regexp, localArch string, platformKeywords map[string][]string) map[string]PlatformDownload {
	downloads := map[string]PlatformDownload{
		platformWindows: {
			Platform:  platformWindows,
			Available: false,
		},
		platformLinux: {
			Platform:  platformLinux,
			Available: false,
		},
		platformMacOS: {
			Platform:  platformMacOS,
			Available: false,
		},
	}

	for _, platform := range supportedPlatforms {
		selected := selectAssetForPlatform(assets, sourceZipURL, basePattern, platform, localArch, platformKeywords)
		downloads[platform] = selected
	}

	return downloads
}

func selectAssetForPlatform(assets []githubAsset, sourceZipURL string, basePattern *regexp.Regexp, platform, localArch string, platformKeywords map[string][]string) PlatformDownload {
	matched := make([]matchedAsset, 0)
	for _, asset := range assets {
		if basePattern == nil || !basePattern.MatchString(asset.Name) {
			continue
		}
		if !matchesPlatform(asset.Name, platform, platformKeywords) {
			continue
		}

		matched = append(matched, matchedAsset{
			asset: asset,
			arch:  detectAssetArch(asset.Name),
		})
	}

	if len(matched) == 0 {
		sourceZipURL = strings.TrimSpace(sourceZipURL)
		if sourceZipURL != "" {
			return PlatformDownload{
				Platform:    platform,
				Available:   true,
				AssetName:   "source-code.zip",
				DownloadURL: applyGitHubProxy(sourceZipURL),
				Arch:        "source",
			}
		}

		return PlatformDownload{
			Platform:  platform,
			Available: false,
		}
	}

	priority := buildArchPriority(localArch)
	for _, arch := range priority {
		for _, candidate := range matched {
			if candidate.arch == arch {
				return PlatformDownload{
					Platform:    platform,
					Available:   true,
					AssetName:   candidate.asset.Name,
					DownloadURL: applyGitHubProxy(candidate.asset.BrowserDownloadURL),
					Arch:        candidate.arch,
				}
			}
		}
	}

	first := matched[0]
	return PlatformDownload{
		Platform:    platform,
		Available:   true,
		AssetName:   first.asset.Name,
		DownloadURL: applyGitHubProxy(first.asset.BrowserDownloadURL),
		Arch:        first.arch,
	}
}

func compileCaseInsensitiveRegex(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile("(?i:" + normalizeMatchPattern(pattern) + ")")
}

func normalizeMatchPattern(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	for strings.Contains(pattern, `\\`) {
		pattern = strings.ReplaceAll(pattern, `\\`, `\`)
	}
	return pattern
}

func matchesPlatform(assetName, platform string, platformKeywords map[string][]string) bool {
	matchedPlatforms := detectAssetPlatforms(assetName, platformKeywords)
	if len(matchedPlatforms) != 1 {
		return false
	}
	return matchedPlatforms[0] == platform
}

func detectAssetPlatforms(assetName string, platformKeywords map[string][]string) []string {
	if platformKeywords == nil {
		return nil
	}

	tokens := tokenizeAssetName(assetName)
	if len(tokens) == 0 {
		return nil
	}

	matched := make([]string, 0, len(supportedPlatforms))
	for _, platform := range supportedPlatforms {
		keywords := platformKeywords[platform]
		if len(keywords) == 0 {
			continue
		}

		for _, keyword := range keywords {
			if matchesKeywordInTokens(tokens, keyword) {
				matched = append(matched, platform)
				break
			}
		}
	}

	return matched
}

func tokenizeAssetName(assetName string) []string {
	if strings.TrimSpace(assetName) == "" {
		return nil
	}

	lowerName := strings.ToLower(assetName)
	normalized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return ' '
	}, lowerName)

	return strings.Fields(normalized)
}

func matchesKeywordInTokens(tokens []string, keyword string) bool {
	if len(tokens) == 0 || keyword == "" {
		return false
	}

	for _, token := range tokens {
		if token == keyword {
			return true
		}

		// For short keywords (e.g. "win", "mac"), only allow digit suffix/prefix variants like win64.
		if len(keyword) <= 3 {
			if strings.HasPrefix(token, keyword) && isDigits(token[len(keyword):]) {
				return true
			}
			if strings.HasSuffix(token, keyword) && isDigits(token[:len(token)-len(keyword)]) {
				return true
			}
			continue
		}

		if strings.HasPrefix(token, keyword) || strings.HasSuffix(token, keyword) {
			return true
		}
	}

	return false
}

func isDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func detectAssetArch(assetName string) string {
	lowerName := strings.ToLower(assetName)
	for _, matcher := range archMatchers {
		for _, pattern := range matcher.patterns {
			if pattern.MatchString(lowerName) {
				return matcher.arch
			}
		}
	}

	return ""
}

func buildArchPriority(localArch string) []string {
	localArch = normalizeArch(localArch)
	base := []string{archAMD64, archARM64, arch386}
	if localArch == "" {
		return base
	}

	priority := []string{localArch}
	for _, arch := range base {
		if arch != localArch {
			priority = append(priority, arch)
		}
	}

	return priority
}

func normalizeArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case archAMD64, "x86_64", "x64":
		return archAMD64
	case archARM64, "aarch64":
		return archARM64
	case arch386, "x86", "i386", "i686":
		return arch386
	default:
		return ""
	}
}

func isValidRepo(repo string) bool {
	parts := strings.Split(strings.TrimSpace(repo), "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

func buildRepoPath(repo string) string {
	parts := strings.Split(strings.TrimSpace(repo), "/")
	if len(parts) != 2 {
		return ""
	}
	return url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1])
}

func resolveSourceCodeZipURL(repo string, release githubRelease) string {
	repoPath := buildRepoPath(repo)
	tag := strings.TrimSpace(release.TagName)
	if repoPath != "" && tag != "" {
		return fmt.Sprintf("https://github.com/%s/archive/refs/tags/%s.zip", repoPath, url.PathEscape(tag))
	}

	return strings.TrimSpace(release.ZipballURL)
}

func applyGitHubProxy(rawURL string) string {
	settings := globalCDNSettings.getSnapshot()
	return resolveGitHubURL(rawURL, settings)
}

func resolveGitHubURL(rawURL string, settings CDNSettings) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	unwrappedURL := unwrapKnownCDNPrefixes(rawURL, settings)
	parsed, err := url.Parse(unwrappedURL)
	if err != nil || parsed.Hostname() == "" {
		return rawURL
	}
	if !isGitHubHost(parsed.Hostname()) {
		return rawURL
	}

	if !settings.Enabled {
		return unwrappedURL
	}

	selectedSource := resolveSelectedCDNSource(settings)
	if selectedSource == "" {
		return unwrappedURL
	}

	return selectedSource + unwrappedURL
}

func resolveSelectedCDNSource(settings CDNSettings) string {
	allSources := append(cloneStringSlice(settings.BuiltinSources), settings.CustomSources...)
	if len(allSources) == 0 {
		return ""
	}

	selected := strings.TrimSpace(settings.SelectedSource)
	if selected == "" {
		return allSources[0]
	}

	normalized, err := normalizeCDNSourceURL(selected)
	if err != nil {
		return allSources[0]
	}
	if containsString(allSources, normalized) {
		return normalized
	}

	return allSources[0]
}

func unwrapKnownCDNPrefixes(rawURL string, settings CDNSettings) string {
	normalized := strings.TrimSpace(rawURL)
	if normalized == "" {
		return normalized
	}

	knownSources := append(cloneStringSlice(settings.BuiltinSources), settings.CustomSources...)
	if len(knownSources) == 0 {
		return normalized
	}

	for {
		changed := false
		for _, source := range knownSources {
			source = strings.TrimSpace(source)
			if source == "" {
				continue
			}
			if strings.HasPrefix(normalized, source) {
				next := strings.TrimPrefix(normalized, source)
				if next == "" {
					continue
				}
				normalized = next
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	return normalized
}

func isGitHubHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}

	// Avatar URLs should stay direct to avoid redirect/proxy issues.
	if host == "avatars.githubusercontent.com" || host == "github.githubassets.com" {
		return false
	}

	switch host {
	case "github.com", "api.github.com", "raw.githubusercontent.com", "codeload.github.com", "objects.githubusercontent.com", "githubusercontent.com":
		return true
	}

	return strings.HasSuffix(host, ".github.com") || strings.HasSuffix(host, ".githubusercontent.com")
}

func buildReadmeRawCandidates(repo string) []readmeCandidate {
	parts := strings.Split(strings.TrimSpace(repo), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil
	}

	owner := url.PathEscape(parts[0])
	repoName := url.PathEscape(parts[1])
	base := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s", owner, repoName)
	branches := []string{"main", "master"}
	filePaths := []string{"readme.md", "README.md"}

	candidates := make([]readmeCandidate, 0, len(branches)*len(filePaths))
	for _, branch := range branches {
		for _, filePath := range filePaths {
			rawURL := fmt.Sprintf("%s/%s/%s", base, branch, filePath)
			candidates = append(candidates, readmeCandidate{
				URL:      rawURL,
				Branch:   branch,
				FilePath: filePath,
			})
		}
	}

	return candidates
}

func buildGitHubAvatarURL(owner string) string {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return defaultAppPlaceholder
	}
	return "https://avatars.githubusercontent.com/" + url.PathEscape(owner)
}

func (a *App) nextTaskID() string {
	sequence := atomic.AddUint64(&a.taskCounter, 1)
	return "task-" + strconv.FormatInt(time.Now().UnixMilli(), 10) + "-" + strconv.FormatUint(sequence, 10)
}

func (a *App) setTask(taskID string, snapshot DownloadTaskSnapshot) {
	a.downloadTasksLock.Lock()
	defer a.downloadTasksLock.Unlock()

	a.downloadTasks[taskID] = &downloadTaskState{
		snapshot: snapshot,
	}
}

func (a *App) getTask(taskID string) *downloadTaskState {
	a.downloadTasksLock.RLock()
	defer a.downloadTasksLock.RUnlock()
	return a.downloadTasks[taskID]
}

func (a *App) runDownloadTask(taskID string) {
	state := a.getTask(taskID)
	if state == nil {
		return
	}

	initialSnapshot := a.readTaskSnapshot(state)
	a.updateTask(taskID, func(snapshot *DownloadTaskSnapshot) {
		snapshot.Status = taskStatusInProgress
	})

	if err := os.MkdirAll(filepath.Dir(initialSnapshot.FilePath), 0o755); err != nil {
		a.failTask(taskID, fmt.Errorf("failed to prepare download directory: %w", err))
		return
	}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, initialSnapshot.DownloadURL, nil)
	if err != nil {
		a.failTask(taskID, fmt.Errorf("failed to create request: %w", err))
		return
	}
	request.Header.Set("User-Agent", "ossam-app")
	log.Printf("download request start task_id=%s method=%s url=%q", taskID, request.Method, request.URL.String())

	resp, err := a.downloadClient.Do(request)
	if err != nil {
		a.failTask(taskID, fmt.Errorf("failed to start download: %w", err))
		return
	}
	defer resp.Body.Close()
	log.Printf(
		"download response task_id=%s status=%d content_type=%q content_length=%q location=%q server=%q",
		taskID,
		resp.StatusCode,
		resp.Header.Get("Content-Type"),
		resp.Header.Get("Content-Length"),
		resp.Header.Get("Location"),
		resp.Header.Get("Server"),
	)

	if resp.StatusCode >= http.StatusBadRequest {
		snippet, snippetErr := readResponseBodySnippet(resp.Body, downloadErrorBodyMax)
		if snippetErr != nil {
			log.Printf("download response body read failed task_id=%s err=%v", taskID, snippetErr)
		}
		a.failTask(taskID, fmt.Errorf("download request failed: status=%d url=%s body=%s", resp.StatusCode, request.URL.String(), snippet))
		return
	}

	tempFile, err := os.Create(initialSnapshot.TempFile)
	if err != nil {
		a.failTask(taskID, fmt.Errorf("failed to create temp file: %w", err))
		return
	}
	tempFileClosed := false
	defer func() {
		if !tempFileClosed {
			_ = tempFile.Close()
		}
	}()

	totalBytes := resp.ContentLength
	a.updateTask(taskID, func(snapshot *DownloadTaskSnapshot) {
		snapshot.TotalBytes = totalBytes
		snapshot.ETag = resp.Header.Get("ETag")
		snapshot.AcceptRanges = resp.Header.Get("Accept-Ranges")
	})

	buffer := make([]byte, 32*1024)
	var downloadedBytes int64
	lastLoggedProgressBucket := -1

	for {
		readSize, readErr := resp.Body.Read(buffer)
		if readSize > 0 {
			written, writeErr := tempFile.Write(buffer[:readSize])
			if writeErr != nil {
				a.failTask(taskID, fmt.Errorf("failed to write download file: %w", writeErr))
				return
			}
			if written != readSize {
				a.failTask(taskID, errors.New("failed to write complete download chunk"))
				return
			}

			downloadedBytes += int64(written)
			progress := 0
			if totalBytes > 0 {
				progress = int((downloadedBytes * 100) / totalBytes)
				if progress > 99 {
					progress = 99
				}
				progressBucket := progress / 10
				if progressBucket > lastLoggedProgressBucket {
					lastLoggedProgressBucket = progressBucket
					log.Printf(
						"download progress task_id=%s progress=%d downloaded=%d total=%d",
						taskID,
						progress,
						downloadedBytes,
						totalBytes,
					)
				}
			}

			a.updateTask(taskID, func(snapshot *DownloadTaskSnapshot) {
				snapshot.DownloadedBytes = downloadedBytes
				snapshot.ResumeOffset = downloadedBytes
				snapshot.Progress = progress
			})
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			a.failTask(taskID, fmt.Errorf("download interrupted: %w", readErr))
			return
		}
	}

	if err := tempFile.Sync(); err != nil {
		a.failTask(taskID, fmt.Errorf("failed to flush download file: %w", err))
		return
	}
	if err := tempFile.Close(); err != nil {
		a.failTask(taskID, fmt.Errorf("failed to close temp file: %w", err))
		return
	}
	tempFileClosed = true

	if err := os.Rename(initialSnapshot.TempFile, initialSnapshot.FilePath); err != nil {
		a.failTask(taskID, fmt.Errorf("failed to finalize download file: %w", err))
		return
	}

	a.updateTask(taskID, func(snapshot *DownloadTaskSnapshot) {
		snapshot.Status = taskStatusCompleted
		snapshot.Progress = 100
		snapshot.Error = ""
		snapshot.DownloadedBytes = downloadedBytes
		if snapshot.TotalBytes <= 0 {
			snapshot.TotalBytes = downloadedBytes
		}
	})
	log.Printf(
		"download completed task_id=%s file=%q path=%q downloaded=%d",
		taskID,
		initialSnapshot.FileName,
		initialSnapshot.FilePath,
		downloadedBytes,
	)
}

func (a *App) failTask(taskID string, err error) {
	state := a.getTask(taskID)
	if state != nil {
		snapshot := a.readTaskSnapshot(state)
		log.Printf(
			"download failed task_id=%s platform=%s file=%q url=%q err=%v",
			taskID,
			snapshot.Platform,
			snapshot.FileName,
			snapshot.DownloadURL,
			err,
		)
	} else {
		log.Printf("download failed task_id=%s err=%v", taskID, err)
	}

	a.updateTask(taskID, func(snapshot *DownloadTaskSnapshot) {
		snapshot.Status = taskStatusFailed
		snapshot.Error = err.Error()
	})
}

func (a *App) readTaskSnapshot(state *downloadTaskState) DownloadTaskSnapshot {
	state.lock.RLock()
	defer state.lock.RUnlock()
	return state.snapshot
}

func (a *App) updateTask(taskID string, updateFn func(snapshot *DownloadTaskSnapshot)) {
	state := a.getTask(taskID)
	if state == nil {
		return
	}

	state.lock.Lock()
	defer state.lock.Unlock()

	updateFn(&state.snapshot)
	state.snapshot.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
}

func sanitizeFileName(fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return ""
	}

	fileName = strings.ReplaceAll(fileName, "\\", "/")
	fileName = filepath.Base(fileName)
	fileName = strings.TrimSpace(fileName)
	if fileName == "." || fileName == ".." {
		return ""
	}

	return fileName
}

func makeUniqueFilePath(directory, fileName string) string {
	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	if base == "" {
		base = "download"
	}

	target := filepath.Join(directory, base+ext)
	if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
		return target
	}

	for index := 1; ; index++ {
		candidate := filepath.Join(directory, fmt.Sprintf("%s (%d)%s", base, index, ext))
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
}

func readResponseBodySnippet(reader io.Reader, limit int64) (string, error) {
	if limit <= 0 {
		limit = downloadErrorBodyMax
	}

	content, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return "", err
	}

	truncated := int64(len(content)) > limit
	if truncated {
		content = content[:limit]
	}

	snippet := strings.TrimSpace(string(content))
	if snippet == "" {
		snippet = "<empty>"
	}
	if truncated {
		snippet += "...(truncated)"
	}

	return snippet, nil
}
