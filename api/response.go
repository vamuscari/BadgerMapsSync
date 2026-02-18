package api

import "net/http"

// APIResponse contains parsed response data and raw HTTP details.
type APIResponse[T any] struct {
	Data       T
	Raw        []byte
	StatusCode int
	Headers    http.Header
}
