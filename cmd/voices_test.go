package cmd

import "testing"

func TestVoices_ReadsBothShapes(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			"keyed",
			`{"voices":[{"id":"alloy","name":"Alloy","accent":"US","gender":"female"}]}`,
			"ID           NAME        ACCENT      GENDER\nalloy        Alloy       US          female\n",
		},
		{
			"list object",
			`{"object":"list","data":[{"object":"voice","id":"alloy","name":"Alloy","accent":"US","gender":"female"}],"has_more":false}`,
			"ID           NAME        ACCENT      GENDER\nalloy        Alloy       US          female\n",
		},
		{"keyed empty", `{"voices":[]}`, "No voices available\n"},
		{"list object empty", `{"object":"list","data":[],"has_more":false}`, "No voices available\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serveJSON(t, tt.body)
			out := captureOutput(t, voicesCmd)

			if err := runVoices(voicesCmd, nil); err != nil {
				t.Fatalf("runVoices: %v", err)
			}

			if out.String() != tt.want {
				t.Errorf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}
