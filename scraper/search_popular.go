package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/embedtools/instagram-scraper/internal"
	"github.com/embedtools/instagram-scraper/types"
)

const (
	popularGraphQLDocID    = "25415354221409251"
	popularGraphQLEndpoint = "https://www.instagram.com/api/graphql"
	popularItemsPerPage    = 12
	popularMaxPages        = 2
)

// SearchPopular scrapes trending/popular content for a keyword.
func (c *Client) SearchPopular(ctx context.Context, in *types.SearchPopularInput) (*types.SearchPopularOutput, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}

	keyword := extractKeyword(in.Keyword)
	if keyword == "" {
		return nil, fmt.Errorf("%w: could not extract keyword", ErrInvalidURL)
	}

	if c.curlBin == "" {
		return nil, fmt.Errorf("%w: curl-impersonate binary required for popular search", ErrUpstreamChanged)
	}

	// Fetch page HTML for tokens and SSR data
	pageURL := fmt.Sprintf("https://www.instagram.com/popular/%s/", keyword)
	html, err := internal.FetchPageWithCurl(ctx, c.curlBin, c.proxyURL, pageURL, "ig_popular_")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBlocked, err)
	}

	checkLen := internal.MinInt(5000, len(html))
	if strings.Contains(html[:checkLen], "accounts/login") && !strings.Contains(html, "xig_logged_out_popular_search") {
		return nil, ErrBlocked
	}

	tokens := internal.ExtractTokens(html)
	if tokens.LSD == "" || tokens.CSRFToken == "" {
		return nil, fmt.Errorf("%w: failed to extract session tokens", ErrUpstreamChanged)
	}

	// Extract embedded SSR data (page 1)
	dataBytes, err := internal.ExtractEmbeddedJSON(html, `"xig_logged_out_popular_search_media_info":`)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
	}

	var page1Data internal.PopularPageData
	if err := json.Unmarshal(dataBytes, &page1Data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
	}

	allPosts := make([]map[string]interface{}, 0, popularItemsPerPage*popularMaxPages)
	for _, edge := range page1Data.Edges {
		allPosts = append(allPosts, internal.NodeToMap(edge.Node))
	}

	// Fetch page 2 if available
	if page1Data.PageInfo.HasNextPage && page1Data.PageInfo.EndCursor != "" {
		variables := map[string]interface{}{
			"after":       page1Data.PageInfo.EndCursor,
			"debug":       nil,
			"keyword":     keyword,
			"media_count": popularItemsPerPage,
		}
		variablesJSON, _ := json.Marshal(variables)

		formData := internal.BuildPopularFormData(popularGraphQLDocID, variablesJSON, tokens)
		referer := fmt.Sprintf("https://www.instagram.com/popular/%s/", keyword)

		output, err := internal.FetchGraphQLWithCurl(ctx, c.curlBin, c.proxyURL, popularGraphQLEndpoint, formData, tokens, referer, "ig_popular_gql_")
		if err == nil {
			if page2Data, err := internal.ParsePopularGraphQLResponse(output); err == nil {
				for _, edge := range page2Data.Edges {
					allPosts = append(allPosts, internal.NodeToMap(edge.Node))
				}
			}
		}
	}

	if in.Limit > 0 && in.Limit < len(allPosts) {
		allPosts = allPosts[:in.Limit]
	}

	return &types.SearchPopularOutput{
		Keyword:    keyword,
		TotalCount: len(allPosts),
		Posts:      allPosts,
	}, nil
}

func extractKeyword(input string) string {
	input = strings.TrimSpace(input)

	if !strings.Contains(input, "/") && !strings.Contains(input, "http") {
		return input
	}

	u, err := url.Parse(input)
	if err != nil {
		return ""
	}

	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) >= 2 && parts[0] == "popular" {
		return parts[1]
	}

	return ""
}
