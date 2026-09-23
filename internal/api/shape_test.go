package api

import (
	"testing"
)

type shapeItem struct {
	ID string `json:"id"`
}

func TestUnmarshalObject(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"wrapped", `{"episode":{"id":"ep_1","title":"A"}}`, "ep_1"},
		{"bare with object field", `{"object":"episode","id":"ep_2","title":"B"}`, "ep_2"},
		{"bare without object field", `{"id":"ep_3"}`, "ep_3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got shapeItem
			if err := UnmarshalObject([]byte(tt.body), "episode", &got); err != nil {
				t.Fatalf("UnmarshalObject: %v", err)
			}
			if got.ID != tt.want {
				t.Errorf("ID = %q, want %q", got.ID, tt.want)
			}
		})
	}
}

func TestUnmarshalObject_RejectsNonObject(t *testing.T) {
	var got shapeItem
	if err := UnmarshalObject([]byte(`[1,2]`), "episode", &got); err == nil {
		t.Error("expected an error for a JSON array")
	}
}

func TestUnmarshalList(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"keyed", `{"episodes":[{"id":"ep_1"},{"id":"ep_2"}]}`, []string{"ep_1", "ep_2"}},
		{"keyed empty", `{"episodes":[]}`, []string{}},
		{"list object", `{"object":"list","data":[{"id":"ep_3"}],"has_more":false}`, []string{"ep_3"}},
		{"list object empty", `{"object":"list","data":[],"has_more":false}`, []string{}},
		{"no items", `{}`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []shapeItem
			if err := UnmarshalList([]byte(tt.body), "episodes", &got); err != nil {
				t.Fatalf("UnmarshalList: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i, id := range tt.want {
				if got[i].ID != id {
					t.Errorf("[%d].ID = %q, want %q", i, got[i].ID, id)
				}
			}
		})
	}
}
