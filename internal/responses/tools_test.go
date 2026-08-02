package responses

import (
	"reflect"
	"testing"
)

func TestToolNames(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{{
		name: "a function tool is named at its top level",
		body: `{"input":[],"tools":[{"type":"function","name":"web_search"}]}`,
		want: []string{"web_search"},
	}, {
		name: "every named tool contributes",
		body: `{"tools":[{"name":"one"},{"name":"two"},{"name":"three"}]}`,
		want: []string{"one", "two", "three"},
	}, {
		name: "a built-in tool identified by type alone contributes none",
		body: `{"tools":[{"type":"web_search_preview"},{"type":"function","name":"grep"}]}`,
		want: []string{"grep"},
	}, {
		name: "an empty array names nothing",
		body: `{"input":[],"tools":[]}`,
		want: []string{},
	}, {
		name: "a body with no tools names nothing",
		body: `{"input":[{"role":"user","content":"hi"}]}`,
		want: []string{},
	}, {
		name: "a body that is not JSON names nothing",
		body: `not json`,
		want: nil,
	}, {
		name: "a tools value of the wrong shape names nothing",
		body: `{"tools":"all of them"}`,
		want: nil,
	}}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := ToolNames([]byte(testCase.body))
			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("ToolNames = %#v, want %#v", got, testCase.want)
			}
		})
	}
}

// A tool's parameters can be arbitrarily deep and chancery never looks inside them, so
// reading a name must not disturb what the backend receives.
func TestToolNamesLeavesTheBodyAlone(t *testing.T) {
	body := []byte(`{"input":[],"tools":[{"type":"function","name":"grep",` +
		`"parameters":{"type":"object","properties":{"pattern":{"type":"string"}}}}]}`)
	before := string(body)
	ToolNames(body)
	if string(body) != before {
		t.Errorf("body changed:\ngot  %s\nwant %s", body, before)
	}
}
