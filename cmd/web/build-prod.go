//go:build !dev

package main

import (
	"fmt"
	"net/http"

	"github.com/Another0Noob/mangadex-import/web"
)

func startServer() error {
	mux := http.NewServeMux()
	web.HandleBack(mux)
	if err := web.HandleFront(mux); err != nil {
		return err
	}
	fmt.Println("Server running at: http://localhost:39039/")
	if err := http.ListenAndServe(":39039", mux); err != nil {
		return err
	}
	return nil
}
