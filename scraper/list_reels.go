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
	clipsAPIEndpoint     = "https://www.instagram.com/api/v1/clips/user/"
	reelsPerPage         = 12
	reelsMaxPages        = 100
	reelsPaginationDelay = 5 * time.Second
)

// ListReels streams reels from an Instagram profile.
func (c *Client) ListReels(ctx context.Context, in *types.ListReelsInput, emit func(item *types.ReelItem) error) (*types.ReelsSummary, error) {
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
	if userID == "" {
		return nil, fmt.Errorf("%w: could not get user ID", ErrNotFound)
	}

	emitted := 0
	var maxID string
	currentSession := session
	moreAvailable := false

	pagesToFetch := reelsMaxPages
	if in.MaxPages > 0 && in.MaxPages < pagesToFetch {
		pagesToFetch = in.MaxPages
	}

	for page := 0; page < pagesToFetch; page++ {
		if ctx.Err() != nil {
			break
		}

		if page > 0 && page%4 == 0 {
			if newSession, err := internal.GetSession(ctx, c.http, c.proxyURL); err == nil {
				currentSession = newSession
			}
		}

		reels, pagingInfo, err := c.fetchReelsPage(ctx, userID, maxID, currentSession)
		if err != nil {
			break
		}

		for _, reel := range reels {
			if in.Since > 0 {
				if takenAt, ok := reel["taken_at"].(float64); ok && int64(takenAt) < in.Since {
					return &types.ReelsSummary{UserID: userID, Username: username, Emitted: emitted, HasMore: false}, nil
				}
			}
			if err := emit(&types.ReelItem{Data: reel}); err != nil {
				return nil, err
			}
			emitted++
			if in.Limit > 0 && emitted >= in.Limit {
				return &types.ReelsSummary{UserID: userID, Username: username, Emitted: emitted, HasMore: pagingInfo.MoreAvailable}, nil
			}
		}

		moreAvailable = pagingInfo.MoreAvailable
		maxID = pagingInfo.MaxID

		if !moreAvailable || maxID == "" {
			break
		}

		time.Sleep(reelsPaginationDelay)
	}

	return &types.ReelsSummary{UserID: userID, Username: username, Emitted: emitted, HasMore: moreAvailable}, nil
}

type clipsPagingInfo struct {
	MaxID         string `json:"max_id"`
	MoreAvailable bool   `json:"more_available"`
}

func (c *Client) fetchReelsPage(ctx context.Context, userID, maxID string, session *internal.Session) ([]map[string]interface{}, *clipsPagingInfo, error) {
	formData := url.Values{}
	formData.Set("include_feed_video", "true")
	formData.Set("page_size", fmt.Sprintf("%d", reelsPerPage))
	formData.Set("target_user_id", userID)
	if maxID != "" {
		formData.Set("max_id", maxID)
	}

	for attempt := 0; attempt < internal.MaxRetries; attempt++ {
		httpClient := c.http
		if c.proxyURL != "" {
			httpClient = internal.CreateRotatingProxyClient(c.proxyURL, attempt)
			defer httpClient.CloseIdleConnections()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, clipsAPIEndpoint, strings.NewReader(formData.Encode()))
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
		}

		req.Header.Set("Accept", "*/*")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "https://www.instagram.com")
		req.Header.Set("Referer", "https://www.instagram.com/")
		req.Header.Set("User-Agent", internal.DefaultUserAgent)
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
			var clipsResp struct {
				Items []struct {
					Media map[string]interface{} `json:"media"`
				} `json:"items"`
				PagingInfo clipsPagingInfo `json:"paging_info"`
				Status     string          `json:"status"`
			}

			if err := json.Unmarshal(body, &clipsResp); err != nil {
				continue
			}

			reels := make([]map[string]interface{}, 0, len(clipsResp.Items))
			for _, item := range clipsResp.Items {
				if item.Media != nil {
					reels = append(reels, item.Media)
				}
			}

			return reels, &clipsResp.PagingInfo, nil
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			time.Sleep(internal.BaseBackoffDelay * time.Duration(1<<attempt))
			continue
		}
	}

	return nil, nil, fmt.Errorf("%w: clips fetch failed", ErrUpstreamChanged)
}
