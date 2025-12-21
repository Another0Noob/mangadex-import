package web

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/Another0Noob/mangadex-import/web/backend"
)

//go:embed all:frontend-vite/dist
var distFS embed.FS

func HandleBack(mux *http.ServeMux) {
	api := backend.NewMangaAPI()
	mux.HandleFunc("/api/follow", api.HandleFollow)
	mux.HandleFunc("/api/progress", api.HandleProgress)
	mux.HandleFunc("/api/cancel", api.HandleCancel)
	mux.HandleFunc("/api/queue", api.HandleQueue)
	mux.HandleFunc("/api/queue/subscribe", api.HandleQueueSubscribe)
}

func HandleFront(mux *http.ServeMux) error {
	efs, err := fs.Sub(distFS, "frontend-vite/dist")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(efs)))
	return nil
}
