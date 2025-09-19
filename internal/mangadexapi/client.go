package mangadexapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

const (
	baseURL   = "https://api.mangadex.org"
	userAgent = "MangaDex-Import/0.1 (https://github.com/Another0Noob/mangadex-import)"
)
const (
	rateLimitRequests = 5
	rateLimitDuration = time.Second
)

// Client is the MangaDex API client.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	userAgent   string
	rateLimiter *rate.Limiter

	token *Token
}

// NewClient creates a new MangaDex API client.
func NewClient() *Client {
	return &Client{
		httpClient:  &http.Client{},
		baseURL:     baseURL,
		userAgent:   userAgent,
		rateLimiter: rate.NewLimiter(rate.Every(rateLimitDuration/time.Duration(rateLimitRequests)), rateLimitRequests),
	}
}

// SetToken sets the authentication token for the client.
func (c *Client) SetToken(token *Token) {
	c.token = token
}

// doRequest performs an HTTP request to the MangaDex API.
func (c *Client) doRequest(method, endpoint string, params url.Values, body interface{}) (*http.Response, error) {
	// Rate limiting
	if err := c.rateLimiter.Wait(context.TODO()); err != nil {
		return nil, fmt.Errorf("rate limit error: %w", err)
	}

	// Build URL
	fullURL := c.baseURL + endpoint
	if params != nil {
		fullURL += "?" + params.Encode()
	}

	// Prepare body
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != nil {
		req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}

