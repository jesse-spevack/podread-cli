package cmd

import "testing"

func TestFormatAuthStatus(t *testing.T) {
	credits := 42
	limit := 100000

	tests := []struct {
		name string
		resp authStatusResponse
		want string
	}{
		{
			name: "unlimited tier omits quota lines",
			resp: authStatusResponse{Email: "u@example.com", Tier: "unlimited"},
			want: "Logged in as u@example.com (unlimited)",
		},
		{
			name: "credit-path user shows quota lines",
			resp: authStatusResponse{
				Email:            "f@example.com",
				Tier:             "free",
				CreditsRemaining: &credits,
				CharacterLimit:   &limit,
			},
			want: "Logged in as f@example.com (free)\nCredits remaining: 42\nCharacter limit: 100000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAuthStatus(tt.resp)
			if got != tt.want {
				t.Errorf("formatAuthStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}
