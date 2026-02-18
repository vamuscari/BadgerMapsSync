package api

import (
	"badgermaps/api/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type requestCapture struct {
	Method        string
	Path          string
	RawQuery      string
	Authorization string
	ContentType   string
	Body          string
	Form          url.Values
}

// Harness helpers.
func captureRequest(t *testing.T, r *http.Request) requestCapture {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}

	form, _ := url.ParseQuery(string(body))
	return requestCapture{
		Method:        r.Method,
		Path:          r.URL.Path,
		RawQuery:      r.URL.RawQuery,
		Authorization: r.Header.Get("Authorization"),
		ContentType:   r.Header.Get("Content-Type"),
		Body:          string(body),
		Form:          form,
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, payload); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func newTestAPIServer(t *testing.T, routes map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		if handler, ok := routes[key]; ok {
			handler(w, r)
			return
		}
		keyWithQuery := fmt.Sprintf("%s %s", r.Method, r.URL.RequestURI())
		if handler, ok := routes[keyWithQuery]; ok {
			handler(w, r)
			return
		}
		http.NotFound(w, r)
	}))
}

func newTestClient(serverURL string) *APIClient {
	return &APIClient{
		BaseURL:   serverURL,
		APIKey:    "test-key",
		client:    http.DefaultClient,
		endpoints: NewEndpoints(serverURL),
	}
}

func assertRequestBasics(t *testing.T, got requestCapture, wantMethod, wantPath, wantContentType string) {
	t.Helper()
	if got.Method != wantMethod {
		t.Fatalf("method: got %s want %s", got.Method, wantMethod)
	}
	if got.Path != wantPath {
		t.Fatalf("path: got %s want %s", got.Path, wantPath)
	}
	if got.Authorization != "Token test-key" {
		t.Fatalf("authorization: got %q", got.Authorization)
	}
	if wantContentType != "" && got.ContentType != wantContentType {
		t.Fatalf("content-type: got %q want %q", got.ContentType, wantContentType)
	}
}

// Shared internal helper tests.
func TestEncodeFormData(t *testing.T) {
	encoded := encodeFormData(map[string]string{"extra_fields[Meeting Notes]": "hello world"})
	vals, err := url.ParseQuery(encoded)
	if err != nil {
		t.Fatalf("parse encoded form: %v", err)
	}
	if vals.Get("extra_fields[Meeting Notes]") != "hello world" {
		t.Fatalf("unexpected form encoding: %s", encoded)
	}
}

func TestHasNonNullValue(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		valid bool
	}{
		{name: "empty", in: "", valid: false},
		{name: "spaces", in: "   ", valid: false},
		{name: "null", in: "null", valid: false},
		{name: "nil", in: "nil", valid: false},
		{name: "value", in: "abc", valid: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasNonNullValue(tt.in); got != tt.valid {
				t.Fatalf("hasNonNullValue(%q) got %v want %v", tt.in, got, tt.valid)
			}
		})
	}
}

func TestApplyAuthHeaders(t *testing.T) {
	client := newTestClient("https://example.com")
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	client.applyAuthHeaders(req, "application/json")
	if got := req.Header.Get("Authorization"); got != "Token test-key" {
		t.Fatalf("authorization got %q", got)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type got %q", got)
	}
}

func TestResponsePreview(t *testing.T) {
	if got := responsePreview([]byte("abcdef"), 4); got != "abcd..." {
		t.Fatalf("preview got %q", got)
	}
	if got := responsePreview([]byte("abc"), 4); got != "abc" {
		t.Fatalf("preview got %q", got)
	}
}

