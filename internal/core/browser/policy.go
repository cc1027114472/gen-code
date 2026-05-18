package browser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	envAllowedHosts       = "GENCODE_BROWSER_ALLOWED_HOSTS"
	envAllowedHostsLegacy = "GEN_CODE_BROWSER_ALLOWED_HOSTS"
	envPolicyFile         = "GENCODE_BROWSER_POLICY_FILE"
	envPolicyFileLegacy   = "GEN_CODE_BROWSER_POLICY_FILE"
)

type Policy struct {
	hostRules map[string]hostRule
}

type hostRule struct {
	allowed         bool
	sessionRequired bool
	sessionProfile  *SessionProfile
	sessionErr      error
}

type SessionProfile struct {
	Cookies []SessionCookie
}

type SessionCookie struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Domain      string `json:"domain,omitempty"`
	Path        string `json:"path,omitempty"`
	Secure      bool   `json:"secure,omitempty"`
	HTTPOnly    bool   `json:"httpOnly,omitempty"`
	SameSite    string `json:"sameSite,omitempty"`
	ExpiresUnix int64  `json:"expiresUnix,omitempty"`
}

type policyFileConfig struct {
	AllowedHosts []string                    `json:"allowedHosts"`
	Hosts        map[string]hostPolicyConfig `json:"hosts"`
}

type hostPolicyConfig struct {
	SessionRequired bool                `json:"sessionRequired"`
	Cookies         []sessionCookieSpec `json:"cookies"`
}

type sessionCookieSpec struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Domain      string `json:"domain,omitempty"`
	Path        string `json:"path,omitempty"`
	Secure      bool   `json:"secure,omitempty"`
	HTTPOnly    bool   `json:"httpOnly,omitempty"`
	SameSite    string `json:"sameSite,omitempty"`
	ExpiresUnix int64  `json:"expiresUnix,omitempty"`
}

func defaultPolicy() Policy {
	return newPolicyFromSources(
		firstNonEmptyEnv(envAllowedHosts, envAllowedHostsLegacy),
		firstNonEmptyEnv(envPolicyFile, envPolicyFileLegacy),
	)
}

func newPolicyFromSources(allowedHosts string, policyFile string) Policy {
	policy := Policy{
		hostRules: map[string]hostRule{},
	}
	policy.allowHost("localhost")
	policy.allowHost("127.0.0.1")
	policy.loadAllowedHosts(allowedHosts)
	policy.loadPolicyFile(policyFile)
	return policy
}

func (p Policy) allowsURL(parsed *url.URL) bool {
	if parsed == nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return false
	}
	if isLocalBrowserHost(host) {
		return true
	}
	if scheme != "https" {
		return false
	}
	rule, ok := p.hostRules[host]
	return ok && rule.allowed
}

func (p Policy) sessionProfileForHost(rawHost string) (SessionProfile, bool, error) {
	host, err := normalizePolicyHost(rawHost)
	if err != nil {
		return SessionProfile{}, false, nil
	}
	rule, ok := p.hostRules[host]
	if !ok || !rule.allowed {
		return SessionProfile{}, false, nil
	}
	if !rule.sessionRequired {
		return SessionProfile{}, false, nil
	}
	if rule.sessionErr != nil {
		return SessionProfile{}, true, rule.sessionErr
	}
	if rule.sessionProfile == nil || len(rule.sessionProfile.Cookies) == 0 {
		return SessionProfile{}, true, fmt.Errorf("missing cookie bootstrap profile for %s", host)
	}
	return cloneSessionProfile(*rule.sessionProfile), true, nil
}

func (p *Policy) allowHost(rawHost string) {
	host, err := normalizePolicyHost(rawHost)
	if err != nil {
		return
	}
	rule := p.hostRules[host]
	rule.allowed = true
	p.hostRules[host] = rule
}

func (p *Policy) loadAllowedHosts(value string) {
	for _, entry := range strings.Split(value, ",") {
		p.allowHost(entry)
	}
}

