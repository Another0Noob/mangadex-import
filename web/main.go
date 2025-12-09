package main

import (
	"fmt"
	"net/http"

	"github.com/Another0Noob/mangadex-import/web/backend"
)

func main() {
	runApi()
	http.Handle("/", ServeWeb())
	fmt.Println("Server running at: http://localhost:8080/")
	http.ListenAndServe(":8080", nil)
}

func runApi() {
	api := backend.NewMangaAPI()

	http.HandleFunc("/api/follow", api.HandleFollow)
	http.HandleFunc("/api/progress", api.HandleProgress)
	http.HandleFunc("/api/cancel", api.HandleCancel)
	http.HandleFunc("/api/queue", api.HandleQueue)
	http.HandleFunc("/api/queue/subscribe", api.HandleQueueSubscribe)
}
