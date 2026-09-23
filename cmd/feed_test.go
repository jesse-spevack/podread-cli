package cmd

import "testing"

func TestFeed_ReadsBothShapes(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"flat", `{"feed_url":"https://podread.app/feeds/abc.xml"}`},
		{"with object", `{"object":"feed","feed_url":"https://podread.app/feeds/abc.xml"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serveJSON(t, tt.body)
			out := captureOutput(t, feedCmd)

			if err := runFeed(feedCmd, nil); err != nil {
				t.Fatalf("runFeed: %v", err)
			}

			want := "https://podread.app/feeds/abc.xml\n"
			if out.String() != want {
				t.Errorf("output = %q, want %q", out.String(), want)
			}
		})
	}
}
