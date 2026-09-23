package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
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
		name string
		err  APIError
		want string
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
	tests := []struct {
		name        string
		body        string
		wantType    string
		wantCode    string
		wantMessage string
		wantParam   string
	}{
		{
			name:        "error object",
			body:        `{"error":{"type":"invalid_request_error","code":"resource_missing","message":"No episode with that id.","param":"id"}}`,
			wantType:    "invalid_request_error",
			wantCode:    "resource_missing",
			wantMessage: "No episode with that id.",
			wantParam:   "id",
		},
		{
			name:        "error object with extra data",
			body:        `{"error":{"type":"payment_error","code":"insufficient_credits","message":"Buy credits.","credits_remaining":0,"upgrade_url":"https://podread.app/upgrade"}}`,
			wantType:    "payment_error",
			wantCode:    "insufficient_credits",
			wantMessage: "Buy credits.",
		},
		{
			name: "string error",
			body: `{"error":"episode not found"}`,
		},
		{
			name: "no body",
			body: "",
		},
		{
			name: "html body",
			body: "<html>not found</html>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				io.WriteString(w, tt.body)
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
			if apiErr.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", apiErr.Type, tt.wantType)
			}
			if apiErr.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", apiErr.Code, tt.wantCode)
			}
			if apiErr.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", apiErr.Message, tt.wantMessage)
			}
			if apiErr.Param != tt.wantParam {
				t.Errorf("Param = %q, want %q", apiErr.Param, tt.wantParam)
			}
		})
	}
}

func TestAPIError_HasCode(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want bool
	}{
		{"code field", APIError{StatusCode: 400, Code: "authorization_pending", Message: "The user has not approved the code."}, true},
		{"code as the message only", APIError{StatusCode: 400, Message: "authorization_pending"}, false},
		{"other code", APIError{StatusCode: 400, Code: "expired_token", Message: "authorization_pending was earlier"}, false},
		{"empty", APIError{StatusCode: 400}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.HasCode("authorization_pending"); got != tt.want {
				t.Errorf("HasCode = %v, want %v", got, tt.want)
			}
		})
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

func TestClient_PostFile_SendsMultipartForm(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "report.pdf")
	if err := os.WriteFile(filePath, []byte("%PDF-1.7 body"), 0o600); err != nil {
		t.Fatal(err)
	}

	var gotAuth, gotSourceType, gotFilename, gotContent string
	var parseErr error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The client must send the body again after a 307 redirect.
		if r.URL.Path == "/old/episodes" {
			http.Redirect(w, r, "/api/v1/episodes", http.StatusTemporaryRedirect)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		if parseErr = r.ParseMultipartForm(1 << 20); parseErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		gotSourceType = r.FormValue("source_type")
		file, header, err := r.FormFile("file")
		if err != nil {
			parseErr = err
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		gotFilename, gotContent = header.Filename, string(data)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"ep_123"}`))
	}))
	defer server.Close()

	c := &Client{baseURL: server.URL, token: "test-token", httpClient: &http.Client{}}

	var got struct {
		ID string `json:"id"`
	}
	if err := c.PostFile("/old/episodes", map[string]string{"source_type": "file"}, filePath, &got); err != nil {
		t.Fatalf("PostFile: %v (server parse error: %v)", err, parseErr)
	}
	if got.ID != "ep_123" {
		t.Errorf("ID = %q, want ep_123", got.ID)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotSourceType != "file" {
		t.Errorf("source_type = %q, want file", gotSourceType)
	}
	if gotFilename != "report.pdf" || gotContent != "%PDF-1.7 body" {
		t.Errorf("file = %q with %q", gotFilename, gotContent)
	}
}

func TestClient_PostFile_MissingFile(t *testing.T) {
	c := &Client{baseURL: "http://127.0.0.1:0", httpClient: &http.Client{}}

	err := c.PostFile("/api/v1/episodes", nil, filepath.Join(t.TempDir(), "missing.pdf"), nil)
	if err == nil || !strings.Contains(err.Error(), "opening file") {
		t.Errorf("err = %v, want opening file error", err)
	}
}
