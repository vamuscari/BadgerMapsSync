package server

import (
	"badgermaps/app"
	appserver "badgermaps/app/server"
	"badgermaps/database"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

func WebhookLoggingMiddleware(next http.Handler, a *app.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Config.Server.LogRequests {
			next.ServeHTTP(w, r)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, appserver.MaxWebhookBodyBytes)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "can't read body", http.StatusInternalServerError)
			return
		}

		// Restore the body so the next handler can read it
		r.Body = io.NopCloser(bytes.NewReader(body))

		headers, _ := json.Marshal(r.Header)

		database.LogWebhook(a.DB, time.Now(), r.Method, r.RequestURI, string(headers), string(body))

		next.ServeHTTP(w, r)
	})
}
