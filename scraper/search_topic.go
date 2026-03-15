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

// SearchTopic scrapes content for a known Instagram explore topic.
func (c *Client) SearchTopic(ctx context.Context, in *types.SearchTopicInput) (*types.SearchTopicOutput, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}

	topic := resolveTopic(in.Topic)
	if topic == nil {
		return nil, fmt.Errorf("%w: could not resolve topic", ErrInvalidURL)
	}

	if c.curlBin == "" {
		return nil, fmt.Errorf("%w: curl-impersonate binary required for topic search", ErrUpstreamChanged)
	}

	// Fetch a page to get session tokens
	pageURL := fmt.Sprintf("https://www.instagram.com/popular/%s/", topic.Slug)
	html, err := internal.FetchPageWithCurl(ctx, c.curlBin, c.proxyURL, pageURL, "ig_topic_")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBlocked, err)
	}

	tokens := internal.ExtractTokens(html)
	if tokens.LSD == "" || tokens.CSRFToken == "" {
		return nil, fmt.Errorf("%w: failed to extract session tokens", ErrUpstreamChanged)
	}

	// Fetch page 1 via GraphQL
	page1Data, err := fetchTopicGraphQLPage(ctx, c.curlBin, c.proxyURL, topic.Slug, "", tokens)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
	}

	allPosts := make([]map[string]interface{}, 0, popularItemsPerPage*popularMaxPages)
	for _, edge := range page1Data.Edges {
		allPosts = append(allPosts, internal.NodeToMap(edge.Node))
	}

	// Fetch page 2 if available
	if page1Data.PageInfo.HasNextPage && page1Data.PageInfo.EndCursor != "" {
		if page2Data, err := fetchTopicGraphQLPage(ctx, c.curlBin, c.proxyURL, topic.Slug, page1Data.PageInfo.EndCursor, tokens); err == nil {
			for _, edge := range page2Data.Edges {
				allPosts = append(allPosts, internal.NodeToMap(edge.Node))
			}
		}
	}

	if in.Limit > 0 && in.Limit < len(allPosts) {
		allPosts = allPosts[:in.Limit]
	}

	return &types.SearchTopicOutput{
		TopicID:    topic.ID,
		TopicSlug:  topic.Slug,
		TopicName:  topic.Name,
		Category:   topic.Category,
		TotalCount: len(allPosts),
		Posts:      allPosts,
	}, nil
}

func resolveTopic(input string) *internal.Topic {
	input = strings.TrimSpace(input)

	if topic := internal.GetTopicBySlug(input); topic != nil {
		return topic
	}
	if topic := internal.GetTopicByID(input); topic != nil {
		return topic
	}

	if strings.Contains(input, "/") {
		u, err := url.Parse(input)
		if err != nil {
			return nil
		}
		path := strings.Trim(u.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) >= 4 && parts[0] == "explore" && parts[1] == "topics" {
			topicID := parts[2]
			topicSlug := parts[3]
			if topic := internal.GetTopicByID(topicID); topic != nil {
				return topic
			}
			return &internal.Topic{
				ID:   topicID,
				Slug: topicSlug,
				Name: topicSlug,
				URL:  fmt.Sprintf("/explore/topics/%s/%s/", topicID, topicSlug),
			}
		}
	}

	return nil
}

func fetchTopicGraphQLPage(ctx context.Context, curlBin, proxyURL, keyword, cursor string, tokens internal.SessionTokens) (*internal.PopularPageData, error) {
	variables := map[string]interface{}{
		"debug":       nil,
		"keyword":     keyword,
		"media_count": popularItemsPerPage,
	}
	if cursor != "" {
		variables["after"] = cursor
	}
	variablesJSON, _ := json.Marshal(variables)

	formData := internal.BuildPopularFormData(popularGraphQLDocID, variablesJSON, tokens)
	referer := fmt.Sprintf("https://www.instagram.com/popular/%s/", keyword)

	output, err := internal.FetchGraphQLWithCurl(ctx, curlBin, proxyURL, popularGraphQLEndpoint, formData, tokens, referer, "ig_topic_gql_")
	if err != nil {
		return nil, err
	}

	return internal.ParsePopularGraphQLResponse(output)
}
