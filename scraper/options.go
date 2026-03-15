package scraper

import "net/http"

// Option configures the Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for API requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.http = c
	}
}

// WithProxyURL sets the proxy URL for per-request session rotation.
func WithProxyURL(url string) Option {
	return func(cl *Client) {
		cl.proxyURL = url
	}
}

// WithCurlBinPath sets the path to the curl-impersonate-chrome binary.
func WithCurlBinPath(path string) Option {
	return func(cl *Client) {
		cl.curlBin = path
	}
}
