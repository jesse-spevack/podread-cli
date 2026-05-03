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
