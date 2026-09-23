package cmd

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jspevack/podread-cli/internal/api"
)

const (
	defaultOpenAPIURL = "https://podread.app/api/v1/openapi.json"
	specTimeout       = 15 * time.Second
	fetchHint         = "run `go test -short ./...` to skip the contract check, or set PODREAD_OPENAPI_URL to another spec"
)

type contractCase struct {
	name  string
	value any
	path  []string
}

// Each path goes through an operation, so a spec that repoints one at a thinner schema fails.
var contractCases = []contractCase{
	{
		"createEpisode request",
		episodeCreateRequest{},
		[]string{"paths", "/api/v1/episodes", "post", "requestBody", "content", "application/json", "schema", "properties"},
	},
	{
		"createEpisode response",
		episodeResponse{},
		[]string{"paths", "/api/v1/episodes", "post", "responses", "201", "content", "application/json", "schema", "properties"},
	},
	{
		"getEpisode response",
		episodeResponse{},
		[]string{"paths", "/api/v1/episodes/{id}", "get", "responses", "200", "content", "application/json", "schema", "properties"},
	},
	{
		"listEpisodes response",
		episodeListResponse{},
		[]string{"paths", "/api/v1/episodes", "get", "responses", "200", "content", "application/json", "schema", "properties"},
	},
	{
		"listEpisodes episode",
		episodeResponse{},
		[]string{"paths", "/api/v1/episodes", "get", "responses", "200", "content", "application/json", "schema", "properties", "data", "items", "properties"},
	},
	{
		"listVoices response",
		voicesListResponse{},
		[]string{"paths", "/api/v1/voices", "get", "responses", "200", "content", "application/json", "schema", "properties"},
	},
	{
		"voice",
		voiceResponse{},
		[]string{"paths", "/api/v1/voices", "get", "responses", "200", "content", "application/json", "schema", "properties", "data", "items", "properties"},
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
	{
		"error",
		api.ErrorDetail{},
		[]string{"paths", "/api/v1/episodes", "post", "responses", "422", "content", "application/json", "schema", "properties", "error", "properties"},
	},
}

// notInSpec names the structs the contract check cannot cover, and why.
var notInSpec = map[string]string{
	"deviceCodeResponse":  "the spec documents no POST /api/v1/auth/device_codes",
	"deviceTokenRequest":  "the spec documents no POST /api/v1/auth/device_tokens",
	"deviceTokenResponse": "the spec documents no POST /api/v1/auth/device_tokens",
}

func TestAPIContract(t *testing.T) {
	if testing.Short() {
		t.Skip("the contract check needs the live OpenAPI spec")
	}

	spec := fetchSpec(t)

	for _, tt := range contractCases {
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

func TestContractCoversEveryStruct(t *testing.T) {
	covered := make(map[string]bool, len(contractCases))
	for _, tt := range contractCases {
		covered[reflect.TypeOf(tt.value).Name()] = true
	}

	for _, name := range structsWithJSONTags(t) {
		if covered[name] || notInSpec[name] != "" {
			continue
		}
		t.Errorf("%s carries json tags but no contract case, add one to contractCases or list it in notInSpec with a reason", name)
	}
}

func structsWithJSONTags(t *testing.T) []string {
	t.Helper()

	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("listing the cmd package: %v", err)
	}

	fset := token.NewFileSet()
	var names []string
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}

		ast.Inspect(file, func(node ast.Node) bool {
			spec, ok := node.(*ast.TypeSpec)
			if !ok {
				return true
			}
			structType, ok := spec.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range structType.Fields.List {
				if field.Tag != nil && strings.Contains(field.Tag.Value, `json:"`) {
					names = append(names, spec.Name.Name)
					return false
				}
			}
			return true
		})
	}

	return names
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
		t.Fatalf("fetching the spec from %s: %v\n%s", url, err, fetchHint)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fetching the spec from %s: status %d, want 200\n%s", url, resp.StatusCode, fetchHint)
	}

	var spec map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		t.Fatalf("decoding the spec from %s: %v\n%s", url, err, fetchHint)
	}

	return spec
}

func jsonTags(value any) []string {
	structType := reflect.TypeOf(value)

	var tags []string
	for i := range structType.NumField() {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}

		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
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
