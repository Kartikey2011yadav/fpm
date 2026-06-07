package tls

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
)

// InsecureHostTransport wraps an http.Transport to skip TLS verification
// for specific hosts, while maintaining strict verification for all others.
type InsecureHostTransport struct {
	Base  *http.Transport
	hosts map[string]bool
}

// NewInsecureHostTransport creates a transport that bypasses TLS verification
// for the given hosts. Hosts can be "hostname" or "hostname:port".
func NewInsecureHostTransport(base *http.Transport, hosts []string) *InsecureHostTransport {
	hostMap := make(map[string]bool, len(hosts))
	for _, h := range hosts {
		hostMap[NormalizeHost(h)] = true
	}
	return &InsecureHostTransport{
		Base:  base,
		hosts: hostMap,
	}
}

func (t *InsecureHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := NormalizeHost(req.URL.Host)
	if !t.hosts[host] {
		return t.Base.RoundTrip(req)
	}

	insecure := t.Base.Clone()
	if insecure.TLSClientConfig == nil {
		insecure.TLSClientConfig = &tls.Config{}
	}
	insecure.TLSClientConfig.InsecureSkipVerify = true
	defer insecure.CloseIdleConnections()
	return insecure.RoundTrip(req)
}

// NormalizeHost lowercases the host and strips default ports (80, 443).
func NormalizeHost(host string) string {
	h := strings.ToLower(strings.TrimSpace(host))

	// Strip scheme if present
	if idx := strings.Index(h, "://"); idx >= 0 {
		h = h[idx+3:]
	}

	hostname, port, err := net.SplitHostPort(h)
	if err != nil {
		return h
	}
	if port == "443" || port == "80" {
		return hostname
	}
	return h
}
