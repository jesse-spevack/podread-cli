package cmd

import "testing"

func TestFeed_PrintsFeedURL(t *testing.T) {
	serveJSON(t, `{"object":"feed","feed_url":"https://podread.app/feeds/abc.xml"}`)
	out := captureOutput(t, feedCmd)

	if err := runFeed(feedCmd, nil); err != nil {
		t.Fatalf("runFeed: %v", err)
	}

	want := "https://podread.app/feeds/abc.xml\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}
