package mangadexapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
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

// doRequest performs an HTTP request to the MangaDex API (raw, no JSON decoding).
func (c *Client) doRequest(ctx context.Context, method, endpoint string, params url.Values, body interface{}) (*http.Response, error) {
	// Rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit error: %w", err)
	}

	// Build URL
	fullURL := c.baseURL + endpoint
	if len(params) > 0 {
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
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != nil {
		req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}

var ErrNotFound = errors.New("mangadex: not found")

// doEnvelope runs the request, reads body, parses (or synthesizes) an Envelope,
// normalizing 404 handling and returning ErrNotFound where appropriate.
func (c *Client) doEnvelope(ctx context.Context, method, endpoint string, params url.Values, body any) (*Envelope, []byte, error) {
	resp, err := c.doRequest(ctx, method, endpoint, params, body)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read body: %w", err)
	}

	// Fast 404 (no JSON needed)
	if resp.StatusCode == http.StatusNotFound {
		return nil, b, ErrNotFound
	}

	var env Envelope
	// Try to decode JSON regardless of status to inspect API errors
	if json.Unmarshal(b, &env) != nil {
		// If not JSON and not success -> raw error
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, b, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(b))
		}
		// If success but envelope missing -> treat as protocol error
		return nil, b, fmt.Errorf("decode envelope: not valid JSON (status %d): %s", resp.StatusCode, string(b))
	}

	// Non-2xx: inspect API errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if len(env.Errors) > 0 {
			first := env.Errors[0]
			if first.Status == http.StatusNotFound {
				return &env, b, ErrNotFound
			}
			return &env, b, fmt.Errorf("api error (%d): %s: %s", first.Status, first.Title, first.Detail)
		}
		return &env, b, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(b))
	}

	// Result-level error (even with HTTP 2xx)
	if env.Result == "error" {
		if len(env.Errors) > 0 {
			first := env.Errors[0]
			if first.Status == http.StatusNotFound {
				return &env, b, ErrNotFound
			}
			return &env, b, fmt.Errorf("api error: %s (%d): %s", first.Title, first.Status, first.Detail)
		}
		return &env, b, fmt.Errorf("api error: result=error with no details")
	}

	return &env, b, nil
}

func (c *Client) doInto(ctx context.Context, method, endpoint string, params url.Values, body any, out any) error {
	_, b, err := c.doEnvelope(ctx, method, endpoint, params, body)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// Updated doData uses shared doEnvelope
func (c *Client) doData(ctx context.Context, method, endpoint string, params url.Values, body any, out any) error {
	env, _, err := c.doEnvelope(ctx, method, endpoint, params, body)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if env == nil || len(env.Data) == 0 { // tolerate empty data
		return nil
	}
	if err := decodeData(env.Data, out); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}
	return nil
}

// Updated doCheck also uses doEnvelope
func (c *Client) doCheck(ctx context.Context, method, endpoint string, params url.Values) error {
	env, _, err := c.doEnvelope(ctx, method, endpoint, params, nil)
	if err != nil {
		return err
	}
	// If env.Result=="ok" we are good. Any other (already filtered errors) unexpected.
	if env != nil && env.Result != "ok" {
		return fmt.Errorf("unexpected result value: %q", env.Result)
	}
	return nil
}

// decodeData handles either an object or collection style automatically if target is slice.
func decodeData(raw json.RawMessage, out any) error {
	if len(raw) == 0 { // accept empty
		return nil
	}
	if err := json.Unmarshal(raw, out); err == nil {
		return nil
	}
	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &wrapper) == nil && len(wrapper.Data) > 0 {
		if err := json.Unmarshal(wrapper.Data, out); err == nil {
			return nil
		}
	}
	return fmt.Errorf("unhandled data shape: %s", string(raw))
}

// ToValues converts QueryParams to url.Values for the request.
func (q QueryParams) ToValues() url.Values {
	v := url.Values{}
	rv := reflect.ValueOf(q)
	rt := reflect.TypeOf(q)

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		tag := sf.Tag.Get("url")
		if tag == "" {
			continue
		}
		parts := strings.Split(tag, ",")
		name := parts[0]
		omitempty := false
		for _, p := range parts[1:] {
			if p == "omitempty" {
				omitempty = true
			}
		}
		fv := rv.Field(i)

		if omitempty && isZeroValue(fv) {
			continue
		}

		// Special case for order: support struct OR map[string]string
		if name == "order" {
			switch fv.Kind() {
			case reflect.Struct:
				addOrderParams(v, fv)
				continue
			case reflect.Map:
				for _, mk := range fv.MapKeys() {
					mv := fv.MapIndex(mk)
					if isZeroValue(mv) {
						continue
					}
					v.Add("order["+fmt.Sprint(mk.Interface())+"]", valueToString(mv))
				}
				continue
			}
		}

		switch fv.Kind() {
		case reflect.Slice, reflect.Array:
			if fv.Len() == 0 && omitempty {
				continue
			}
			for j := 0; j < fv.Len(); j++ {
				item := fv.Index(j)
				if isZeroValue(item) {
					continue
				}
				v.Add(name, valueToString(item))
			}
		default:
			v.Add(name, valueToString(fv))
		}
	}
	return v
}

func addOrderParams(v url.Values, orderVal reflect.Value) {
	ot := orderVal.Type()
	for i := 0; i < ot.NumField(); i++ {
		sf := ot.Field(i)
		tag := sf.Tag.Get("url")
		if tag == "" {
			continue
		}
		parts := strings.Split(tag, ",")
		fieldName := parts[0]
		omitempty := false
		for _, p := range parts[1:] {
			if p == "omitempty" {
				omitempty = true
			}
		}
		fv := orderVal.Field(i)
		if omitempty && isZeroValue(fv) {
			continue
		}
		v.Add("order["+fieldName+"]", valueToString(fv))
	}
}

func isZeroValue(v reflect.Value) bool {
	// For slices / maps / arrays / ptr / interface
	switch v.Kind() {
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	}
	// Use IsZero (Go 1.18+)
	return v.IsZero()
}

func valueToString(v reflect.Value) string {
	return anyToString(v.Interface())
}

func anyToString(a any) string {
	switch t := a.(type) {
	case fmt.Stringer:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}
