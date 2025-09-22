package mangadexapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

func (c *Client) GetMangaList(ctx context.Context, qp QueryParams) ([]Manga, error) {
	params := qp.ToValues()
	var list []Manga
	if err := c.doData(ctx, http.MethodGet, "/manga", params, nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}
func (c *Client) GetManga(ctx context.Context, id string, qp QueryParams) (*Manga, error) {
	params := qp.ToValues()
	params.Del("id")

	var m Manga
	if err := c.doData(ctx, http.MethodGet, "/manga/"+id, params, nil, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *Client) GetFollowedMangaList(ctx context.Context, qp QueryParams) ([]Manga, error) {
	params := qp.ToValues()
	var list []Manga
	if err := c.doData(ctx, http.MethodGet, "/user/follows/manga", params, nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) CheckFollowedManga(ctx context.Context, id string, qp QueryParams) (bool, error) {
	params := qp.ToValues()
	params.Del("id")
	err := c.doCheck(ctx, http.MethodGet, "/user/follows/manga/"+id, params)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return false, err
}

func (c *Client) GetMangaStatusList(ctx context.Context, qp QueryParams) (map[string]ReadingStatus, error) {
	params := qp.ToValues()
	params.Del("id")

	m := make(map[string]ReadingStatus)
	_, b, err := c.doEnvelope(ctx, http.MethodGet, "/manga/status", params, nil)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Statuses json.RawMessage `json:"statuses"`
	}
	if err := json.Unmarshal(b, &wrapper); err != nil {
		return nil, err
	}
	if len(wrapper.Statuses) > 0 {
		if err := json.Unmarshal(wrapper.Statuses, &m); err != nil {
			return nil, err
		}
	}
	return m, nil
}
