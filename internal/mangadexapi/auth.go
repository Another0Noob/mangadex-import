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

	"gopkg.in/ini.v1"
)

const authURL = "https://auth.mangadex.org/realms/mangadex/protocol/openid-connect/token"

func (c *Client) LoadAuth(path string) error {
	var m AuthForm
	cfg, err := ini.Load(path)
	if err != nil {
		return fmt.Errorf("load auth config %s: %w", path, err)
	}
	sec := cfg.Section("mangadex")
	m.Username = sec.Key("username").String()
	m.Password = sec.Key("password").String()
	m.ClientID = sec.Key("client_id").String()
	m.ClientSecret = sec.Key("client_secret").String()
	c.auth = m
	return nil
}

func (c *Client) Authenticate(ctx context.Context) error {
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", c.auth.Username)
	form.Set("password", c.auth.Password)
	form.Set("client_id", c.auth.ClientID)
	form.Set("client_secret", c.auth.ClientSecret)

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

func (c *Client) RefreshToken(ctx context.Context) error {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", c.token.RefreshToken)
	form.Set("client_id", c.auth.ClientID)
	form.Set("client_secret", c.auth.ClientSecret)

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

func (c *Client) EnsureToken(ctx context.Context) error {
	if time.Until(c.token.Expiry) < time.Minute {
		err := c.RefreshToken(ctx)
		if err != nil {
			return fmt.Errorf("refresh token: %w", err)
		}
	}
	return nil
}
