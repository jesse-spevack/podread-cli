package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPathEscape(t *testing.T) {
	// Verify that url.PathEscape handles IDs with special characters.
	tests := []struct {
		input string
		want  string
	}{
		{"abc123", "abc123"},
		{"id/with/slashes", "id%2Fwith%2Fslashes"},
		{"id with spaces", "id%20with%20spaces"},
		{"id?query=yes", "id%3Fquery=yes"},
		{"100%done", "100%25done"},
	}

	for _, tt := range tests {
		got := url.PathEscape(tt.input)
		if got != tt.want {
			t.Errorf("PathEscape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name   string
		err    APIError
		want   string
	}{
		{
			name: "with message",
			err:  APIError{StatusCode: 404, Message: "not found"},
			want: "API error (404): not found",
		},
		{
			name: "without message",
			err:  APIError{StatusCode: 500},
			want: "API error (500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClient_Do_ParsesJSON(t *testing.T) {
	type testResp struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testResp{Name: "test", ID: 42})
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: &http.Client{},
	}

	var got testResp
	if err := c.Get("/test", &got); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Name != "test" || got.ID != 42 {
		t.Errorf("Get result = %+v, want {Name:test ID:42}", got)
	}
}

func TestClient_Do_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "episode not found"})
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: &http.Client{},
	}

	var result map[string]interface{}
	err := c.Get("/episodes/missing", &result)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Message != "episode not found" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "episode not found")
	}
}

func TestClient_Do_SetsAuthHeader(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		token:      "my-secret",
		httpClient: &http.Client{},
	}

	c.Get("/test", nil)

	if gotAuth != "Bearer my-secret" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer my-secret")
	}
}

func TestClient_Do_NoAuthWhenEmpty(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		token:      "",
		httpClient: &http.Client{},
	}

	c.Get("/test", nil)

	if gotAuth != "" {
		t.Errorf("Authorization header = %q, want empty", gotAuth)
	}
}

func TestClient_Do_LimitsResponseBody(t *testing.T) {
	// Serve a response larger than maxResponseSize.
	largeBody := strings.Repeat("x", maxResponseSize+1000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, largeBody)
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		token:      "",
		httpClient: &http.Client{},
	}

	// Request without a result struct — just reading the body.
	// The body gets read (up to limit) but since we don't unmarshal,
	// it should not error. The key is that it doesn't try to read
	// the full (oversized) body into memory.
	err := c.Get("/large", nil)
	if err != nil {
		t.Fatalf("unexpected error for large response with nil result: %v", err)
	}
}

func TestClient_CheckRedirect_StripsAuth(t *testing.T) {
	// Create two servers: one that redirects to the other.
	var gotAuth string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/final", http.StatusFound)
	}))
	defer redirector.Close()

	c := &Client{
		baseURL: redirector.URL,
		token:   "secret-token",
		httpClient: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 0 && req.URL.Host != via[0].URL.Host {
					req.Header.Del("Authorization")
				}
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}

	c.Get("/redirect", nil)

	if gotAuth != "" {
		t.Errorf("Authorization header leaked to different host: %q", gotAuth)
	}
}

func TestClient_Post_SendsJSON(t *testing.T) {
	type reqBody struct {
		Name string `json:"name"`
	}
	type respBody struct {
		ID string `json:"id"`
	}

	var gotContentType string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respBody{ID: "ep-123"})
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		token:      "tok",
		httpClient: &http.Client{},
	}

	var result respBody
	if err := c.Post("/episodes", reqBody{Name: "test"}, &result); err != nil {
		t.Fatalf("Post: %v", err)
	}

	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}

	var parsed reqBody
	if err := json.Unmarshal(gotBody, &parsed); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if parsed.Name != "test" {
		t.Errorf("request body Name = %q, want %q", parsed.Name, "test")
	}
	if result.ID != "ep-123" {
		t.Errorf("response ID = %q, want %q", result.ID, "ep-123")
	}
}
