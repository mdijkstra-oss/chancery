package fn

import (
	"slices"
	"strconv"
	"testing"
)

func TestMap(t *testing.T) {
	tests := []struct {
		name  string
		input []int
		want  []string
	}{
		{name: "empty", input: nil, want: []string{}},
		{name: "ints to strings", input: []int{1, 2, 3}, want: []string{"1", "2", "3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Map(tt.input, strconv.Itoa)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("Map = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter(t *testing.T) {
	tests := []struct {
		name  string
		input []int
		want  []int
	}{
		{name: "empty", input: nil, want: []int{}},
		{name: "keeps evens", input: []int{1, 2, 3, 4}, want: []int{2, 4}},
		{name: "none match", input: []int{1, 3, 5}, want: []int{}},
	}
	isEven := func(n int) bool { return n%2 == 0 }
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Filter(tt.input, isEven)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("Filter = %v, want %v", got, tt.want)
			}
		})
	}
}
