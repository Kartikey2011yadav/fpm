package auth

import (
	"net/http"
)

type AuthTransport struct {
	Base          http.RoundTripper
	Authenticator *Authenticator
}

func NewAuthTransport(auth *Authenticator) *AuthTransport {
	return &AuthTransport{
		Base:          http.DefaultTransport,
		Authenticator: auth,
	}
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request to avoid modifying the original
	clone := req.Clone(req.Context())

	// Apply authentication
	t.Authenticator.Authenticate(clone)

	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	resp, err := base.RoundTrip(clone)
	if err != nil {
		return nil, err
	}

	// Handle 401 - could retry with different credentials
	if resp.StatusCode == http.StatusUnauthorized {
		// For now, just return the 401 — future: retry with keyring
		return resp, nil
	}

	return resp, nil
}

func NewAuthenticatedClient(auth *Authenticator) *http.Client {
	return &http.Client{
		Transport: NewAuthTransport(auth),
	}
}
