//go:build dev

package main

import (
	"net/http"

	"github.com/Another0Noob/mangadex-import/web"
)

func startServer() error {
	mux := http.NewServeMux()
	web.RunApi(mux)
	if err := http.ListenAndServe(":39039", mux); err != nil {
		return err
	}
	return nil
}
