package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/embedtools/instagram-scraper/internal"
	"github.com/embedtools/instagram-scraper/types"
)

const (
	postsDocID           = "7950326061742207"
	postsPerPage         = 12
	postsMaxPages        = 100
	postsPaginationDelay = 5 * time.Second
)

// ListProfilePosts streams posts from an Instagram profile.
func (c *Client) ListProfilePosts(ctx context.Context, in *types.ListProfilePostsInput, emit func(item *types.ProfilePostItem) error) (*types.ProfilePostsSummary, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}

	username := extractUsername(in.URL)
	if username == "" {
		return nil, fmt.Errorf("%w: could not extract username", ErrInvalidURL)
	}

	session, err := internal.GetSession(ctx, c.http, c.proxyURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBlocked, err)
	}

	profileData, err := internal.FetchProfileData(ctx, c.http, c.proxyURL, username, session)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}

	userID := getStr(profileData, "id")
	totalPosts := 0
	emitted := 0

	timelineMedia := getMap2(profileData, "edge_owner_to_timeline_media")
	totalPosts = getInt2(timelineMedia, "count")
	pageInfo := getMap2(timelineMedia, "page_info")
	hasNextPage := getBool2(pageInfo, "has_next_page")
	endCursor := getStr(pageInfo, "end_cursor")

	edges := getArray2(timelineMedia, "edges")
	for _, edge := range edges {
		if edgeMap, ok := edge.(map[string]interface{}); ok {
			if node, ok := edgeMap["node"].(map[string]interface{}); ok {
				if in.Since > 0 {
					if takenAt, ok := node["taken_at_timestamp"].(float64); ok && int64(takenAt) < in.Since {
						return &types.ProfilePostsSummary{UserID: userID, Username: username, TotalPosts: totalPosts, Emitted: emitted, HasMore: false}, nil
					}
				}
				if err := emit(&types.ProfilePostItem{Data: node}); err != nil {
					return nil, err
				}
				emitted++
				if in.Limit > 0 && emitted >= in.Limit {
					return &types.ProfilePostsSummary{UserID: userID, Username: username, TotalPosts: totalPosts, Emitted: emitted, HasMore: hasNextPage}, nil
				}
			}
		}
	}

	pagesToFetch := postsMaxPages
	if in.MaxPages > 0 && in.MaxPages < pagesToFetch {
		pagesToFetch = in.MaxPages
	}

	currentSession := session
	for page := 0; page < pagesToFetch && hasNextPage && endCursor != "" && userID != ""; page++ {
		if ctx.Err() != nil {
			break
		}

		if page > 0 && page%4 == 0 {
			if newSession, err := internal.GetSession(ctx, c.http, c.proxyURL); err == nil {
				currentSession = newSession
			}
		}

		posts, pi, err := c.fetchPostsPage(ctx, userID, endCursor, currentSession)
		if err != nil {
			break
		}

		for _, post := range posts {
			if in.Since > 0 {
				if takenAt, ok := post["taken_at_timestamp"].(float64); ok && int64(takenAt) < in.Since {
					return &types.ProfilePostsSummary{UserID: userID, Username: username, TotalPosts: totalPosts, Emitted: emitted, HasMore: false}, nil
				}
			}
			if err := emit(&types.ProfilePostItem{Data: post}); err != nil {
				return nil, err
			}
			emitted++
			if in.Limit > 0 && emitted >= in.Limit {
				return &types.ProfilePostsSummary{UserID: userID, Username: username, TotalPosts: totalPosts, Emitted: emitted, HasMore: pi != nil && pi.HasNextPage}, nil
			}
		}

		if pi == nil || !pi.HasNextPage || pi.EndCursor == "" {
			hasNextPage = false
			break
		}
		hasNextPage = pi.HasNextPage
		endCursor = pi.EndCursor

		time.Sleep(postsPaginationDelay)
	}

	return &types.ProfilePostsSummary{UserID: userID, Username: username, TotalPosts: totalPosts, Emitted: emitted, HasMore: hasNextPage}, nil
}

type postsPageInfo struct {
	HasNextPage bool
	EndCursor   string
}

func (c *Client) fetchPostsPage(ctx context.Context, userID, cursor string, session *internal.Session) ([]map[string]interface{}, *postsPageInfo, error) {
	variables := map[string]interface{}{
		"id":    userID,
		"first": postsPerPage,
		"after": cursor,
	}
	variablesJSON, _ := json.Marshal(variables)

	apiURL := fmt.Sprintf("https://www.instagram.com/graphql/query/?doc_id=%s&variables=%s",
		postsDocID, url.QueryEscape(string(variablesJSON)))

	for attempt := 0; attempt < internal.MaxRetries; attempt++ {
		httpClient := c.http
		if c.proxyURL != "" {
			httpClient = internal.CreateRotatingProxyClient(c.proxyURL, attempt)
			defer httpClient.CloseIdleConnections()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
		}

		req.Header.Set("Accept", "*/*")
		req.Header.Set("User-Agent", internal.DefaultUserAgent)
		req.Header.Set("Referer", "https://www.instagram.com/")
		req.Header.Set("X-CSRFToken", session.CSRFToken)
		req.Header.Set("X-IG-App-ID", internal.IGAppID)
		req.AddCookie(&http.Cookie{Name: "csrftoken", Value: session.CSRFToken})
		if session.Mid != "" {
			req.AddCookie(&http.Cookie{Name: "mid", Value: session.Mid})
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			time.Sleep(internal.BaseBackoffDelay * time.Duration(1<<attempt))
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var gqlResp struct {
				Data struct {
					User struct {
						EdgeOwnerToTimelineMedia struct {
							PageInfo struct {
								HasNextPage bool   `json:"has_next_page"`
								EndCursor   string `json:"end_cursor"`
							} `json:"page_info"`
							Edges []struct {
								Node map[string]interface{} `json:"node"`
							} `json:"edges"`
						} `json:"edge_owner_to_timeline_media"`
					} `json:"user"`
				} `json:"data"`
			}

			if err := json.Unmarshal(body, &gqlResp); err != nil {
				continue
			}

			edges := gqlResp.Data.User.EdgeOwnerToTimelineMedia.Edges
			posts := make([]map[string]interface{}, 0, len(edges))
			for _, edge := range edges {
				posts = append(posts, edge.Node)
			}

			pi := &postsPageInfo{
				HasNextPage: gqlResp.Data.User.EdgeOwnerToTimelineMedia.PageInfo.HasNextPage,
				EndCursor:   gqlResp.Data.User.EdgeOwnerToTimelineMedia.PageInfo.EndCursor,
			}

			return posts, pi, nil
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			time.Sleep(internal.BaseBackoffDelay * time.Duration(1<<attempt))
			continue
		}

		// Check for login-required response
		var errResp struct {
			RequireLogin bool `json:"require_login"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.RequireLogin {
			return nil, nil, fmt.Errorf("%w: login required", ErrBlocked)
		}
	}

	return nil, nil, fmt.Errorf("%w: pagination fetch failed", ErrUpstreamChanged)
}

// Map helper functions (avoid conflicts with other files)
func getStr(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt2(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getBool2(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getMap2(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return map[string]interface{}{}
}

func getArray2(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key].([]interface{}); ok {
		return v
	}
	return nil
}
