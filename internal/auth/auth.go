package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type CredentialType int

const (
	CredBasic  CredentialType = iota
	CredBearer
	CredToken
)

type Credential struct {
	Type     CredentialType
	Username string
	Password string
	Token    string
}

func (c *Credential) Apply(req *http.Request) {
	switch c.Type {
	case CredBasic:
		req.SetBasicAuth(c.Username, c.Password)
	case CredBearer:
		req.Header.Set("Authorization", "Bearer "+c.Token)
	case CredToken:
		req.SetBasicAuth("__token__", c.Token)
	}
}

type Authenticator struct {
	netrcPath   string
	credentials map[string]*Credential
}

func NewAuthenticator() *Authenticator {
	return &Authenticator{
		credentials: make(map[string]*Credential),
	}
}

func (a *Authenticator) Authenticate(req *http.Request) {
	host := req.URL.Host

	cred := a.resolve(req.URL)
	if cred != nil {
		cred.Apply(req)
	}
	_ = host
}

func (a *Authenticator) resolve(u *url.URL) *Credential {
	host := u.Host

	// 1. URL-embedded credentials
	if u.User != nil {
		pass, _ := u.User.Password()
		return &Credential{Type: CredBasic, Username: u.User.Username(), Password: pass}
	}

	// 2. Environment variables (FPM_INDEX_<NAME>_USERNAME / PASSWORD or generic)
	if cred := a.fromEnv(host); cred != nil {
		return cred
	}

	// 3. Stored credentials (from config)
	if cred, ok := a.credentials[host]; ok {
		return cred
	}

	// 4. Netrc
	if cred := a.fromNetrc(host); cred != nil {
		return cred
	}

	return nil
}

func (a *Authenticator) fromEnv(host string) *Credential {
	// Check generic FPM_INDEX_TOKEN
	if token := os.Getenv("FPM_INDEX_TOKEN"); token != "" {
		return &Credential{Type: CredToken, Token: token}
	}

	// Check per-host: FPM_INDEX_<HOST>_TOKEN
	envKey := "FPM_INDEX_" + strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(host, ".", "_"), "-", "_")) + "_TOKEN"
	if token := os.Getenv(envKey); token != "" {
		return &Credential{Type: CredToken, Token: token}
	}

	// Check username/password
	userKey := "FPM_INDEX_USERNAME"
	passKey := "FPM_INDEX_PASSWORD"
	if user := os.Getenv(userKey); user != "" {
		return &Credential{Type: CredBasic, Username: user, Password: os.Getenv(passKey)}
	}

	return nil
}

func (a *Authenticator) fromNetrc(host string) *Credential {
	cred, err := LookupNetrc(host)
	if err != nil || cred == nil {
		return nil
	}
	return cred
}

func (a *Authenticator) SetCredential(host string, cred *Credential) {
	a.credentials[host] = cred
}

func BasicAuthHeader(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func BearerAuthHeader(token string) string {
	return fmt.Sprintf("Bearer %s", token)
}
