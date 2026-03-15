package scraper

import (
	"context"

	"github.com/embedtools/instagram-scraper/internal"
	"github.com/embedtools/instagram-scraper/types"
)

// ListTopics returns all known Instagram explore topics, optionally filtered by category.
func (c *Client) ListTopics(_ context.Context, in *types.ListTopicsInput) (*types.ListTopicsOutput, error) {
	var topics []types.TopicInfo

	if in.Category != "" {
		byCategory := internal.TopicsByCategory()
		for _, t := range byCategory[in.Category] {
			topics = append(topics, types.TopicInfo{
				ID:       t.ID,
				Slug:     t.Slug,
				Name:     t.Name,
				Category: t.Category,
				URL:      t.URL,
			})
		}
	} else {
		for _, t := range internal.AllTopics {
			topics = append(topics, types.TopicInfo{
				ID:       t.ID,
				Slug:     t.Slug,
				Name:     t.Name,
				Category: t.Category,
				URL:      t.URL,
			})
		}
	}

	return &types.ListTopicsOutput{
		Topics: topics,
		Count:  len(topics),
	}, nil
}
