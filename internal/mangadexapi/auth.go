package mangadexapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const authURL = "https://auth.mangadex.org/realms/mangadex/protocol/openid-connect/token"

func Authenticate(username, password, clientID, clientSecret string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", username)
	form.Set("password", password)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	resp, err := http.PostForm(authURL, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth failed: %s", resp.Status)
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

func RefreshToken(refreshToken, clientID, clientSecret string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	resp, err := http.PostForm(authURL, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed: %s", resp.Status)
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}
