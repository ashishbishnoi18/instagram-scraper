package types

// GetProfileInput holds input for fetching an Instagram profile.
type GetProfileInput struct {
	URL    string   `json:"url"`
	Fields []string `json:"fields,omitempty"`
}

// GetPostInput holds input for fetching an Instagram post.
type GetPostInput struct {
	URL string `json:"url"`
}

// ListProfilePostsInput holds input for listing profile posts.
type ListProfilePostsInput struct {
	URL      string `json:"url"`
	Limit    int    `json:"limit,omitempty"`
	Since    int64  `json:"since,omitempty"`
	MaxPages int    `json:"max_pages,omitempty"`
}

// ListReelsInput holds input for listing profile reels.
type ListReelsInput struct {
	URL      string `json:"url"`
	Limit    int    `json:"limit,omitempty"`
	Since    int64  `json:"since,omitempty"`
	MaxPages int    `json:"max_pages,omitempty"`
}

// SearchPopularInput holds input for searching popular content.
type SearchPopularInput struct {
	Keyword string `json:"keyword"`
	Limit   int    `json:"limit,omitempty"`
}

// ListTopicsInput holds input for listing explore topics.
type ListTopicsInput struct {
	Category string `json:"category,omitempty"`
}

// SearchTopicInput holds input for searching topic content.
type SearchTopicInput struct {
	Topic string `json:"topic"` // slug, ID, or URL
	Limit int    `json:"limit,omitempty"`
}

// SearchHashtagInput holds input for searching hashtag content.
type SearchHashtagInput struct {
	Hashtag string `json:"hashtag"`
	Limit   int    `json:"limit,omitempty"`
}
