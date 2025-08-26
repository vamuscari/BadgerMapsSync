package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"

	"badgermapscli/app"
)

func withSignatureValidation(h http.HandlerFunc, App *app.State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if App.Config.WebhookSecret == "" {
			h.ServeHTTP(w, r)
			return
		}

		signature := r.Header.Get("X-BadgerMaps-Signature")
		if signature == "" {
			http.Error(w, "missing signature", http.StatusUnauthorized)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "can't read body", http.StatusInternalServerError)
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body)) // a little trick to be able to read the body again

		mac := hmac.New(sha256.New, []byte(App.Config.WebhookSecret))
		mac.Write(body)
		expectedMAC := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
			http.Error(w, fmt.Sprintf("invalid signature"), http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	}
}
