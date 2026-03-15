package outfmt

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestOutputJSON(t *testing.T) {
	tests := []struct {
		name string
		data any
		want string
	}{
		{
			name: "simple map",
			data: map[string]string{"key": "value"},
			want: `"key": "value"`,
		},
		{
			name: "slice",
			data: []string{"a", "b"},
			want: `"a"`,
		},
		{
			name: "nested",
			data: map[string]any{"state": "on", "attrs": map[string]any{"brightness": 128}},
			want: `"state": "on"`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			OutputJSON(tc.data, &buf)
			if !strings.Contains(buf.String(), tc.want) {
				t.Errorf("OutputJSON() = %q, want to contain %q", buf.String(), tc.want)
			}
			// Must be valid JSON.
			var v any
			if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
				t.Errorf("OutputJSON() produced invalid JSON: %v\noutput: %s", err, buf.String())
			}
		})
	}
}

func TestOutputPlain(t *testing.T) {
	tests := []struct {
		name   string
		fields [][2]string
		want   []string
	}{
		{
			name:   "single field",
			fields: [][2]string{{"entity_id", "light.living_room"}},
			want:   []string{"entity_id\tlight.living_room"},
		},
		{
			name: "multiple fields",
			fields: [][2]string{
				{"state", "on"},
				{"brightness", "128"},
			},
			want: []string{"state\ton", "brightness\t128"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			OutputPlain(tc.fields, &buf)
			out := buf.String()
			for _, w := range tc.want {
				if !strings.Contains(out, w) {
					t.Errorf("OutputPlain() = %q, want to contain %q", out, w)
				}
			}
		})
	}
}

func TestOutput_JSON(t *testing.T) {
	var buf bytes.Buffer
	Output(JSON, map[string]string{"ok": "true"}, &buf)
	var v any
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Errorf("Output(JSON) produced invalid JSON: %v", err)
	}
}

func TestOutput_Human(t *testing.T) {
	var buf bytes.Buffer
	Output(Human, "hello", &buf)
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("Output(Human) = %q, want to contain 'hello'", buf.String())
	}
}
