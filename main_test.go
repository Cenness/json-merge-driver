package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

type mergeTest struct {
	name           string
	base           string
	ours           string
	theirs         string
	expected       string
	expectedConflict bool
}

func mustDecode(t *testing.T, src string) interface{} {
	t.Helper()
	if src == "" {
		return nil
	}
	var v interface{}
	if err := json.Unmarshal([]byte(src), &v); err != nil {
		t.Fatalf("failed to decode JSON %q: %v", src, err)
	}
	return v
}

func mustEncode(t *testing.T, v interface{}) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to encode JSON: %v", err)
	}
	return string(data)
}

func TestMerge(t *testing.T) {
	tests := []mergeTest{
		{
			name: "no changes",
			base: `{"a": 1}`,
			ours: `{"a": 1}`,
			theirs: `{"a": 1}`,
			expected: `{"a":1}`,
			expectedConflict: false,
		},
		{
			name: "only ours changed",
			base: `{"a": 1}`,
			ours: `{"a": 2}`,
			theirs: `{"a": 1}`,
			expected: `{"a":2}`,
			expectedConflict: false,
		},
		{
			name: "only theirs changed",
			base: `{"a": 1}`,
			ours: `{"a": 1}`,
			theirs: `{"a": 2}`,
			expected: `{"a":2}`,
			expectedConflict: false,
		},
		{
			name: "scalar conflict",
			base: `{"a": 1}`,
			ours: `{"a": 2}`,
			theirs: `{"a": 3}`,
			expected: `{"a":2}`,
			expectedConflict: true,
		},
		{
			name: "nested map merge no conflict",
			base: `{"a": {"b": 1}}`,
			ours: `{"a": {"b": 1, "c": 2}}`,
			theirs: `{"a": {"b": 1}}`,
			expected: `{"a":{"b":1,"c":2}}`,
			expectedConflict: false,
		},
		{
			name: "type mismatch conflict",
			base: `{"a": 1}`,
			ours: `{"a": [1]}`,
			theirs: `{"a": false}`,
			expected: `{"a":[1]}`,
			expectedConflict: true,
		},
		{
			name: "array length conflict",
			base: `{"a": [1, 2]}`,
			ours: `{"a": [1, 2, 3]}`,
			theirs: `{"a": [1, 2, 3, 4]}`,
			expected: `{"a":[1,2,3]}`,
			expectedConflict: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := mustDecode(t, tt.base)
			ours := mustDecode(t, tt.ours)
			theirs := mustDecode(t, tt.theirs)

			merged, conflict := merge(base, ours, theirs)
			if conflict != tt.expectedConflict {
				t.Fatalf("expected conflict=%v, got %v", tt.expectedConflict, conflict)
			}

			expected := mustDecode(t, tt.expected)
			if !reflect.DeepEqual(merged, expected) {
				gotJSON := mustEncode(t, merged)
				expectedJSON := mustEncode(t, expected)
				t.Fatalf("unexpected merge result:\n got: %s\nwant: %s", gotJSON, expectedJSON)
			}
		})
	}
}
