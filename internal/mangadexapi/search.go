package mangadexapi

import (
	"context"
	"net/http"
)

func (c *Client) GetMangaList(ctx context.Context, qp QueryParams) ([]Manga, error) {
	params := qp.ToValues()
	var list []Manga
	if err := c.doJSON(ctx, http.MethodGet, "/manga", params, nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}
