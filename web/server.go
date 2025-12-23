package web

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

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
}

func HandleFront(mux *http.ServeMux) {
	efs, err := fs.Sub(distFS, "frontend-vite/dist")
	if err != nil {
		log.Fatalf("Couldn't serve front: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(efs)))
}

func RunServer(mux *http.ServeMux) {
	server := &http.Server{
		Addr:    ":39039",
		Handler: mux,
	}

	go func() {
		log.Println("Starting server on http://localhost:39039/")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen: %v\n", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Step 6: Gracefully shut down the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
