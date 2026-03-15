package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// SessionTokens holds extracted session tokens from Instagram page HTML.
type SessionTokens struct {
	LSD       string
	CSRFToken string
	HSI       string
	Rev       string
}

var (
	lsdRe  = regexp.MustCompile(`"LSD",\[\],\{"token":"([^"]+)"`)
	csrfRe = regexp.MustCompile(`"csrf_token":"([^"]+)"`)
	hsiRe  = regexp.MustCompile(`"hsi":"(\d+)"`)
	revRe  = regexp.MustCompile(`"revision":(\d+)`)
)

// ExtractTokens extracts session tokens from HTML page source.
func ExtractTokens(html string) SessionTokens {
	tokens := SessionTokens{}
	if match := lsdRe.FindStringSubmatch(html); len(match) > 1 {
		tokens.LSD = match[1]
	}
	if match := csrfRe.FindStringSubmatch(html); len(match) > 1 {
		tokens.CSRFToken = match[1]
	}
	if match := hsiRe.FindStringSubmatch(html); len(match) > 1 {
		tokens.HSI = match[1]
	}
	if match := revRe.FindStringSubmatch(html); len(match) > 1 {
		tokens.Rev = match[1]
	}
	return tokens
}

// FetchPageWithCurl fetches a page using curl-impersonate with proxy, writing to temp file.
func FetchPageWithCurl(ctx context.Context, curlBin, proxyURL, pageURL, prefix string) (string, error) {
	tmpFile, err := os.CreateTemp("", prefix+"*.html")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	args := []string{
		"-s", "-L",
		"-H", "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"-H", "Accept-Language: en-US,en;q=0.9",
		"-o", tmpPath,
		pageURL,
	}

	if proxyURL != "" {
		args = append([]string{"-x", proxyURL}, args...)
	}

	cmd := exec.CommandContext(ctx, curlBin, args...)
	cmd.Env = os.Environ()
	_ = cmd.Run() // Ignore exit code - curl-impersonate may return non-zero on success

	output, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to read curl output: %w", err)
	}

	if len(output) < 10000 {
		return "", fmt.Errorf("response too small, likely blocked: %d bytes", len(output))
	}

	return string(output), nil
}

// FetchGraphQLWithCurl performs a GraphQL POST request via curl-impersonate.
func FetchGraphQLWithCurl(ctx context.Context, curlBin, proxyURL, endpoint string, formData url.Values, tokens SessionTokens, referer, prefix string) ([]byte, error) {
	cookies := fmt.Sprintf("csrftoken=%s", tokens.CSRFToken)

	args := []string{
		"-s",
		"-X", "POST",
		"-H", "Content-Type: application/x-www-form-urlencoded",
		"-H", fmt.Sprintf("X-CSRFToken: %s", tokens.CSRFToken),
		"-H", fmt.Sprintf("X-IG-App-ID: %s", IGAppID),
		"-H", fmt.Sprintf("X-FB-LSD: %s", tokens.LSD),
		"-H", "X-Requested-With: XMLHttpRequest",
		"-H", "Origin: https://www.instagram.com",
		"-H", fmt.Sprintf("Referer: %s", referer),
		"-H", "Accept: */*",
		"-b", cookies,
		"-d", formData.Encode(),
		endpoint,
	}

	if proxyURL != "" {
		args = append([]string{"-x", proxyURL}, args...)
	}

	tmpFile, err := os.CreateTemp("", prefix+"*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	args = append(args, "-o", tmpPath)

	cmd := exec.CommandContext(ctx, curlBin, args...)
	cmd.Env = os.Environ()
	_ = cmd.Run()

	output, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read graphql response: %w", err)
	}

	return output, nil
}

// ExtractEmbeddedJSON finds a JSON object in HTML by marker, using brace-matching.
func ExtractEmbeddedJSON(html, marker string) ([]byte, error) {
	startIdx := strings.Index(html, marker)
	if startIdx == -1 {
		return nil, fmt.Errorf("marker %q not found", marker)
	}

	searchStart := startIdx + len(marker)
	braceCount := 0
	endIdx := searchStart

	for i := searchStart; i < len(html); i++ {
		switch html[i] {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				endIdx = i + 1
			}
		}
		if braceCount == 0 && endIdx > searchStart {
			break
		}
	}

	return []byte(html[searchStart:endIdx]), nil
}