func TestDoJSON(t *testing.T) {
	server := newTestAPIServer(t, map[string]http.HandlerFunc{
		"GET /ok": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "1")
			writeJSON(t, w, http.StatusOK, `{"id":7}`)
		},
		"GET /bad-status": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusTeapot, `{"error":"nope"}`)
		},
		"GET /bad-json": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, `{`)
		},
	})
	defer server.Close()

	client := newTestClient(server.URL)

	t.Run("success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/ok", nil)
		res, err := doJSON[struct {
			ID int `json:"id"`
		}](client, req, http.StatusOK, "decode failed")
		if err != nil {
			t.Fatalf("doJSON error: %v", err)
		}
		if res.Data.ID != 7 {
			t.Fatalf("id got %d", res.Data.ID)
		}
		if res.Headers.Get("X-Test") != "1" {
			t.Fatalf("header clone missing")
		}
		if string(res.Raw) != `{"id":7}` {
			t.Fatalf("raw body mismatch")
		}
	})

	t.Run("status mismatch", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/bad-status", nil)
		_, err := doJSON[map[string]any](client, req, http.StatusOK, "decode failed")
		if err == nil || !strings.Contains(err.Error(), "unexpected status") {
			t.Fatalf("expected unexpected status error, got %v", err)
		}
	})

	t.Run("decode error", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/bad-json", nil)
		_, err := doJSON[map[string]any](client, req, http.StatusOK, "decode failed")
		if err == nil || !strings.Contains(err.Error(), "decode failed") {
			t.Fatalf("expected decode error, got %v", err)
		}
	})
}

// Endpoint tests.
func TestAPIClient_GetAccounts(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK, body: `[{"id":101}]`},
		{name: "status error", statusCode: http.StatusInternalServerError, body: `{"error":"boom"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq requestCapture
			server := newTestAPIServer(t, map[string]http.HandlerFunc{
				"GET /customers/": func(w http.ResponseWriter, r *http.Request) {
					gotReq = captureRequest(t, r)
					writeJSON(t, w, tt.statusCode, tt.body)
				},
			})
			defer server.Close()

			client := newTestClient(server.URL)
			res, err := client.GetAccounts()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetAccounts error: %v", err)
			}
			assertRequestBasics(t, gotReq, http.MethodGet, "/customers/", "application/json")
			if len(res.Data) != 1 || !res.Data[0].AccountId.Valid || res.Data[0].AccountId.Int64 != 101 {
				t.Fatalf("unexpected accounts payload: %+v", res.Data)
			}
		})
	}
}

func TestAPIClient_GetAccountIDs(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK, body: `[{"id":1},{"id":2}]`},
		{name: "bad status", statusCode: http.StatusBadRequest, body: `{"error":"bad"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq requestCapture
			server := newTestAPIServer(t, map[string]http.HandlerFunc{
				"GET /customers/": func(w http.ResponseWriter, r *http.Request) {
					gotReq = captureRequest(t, r)
					writeJSON(t, w, tt.statusCode, tt.body)
				},
			})
			defer server.Close()

			res, err := newTestClient(server.URL).GetAccountIDs()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetAccountIDs error: %v", err)
			}
			assertRequestBasics(t, gotReq, http.MethodGet, "/customers/", "application/json")
			if len(res.Data) != 2 || res.Data[0] != 1 || res.Data[1] != 2 {
				t.Fatalf("unexpected ids: %+v", res.Data)
			}
		})
	}
}

func TestAPIClient_GetCheckinIDs(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK, body: `[{"id":9},{"id":11}]`},
		{name: "invalid json", statusCode: http.StatusOK, body: `{`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq requestCapture
			server := newTestAPIServer(t, map[string]http.HandlerFunc{
				"GET /appointments/": func(w http.ResponseWriter, r *http.Request) {
					gotReq = captureRequest(t, r)
					writeJSON(t, w, tt.statusCode, tt.body)
				},
			})
			defer server.Close()

			res, err := newTestClient(server.URL).GetCheckinIDs()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetCheckinIDs error: %v", err)
			}
			assertRequestBasics(t, gotReq, http.MethodGet, "/appointments/", "application/json")
			if len(res.Data) != 2 || res.Data[0] != 9 || res.Data[1] != 11 {
				t.Fatalf("unexpected ids: %+v", res.Data)
			}
		})
	}
}

func TestAPIClient_GetRoutes(t *testing.T) {
	server := newTestAPIServer(t, map[string]http.HandlerFunc{
		"GET /routes/": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, `[{"id":33}]`)
		},
	})
	defer server.Close()

	res, err := newTestClient(server.URL).GetRoutes()
	if err != nil {
		t.Fatalf("GetRoutes error: %v", err)
	}
	if len(res.Data) != 1 || !res.Data[0].RouteId.Valid || res.Data[0].RouteId.Int64 != 33 {
		t.Fatalf("unexpected routes: %+v", res.Data)
	}
}

