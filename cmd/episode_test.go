package cmd

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestEpisodeListPath(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		page      int
		wantLimit string
		wantPage  string
	}{
		{"defaults", 10, 1, "10", "1"},
		{"custom", 50, 3, "50", "3"},
		{"max", 100, 7, "100", "7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := episodeListPath(tt.limit, tt.page)

			if !strings.HasPrefix(got, "/api/v1/episodes?") {
				t.Errorf("path = %q, want prefix /api/v1/episodes?", got)
			}

			q, err := url.ParseQuery(strings.TrimPrefix(got, "/api/v1/episodes?"))
			if err != nil {
				t.Fatalf("ParseQuery: %v", err)
			}

			if q.Get("limit") != tt.wantLimit {
				t.Errorf("limit = %q, want %q", q.Get("limit"), tt.wantLimit)
			}
			if q.Get("page") != tt.wantPage {
				t.Errorf("page = %q, want %q", q.Get("page"), tt.wantPage)
			}
		})
	}
}

func TestEpisodeCreateRequest_JSON(t *testing.T) {
	tests := []struct {
		name     string
		req      episodeCreateRequest
		contains []string
		omits    []string
	}{
		{
			name: "url with title and author",
			req: episodeCreateRequest{
				SourceType: "url",
				URL:        "https://example.com/article",
				Title:      "An Article",
				Author:     "Jane Doe",
			},
			contains: []string{
				`"source_type":"url"`,
				`"url":"https://example.com/article"`,
				`"title":"An Article"`,
				`"author":"Jane Doe"`,
			},
			omits: []string{`"text"`, `"voice"`},
		},
		{
			name: "text with author",
			req: episodeCreateRequest{
				SourceType: "text",
				Text:       "hello world",
				Title:      "Greeting",
				Author:     "Alice",
				Voice:      "alloy",
			},
			contains: []string{
				`"source_type":"text"`,
				`"text":"hello world"`,
				`"title":"Greeting"`,
				`"author":"Alice"`,
				`"voice":"alloy"`,
			},
			omits: []string{`"url"`},
		},
		{
			name: "author omitted when empty",
			req: episodeCreateRequest{
				SourceType: "url",
				URL:        "https://example.com/article",
			},
			contains: []string{
				`"source_type":"url"`,
				`"url":"https://example.com/article"`,
			},
			omits: []string{`"author"`, `"title"`, `"text"`, `"voice"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			got := string(data)
			for _, s := range tt.contains {
				if !strings.Contains(got, s) {
					t.Errorf("body %s missing %s", got, s)
				}
			}
			for _, s := range tt.omits {
				if strings.Contains(got, s) {
					t.Errorf("body %s should omit %s", got, s)
				}
			}
		})
	}
}

func TestEpisodeCreateRequest_FormFields(t *testing.T) {
	req := episodeCreateRequest{SourceType: "file", Title: "Report", Voice: "felix"}

	got := req.formFields()

	want := map[string]string{"source_type": "file", "title": "Report", "voice": "felix"}
	if len(got) != len(want) {
		t.Errorf("formFields() = %v, want %v", got, want)
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("formFields()[%q] = %q, want %q", key, got[key], value)
		}
	}
}

func TestEpisodeStatus_PrintsEpisode(t *testing.T) {
	serveJSON(t, `{"object":"episode","id":"ep_1","title":"A Title","status":"complete","created_at":"2026-09-22T00:00:00Z"}`)
	out := captureOutput(t, episodeStatusCmd)

	if err := runEpisodeStatus(episodeStatusCmd, []string{"ep_1"}); err != nil {
		t.Fatalf("runEpisodeStatus: %v", err)
	}

	want := "ID:     ep_1\nTitle:  A Title\nStatus: complete\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}

func TestEpisodeList_ReadsListObject(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			"two episodes",
			`{"object":"list","data":[{"object":"episode","id":"ep_1","title":"One","status":"complete","created_at":""},{"object":"episode","id":"ep_2","title":"","status":"processing","created_at":""}],"has_more":false,"page":1,"limit":10,"total":2}`,
			"ID  STATUS        TITLE\nep_1  complete      One\nep_2  processing    (untitled)\n",
		},
		{"empty", `{"object":"list","data":[],"has_more":false,"page":1,"limit":10,"total":0}`, "No episodes found\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serveJSON(t, tt.body)
			out := captureOutput(t, episodeListCmd)

			if err := runEpisodeList(episodeListCmd, nil); err != nil {
				t.Fatalf("runEpisodeList: %v", err)
			}

			if out.String() != tt.want {
				t.Errorf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestEpisodeCreate_PrintsEpisode(t *testing.T) {
	serveJSON(t, `{"object":"episode","id":"ep_1","title":"A Title","status":"pending","created_at":""}`)
	out := captureOutput(t, episodeCreateCmd)
	episodeCreateCmd.Flags().Set("url", "https://example.com/article")
	episodeCreateCmd.Flags().Set("no-wait", "true")

	if err := runEpisodeCreate(episodeCreateCmd, nil); err != nil {
		t.Fatalf("runEpisodeCreate: %v", err)
	}

	want := "ID:     ep_1\nTitle:  A Title\nStatus: pending\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}

func TestEpisodeStatus_PrintsErrorMessage(t *testing.T) {
	serveStatus(t, 404, `{"error":{"type":"invalid_request_error","code":"resource_missing","message":"No episode with that id.","param":"id"}}`)
	captureOutput(t, episodeStatusCmd)

	err := runEpisodeStatus(episodeStatusCmd, []string{"ep_1"})
	if err == nil {
		t.Fatal("expected an error")
	}

	want := "fetching episode: API error (404): No episode with that id."
	if err.Error() != want {
		t.Errorf("err = %q, want %q", err.Error(), want)
	}
}
