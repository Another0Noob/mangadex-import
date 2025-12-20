package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/Another0Noob/mangadex-import/web/backend"
)

//go:embed all:frontend-vite/dist
var distFS embed.FS

func DistFS() fs.FS {
	efs, err := fs.Sub(distFS, "frontend-vite/dist")
	if err != nil {
		// Try without the "frontend/" prefix
		efs, err = fs.Sub(distFS, "dist")
		if err != nil {
			panic(fmt.Sprintf("unable to serve frontend: %v", err))
		}
	}
	return efs
}

func RunApi(mux *http.ServeMux) {
	api := backend.NewMangaAPI()
	mux.HandleFunc("/api/follow", api.HandleFollow)
	mux.HandleFunc("/api/progress", api.HandleProgress)
	mux.HandleFunc("/api/cancel", api.HandleCancel)
	mux.HandleFunc("/api/queue", api.HandleQueue)
	mux.HandleFunc("/api/queue/subscribe", api.HandleQueueSubscribe)
}
