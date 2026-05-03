package cmd

import (
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
