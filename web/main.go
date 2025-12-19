package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/Another0Noob/mangadex-import/web/backend"
	"github.com/olivere/vite"
)

//go:embed all:frontend-vite/dist
var distFS embed.FS

var isDev = flag.Bool("dev", false, "Enable development mode")

func main() {
	flag.Parse()

	// Setup API routes
	runApi()

	// Setup Vite/Frontend handling
	handler, err := createViteHandler()
	if err != nil {
		panic(err)
	}
	http.Handle("/", handler)

	fmt.Printf("Server running at: http://localhost:3939/ (dev mode: %v)\n", *isDev)
	http.ListenAndServe(":3939", nil)
}

func runApi() {
	api := backend.NewMangaAPI()
	http.HandleFunc("/api/follow", api.HandleFollow)
	http.HandleFunc("/api/progress", api.HandleProgress)
	http.HandleFunc("/api/cancel", api.HandleCancel)
	http.HandleFunc("/api/queue", api.HandleQueue)
	http.HandleFunc("/api/queue/subscribe", api.HandleQueueSubscribe)
}

// DistFS returns the embedded dist filesystem for production
func DistFS() fs.FS {
	efs, err := fs.Sub(distFS, "frontend-vite/dist")
	if err != nil {
		panic(fmt.Sprintf("unable to serve frontend: %v", err))
	}
	return efs
}

// createViteHandler creates the appropriate Vite handler based on mode
func createViteHandler() (http.Handler, error) {
	if *isDev {
		// Development mode: proxy to Vite dev server
		return vite.NewHandler(vite.Config{
			FS:        os.DirFS("./frontend-vite"),        // Source directory
			IsDev:     true,                               // Enable dev mode
			PublicFS:  os.DirFS("./frontend-vite/public"), // Optional: public assets
			ViteURL:   "http://localhost:5173",            // Vite dev server
			ViteEntry: "src/main.ts",                      // Entry point
		})
	}

	// Production mode: serve embedded files
	return vite.NewHandler(vite.Config{
		FS:        DistFS(),      // Embedded dist directory
		IsDev:     false,         // Disable dev mode
		ViteEntry: "src/main.ts", // Entry point
	})
}
