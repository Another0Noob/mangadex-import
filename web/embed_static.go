//go:build !dev
// +build !dev

package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed frontend/*
var webFiles embed.FS

// ServeWeb returns an http.Handler that serves the embedded web files
func ServeWeb() http.Handler {
	stripped, err := fs.Sub(webFiles, "frontend")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(stripped))
}
