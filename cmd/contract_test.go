package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	defaultOpenAPIURL = "https://podread.app/api/v1/openapi.json"
	specTimeout       = 15 * time.Second
)

// TestAPIContract compares every json tag the CLI sends or reads against the
// published OpenAPI spec. Run `go test -short` to skip it when offline.
func TestAPIContract(t *testing.T) {
	if testing.Short() {
		t.Skip("the contract check needs the live OpenAPI spec")
	}

	spec := fetchSpec(t)

	tests := []struct {
		name  string
		value any
		path  []string
	}{
		{
			"episode",
			episodeResponse{},
			[]string{"components", "schemas", "Episode", "properties"},
		},
		{
			"createEpisode request",
			episodeCreateRequest{},
			[]string{"paths", "/api/v1/episodes", "post", "requestBody", "content", "application/json", "schema", "properties"},
		},
		{
			"getEpisode response",
			episodeShowResponse{},
			[]string{"paths", "/api/v1/episodes/{id}", "get", "responses", "200", "content", "application/json", "schema", "properties"},
		},
		{
			"listEpisodes response",
			episodeListResponse{},
			[]string{"paths", "/api/v1/episodes", "get", "responses", "200", "content", "application/json", "schema", "properties"},
		},
		{
			"listVoices response",
			voicesListResponse{},
			[]string{"paths", "/api/v1/voices", "get", "responses", "200", "content", "application/json", "schema", "properties"},
		},
		{
			"voice",
			voiceResponse{},
			[]string{"paths", "/api/v1/voices", "get", "responses", "200", "content", "application/json", "schema", "properties", "voices", "items", "properties"},
		},
		{
			"getFeed response",
			feedResponse{},
			[]string{"paths", "/api/v1/feed", "get", "responses", "200", "content", "application/json", "schema", "properties"},
		},
		{
			"getAuthStatus response",
			authStatusResponse{},
			[]string{"paths", "/api/v1/auth/status", "get", "responses", "200", "content", "application/json", "schema", "properties"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties := resolve(t, spec, tt.path)

			for _, tag := range jsonTags(tt.value) {
				if _, ok := properties[tag]; !ok {
					t.Errorf("the CLI uses %q, which the spec does not document at %s", tag, strings.Join(tt.path, "."))
				}
			}
		})
	}
}

func fetchSpec(t *testing.T) map[string]any {
	t.Helper()

	url := defaultOpenAPIURL
	if override := os.Getenv("PODREAD_OPENAPI_URL"); override != "" {
		url = override
	}

	client := &http.Client{Timeout: specTimeout}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("fetching the spec from %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fetching the spec from %s: status %d, want 200", url, resp.StatusCode)
	}

	var spec map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		t.Fatalf("decoding the spec from %s: %v", url, err)
	}

	return spec
}

func jsonTags(value any) []string {
	structType := reflect.TypeOf(value)

	var tags []string
	for i := range structType.NumField() {
		name, _, _ := strings.Cut(structType.Field(i).Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		tags = append(tags, name)
	}

	return tags
}

func resolve(t *testing.T, spec map[string]any, path []string) map[string]any {
	t.Helper()

	node := spec
	for i, key := range path {
		child, ok := node[key].(map[string]any)
		if !ok {
			t.Fatalf("the spec has no object at %s", strings.Join(path[:i+1], "."))
		}
		node = deref(t, spec, child)
	}

	return node
}

func deref(t *testing.T, spec map[string]any, node map[string]any) map[string]any {
	t.Helper()

	ref, ok := node["$ref"].(string)
	if !ok {
		return node
	}
	if !strings.HasPrefix(ref, "#/") {
		t.Fatalf("the spec uses an external reference: %s", ref)
	}

	target := spec
	for _, key := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		child, ok := target[key].(map[string]any)
		if !ok {
			t.Fatalf("the spec cannot resolve %s", ref)
		}
		target = child
	}

	return target
}
