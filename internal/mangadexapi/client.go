package mangadexapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	if err := c.rateLimiter.Wait(nil); err != nil {
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

// parseResponse parses the HTTP response and decodes the JSON into the provided result interface.
func parseResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Attempt to parse error message from response body
		var apiError struct {
			Result  string `json:"result"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &apiError); err == nil && apiError.Message != "" {
			return fmt.Errorf("API error: %s", apiError.Message)
		}
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return nil
}

// buildQueryParams constructs URL query parameters from a map.
func buildQueryParams(params map[string]string) url.Values {
	values := url.Values{}
	for key, value := range params {
		if strings.TrimSpace(value) != "" {
			values.Set(key, value)
		}
	}
	return values
}

// addQueryParam adds a single query parameter to the URL values if the value is not empty.
func addQueryParam(values url.Values, key, value string) {
	if strings.TrimSpace(value) != "" {
		values.Set(key, value)
	}
}

// addQueryParams adds multiple query parameters to the URL values from a map.
func addQueryParams(values url.Values, params map[string]string) {
	for key, value := range params {
		addQueryParam(values, key, value)
	}
}

// addListQueryParam adds a list of values as a comma-separated query parameter.
func addListQueryParam(values url.Values, key string, list []string) {
	if len(list) > 0 {
		values.Set(key, strings.Join(list, ","))
	}
}

// addIntQueryParam adds an integer query parameter if the value is greater than zero.
func addIntQueryParam(values url.Values, key string, value int) {
	if value > 0 {
		values.Set(key, fmt.Sprintf("%d", value))
	}
}

// addBoolQueryParam adds a boolean query parameter.
func addBoolQueryParam(values url.Values, key string, value bool) {
	values.Set(key, fmt.Sprintf("%t", value))
}
