package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCertPoolNoEnv(t *testing.T) {
	t.Setenv("SSL_CERT_FILE", "")
	t.Setenv("SSL_CERT_DIR", "")

	pool, err := LoadCertPool()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool != nil {
		t.Error("expected nil pool when no env vars set")
	}
}

func TestLoadCertPoolFile(t *testing.T) {
	certPEM := generateTestCACert(t)
	certFile := filepath.Join(t.TempDir(), "ca.pem")
	os.WriteFile(certFile, certPEM, 0644)

	t.Setenv("SSL_CERT_FILE", certFile)
	t.Setenv("SSL_CERT_DIR", "")

	pool, err := LoadCertPool()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
}

func TestLoadCertPoolFileMissing(t *testing.T) {
	t.Setenv("SSL_CERT_FILE", "/nonexistent/path/ca.pem")
	t.Setenv("SSL_CERT_DIR", "")

	_, err := LoadCertPool()
	if err == nil {
		t.Error("expected error for missing SSL_CERT_FILE")
	}
}

func TestLoadCertPoolFileInvalid(t *testing.T) {
	certFile := filepath.Join(t.TempDir(), "bad.pem")
	os.WriteFile(certFile, []byte("not a certificate"), 0644)

	t.Setenv("SSL_CERT_FILE", certFile)
	t.Setenv("SSL_CERT_DIR", "")

	_, err := LoadCertPool()
	if err == nil {
		t.Error("expected error for invalid PEM in SSL_CERT_FILE")
	}
}

func TestLoadCertPoolDir(t *testing.T) {
	certPEM := generateTestCACert(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "corp-ca.pem"), certPEM, 0644)
	os.WriteFile(filepath.Join(dir, "not-a-cert.txt"), []byte("hello"), 0644)

	t.Setenv("SSL_CERT_FILE", "")
	t.Setenv("SSL_CERT_DIR", dir)

	pool, err := LoadCertPool()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
}

func TestLoadCertPoolDirEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SSL_CERT_FILE", "")
	t.Setenv("SSL_CERT_DIR", dir)

	_, err := LoadCertPool()
	if err == nil {
		t.Error("expected error for empty SSL_CERT_DIR")
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"example.com", "example.com"},
		{"Example.COM", "example.com"},
		{"example.com:443", "example.com"},
		{"example.com:80", "example.com"},
		{"example.com:8080", "example.com:8080"},
		{"https://example.com", "example.com"},
		{"https://Example.COM:443", "example.com"},
		{"192.168.1.1:9000", "192.168.1.1:9000"},
		{"  example.com  ", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeHost(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeHost(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestInsecureHostTransport(t *testing.T) {
	// Create a TLS server with a self-signed cert (clients will reject it normally)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Extract host from server URL
	host := server.Listener.Addr().String()

	// Base transport that does NOT trust the server's self-signed cert
	base := http.DefaultTransport.(*http.Transport).Clone()

	// Without insecure host: should fail
	client := &http.Client{Transport: base}
	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("expected TLS error for self-signed cert without insecure bypass")
	}

	// With insecure host: should succeed
	transport := NewInsecureHostTransport(base, []string{host})
	client = &http.Client{Transport: transport}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("expected success with insecure host bypass, got: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestInsecureHostTransportNonMatchingHost(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	base := http.DefaultTransport.(*http.Transport).Clone()

	// Insecure host list does NOT include the server's host
	transport := NewInsecureHostTransport(base, []string{"other-host.example.com"})
	client := &http.Client{Transport: transport}
	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("expected TLS error for non-matching insecure host")
	}
}

func generateTestCACert(t *testing.T) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
}
