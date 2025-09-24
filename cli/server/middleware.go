package server

import (
	"badgermaps/app"
	"badgermaps/database"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

func WebhookLoggingMiddleware(next http.Handler, a *app.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Config.Server.LogRequests {
			next.ServeHTTP(w, r)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "can't read body", http.StatusInternalServerError)
			return
		}

		// Restore the body so the next handler can read it
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		headers, _ := json.Marshal(r.Header)

		database.LogWebhook(a.DB, time.Now(), r.Method, r.RequestURI, string(headers), string(body))

		next.ServeHTTP(w, r)
	})
}
