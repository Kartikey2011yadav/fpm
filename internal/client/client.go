package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kartikeyyadav/fpm/internal/config"
	fpmtls "github.com/kartikeyyadav/fpm/internal/tls"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

type RegistryClient struct {
	httpClient  *http.Client
	indexes     []config.IndexConfig
	cacheDir    string
	concurrency int
	semaphore   chan struct{}
	mu          sync.Mutex
	transport   *http.Transport
}

type ClientOptions struct {
	Indexes     []config.IndexConfig
	CacheDir    string
	Concurrency int
	Timeout     time.Duration
	Network     config.NetworkConfig
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

	transport := http.DefaultTransport.(*http.Transport).Clone()

	// TLS configuration (precedence: SSL_CERT_FILE/DIR > system > bundled fallback)
	if os.Getenv("FPM_INSECURE") == "1" || os.Getenv("FPM_INSECURE") == "true" {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		pool, _ := fpmtls.LoadCertPool()
		if pool != nil {
			transport.TLSClientConfig = &tls.Config{RootCAs: pool}
		}
	}

	// Per-host insecure bypass
	var rt http.RoundTripper = transport
	if len(opts.Network.AllowInsecureHost) > 0 {
		rt = fpmtls.NewInsecureHostTransport(transport, opts.Network.AllowInsecureHost)
	}

	return &RegistryClient{
		httpClient: &http.Client{
			Timeout:   opts.Timeout,
			Transport: rt,
		},
		indexes:     opts.Indexes,
		cacheDir:    opts.CacheDir,
		concurrency: opts.Concurrency,
		semaphore:   make(chan struct{}, opts.Concurrency),
		transport:   transport,
	}
}

// doWithRetry wraps httpClient.Do with retry logic using exponential backoff.
// It retries on transient errors: timeouts, 429, 500, 502, 503, connection refused,
// and DNS temporary errors. It does not retry on 400, 401, 403, 404, 422.
func (c *RegistryClient) doWithRetry(req *http.Request) (*http.Response, error) {
	const maxRetries = 3
	backoffs := []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(backoffs[attempt-1])
			// Reset request body if needed (GET requests have no body)
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				req.Body = body
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if isRetryableError(err) && attempt < maxRetries {
				lastErr = err
				continue
			}
			return nil, err
		}

		// Check for retryable status codes
		if isRetryableStatus(resp.StatusCode) && attempt < maxRetries {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// isRetryableError checks if a network error is transient and worth retrying.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for timeout
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Check for DNS temporary errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.Temporary() {
		return true
	}

	// Check for connection refused / reset
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "i/o timeout") {
		return true
	}

	return false
}

// isRetryableStatus returns true for HTTP status codes that indicate a transient error.
func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable:  // 503
		return true
	}
	return false
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
	var lastErr error
	for _, index := range c.indexes {
		detail, err := c.fetchFromIndex(ctx, index, name)
		if err != nil {
			lastErr = err
			continue
		}
		// Cache the result
		c.writeCache(name, detail)
		return detail, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", name.Raw(), lastErr)
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
	resp, err := c.doWithRetry(req)
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
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}

	// Check content type — PyPI may return HTML instead of JSON
	contentType := resp.Header.Get("Content-Type")

	var detail SimpleProjectDetail
	if strings.Contains(contentType, "json") {
		if err := json.Unmarshal(body, &detail); err != nil {
			return nil, fmt.Errorf("parsing JSON from %s: %w", url, err)
		}
	} else {
		// Parse HTML Simple API response (PEP 503 fallback)
		files, err := parseSimpleHTML(string(body), url)
		if err != nil {
			return nil, fmt.Errorf("parsing HTML from %s: %w", url, err)
		}
		detail.Files = files
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
	resp, err := c.doWithRetry(req)
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

// parseSimpleHTML parses PEP 503 HTML Simple API response.
// Extracts <a> tags with href pointing to wheel/sdist files.
var linkRegex = regexp.MustCompile(`<a\s+[^>]*href="([^"]+)"[^>]*>([^<]+)</a>`)
var hashRegex = regexp.MustCompile(`#sha256=([a-f0-9]+)`)
var requiresPythonRegex = regexp.MustCompile(`data-requires-python="([^"]*)"`)

func parseSimpleHTML(html, baseURL string) ([]SimpleFile, error) {
	matches := linkRegex.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no links found in HTML response")
	}

	var files []SimpleFile
	for _, match := range matches {
		href := match[1]
		filename := match[2]

		// Skip non-wheel non-tar files
		if !strings.HasSuffix(filename, ".whl") && !strings.HasSuffix(filename, ".tar.gz") {
			continue
		}

		// Resolve relative URLs
		fileURL := href
		if strings.HasPrefix(href, "/") {
			// Absolute path — prepend scheme+host from base
			parts := strings.SplitN(baseURL, "/", 4)
			if len(parts) >= 3 {
				fileURL = parts[0] + "//" + parts[2] + href
			}
		} else if !strings.HasPrefix(href, "http") {
			fileURL = strings.TrimSuffix(baseURL, "/") + "/" + href
		}

		// Extract hash from URL fragment
		hashes := make(map[string]string)
		if hashMatch := hashRegex.FindStringSubmatch(href); len(hashMatch) == 2 {
			hashes["sha256"] = hashMatch[1]
			// Remove fragment from URL
			if idx := strings.Index(fileURL, "#"); idx > 0 {
				fileURL = fileURL[:idx]
			}
		}

		// Extract requires-python from the surrounding context
		requiresPython := ""
		lineStart := strings.LastIndex(html[:strings.Index(html, href)], "<a")
		if lineStart >= 0 {
			lineEnd := strings.Index(html[lineStart:], "</a>") + lineStart
			if lineEnd > lineStart {
				segment := html[lineStart:lineEnd]
				if rpMatch := requiresPythonRegex.FindStringSubmatch(segment); len(rpMatch) == 2 {
					requiresPython = strings.ReplaceAll(rpMatch[1], "&gt;", ">")
					requiresPython = strings.ReplaceAll(requiresPython, "&lt;", "<")
					requiresPython = strings.ReplaceAll(requiresPython, "&amp;", "&")
				}
			}
		}

		files = append(files, SimpleFile{
			Filename:       filename,
			URL:            fileURL,
			Hashes:         hashes,
			RequiresPython: requiresPython,
		})
	}

	return files, nil
}
