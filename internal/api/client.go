// Package api provides an HTTP client for the podread.app API.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jspevack/podread-cli/internal/config"
)

const (
	// maxResponseSize is the maximum allowed HTTP response body size (10 MB).
	maxResponseSize = 10 * 1024 * 1024
)

// Build info variables, set at build time via ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

const (
	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 30 * time.Second
)

// Client is an HTTP client for the podread API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (%d)", e.StatusCode)
}

// NewClient creates a new API client. If token is empty, requests are sent
// without authentication (used for the device code flow).
func NewClient(token string) *Client {
	return &Client{
		baseURL: config.BaseURL(),
		token:   token,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 0 && req.URL.Host != via[0].URL.Host {
					req.Header.Del("Authorization")
				}
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// NewClientWithTimeout creates a new API client with a custom timeout.
func NewClientWithTimeout(token string, timeout time.Duration) *Client {
	c := NewClient(token)
	c.httpClient.Timeout = timeout
	return c
}

// Get performs an authenticated GET request and decodes the JSON response.
func (c *Client) Get(path string, result interface{}) error {
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

// Post performs an authenticated POST request with a JSON body and decodes
// the JSON response.
func (c *Client) Post(path string, body interface{}, result interface{}) error {
	req, err := c.newRequest(http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

// PostFile performs an authenticated multipart POST request. It sends each
// field as a form value and the file at filePath as the "file" part, then
// decodes the JSON response.
func (c *Client) PostFile(path string, fields map[string]string, filePath string, result interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		if value == "" {
			continue
		}
		if err := writer.WriteField(name, value); err != nil {
			return fmt.Errorf("encoding form field %s: %w", name, err)
		}
	}
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("encoding file part: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("encoding multipart body: %w", err)
	}

	req, err := c.newRequest(http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(&body)
	req.ContentLength = int64(body.Len())
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return c.do(req, result)
}

// Delete performs an authenticated DELETE request.
func (c *Client) Delete(path string) error {
	req, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "podread-cli/"+Version)
	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return req, nil
}

func (c *Client) do(req *http.Request, result interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			apiErr.Message = errResp.Error
		}
		return apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}
