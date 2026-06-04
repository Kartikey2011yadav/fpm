package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

type RegistryClient struct {
	httpClient  *http.Client
	indexes     []config.IndexConfig
	cacheDir    string
	concurrency int
	semaphore   chan struct{}
	mu          sync.Mutex
}

type ClientOptions struct {
	Indexes     []config.IndexConfig
	CacheDir    string
	Concurrency int
	Timeout     time.Duration
}

func New(opts ClientOptions) *RegistryClient {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 50
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.CacheDir == "" {
		opts.CacheDir = filepath.Join(config.CacheDir(), "http")
	}
	if len(opts.Indexes) == 0 {
		opts.Indexes = []config.IndexConfig{
			{Name: "pypi", URL: "https://pypi.org/simple", Default: true},
		}
	}

	return &RegistryClient{
		httpClient: &http.Client{
			Timeout: opts.Timeout,
		},
		indexes:     opts.Indexes,
		cacheDir:    opts.CacheDir,
		concurrency: opts.Concurrency,
		semaphore:   make(chan struct{}, opts.Concurrency),
	}
}

// PEP 691 JSON Simple API response types
type SimpleProjectDetail struct {
	Meta  SimpleMeta   `json:"meta"`
	Name  string       `json:"name"`
	Files []SimpleFile `json:"files"`
}

type SimpleMeta struct {
	APIVersion string `json:"api-version"`
}

type SimpleFile struct {
	Filename      string            `json:"filename"`
	URL           string            `json:"url"`
	Hashes        map[string]string `json:"hashes"`
	RequiresPython string           `json:"requires-python"`
	Yanked        interface{}       `json:"yanked"`
	CoreMetadata  interface{}       `json:"core-metadata"`
	Size          int64             `json:"size"`
	UploadTime    string            `json:"upload-time"`
}

func (f SimpleFile) IsYanked() bool {
	if f.Yanked == nil {
		return false
	}
	switch v := f.Yanked.(type) {
	case bool:
		return v
	case string:
		return v != ""
	default:
		return false
	}
}

func (f SimpleFile) SHA256() string {
	if h, ok := f.Hashes["sha256"]; ok {
		return h
	}
	return ""
}

func (c *RegistryClient) FetchPackageVersions(ctx context.Context, name types.PackageName) (*SimpleProjectDetail, error) {
	// Try cache first
	cached, err := c.readCache(name)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Fetch from registries
	for _, index := range c.indexes {
		detail, err := c.fetchFromIndex(ctx, index, name)
		if err != nil {
			continue
		}
		// Cache the result
		c.writeCache(name, detail)
		return detail, nil
	}

	return nil, fmt.Errorf("package %q not found in any index", name.Raw())
}

func (c *RegistryClient) fetchFromIndex(ctx context.Context, index config.IndexConfig, name types.PackageName) (*SimpleProjectDetail, error) {
	url := fmt.Sprintf("%s/%s/", index.URL, name.Normalized())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// PEP 691: Request JSON format
	req.Header.Set("Accept", "application/vnd.pypi.simple.v1+json, application/json;q=0.9, text/html;q=0.1")
	req.Header.Set("User-Agent", "fpm/0.1.0")

	c.semaphore <- struct{}{}
	resp, err := c.httpClient.Do(req)
	<-c.semaphore

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var detail SimpleProjectDetail
	if err := json.Unmarshal(body, &detail); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &detail, nil
}

func (c *RegistryClient) DownloadWheel(ctx context.Context, file SimpleFile, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", file.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "fpm/0.1.0")

	c.semaphore <- struct{}{}
	resp, err := c.httpClient.Do(req)
	<-c.semaphore

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d downloading %s", resp.StatusCode, file.Filename)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// HTTP cache helpers
func (c *RegistryClient) cachePathFor(name types.PackageName) string {
	return filepath.Join(c.cacheDir, name.Normalized()+".json")
}

func (c *RegistryClient) readCache(name types.PackageName) (*SimpleProjectDetail, error) {
	path := c.cachePathFor(name)
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Cache TTL: 10 minutes (matching PyPI's Cache-Control)
	if time.Since(info.ModTime()) > 10*time.Minute {
		return nil, fmt.Errorf("cache expired")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var detail SimpleProjectDetail
	return &detail, json.Unmarshal(data, &detail)
}

func (c *RegistryClient) writeCache(name types.PackageName, detail *SimpleProjectDetail) {
	path := c.cachePathFor(name)
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.Marshal(detail)
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0644)
}
