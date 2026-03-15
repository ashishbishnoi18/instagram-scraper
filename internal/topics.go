package internal

// Topic represents an Instagram explore topic.
type Topic struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Category string `json:"category"`
	URL      string `json:"url"`
}

// AllTopics contains all discovered Instagram explore topics.
var AllTopics = []Topic{
	{ID: "10155994924430727", Slug: "music-audio", Name: "Music & Audio", Category: "Music", URL: "/explore/topics/10155994924430727/music-audio/"},
	{ID: "1004784969545988", Slug: "country-music", Name: "Country Music", Category: "Music", URL: "/explore/topics/1004784969545988/country-music/"},
	{ID: "943656532390416", Slug: "hip-hop-rap-music", Name: "Hip Hop & Rap", Category: "Music", URL: "/explore/topics/943656532390416/hip-hop-rap-music/"},
	{ID: "206741089697034", Slug: "k-pop-music", Name: "K-Pop", Category: "Music", URL: "/explore/topics/206741089697034/k-pop-music/"},
	{ID: "10156104410190727", Slug: "fashion-beauty", Name: "Fashion & Beauty", Category: "Fashion", URL: "/explore/topics/10156104410190727/fashion-beauty/"},
	{ID: "917256014984692", Slug: "photography", Name: "Photography", Category: "Photography", URL: "/explore/topics/917256014984692/photography/"},
	{ID: "1283274535024498", Slug: "sports", Name: "Sports", Category: "Sports", URL: "/explore/topics/1283274535024498/sports/"},
	{ID: "611682099330820", Slug: "gaming", Name: "Gaming", Category: "Games", URL: "/explore/topics/611682099330820/gaming/"},
	{ID: "2381024672140869", Slug: "tv-movies", Name: "TV & Movies", Category: "TV & Movies", URL: "/explore/topics/2381024672140869/tv-movies/"},
}

// TopicsByCategory groups topics by category.
func TopicsByCategory() map[string][]Topic {
	result := make(map[string][]Topic)
	for _, t := range AllTopics {
		result[t.Category] = append(result[t.Category], t)
	}
	return result
}

// GetTopicBySlug looks up a topic by slug.
func GetTopicBySlug(slug string) *Topic {
	for i := range AllTopics {
		if AllTopics[i].Slug == slug {
			return &AllTopics[i]
		}
	}
	return nil
}

// GetTopicByID looks up a topic by ID.
func GetTopicByID(id string) *Topic {
	for i := range AllTopics {
		if AllTopics[i].ID == id {
			return &AllTopics[i]
		}
	}
	return nil
}