func (p *Policy) loadPolicyFile(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	var config policyFileConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return
	}
	for _, host := range config.AllowedHosts {
		p.allowHost(host)
	}
	for rawHost, hostConfig := range config.Hosts {
		host, err := normalizePolicyHost(rawHost)
		if err != nil {
			continue
		}
		rule := p.hostRules[host]
		rule.allowed = true
		if hostConfig.SessionRequired || len(hostConfig.Cookies) > 0 {
			rule.sessionRequired = true
		}
		if len(hostConfig.Cookies) > 0 {
			profile, err := buildSessionProfile(hostConfig.Cookies)
			if err != nil {
				rule.sessionErr = err
				rule.sessionProfile = nil
			} else {
				rule.sessionErr = nil
				rule.sessionProfile = &profile
			}
		} else if hostConfig.SessionRequired {
			rule.sessionErr = fmt.Errorf("missing cookie bootstrap profile for %s", host)
			rule.sessionProfile = nil
		}
		p.hostRules[host] = rule
	}
}

func buildSessionProfile(specs []sessionCookieSpec) (SessionProfile, error) {
	cookies := make([]SessionCookie, 0, len(specs))
	for index, spec := range specs {
		cookie, err := normalizeSessionCookie(spec)
		if err != nil {
			return SessionProfile{}, fmt.Errorf("invalid cookie %d: %w", index, err)
		}
		cookies = append(cookies, cookie)
	}
	return SessionProfile{Cookies: cookies}, nil
}

func normalizeSessionCookie(spec sessionCookieSpec) (SessionCookie, error) {
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return SessionCookie{}, fmt.Errorf("cookie name is required")
	}
	sameSite, err := normalizeSameSite(spec.SameSite)
	if err != nil {
		return SessionCookie{}, err
	}
	path := strings.TrimSpace(spec.Path)
	if path == "" {
		path = "/"
	}
	domain := strings.TrimSpace(spec.Domain)
	if domain != "" {
		normalizedDomain, err := normalizePolicyHost(domain)
		if err != nil {
			return SessionCookie{}, fmt.Errorf("invalid cookie domain %q", spec.Domain)
		}
		domain = normalizedDomain
	}
	return SessionCookie{
		Name:        name,
		Value:       spec.Value,
		Domain:      domain,
		Path:        path,
		Secure:      spec.Secure,
		HTTPOnly:    spec.HTTPOnly,
		SameSite:    sameSite,
		ExpiresUnix: spec.ExpiresUnix,
	}, nil
}

func normalizeSameSite(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return "", nil
	case "strict":
		return "Strict", nil
	case "lax":
		return "Lax", nil
	case "none":
		return "None", nil
	default:
		return "", fmt.Errorf("invalid sameSite value %q", value)
	}
}

func normalizeURL(raw string) (string, error) {
	return normalizeURLWithPolicy(raw, defaultPolicy())
}

func normalizeURLWithPolicy(raw string, policy Policy) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("%w: empty url", ErrURLNotAllowed)
	}
	if strings.HasPrefix(value, "localhost:") || strings.HasPrefix(value, "127.0.0.1:") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrURLNotAllowed, err)
	}
	if !allowedURLWithPolicy(parsed, policy) {
		return "", fmt.Errorf("%w: %s", ErrURLNotAllowed, value)
	}
	return parsed.String(), nil
}

func allowedURL(parsed *url.URL) bool {
	return allowedURLWithPolicy(parsed, defaultPolicy())
}

func allowedURLWithPolicy(parsed *url.URL, policy Policy) bool {
	return policy.allowsURL(parsed)
}

func isLocalBrowserHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1"
}

func normalizePolicyHost(rawHost string) (string, error) {
	host := strings.ToLower(strings.TrimSpace(rawHost))
	if host == "" {
		return "", fmt.Errorf("host is required")
	}
	if strings.Contains(host, "://") || strings.ContainsAny(host, "/?#") || strings.Contains(host, ":") {
		return "", fmt.Errorf("invalid host %q", rawHost)
	}
	parsed, err := url.Parse("https://" + host)
	if err != nil || parsed.Hostname() == "" || parsed.Hostname() != host || parsed.Port() != "" {
		return "", fmt.Errorf("invalid host %q", rawHost)
	}
	return host, nil
}

func firstNonEmptyEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func cloneSessionProfile(profile SessionProfile) SessionProfile {
	if len(profile.Cookies) == 0 {
		return SessionProfile{}
	}
	cookies := make([]SessionCookie, len(profile.Cookies))
	copy(cookies, profile.Cookies)
	return SessionProfile{Cookies: cookies}
}
