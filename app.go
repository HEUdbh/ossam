package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
		downloadClient:   &http.Client{},
		githubAPIBaseURL: "https://api.github.com",
		downloadTasks:    make(map[string]*downloadTaskState),
	}
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
	Name    string `json:"name"`
	Repo    string `json:"repo"`
	Photo   string `json:"photo"`
	Match   string `json:"match"`
	Summary string `json:"summary"`
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

const (
	appsConfigPath        = "appsconfig.json"
	defaultAppPlaceholder = "https://github.githubassets.com/favicons/favicon.png"

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

	repoStarsWorkers  = 8
	repoStarsCacheEnv = "OSSAM_STARS_CACHE_PATH"
)

var platformKeywords = map[string][]*regexp.Regexp{
	platformWindows: {
		regexp.MustCompile(`(^|[^a-z0-9])(windows|win32|win64|win)([^a-z0-9]|$)`),
	},
	platformLinux: {
		regexp.MustCompile(`(^|[^a-z0-9])(linux|gnu|musl)([^a-z0-9]|$)`),
	},
	platformMacOS: {
		regexp.MustCompile(`(^|[^a-z0-9])(darwin|macos|osx|mac)([^a-z0-9]|$)`),
	},
}

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

// GetAppsConfig loads app catalog data from the local appsconfig.json file.
func (a *App) GetAppsConfig() (*AppsConfig, error) {
	return loadAppsConfigFromFile(appsConfigPath)
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
		return photo
	}

	parts := strings.Split(strings.TrimSpace(repo), "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return fmt.Sprintf("https://github.com/%s.png", parts[0])
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

	pattern, err := compileCaseInsensitiveRegex(match)
	if err != nil {
		return nil, fmt.Errorf("invalid match regular expression: %w", err)
	}

	releases, err := a.fetchReleases(repo)
	if err != nil {
		return nil, err
	}

	release, err := selectLatestRelease(releases)
	if err != nil {
		return nil, err
	}

	readme, readmeErr := a.fetchReadme(repo)
	if readmeErr != nil {
		// README failure should not block download capabilities.
		readme = ""
	}

	detail := &AppReleaseDetail{
		Repo:        repo,
		Match:       match,
		ReleaseTag:  release.TagName,
		ReleaseName: release.Name,
		ReleaseBody: release.Body,
		Readme:      readme,
		Downloads:   buildPlatformDownloads(release.Assets, pattern, normalizeArch(goruntime.GOARCH)),
	}

	if !release.PublishedAt.IsZero() {
		detail.ReleasePublishedAt = release.PublishedAt.UTC().Format(time.RFC3339)
	}

	return detail, nil
}

// StartDownload starts a download task asynchronously and returns the task snapshot immediately.
func (a *App) StartDownload(request StartDownloadRequest) (*DownloadTaskSnapshot, error) {
	downloadURL := strings.TrimSpace(request.DownloadURL)
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
	endpoint := fmt.Sprintf(
		"%s/repos/%s/releases?per_page=30",
		strings.TrimRight(a.githubAPIBaseURL, "/"),
		buildRepoPath(repo),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := a.newGitHubRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := a.apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read releases response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("github release API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse release response: %w", err)
	}

	return releases, nil
}

func (a *App) fetchReadme(repo string) (string, error) {
	endpoint := fmt.Sprintf(
		"%s/repos/%s/readme",
		strings.TrimRight(a.githubAPIBaseURL, "/"),
		buildRepoPath(repo),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := a.newGitHubRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.raw")

	resp, err := a.apiClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch readme: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read readme response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("github readme API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return string(body), nil
}

func (a *App) fetchRepoStars(repo string) (int, error) {
	endpoint := fmt.Sprintf(
		"%s/repos/%s",
		strings.TrimRight(a.githubAPIBaseURL, "/"),
		buildRepoPath(repo),
	)

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

func buildPlatformDownloads(assets []githubAsset, basePattern *regexp.Regexp, localArch string) map[string]PlatformDownload {
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

	for _, platform := range []string{platformWindows, platformLinux, platformMacOS} {
		selected := selectAssetForPlatform(assets, basePattern, platform, localArch)
		downloads[platform] = selected
	}

	return downloads
}

func selectAssetForPlatform(assets []githubAsset, basePattern *regexp.Regexp, platform, localArch string) PlatformDownload {
	matched := make([]matchedAsset, 0)
	for _, asset := range assets {
		if !basePattern.MatchString(asset.Name) {
			continue
		}
		if !matchesPlatform(asset.Name, platform) {
			continue
		}

		matched = append(matched, matchedAsset{
			asset: asset,
			arch:  detectAssetArch(asset.Name),
		})
	}

	if len(matched) == 0 {
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
					DownloadURL: candidate.asset.BrowserDownloadURL,
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
		DownloadURL: first.asset.BrowserDownloadURL,
		Arch:        first.arch,
	}
}

func compileCaseInsensitiveRegex(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile("(?i:" + pattern + ")")
}

func matchesPlatform(assetName, platform string) bool {
	patterns, ok := platformKeywords[platform]
	if !ok {
		return false
	}

	lowerName := strings.ToLower(assetName)
	for _, pattern := range patterns {
		if pattern.MatchString(lowerName) {
			return true
		}
	}

	return false
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

	resp, err := a.downloadClient.Do(request)
	if err != nil {
		a.failTask(taskID, fmt.Errorf("failed to start download: %w", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		a.failTask(taskID, fmt.Errorf("download request failed with status %d", resp.StatusCode))
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
}

func (a *App) failTask(taskID string, err error) {
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
