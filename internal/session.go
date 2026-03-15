package internal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	IGAppID            = "936619743392459"
	ProfileAPIEndpoint = "https://i.instagram.com/api/v1/users/web_profile_info/"
	DefaultUserAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	MaxRetries         = 3
	BaseBackoffDelay   = 1 * time.Second
)

// Session holds Instagram session cookies.
type Session struct {
	CSRFToken string
	Mid       string
}

// GenerateSessionID creates a random session ID for proxy rotation.
func GenerateSessionID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateRotatingProxyClient creates a new HTTP client with a fresh session ID for IP rotation.
func CreateRotatingProxyClient(baseProxyURL string, attempt int) *http.Client {
	if baseProxyURL == "" {
		return &http.Client{Timeout: 90 * time.Second}
	}

	sessionID := GenerateSessionID()
	rotatedProxyURL := baseProxyURL

	if atIdx := strings.LastIndex(baseProxyURL, "@"); atIdx != -1 {
		rotatedProxyURL = baseProxyURL[:atIdx] + "_session-" + sessionID + baseProxyURL[atIdx:]
	}

	proxyURL, err := url.Parse(rotatedProxyURL)
	if err != nil {
		return &http.Client{Timeout: 90 * time.Second}
	}

	transport := &http.Transport{
		Proxy:               http.ProxyURL(proxyURL),
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 30 * time.Second,
		ForceAttemptHTTP2:   true,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   90 * time.Second,
	}
}

// GetSession fetches Instagram homepage to get session cookies.
func GetSession(ctx context.Context, httpClient *http.Client, proxyURL string) (*Session, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.instagram.com/", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating session request: %w", err)
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("User-Agent", DefaultUserAgent)

	client := httpClient
	if proxyURL != "" {
		client = CreateRotatingProxyClient(proxyURL, 0)
		defer client.CloseIdleConnections()
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching session: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	session := &Session{}
	for _, cookie := range resp.Cookies() {
		switch cookie.Name {
		case "csrftoken":
			session.CSRFToken = cookie.Value
		case "mid":
			session.Mid = cookie.Value
		}
	}

	if session.CSRFToken == "" {
		return nil, fmt.Errorf("failed to get CSRF token from Instagram")
	}

	return session, nil
}

// FetchProfileData fetches profile data from Instagram API.
func FetchProfileData(ctx context.Context, httpClient *http.Client, proxyURL, username string, session *Session) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s?username=%s", ProfileAPIEndpoint, url.QueryEscape(username))

	for attempt := 0; attempt < MaxRetries; attempt++ {
		client := httpClient
		if proxyURL != "" {
			client = CreateRotatingProxyClient(proxyURL, attempt)
			defer client.CloseIdleConnections()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("x-ig-app-id", IGAppID)
		req.Header.Set("User-Agent", DefaultUserAgent)
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Referer", "https://www.instagram.com/"+username+"/")
		req.Header.Set("X-CSRFToken", session.CSRFToken)

		req.AddCookie(&http.Cookie{Name: "csrftoken", Value: session.CSRFToken})
		if session.Mid != "" {
			req.AddCookie(&http.Cookie{Name: "mid", Value: session.Mid})
		}

		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(BaseBackoffDelay * time.Duration(1<<attempt))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var apiResp struct {
				Data struct {
					User map[string]interface{} `json:"user"`
				} `json:"data"`
				Status string `json:"status"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
				continue
			}
			if apiResp.Data.User == nil {
				return nil, fmt.Errorf("user not found")
			}
			return apiResp.Data.User, nil
		}

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("user not found")
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			time.Sleep(BaseBackoffDelay * time.Duration(1<<attempt))
			continue
		}
	}

	return nil, fmt.Errorf("failed to get profile after %d attempts", MaxRetries)
}

// MinInt returns the smaller of two ints.
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
