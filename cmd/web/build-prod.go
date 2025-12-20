//go:build !dev

package main

import (
	"fmt"
	"net/http"

	"github.com/Another0Noob/mangadex-import/web"
)

func startServer() error {
	mux := http.NewServeMux()
	web.RunApi(mux)
	mux.Handle("/", http.FileServer(http.FS(web.DistFS())))
	fmt.Println("Server running at: http://localhost:39039/")
	if err := http.ListenAndServe(":39039", mux); err != nil {
		return err
	}
	return nil
}