func TestAPIClient_GetCheckins(t *testing.T) {
	server := newTestAPIServer(t, map[string]http.HandlerFunc{
		"GET /appointments/": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, `[{"id":44}]`)
		},
	})
	defer server.Close()

	res, err := newTestClient(server.URL).GetCheckins()
	if err != nil {
		t.Fatalf("GetCheckins error: %v", err)
	}
	if len(res.Data) != 1 || !res.Data[0].CheckinId.Valid || res.Data[0].CheckinId.Int64 != 44 {
		t.Fatalf("unexpected checkins: %+v", res.Data)
	}
}

func TestAPIClient_GetUserProfile(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK, body: `{"id":7,"datafields":[{"name":"cn"}]}`},
		{name: "invalid json", statusCode: http.StatusOK, body: `{`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq requestCapture
			server := newTestAPIServer(t, map[string]http.HandlerFunc{
				"GET /profiles/": func(w http.ResponseWriter, r *http.Request) {
					gotReq = captureRequest(t, r)
					writeJSON(t, w, tt.statusCode, tt.body)
				},
			})
			defer server.Close()

			res, err := newTestClient(server.URL).GetUserProfile()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetUserProfile error: %v", err)
			}
			assertRequestBasics(t, gotReq, http.MethodGet, "/profiles/", "application/json")
			if !res.Data.ProfileId.Valid || res.Data.ProfileId.Int64 != 7 {
				t.Fatalf("unexpected profile id: %+v", res.Data.ProfileId)
			}
			if len(res.Data.Datafields) != 1 || res.Data.Datafields[0].AccountField.String != "CustomNumeric" {
				t.Fatalf("expected datafield mapping, got %+v", res.Data.Datafields)
			}
		})
	}
}

