//go:build !dev

package main

import (
	"net/http"

	"github.com/Another0Noob/mangadex-import/web"
)

func main() {
	mux := http.NewServeMux()
	web.HandleBack(mux)
	web.HandleFront(mux)
	web.RunServer(mux)
}
