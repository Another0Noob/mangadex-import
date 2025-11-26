package mangadexapi

import (
	"context"
	"errors"
	"fmt"
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

func (c *Client) GetFollowedMangaList(ctx context.Context, qp QueryParams) ([]Manga, Stats, error) {
	params := qp.ToValues()
	env, _, err := c.doEnvelope(ctx, http.MethodGet, "/user/follows/manga", params, nil)
	if err != nil {
		return nil, Stats{}, err
	}
	if env == nil || len(env.Data) == 0 { // tolerate empty data
		return nil, Stats{}, nil
	}
	var s Stats
	s.Limit = *env.Limit
	s.Offset = *env.Offset
	s.Total = *env.Total

	var list []Manga
	if err := decodeData(env.Data, &list); err != nil {
		return nil, Stats{}, fmt.Errorf("decode data: %w", err)
	}
	return list, s, nil
}

type Stats struct {
	Limit  int
	Offset int
	Total  int
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

	var wrapper struct {
		Statuses map[string]ReadingStatus `json:"statuses"`
	}
	if err := c.doInto(ctx, http.MethodGet, "/manga/status", params, nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Statuses, nil
}

func (c *Client) GetMangaStatus(ctx context.Context, id string) (*ReadingStatus, error) {
	var wrapper struct {
		Status ReadingStatus `json:"status"`
	}
	if err := c.doInto(ctx, http.MethodGet, "/manga/"+id+"/status", nil, nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Status, nil
}

func (c *Client) FollowManga(ctx context.Context, id string) error {
	if err := c.doCheck(ctx, http.MethodPost, "/manga/"+id+"/follow", nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) UpdateMangaStatus(ctx context.Context, id string, status ReadingStatus) error {
	if status == "" {
		return fmt.Errorf("empty status")
	}
	body := struct {
		Status ReadingStatus `json:"status"`
	}{Status: status}
	var dummy struct{}
	if err := c.doInto(ctx, http.MethodPost, "/manga/"+id+"/status", nil, body, &dummy); err != nil {
		return err
	}
	return nil
}