func TestAPIClient_TestAPIConnection(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK, body: `{"id":55}`},
		{name: "bad status", statusCode: http.StatusUnauthorized, body: `{"error":"bad"}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestAPIServer(t, map[string]http.HandlerFunc{
				"GET /profiles/": func(w http.ResponseWriter, r *http.Request) {
					writeJSON(t, w, tt.statusCode, tt.body)
				},
			})
			defer server.Close()

			client := newTestClient(server.URL)
			err := client.TestAPIConnection()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("TestAPIConnection error: %v", err)
			}
			if !client.connected || client.UserID != 55 {
				t.Fatalf("expected connected true and user id 55")
			}
		})
	}
}

func TestAPIClient_GetAccountDetailed(t *testing.T) {
	server := newTestAPIServer(t, map[string]http.HandlerFunc{
		"GET /customers/99/": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, `{"id":99}`)
		},
	})
	defer server.Close()

	res, err := newTestClient(server.URL).GetAccountDetailed(99)
	if err != nil {
		t.Fatalf("GetAccountDetailed error: %v", err)
	}
	if !res.Data.AccountId.Valid || res.Data.AccountId.Int64 != 99 {
		t.Fatalf("unexpected account: %+v", res.Data)
	}
}

func TestAPIClient_UpdateAccount(t *testing.T) {
	tests := []struct {
		name    string
		input   models.AccountUpload
		status  int
		wantErr bool
	}{
		{name: "success", input: models.AccountUpload{Fields: map[string]string{"last_name": "Updated"}}, status: http.StatusOK},
		{name: "nil fields", input: models.AccountUpload{}, status: http.StatusOK},
		{name: "status error", input: models.AccountUpload{Fields: map[string]string{"last_name": "Updated"}}, status: http.StatusBadRequest, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq requestCapture
			server := newTestAPIServer(t, map[string]http.HandlerFunc{
				"PATCH /customers/99/": func(w http.ResponseWriter, r *http.Request) {
					gotReq = captureRequest(t, r)
					writeJSON(t, w, tt.status, `{"id":99,"last_name":"Updated"}`)
				},
			})
			defer server.Close()

			res, err := newTestClient(server.URL).UpdateAccount(99, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateAccount error: %v", err)
			}
			assertRequestBasics(t, gotReq, http.MethodPatch, "/customers/99/", "application/x-www-form-urlencoded")
			if tt.input.Fields != nil && tt.input.Fields["last_name"] != "" && gotReq.Form.Get("last_name") != "Updated" {
				t.Fatalf("expected form field last_name, got %s", gotReq.Body)
			}
			if !res.Data.AccountId.Valid || res.Data.AccountId.Int64 != 99 {
				t.Fatalf("unexpected account response: %+v", res.Data)
			}
		})
	}
}

func TestAPIClient_DeleteAccount(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{name: "no content", status: http.StatusNoContent},
		{name: "ok", status: http.StatusOK},
		{name: "error", status: http.StatusBadRequest, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestAPIServer(t, map[string]http.HandlerFunc{
				"DELETE /customers/12/": func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.status)
					_, _ = io.WriteString(w, `{"error":"bad"}`)
				},
			})
			defer server.Close()
			err := newTestClient(server.URL).DeleteAccount(12)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAPIClient_CreateAccount(t *testing.T) {
	tests := []struct {
		name    string
		input   models.AccountUpload
		status  int
		wantErr bool
	}{
		{name: "success", input: models.AccountUpload{Fields: map[string]string{"last_name": "Created"}}, status: http.StatusCreated},
		{name: "nil fields", input: models.AccountUpload{}, status: http.StatusCreated},
		{name: "status error", input: models.AccountUpload{Fields: map[string]string{"last_name": "Created"}}, status: http.StatusBadRequest, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq requestCapture
			server := newTestAPIServer(t, map[string]http.HandlerFunc{
				"POST /customers/": func(w http.ResponseWriter, r *http.Request) {
					gotReq = captureRequest(t, r)
					writeJSON(t, w, tt.status, `{"id":201}`)
				},
			})
			defer server.Close()

			res, err := newTestClient(server.URL).CreateAccount(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateAccount error: %v", err)
			}
			assertRequestBasics(t, gotReq, http.MethodPost, "/customers/", "application/x-www-form-urlencoded")
			if tt.input.Fields != nil && tt.input.Fields["last_name"] != "" && gotReq.Form.Get("last_name") != "Created" {
				t.Fatalf("expected encoded field, got %s", gotReq.Body)
			}
			if !res.Data.AccountId.Valid || res.Data.AccountId.Int64 != 201 {
				t.Fatalf("unexpected created account: %+v", res.Data)
			}
		})
	}
}

func TestAPIClient_GetRoute(t *testing.T) {
	server := newTestAPIServer(t, map[string]http.HandlerFunc{
		"GET /routes/5/": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, `{"id":5}`)
		},
	})
	defer server.Close()

	res, err := newTestClient(server.URL).GetRoute(5)
	if err != nil {
		t.Fatalf("GetRoute error: %v", err)
	}
	if !res.Data.RouteId.Valid || res.Data.RouteId.Int64 != 5 {
		t.Fatalf("unexpected route response: %+v", res.Data)
	}
}

func TestAPIClient_GetCheckinsForAccount(t *testing.T) {
	var gotReq requestCapture
	server := newTestAPIServer(t, map[string]http.HandlerFunc{
		"GET /appointments/": func(w http.ResponseWriter, r *http.Request) {
			gotReq = captureRequest(t, r)
			writeJSON(t, w, http.StatusOK, `[{"id":77,"customer":42}]`)
		},
	})
	defer server.Close()

	res, err := newTestClient(server.URL).GetCheckinsForAccount(42)
	if err != nil {
		t.Fatalf("GetCheckinsForAccount error: %v", err)
	}
	if gotReq.RawQuery != "customer_id=42" {
		t.Fatalf("expected customer_id query, got %s", gotReq.RawQuery)
	}
	if len(res.Data) != 1 || !res.Data[0].CheckinId.Valid || res.Data[0].CheckinId.Int64 != 77 {
		t.Fatalf("unexpected checkins response: %+v", res.Data)
	}
}

func TestAPIClient_CreateCheckin(t *testing.T) {
	tests := []struct {
		name      string
		input     models.CheckinUpload
		wantErr   bool
		assertReq func(t *testing.T, req requestCapture)
	}{
		{
			name:    "validation customer",
			input:   models.CheckinUpload{Customer: 0, Type: "test"},
			wantErr: true,
		},
		{
			name:    "validation type",
			input:   models.CheckinUpload{Customer: 10, Type: "  null "},
			wantErr: true,
		},
		{
			name: "success filters null fields",
			input: models.CheckinUpload{
				Customer: 10,
				Type:     " test ",
				Fields: map[string]string{
					"comments": "hello",
					"type":     "ignore",
					"customer": "ignore",
					"empty":    " ",
				},
			},
			assertReq: func(t *testing.T, req requestCapture) {
				if req.Form.Get("customer") != "10" || req.Form.Get("type") != "test" {
					t.Fatalf("missing customer/type fields: %s", req.Body)
				}
				if req.Form.Get("comments") != "hello" {
					t.Fatalf("expected comments in form: %s", req.Body)
				}
				if req.Form.Get("empty") != "" {
					t.Fatalf("empty field should be filtered")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq requestCapture
			server := newTestAPIServer(t, map[string]http.HandlerFunc{
				"POST /appointments/": func(w http.ResponseWriter, r *http.Request) {
					gotReq = captureRequest(t, r)
					writeJSON(t, w, http.StatusCreated, `{"id":300,"customer":10,"type":"test"}`)
				},
			})
			defer server.Close()

			res, err := newTestClient(server.URL).CreateCheckin(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateCheckin error: %v", err)
			}
			assertRequestBasics(t, gotReq, http.MethodPost, "/appointments/", "application/x-www-form-urlencoded")
			if tt.assertReq != nil {
				tt.assertReq(t, gotReq)
			}
			if !res.Data.CheckinId.Valid || res.Data.CheckinId.Int64 != 300 {
				t.Fatalf("unexpected checkin: %+v", res.Data)
			}
		})
	}
}

func TestAPIClient_CreateCustomCheckin(t *testing.T) {
	tests := []struct {
		name         string
		input        models.CustomCheckinUpload
		wantErr      bool
		wantContains []string
		wantExtra    map[string]string
		wantNoExtra  []string
	}{
		{name: "validation customer", input: models.CustomCheckinUpload{Customer: 0, Type: "Phone"}, wantErr: true},
		{name: "validation type", input: models.CustomCheckinUpload{Customer: 1, Type: "  "}, wantErr: true},
		{
			name:         "only raw extra_fields",
			input:        models.CustomCheckinUpload{Customer: 12, Type: "Phone", Fields: map[string]string{"extra_fields": `{"Foo":"Bar"}`}},
			wantContains: []string{"customer=12", "type=Phone", "extra_fields="},
			wantExtra:    map[string]string{"Foo": "Bar", "Log Type": "Phone"},
		},
		{
			name:         "only derived extra fields",
			input:        models.CustomCheckinUpload{Customer: 12, Type: "Phone", Fields: map[string]string{"Meeting Notes": "Discussed"}},
			wantContains: []string{"customer=12", "type=Phone", "extra_fields="},
			wantExtra:    map[string]string{"Log Type": "Phone", "Meeting Notes": "Discussed"},
		},
		{
			name: "typed extra fields",
			input: models.CustomCheckinUpload{
				Customer: 12,
				Type:     "Phone",
				ExtraFields: &models.CustomCheckinExtraFields{
					LogType:      "Inbound",
					MeetingNotes: "Discussed roadmap",
				},
			},
			wantContains: []string{"customer=12", "type=Phone", "extra_fields="},
			wantExtra:    map[string]string{"Log Type": "Inbound", "Meeting Notes": "Discussed roadmap"},
		},
		{
			name: "typed meeting notes optional",
			input: models.CustomCheckinUpload{
				Customer: 12,
				Type:     "Phone",
				ExtraFields: &models.CustomCheckinExtraFields{
					LogType: "Outbound",
				},
			},
			wantContains: []string{"customer=12", "type=Phone", "extra_fields="},
			wantExtra:    map[string]string{"Log Type": "Outbound"},
		},
		{
			name:         "merge raw and derived extra fields",
			input:        models.CustomCheckinUpload{Customer: 12, Type: "Phone", Fields: map[string]string{"extra_fields": `{"Foo":"Bar"}`, "Meeting Notes": "Discussed", "comments": "ok"}},
			wantContains: []string{"comments=ok", "extra_fields="},
			wantExtra:    map[string]string{"Foo": "Bar", "Log Type": "Phone", "Meeting Notes": "Discussed"},
		},
		{
			name: "drop empty extra fields and canonicalize keys",
			input: models.CustomCheckinUpload{
				Customer: 12,
				Type:     "Phone",
				Fields: map[string]string{
					" meeting notes ": "Discussed plan",
					"extra_fields":    `{"Meeting Notes":"   ","Legacy":"ok","  ":"skip"}`,
				},
			},
			wantContains: []string{"customer=12", "type=Phone", "extra_fields="},
			wantExtra:    map[string]string{"Log Type": "Phone", "Meeting Notes": "Discussed plan", "Legacy": "ok"},
			wantNoExtra:  []string{" meeting notes ", "  "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq requestCapture
			server := newTestAPIServer(t, map[string]http.HandlerFunc{
				"POST /appointments/": func(w http.ResponseWriter, r *http.Request) {
					gotReq = captureRequest(t, r)
					writeJSON(t, w, http.StatusCreated, `{"id":301,"customer":12,"type":"Phone"}`)
				},
			})
			defer server.Close()

			res, err := newTestClient(server.URL).CreateCustomCheckin(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateCustomCheckin error: %v", err)
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(gotReq.Body, want) {
					t.Fatalf("body %q missing %q", gotReq.Body, want)
				}
			}

			if gotReq.Form.Get("extra_fields") == "" {
				t.Fatalf("expected extra_fields in form: %s", gotReq.Body)
			}
			if !json.Valid([]byte(gotReq.Form.Get("extra_fields"))) {
				t.Fatalf("extra_fields is not valid json: %s", gotReq.Form.Get("extra_fields"))
			}
			var gotExtra map[string]interface{}
			if err := json.Unmarshal([]byte(gotReq.Form.Get("extra_fields")), &gotExtra); err != nil {
				t.Fatalf("failed to decode extra_fields: %v", err)
			}
			for key, wantVal := range tt.wantExtra {
				gotVal, ok := gotExtra[key]
				if !ok {
					t.Fatalf("extra_fields missing key %q: %+v", key, gotExtra)
				}
				gotString, ok := gotVal.(string)
				if !ok {
					t.Fatalf("extra_fields key %q is not a string: %T", key, gotVal)
				}
				if gotString != wantVal {
					t.Fatalf("extra_fields[%q] = %q, want %q", key, gotString, wantVal)
				}
			}
			for _, key := range tt.wantNoExtra {
				if _, exists := gotExtra[key]; exists {
					t.Fatalf("extra_fields should not contain key %q: %+v", key, gotExtra)
				}
			}
			if !res.Data.CheckinId.Valid || res.Data.CheckinId.Int64 != 301 {
				t.Fatalf("unexpected checkin response: %+v", res.Data)
			}
		})
	}
}

func TestAPIClient_UpdateLocation(t *testing.T) {
	tests := []struct {
		name    string
		input   models.LocationUpload
		wantErr bool
		status  int
	}{
		{name: "success", input: models.LocationUpload{Fields: map[string]string{"city": "Austin"}}, status: http.StatusOK},
		{name: "nil fields", input: models.LocationUpload{}, status: http.StatusOK},
		{name: "status error", input: models.LocationUpload{Fields: map[string]string{"city": "Austin"}}, status: http.StatusBadRequest, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq requestCapture
			server := newTestAPIServer(t, map[string]http.HandlerFunc{
				"PATCH /locations/88/": func(w http.ResponseWriter, r *http.Request) {
					gotReq = captureRequest(t, r)
					writeJSON(t, w, tt.status, `{"id":88,"city":"Austin"}`)
				},
			})
			defer server.Close()

			res, err := newTestClient(server.URL).UpdateLocation(88, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateLocation error: %v", err)
			}
			assertRequestBasics(t, gotReq, http.MethodPatch, "/locations/88/", "application/x-www-form-urlencoded")
			if tt.input.Fields != nil && tt.input.Fields["city"] != "" && gotReq.Form.Get("city") != "Austin" {
				t.Fatalf("city field missing")
			}
			if !res.Data.LocationId.Valid || res.Data.LocationId.Int64 != 88 {
				t.Fatalf("unexpected location response: %+v", res.Data)
			}
		})
	}
}

func TestNewAPIClient(t *testing.T) {
	server := newTestAPIServer(t, map[string]http.HandlerFunc{
		"GET /profiles/": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, `{"id":123}`)
		},
	})
	defer server.Close()

	client := NewAPIClient(&APIConfig{BaseURL: server.URL, APIKey: "test-key"})
	if !client.IsConnected() {
		t.Fatalf("expected client to be connected")
	}
	if client.UserID != 123 {
		t.Fatalf("expected user id 123, got %d", client.UserID)
	}
}
