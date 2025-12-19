package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/Another0Noob/mangadex-import/web/backend"
)

//go:embed all:frontend-vite/dist
var distFS embed.FS

var isDev = flag.Bool("dev", false, "Enable development mode")

func main() {
	flag.Parse()

	mux := http.NewServeMux()

	// Setup API routes
	runApi(mux)

	// Setup Vite/Frontend handling
	handler, err := createViteHandler()
	if err != nil {
		fmt.Printf("Error creating Vite handler: %v\n", err)
		panic(err)
	}
	mux.Handle("/", handler)

	fmt.Printf("Server running at: http://localhost:3939/ (dev mode: %v)\n", *isDev)
	if err := http.ListenAndServe(":3939", mux); err != nil {
		panic(err)
	}
}

func runApi(mux *http.ServeMux) {
	api := backend.NewMangaAPI()
	mux.HandleFunc("/api/follow", api.HandleFollow)
	mux.HandleFunc("/api/progress", api.HandleProgress)
	mux.HandleFunc("/api/cancel", api.HandleCancel)
	mux.HandleFunc("/api/queue", api.HandleQueue)
	mux.HandleFunc("/api/queue/subscribe", api.HandleQueueSubscribe)
}

// DistFS returns the embedded dist filesystem for production
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

func devProxy() http.Handler {
	target, _ := url.Parse("http://localhost:5173")

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ModifyResponse = func(resp *http.Response) error {
		// Optional: allow HMR through Go
		resp.Header.Set("Access-Control-Allow-Origin", "*")
		return nil
	}

	return proxy
}

// createViteHandler creates the appropriate Vite handler based on mode
func createViteHandler() (http.Handler, error) {
	if *isDev {
		http.Handle("/", devProxy())
	}

	return http.FileServer(http.FS(DistFS())), nil
}
