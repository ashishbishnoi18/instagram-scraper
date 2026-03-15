package types

// GetProfileOutput holds the result of a profile fetch.
type GetProfileOutput struct {
	Username  string                 `json:"username"`
	IsPrivate bool                   `json:"is_private"`
	Data      map[string]interface{} `json:"data"`
}

// GetPostOutput holds the result of a post fetch.
type GetPostOutput struct {
	Shortcode string                 `json:"shortcode"`
	Data      map[string]interface{} `json:"data"`
}

// ProfilePostItem is a single post in a profile posts stream.
type ProfilePostItem struct {
	Data map[string]interface{} `json:"data"`
}

// ProfilePostsSummary is the summary returned after streaming profile posts.
type ProfilePostsSummary struct {
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	TotalPosts int    `json:"total_posts"`
	Emitted    int    `json:"emitted"`
	HasMore    bool   `json:"has_more"`
}

// ReelItem is a single reel in a reels stream.
type ReelItem struct {
	Data map[string]interface{} `json:"data"`
}

// ReelsSummary is the summary returned after streaming reels.
type ReelsSummary struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Emitted  int    `json:"emitted"`
	HasMore  bool   `json:"has_more"`
}

// SearchPopularOutput holds the result of a popular search.
type SearchPopularOutput struct {
	Keyword    string                   `json:"keyword"`
	TotalCount int                      `json:"total_count"`
	Posts      []map[string]interface{} `json:"posts"`
}

// ListTopicsOutput holds the result of listing topics.
type ListTopicsOutput struct {
	Topics []TopicInfo `json:"topics"`
	Count  int         `json:"count"`
}

// TopicInfo represents a single topic.
type TopicInfo struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Category string `json:"category"`
	URL      string `json:"url"`
}

// SearchTopicOutput holds the result of a topic search.
type SearchTopicOutput struct {
	TopicID    string                   `json:"topic_id"`
	TopicSlug  string                   `json:"topic_slug"`
	TopicName  string                   `json:"topic_name"`
	Category   string                   `json:"category"`
	TotalCount int                      `json:"total_count"`
	Posts      []map[string]interface{} `json:"posts"`
}

// SearchHashtagOutput holds the result of a hashtag search.
type SearchHashtagOutput struct {
	Hashtag    string                   `json:"hashtag"`
	HashtagID  string                   `json:"hashtag_id"`
	TotalCount int                      `json:"total_count"`
	Posts      []map[string]interface{} `json:"posts"`
}
