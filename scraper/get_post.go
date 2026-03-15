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

const (
	instagramGraphQLURL  = "https://www.instagram.com/graphql/query"
	instagramDocIDPost   = "29599222026389233"
	defaultCSRFTokenPost = "wTB0c7BwcA3mahgAuC9ZTO"
	defaultXFBLSDPost    = "AVpp68Mg2aw"
)

// GetPost fetches an Instagram post by URL or shortcode.
func (c *Client) GetPost(ctx context.Context, in *types.GetPostInput) (*types.GetPostOutput, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}

	shortcode := extractShortcode(in.URL)
	if shortcode == "" || len(shortcode) > 15 {
		return nil, fmt.Errorf("%w: could not extract shortcode", ErrInvalidURL)
	}

	data, err := c.fetchGraphQLPostData(ctx, shortcode)
	if err != nil {
		return nil, err
	}

	return &types.GetPostOutput{
		Shortcode: shortcode,
		Data:      data,
	}, nil
}

func (c *Client) fetchGraphQLPostData(ctx context.Context, shortcode string) (map[string]interface{}, error) {
	variables := map[string]interface{}{
		"shortcode":               shortcode,
		"fetch_tagged_user_count": nil,
		"hoisted_comment_id":      nil,
		"hoisted_reply_id":        nil,
	}
	variablesJSON, _ := json.Marshal(variables)

	formData := url.Values{}
	formData.Set("av", "0")
	formData.Set("__d", "www")
	formData.Set("__user", "0")
	formData.Set("__a", "1")
	formData.Set("__req", "c")
	formData.Set("dpr", "2")
	formData.Set("__ccg", "EXCELLENT")
	formData.Set("__comet_req", "7")
	formData.Set("lsd", defaultXFBLSDPost)
	formData.Set("fb_api_caller_class", "RelayModern")
	formData.Set("fb_api_req_friendly_name", "PolarisPostActionLoadPostQueryQuery")
	formData.Set("variables", string(variablesJSON))
	formData.Set("server_timestamps", "true")
	formData.Set("doc_id", instagramDocIDPost)

	var lastErr error
	for attempt := 0; attempt < internal.MaxRetries; attempt++ {
		httpClient := c.http
		if c.proxyURL != "" {
			httpClient = internal.CreateRotatingProxyClient(c.proxyURL, attempt)
			defer httpClient.CloseIdleConnections()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, instagramGraphQLURL, strings.NewReader(formData.Encode()))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
		}

		req.Header.Set("accept", "*/*")
		req.Header.Set("content-type", "application/x-www-form-urlencoded")
		req.Header.Set("origin", "https://www.instagram.com")
		req.Header.Set("referer", fmt.Sprintf("https://www.instagram.com/p/%s/", shortcode))
		req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")
		req.Header.Set("x-csrftoken", defaultCSRFTokenPost)
		req.Header.Set("x-fb-friendly-name", "PolarisPostActionLoadPostQueryQuery")
		req.Header.Set("x-fb-lsd", defaultXFBLSDPost)
		req.Header.Set("x-ig-app-id", internal.IGAppID)

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
			time.Sleep(internal.BaseBackoffDelay * time.Duration(1<<attempt))
			continue
		}

		switch resp.StatusCode {
		case http.StatusOK:
			var gqlResp struct {
				Data struct {
					XdtShortcodeMedia map[string]interface{} `json:"xdt_shortcode_media"`
				} `json:"data"`
			}
			if decErr := json.NewDecoder(resp.Body).Decode(&gqlResp); decErr != nil {
				resp.Body.Close()
				lastErr = fmt.Errorf("%w: %v", ErrUpstreamChanged, decErr)
				continue
			}
			resp.Body.Close()

			if len(gqlResp.Data.XdtShortcodeMedia) == 0 {
				lastErr = ErrNotFound
				continue
			}
			return gqlResp.Data.XdtShortcodeMedia, nil

		case http.StatusNotFound:
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return nil, ErrNotFound

		case http.StatusTooManyRequests:
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lastErr = ErrRateLimited

		default:
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("%w: status %d", ErrUpstreamChanged, resp.StatusCode)
		}

		if attempt < internal.MaxRetries-1 {
			time.Sleep(internal.BaseBackoffDelay * time.Duration(1<<attempt))
		}
	}

	return nil, lastErr
}

func extractShortcode(input string) string {
	if !strings.Contains(input, "/") && len(input) < 15 {
		return input
	}

	u, err := url.Parse(input)
	if err != nil || u.Scheme == "" || u.Host == "" {
		parts := strings.Split(strings.Trim(input, "/"), "/")
		if len(parts) >= 2 && (parts[0] == "p" || parts[0] == "reel") {
			if len(parts[1]) > 0 {
				return parts[1]
			}
		}
		if len(parts) == 1 && len(parts[0]) < 15 && parts[0] != "" {
			return parts[0]
		}
		return ""
	}

	path := strings.TrimSuffix(u.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) >= 3 && (parts[1] == "p" || parts[1] == "reel") {
		return parts[2]
	} else if len(parts) >= 4 && (parts[2] == "p" || parts[2] == "reel") {
		return parts[3]
	}

	return ""
}