// PopularPageData holds parsed data from popular/topic search pages.
type PopularPageData struct {
	Edges    []PopularEdge `json:"edges"`
	PageInfo struct {
		HasNextPage bool   `json:"has_next_page"`
		EndCursor   string `json:"end_cursor"`
	} `json:"page_info"`
}

// PopularEdge is a single edge in popular/topic results.
type PopularEdge struct {
	Node   PopularNode `json:"node"`
	Cursor string      `json:"cursor"`
}

// PopularNode is the node data for a popular/topic result item.
type PopularNode struct {
	Typename  string `json:"__typename"`
	ID        string `json:"id"`
	Code      string `json:"code"`
	Caption   *struct {
		Text string `json:"text"`
	} `json:"caption"`
	DisplayURI    string `json:"display_uri"`
	PlayCount     int64  `json:"play_count"`
	LikeCount     int64  `json:"like_count"`
	CommentCount  int64  `json:"comment_count"`
	User          *struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		ProfilePicURL string `json:"profile_pic_url"`
		IsVerified    bool   `json:"is_verified"`
	} `json:"user"`
	VideoVersions []struct {
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"video_versions"`
}

// NodeToMap converts a PopularNode to a response map.
func NodeToMap(node PopularNode) map[string]interface{} {
	mediaType := "Video"
	if strings.Contains(node.Typename, "Image") {
		mediaType = "Image"
	}

	postURL := fmt.Sprintf("https://www.instagram.com/reel/%s/", node.Code)
	if mediaType == "Image" {
		postURL = fmt.Sprintf("https://www.instagram.com/p/%s/", node.Code)
	}

	result := map[string]interface{}{
		"type":        mediaType,
		"id":          strings.TrimPrefix(node.ID, "POLARIS_"),
		"shortcode":   node.Code,
		"url":         postURL,
		"display_url": node.DisplayURI,
		"play_count":  node.PlayCount,
	}

	if node.LikeCount > 0 {
		result["like_count"] = node.LikeCount
	}
	if node.CommentCount > 0 {
		result["comment_count"] = node.CommentCount
	}

	if node.Caption != nil {
		result["caption"] = node.Caption.Text
	}

	if node.User != nil {
		result["user"] = map[string]interface{}{
			"id":              node.User.ID,
			"username":        node.User.Username,
			"profile_pic_url": node.User.ProfilePicURL,
			"is_verified":     node.User.IsVerified,
		}
	}

	if len(node.VideoVersions) > 0 {
		result["video_versions"] = node.VideoVersions
	}

	return result
}

// BuildPopularFormData builds the form data for popular/topic GraphQL pagination.
func BuildPopularFormData(docID string, variablesJSON []byte, tokens SessionTokens) url.Values {
	return url.Values{
		"av":                       {"0"},
		"__d":                      {"www"},
		"__user":                   {"0"},
		"__a":                      {"1"},
		"__req":                    {"1"},
		"dpr":                      {"1"},
		"__ccg":                    {"EXCELLENT"},
		"__rev":                    {tokens.Rev},
		"__hsi":                    {tokens.HSI},
		"__comet_req":              {"7"},
		"lsd":                      {tokens.LSD},
		"fb_api_caller_class":      {"RelayModern"},
		"fb_api_req_friendly_name": {"PolarisLoggedOutPopularSearchPagePaginationQuery"},
		"variables":                {string(variablesJSON)},
		"doc_id":                   {docID},
	}
}

// ParsePopularGraphQLResponse parses a popular/topic search GraphQL response.
func ParsePopularGraphQLResponse(data []byte) (*PopularPageData, error) {
	var response struct {
		Data struct {
			XIGLoggedOutPopularSearchMediaInfo *PopularPageData `json:"xig_logged_out_popular_search_media_info"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse graphql response: %w", err)
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s (code: %d)", response.Errors[0].Message, response.Errors[0].Code)
	}

	if response.Data.XIGLoggedOutPopularSearchMediaInfo == nil {
		return nil, fmt.Errorf("no data in graphql response")
	}

	return response.Data.XIGLoggedOutPopularSearchMediaInfo, nil
}
