package tls

import (
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// LoadCertPool builds a certificate pool based on environment variables.
// Follows uv's precedence: if SSL_CERT_FILE or SSL_CERT_DIR is set,
// those certs are the ONLY trusted roots (complete override).
// Returns nil if neither is set (Go will use system + bundled fallback).
func LoadCertPool() (*x509.CertPool, error) {
	certFile := os.Getenv("SSL_CERT_FILE")
	certDir := os.Getenv("SSL_CERT_DIR")

	if certFile == "" && certDir == "" {
		return nil, nil
	}

	pool := x509.NewCertPool()
	loaded := false

	if certFile != "" {
		pem, err := os.ReadFile(certFile)
		if err != nil {
			return nil, fmt.Errorf("reading SSL_CERT_FILE %q: %w", certFile, err)
		}
		if pool.AppendCertsFromPEM(pem) {
			loaded = true
		}
	}

	if certDir != "" {
		dirs := splitCertDirs(certDir)
		for _, dir := range dirs {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				path := filepath.Join(dir, entry.Name())
				pem, err := os.ReadFile(path)
				if err != nil {
					continue
				}
				if pool.AppendCertsFromPEM(pem) {
					loaded = true
				}
			}
		}
	}

	if !loaded {
		return nil, fmt.Errorf("no valid certificates found in SSL_CERT_FILE/SSL_CERT_DIR")
	}

	return pool, nil
}

func splitCertDirs(val string) []string {
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	var dirs []string
	for _, d := range strings.Split(val, sep) {
		d = strings.TrimSpace(d)
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}
