package scraper

import (
	"net/http"
	"time"
)

// Client is the Instagram scraper client.
type Client struct {
	http     *http.Client
	proxyURL string
	curlBin  string
}

// New creates a new Instagram scraper client.
func New(opts ...Option) (*Client, error) {
	c := &Client{
		http: &http.Client{Timeout: 90 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}
