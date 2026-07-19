package prompts

import (
	"strings"
	"testing"
)

func TestDecodeYAMLErrorMessages(t *testing.T) {
	type sample struct {
		Name string `yaml:"name"`
		Size int    `yaml:"size"`
	}
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "valid decodes cleanly", input: "name: x\nsize: 3\n", wantErr: ""},
		{name: "unknown field is named without Go type", input: "name: x\npricing: 5\n", wantErr: `unknown field "pricing"`},
		{name: "type mismatch is preserved", input: "size: notanumber\n", wantErr: "cannot unmarshal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target sample
			err := decodeYAML([]byte(tt.input), &target)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
			if strings.Contains(err.Error(), "prompts.") || strings.Contains(err.Error(), "in type") {
				t.Fatalf("error leaks internal Go type: %q", err.Error())
			}
		})
	}
}
