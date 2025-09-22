package mangadexapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

func (c *Client) GetMangaList(ctx context.Context, qp QueryParams) ([]Manga, error) {
	params := qp.ToValues()
	var list []Manga
	if err := c.doJSON(ctx, http.MethodGet, "/manga", params, nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}
func (c *Client) GetManga(ctx context.Context, id uuid.UUID, qp QueryParams) (*Manga, error) {
	params := qp.ToValues()
	params.Del("id")

	var m Manga
	if err := c.doJSON(ctx, http.MethodGet, "/manga/"+id.String(), params, nil, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
