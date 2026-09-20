package utils

import "testing"

func TestFinancialStatusFromScore(t *testing.T) {
	tests := []struct {
		name  string
		score int
		want  string
	}{
		{name: "high upper", score: 100, want: "High"},
		{name: "high lower", score: 75, want: "High"},
		{name: "average upper", score: 74, want: "Average"},
		{name: "average lower", score: 50, want: "Average"},
		{name: "low upper", score: 49, want: "Low"},
		{name: "low zero", score: 0, want: "Low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FinancialStatusFromScore(tt.score); got != tt.want {
				t.Fatalf("FinancialStatusFromScore(%d) = %q; want %q", tt.score, got, tt.want)
			}
		})
	}
}
