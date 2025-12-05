package testhelpers

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func AssertEqual(t *testing.T, got, want any, msg string, opts ...cmp.Option) {
	t.Helper()

	gotVal := reflect.ValueOf(got)
	wantVal := reflect.ValueOf(want)
	gotIsNil := !gotVal.IsValid() || (gotVal.Kind() == reflect.Ptr && gotVal.IsNil())
	wantIsNil := !wantVal.IsValid() || (wantVal.Kind() == reflect.Ptr && wantVal.IsNil())

	if gotIsNil && wantIsNil {
		return
	}

	if diff := cmp.Diff(want, got, opts...); diff != "" {
		t.Fatalf("%s: mismatch (-want +got):\n%s", msg, diff)
	}
}

func RunMapTests[In comparable, Out any](t *testing.T, cases map[In]Out, fn func(In) Out) {
	for input, expected := range cases {
		t.Run(fmt.Sprintf("%v", input), func(t *testing.T) {
			got := fn(input)
			AssertEqual(t, got, expected, "result")
		})
	}
}

func RunFunctionTests[T any, R any, M any](t *testing.T, tests []struct {
	Name     string
	Input    T
	Expected M
}, testFunc func(T) R, mapFunc ...func(R) M) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := testFunc(tt.Input)
			var actual any
			if len(mapFunc) > 0 {
				actual = mapFunc[0](result)
			} else {
				actual = result
			}
			AssertEqual(t, actual, tt.Expected, "result")
		})
	}
}
