package mangadexapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const authURL = "https://auth.mangadex.org/realms/mangadex/protocol/openid-connect/token"

func (c *Client) Authenticate(ctx context.Context, a AuthForm) error {
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", a.Username)
	form.Set("password", a.Password)
	form.Set("client_id", a.ClientID)
	form.Set("client_secret", a.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("refresh failed: %s: %s", resp.Status, string(b))
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return err
	}
	c.token = &token
	c.token.Expiry = time.Now().Add(15 * time.Minute)
	return nil
}

func (c *Client) RefreshToken(ctx context.Context, a AuthForm) error {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", c.token.RefreshToken)
	form.Set("client_id", a.ClientID)
	form.Set("client_secret", a.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("refresh failed: %s: %s", resp.Status, string(b))
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return err
	}
	if token.AccessToken != "" {
		c.token.AccessToken = token.AccessToken
	}
	if token.RefreshToken != "" {
		c.token.RefreshToken = token.RefreshToken
	}
	c.token.Expiry = time.Now().Add(15 * time.Minute)
	return nil
}

func (c *Client) EnsureToken(ctx context.Context, a AuthForm) error {
	if time.Until(c.token.Expiry) < time.Minute {
		err := c.RefreshToken(ctx, a)
		if err != nil {
			return fmt.Errorf("refresh token: %w", err)
		}
	}
	return nil
}
