package server

import (
	"log"
	"net/http"
)

// logAllRequests is a middleware that logs all incoming requests
func logAllRequests(next http.Handler, debug bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if debug {
			log.Printf("Received request for: %s", r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}
