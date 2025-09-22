package mangadexapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

func (c *Client) GetMangaList(ctx context.Context, qp QueryParams) ([]Manga, error) {
	params := qp.ToValues()
	var list []Manga
	if err := c.doData(ctx, http.MethodGet, "/manga", params, nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}
func (c *Client) GetManga(ctx context.Context, id uuid.UUID, qp QueryParams) (*Manga, error) {
	params := qp.ToValues()
	params.Del("id")

	var m Manga
	if err := c.doData(ctx, http.MethodGet, "/manga/"+id.String(), params, nil, &m); err != nil {
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

func (c *Client) CheckFollowedManga(ctx context.Context, id uuid.UUID, qp QueryParams) (bool, error) {
	params := qp.ToValues()
	params.Del("id")
	err := c.doCheck(ctx, http.MethodGet, "/user/follows/manga/"+id.String(), params)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return false, err
}
