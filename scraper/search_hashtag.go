package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/embedtools/instagram-scraper/internal"
	"github.com/embedtools/instagram-scraper/types"
)

const (
	hashtagGraphQLDocID    = "9510064595728286"
	hashtagGraphQLEndpoint = "https://www.instagram.com/api/graphql"
	hashtagItemsPerPage    = 12
	hashtagMaxPages        = 2
)

var hashtagIDRe = regexp.MustCompile(`"hashtag_id":"(\d+)"`)

// SearchHashtag scrapes content from a hashtag page.
func (c *Client) SearchHashtag(ctx context.Context, in *types.SearchHashtagInput) (*types.SearchHashtagOutput, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}

	hashtag := extractHashtag(in.Hashtag)
	if hashtag == "" {
		return nil, fmt.Errorf("%w: could not extract hashtag", ErrInvalidURL)
	}

	if c.curlBin == "" {
		return nil, fmt.Errorf("%w: curl-impersonate binary required for hashtag search", ErrUpstreamChanged)
	}

	pageURL := fmt.Sprintf("https://www.instagram.com/explore/tags/%s/", hashtag)
	html, err := internal.FetchPageWithCurl(ctx, c.curlBin, c.proxyURL, pageURL, "ig_hashtag_")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBlocked, err)
	}

	checkLen := internal.MinInt(5000, len(html))
	if strings.Contains(html[:checkLen], "accounts/login") && !strings.Contains(html, "xig_hashtag_info") {
		return nil, ErrBlocked
	}

	tokens := internal.ExtractTokens(html)
	if tokens.LSD == "" || tokens.CSRFToken == "" {
		return nil, fmt.Errorf("%w: failed to extract session tokens", ErrUpstreamChanged)
	}

	// Extract hashtag ID
	var hashtagID string
	if match := hashtagIDRe.FindStringSubmatch(html); len(match) > 1 {
		hashtagID = match[1]
	}

	// Extract embedded data (page 1) - try top posts first, then regular media
	var page1Data internal.PopularPageData
	dataBytes, err := internal.ExtractEmbeddedJSON(html, `"edge_hashtag_to_top_posts":`)
	if err != nil {
		dataBytes, err = internal.ExtractEmbeddedJSON(html, `"edge_hashtag_to_media":`)
		if err != nil {
			return nil, fmt.Errorf("%w: hashtag data not found", ErrUpstreamChanged)
		}
	}

	if err := json.Unmarshal(dataBytes, &page1Data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
	}

	allPosts := make([]map[string]interface{}, 0, hashtagItemsPerPage*hashtagMaxPages)
	for _, edge := range page1Data.Edges {
		allPosts = append(allPosts, internal.NodeToMap(edge.Node))
	}

	// Fetch page 2 if available
	if page1Data.PageInfo.HasNextPage && page1Data.PageInfo.EndCursor != "" && hashtagID != "" {
		variables := map[string]interface{}{
			"after":      page1Data.PageInfo.EndCursor,
			"hashtag_id": hashtagID,
			"first":      hashtagItemsPerPage,
		}
		variablesJSON, _ := json.Marshal(variables)

		formData := internal.BuildPopularFormData(hashtagGraphQLDocID, variablesJSON, tokens)
		formData.Set("fb_api_req_friendly_name", "PolarisHashtagPageContentQuery")

		output, err := internal.FetchGraphQLWithCurl(ctx, c.curlBin, c.proxyURL, hashtagGraphQLEndpoint, formData, tokens, pageURL, "ig_hashtag_gql_")
		if err == nil {
			var response struct {
				Data struct {
					XIGHashtagInfo *struct {
						EdgeHashtagToTopPosts *internal.PopularPageData `json:"edge_hashtag_to_top_posts"`
						EdgeHashtagToMedia    *internal.PopularPageData `json:"edge_hashtag_to_media"`
					} `json:"xig_hashtag_info"`
				} `json:"data"`
			}
			if json.Unmarshal(output, &response) == nil && response.Data.XIGHashtagInfo != nil {
				var page2Data *internal.PopularPageData
				if response.Data.XIGHashtagInfo.EdgeHashtagToTopPosts != nil {
					page2Data = response.Data.XIGHashtagInfo.EdgeHashtagToTopPosts
				} else if response.Data.XIGHashtagInfo.EdgeHashtagToMedia != nil {
					page2Data = response.Data.XIGHashtagInfo.EdgeHashtagToMedia
				}
				if page2Data != nil {
					for _, edge := range page2Data.Edges {
						allPosts = append(allPosts, internal.NodeToMap(edge.Node))
					}
				}
			}
		}
	}

	if in.Limit > 0 && in.Limit < len(allPosts) {
		allPosts = allPosts[:in.Limit]
	}

	return &types.SearchHashtagOutput{
		Hashtag:    hashtag,
		HashtagID:  hashtagID,
		TotalCount: len(allPosts),
		Posts:      allPosts,
	}, nil
}

func extractHashtag(input string) string {
	input = strings.TrimSpace(input)
	input = strings.TrimPrefix(input, "#")

	if !strings.Contains(input, "/") && !strings.Contains(input, "http") {
		return strings.ToLower(input)
	}

	u, err := url.Parse(input)
	if err != nil {
		return ""
	}

	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) >= 3 && parts[0] == "explore" && parts[1] == "tags" {
		return strings.ToLower(parts[2])
	}

	return ""
}
