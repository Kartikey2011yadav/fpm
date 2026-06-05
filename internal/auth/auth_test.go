package auth

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestAuthenticatorURLCredentials(t *testing.T) {
	auth := NewAuthenticator()

	req, _ := http.NewRequest("GET", "https://user:pass@pypi.example.com/simple/", nil)
	auth.Authenticate(req)

	user, pass, ok := req.BasicAuth()
	if !ok {
		t.Fatal("expected basic auth to be set")
	}
	if user != "user" || pass != "pass" {
		t.Errorf("got %s:%s, want user:pass", user, pass)
	}
}

func TestAuthenticatorEnvToken(t *testing.T) {
	os.Setenv("FPM_INDEX_TOKEN", "test-token-123")
	defer os.Unsetenv("FPM_INDEX_TOKEN")

	auth := NewAuthenticator()

	req, _ := http.NewRequest("GET", "https://pypi.example.com/simple/", nil)
	auth.Authenticate(req)

	user, pass, ok := req.BasicAuth()
	if !ok {
		t.Fatal("expected basic auth to be set")
	}
	if user != "__token__" || pass != "test-token-123" {
		t.Errorf("got %s:%s, want __token__:test-token-123", user, pass)
	}
}

func TestAuthenticatorStoredCredential(t *testing.T) {
	auth := NewAuthenticator()
	auth.SetCredential("private.pypi.com", &Credential{
		Type:  CredBearer,
		Token: "bearer-token-abc",
	})

	req, _ := http.NewRequest("GET", "https://private.pypi.com/simple/numpy/", nil)
	auth.Authenticate(req)

	authHeader := req.Header.Get("Authorization")
	if authHeader != "Bearer bearer-token-abc" {
		t.Errorf("got %q, want 'Bearer bearer-token-abc'", authHeader)
	}
}

func TestNetrcParsing(t *testing.T) {
	tmpDir := t.TempDir()
	netrcFile := filepath.Join(tmpDir, ".netrc")
	os.WriteFile(netrcFile, []byte(`machine pypi.org
login myuser
password mypass

machine other.com
login otheruser
password otherpass
`), 0600)

	os.Setenv("NETRC", netrcFile)
	defer os.Unsetenv("NETRC")

	cred, err := LookupNetrc("pypi.org")
	if err != nil {
		t.Fatalf("LookupNetrc failed: %v", err)
	}
	if cred == nil {
		t.Fatal("expected credential, got nil")
	}
	if cred.Username != "myuser" || cred.Password != "mypass" {
		t.Errorf("got %s:%s, want myuser:mypass", cred.Username, cred.Password)
	}

	// Non-existent host
	cred2, _ := LookupNetrc("nonexistent.com")
	if cred2 != nil {
		t.Error("expected nil for non-existent host")
	}
}
