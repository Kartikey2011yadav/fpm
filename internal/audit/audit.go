package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/env"
)

const osvAPIURL = "https://api.osv.dev/v1/query"

type Vulnerability struct {
	ID       string   `json:"id"`
	Summary  string   `json:"summary"`
	Severity string   `json:"severity,omitempty"`
	Fixed    string   `json:"fixed,omitempty"`
	Aliases  []string `json:"aliases,omitempty"`
	Link     string   `json:"link,omitempty"`
}

type AuditResult struct {
	Package         string
	Version         string
	Vulnerabilities []Vulnerability
}

type AuditReport struct {
	Results     []AuditResult
	TotalVulns  int
	ScannedPkgs int
}

func Scan(ctx context.Context, packages []env.InstalledPackage) (*AuditReport, error) {
	report := &AuditReport{ScannedPkgs: len(packages)}

	for _, pkg := range packages {
		vulns, err := queryOSV(ctx, pkg.Name.Normalized(), pkg.Version.String())
		if err != nil {
			continue
		}

		if len(vulns) > 0 {
			report.Results = append(report.Results, AuditResult{
				Package:         pkg.Name.Raw(),
				Version:         pkg.Version.String(),
				Vulnerabilities: vulns,
			})
			report.TotalVulns += len(vulns)
		}
	}

	return report, nil
}

func (r *AuditReport) Print() {
	if r.TotalVulns == 0 {
		fmt.Printf("Scanned %d packages — no vulnerabilities found.\n", r.ScannedPkgs)
		return
	}

	fmt.Printf("Found %d vulnerabilities in %d packages (scanned %d):\n\n",
		r.TotalVulns, len(r.Results), r.ScannedPkgs)

	for _, result := range r.Results {
		fmt.Printf("  %s %s\n", result.Package, result.Version)
		for _, vuln := range result.Vulnerabilities {
			severity := vuln.Severity
			if severity == "" {
				severity = "unknown"
			}
			fmt.Printf("    %s [%s] %s\n", vuln.ID, severity, vuln.Summary)
			if vuln.Fixed != "" {
				fmt.Printf("      fix: upgrade to %s\n", vuln.Fixed)
			}
			if vuln.Link != "" {
				fmt.Printf("      %s\n", vuln.Link)
			}
		}
		fmt.Println()
	}
}

// OSV API types
type osvQuery struct {
	Package osvPackage `json:"package"`
	Version string     `json:"version"`
}

type osvPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

type osvResponse struct {
	Vulns []osvVuln `json:"vulns"`
}

type osvVuln struct {
	ID       string        `json:"id"`
	Summary  string        `json:"summary"`
	Aliases  []string      `json:"aliases"`
	Severity []osvSeverity `json:"severity"`
	Affected []osvAffected `json:"affected"`
}

type osvSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

type osvAffected struct {
	Ranges []osvRange `json:"ranges"`
}

type osvRange struct {
	Type   string     `json:"type"`
	Events []osvEvent `json:"events"`
}

type osvEvent struct {
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
}

const osvCacheTTL = 1 * time.Hour

func osvCacheDir() string {
	return filepath.Join(config.CacheDir(), "osv")
}

func queryOSV(ctx context.Context, name, version string) ([]Vulnerability, error) {
	cacheFile := filepath.Join(osvCacheDir(), fmt.Sprintf("%s-%s.json", name, version))
	if data, err := os.ReadFile(cacheFile); err == nil {
		if info, _ := os.Stat(cacheFile); info != nil && time.Since(info.ModTime()) < osvCacheTTL {
			var cached []Vulnerability
			if json.Unmarshal(data, &cached) == nil {
				return cached, nil
			}
		}
	}

	vulns, err := queryOSVRemote(ctx, name, version)
	if err != nil {
		return nil, err
	}

	os.MkdirAll(osvCacheDir(), 0755)
	if data, err := json.Marshal(vulns); err == nil {
		os.WriteFile(cacheFile, data, 0644)
	}
	return vulns, nil
}

func queryOSVRemote(ctx context.Context, name, version string) ([]Vulnerability, error) {
	query := osvQuery{
		Package: osvPackage{Name: name, Ecosystem: "PyPI"},
		Version: version,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", osvAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSV API returned %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var osvResp osvResponse
	if err := json.Unmarshal(respBody, &osvResp); err != nil {
		return nil, err
	}

	var vulns []Vulnerability
	for _, v := range osvResp.Vulns {
		vuln := Vulnerability{
			ID:      v.ID,
			Summary: v.Summary,
			Aliases: v.Aliases,
			Link:    fmt.Sprintf("https://osv.dev/vulnerability/%s", v.ID),
		}

		if len(v.Severity) > 0 {
			vuln.Severity = v.Severity[0].Score
		}

		// Find fix version
		for _, affected := range v.Affected {
			for _, r := range affected.Ranges {
				for _, event := range r.Events {
					if event.Fixed != "" {
						vuln.Fixed = event.Fixed
					}
				}
			}
		}

		vulns = append(vulns, vuln)
	}

	_ = strings.HasPrefix // keep strings import
	return vulns, nil
}
