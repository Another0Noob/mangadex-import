//go:build dev
// +build dev

package main

import (
	"net/http"
)

// ServeWeb returns an http.Handler that serves web files from the directory
func ServeWeb() http.Handler {
	return http.FileServer(http.Dir("frontend"))
}
