package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/embedtools/instagram-scraper/internal"
	"github.com/embedtools/instagram-scraper/types"
)

// GetProfile fetches an Instagram profile by URL or username.
func (c *Client) GetProfile(ctx context.Context, in *types.GetProfileInput) (*types.GetProfileOutput, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}

	username := extractUsername(in.URL)
	if username == "" {
		return nil, fmt.Errorf("%w: could not extract username", ErrInvalidURL)
	}

	profileData, isPrivate, err := c.fetchProfileInfo(ctx, username)
	if err != nil && profileData == nil {
		return nil, err
	}

	// Filter fields if specified
	filtered := profileData
	if len(in.Fields) > 0 && profileData != nil {
		filtered = map[string]interface{}{}
		for _, field := range in.Fields {
			if val, ok := profileData[field]; ok {
				filtered[field] = val
			}
		}
	}

	output := &types.GetProfileOutput{
		Username:  username,
		IsPrivate: isPrivate,
		Data:      filtered,
	}

	if isPrivate {
		return output, ErrPrivateResource
	}

	return output, nil
}

func (c *Client) fetchProfileInfo(ctx context.Context, username string) (map[string]interface{}, bool, error) {
	apiURL := fmt.Sprintf("%s?username=%s", internal.ProfileAPIEndpoint, url.QueryEscape(username))

	var lastErr error
	for attempt := 0; attempt < internal.MaxRetries; attempt++ {
		httpClient := c.http
		if c.proxyURL != "" {
			httpClient = internal.CreateRotatingProxyClient(c.proxyURL, attempt)
			defer httpClient.CloseIdleConnections()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return nil, false, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
		}

		req.Header.Set("x-ig-app-id", internal.IGAppID)
		req.Header.Set("User-Agent", internal.DefaultUserAgent)
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Referer", "https://www.instagram.com/"+username+"/")
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("Sec-Fetch-Site", "same-origin")

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
			time.Sleep(internal.BaseBackoffDelay * time.Duration(1<<attempt))
			continue
		}

		switch resp.StatusCode {
		case http.StatusOK:
			var apiResp struct {
				Data struct {
					User map[string]interface{} `json:"user"`
				} `json:"data"`
				Status string `json:"status"`
			}
			if decErr := json.NewDecoder(resp.Body).Decode(&apiResp); decErr != nil {
				resp.Body.Close()
				lastErr = fmt.Errorf("%w: %v", ErrUpstreamChanged, decErr)
				continue
			}
			resp.Body.Close()

			if apiResp.Data.User == nil {
				return nil, false, ErrNotFound
			}

			if isPrivate, ok := apiResp.Data.User["is_private"].(bool); ok && isPrivate {
				return apiResp.Data.User, true, nil
			}
			return apiResp.Data.User, false, nil

		case http.StatusNotFound:
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return nil, false, ErrNotFound

		case http.StatusTooManyRequests:
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lastErr = ErrRateLimited
			time.Sleep(internal.BaseBackoffDelay * time.Duration(1<<attempt))
			continue

		case http.StatusForbidden:
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lastErr = ErrBlocked

		default:
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("%w: status %d", ErrUpstreamChanged, resp.StatusCode)
		}

		if attempt < internal.MaxRetries-1 {
			time.Sleep(internal.BaseBackoffDelay * time.Duration(1<<attempt))
		}
	}

	return nil, false, lastErr
}

func extractUsername(input string) string {
	input = strings.TrimPrefix(input, "@")

	if !strings.Contains(input, "/") && len(input) > 0 && len(input) <= 30 {
		return input
	}

	u, err := url.Parse(input)
	if err != nil || u.Scheme == "" || u.Host == "" {
		parts := strings.Split(strings.Trim(input, "/"), "/")
		if len(parts) == 1 && len(parts[0]) > 0 && len(parts[0]) <= 30 {
			return parts[0]
		}
		return ""
	}

	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) >= 1 && parts[0] != "" {
		username := parts[0]
		nonUserPaths := map[string]bool{
			"p": true, "reel": true, "tv": true, "stories": true,
			"explore": true, "direct": true, "accounts": true, "emailsignup": true,
		}
		if nonUserPaths[username] {
			return ""
		}
		if len(username) <= 30 {
			return username
		}
	}

	return ""
}
