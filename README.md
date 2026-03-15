# Instagram Scraper Module

A modulemaker-compliant Instagram scraper supporting profiles, posts, reels, popular content, topics, and hashtags.

## Capabilities

| ID | Method | Type | Description |
|----|--------|------|-------------|
| `instagram.profile.get` | `GetProfile` | sync | Fetch profile metadata |
| `instagram.post.get` | `GetPost` | sync | Fetch post by shortcode |
| `instagram.profile-posts.list` | `ListProfilePosts` | stream | Stream profile timeline posts |
| `instagram.reels.list` | `ListReels` | stream | Stream profile reels |
| `instagram.popular.search` | `SearchPopular` | sync | Search popular/trending content |
| `instagram.topics.list` | `ListTopics` | sync | List explore topics (static) |
| `instagram.topic.search` | `SearchTopic` | sync | Search topic content |
| `instagram.hashtag.search` | `SearchHashtag` | sync | Search hashtag content |

## Usage

```go
client, _ := scraper.New(
    scraper.WithProxyURL("http://user:pass@proxy:port"),
    scraper.WithCurlBinPath("/path/to/curl-impersonate-chrome"),
)
result, err := client.GetProfile(ctx, &types.GetProfileInput{URL: "instagram.com/natgeo"})
```
