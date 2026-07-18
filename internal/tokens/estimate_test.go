package tokens

import (
	"encoding/json"
	"testing"
)

func TestEstimate(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int
	}{
		{name: "empty strings", input: []string(nil), want: 0},
		{name: "strings", input: []string{"abcdefgh", "ijklmnop"}, want: 4},
		{name: "raw messages", input: []json.RawMessage{json.RawMessage(`{"role":"user"}`), json.RawMessage(`{"role":"assistant"}`)}, want: 8},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got int
			switch input := test.input.(type) {
			case []string:
				got = Estimate(input)
			case []json.RawMessage:
				got = Estimate(input)
			default:
				panic("unknown input")
			}
			if got != test.want {
				t.Errorf("Estimate() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestEstimateByteCount(t *testing.T) {
	tests := []struct {
		count int
		want  int
	}{
		{count: 0, want: 0},
		{count: 3, want: 0},
		{count: 4, want: 1},
		{count: 9, want: 2},
	}
	for _, test := range tests {
		if got := EstimateByteCount(test.count); got != test.want {
			t.Errorf("EstimateByteCount(%d) = %d, want %d", test.count, got, test.want)
		}
	}
}
