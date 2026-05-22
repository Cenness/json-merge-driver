package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
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
			name: "no changes | no conflict",
			base: `{"a": 1}`,
			ours: `{"a": 1}`,
			theirs: `{"a": 1}`,
			expected: `{"a":1}`,
			expectedConflict: false,
		},
		{
			name: "only ours changed | no conflict",
			base: `{"a": 1}`,
			ours: `{"a": 2}`,
			theirs: `{"a": 1}`,
			expected: `{"a":2}`,
			expectedConflict: false,
		},
		{
			name: "only theirs changed | no conflict",
			base: `{"a": 1}`,
			ours: `{"a": 1}`,
			theirs: `{"a": 2}`,
			expected: `{"a":2}`,
			expectedConflict: false,
		},
		{
			name: "mutual change | conflict",
			base: `{"a": 1}`,
			ours: `{"a": 2}`,
			theirs: `{"a": 3}`,
			expected: `{"a":2}`,
			expectedConflict: true,
		},
		{
			name: "type mismatch | conflict",
			base: `{"a": 1}`,
			ours: `{"a": [1]}`,
			theirs: `{"a": false}`,
			expected: `{"a":[1]}`,
			expectedConflict: true,
		},
		{
			name: "nested map merge | no conflict",
			base: `{"a": {"b": 1},"n": []}`,
			ours: `{"a": {"b": 1, "c": 2}}`,
			theirs: `{"a": {"b": 1,"d": 3}}`,
			expected: `{"a":{"b":1,"c":2,"d":3},"n":null}`,
			expectedConflict: false,
		},
		{
			name: "one sided changes | no conflict",
			base: `{"a": [1, 2],"b": false}`,
			ours: `{"a": [1, 2, 3],"b": false}`,
			theirs: `{"a": [1, 2],"b": true}`,
			expected: `{"a":[1,2,3],"b": true}`,
			expectedConflict: false,
		},
		{
			name: "two sided array expand | conflict",
			base: `{"a": [1, 2]}`,
			ours: `{"a": [1, 2, 3]}`,
			theirs: `{"a": [1, 2, 4]}`,
			expected: `{"a":[1,2,3]}`,
			expectedConflict: true,
		},
		{
			name: "two sided array length change | conflict",
			base: `{"a": [1, 2]}`,
			ours: `{"a": [1, 2, 3]}`,
			theirs: `{"a": [1]}`,
			expected: `{"a":[1,2,3]}`,
			expectedConflict: true,
		},
		{
			name: "mutual array change | conflict",
			base: `{"a": [1, 2]}`,
			ours: `{"a": [1, 3]}`,
			theirs: `{"a": [1, 4]}`,
			expected: `{"a":[1,3]}`,
			expectedConflict: true,
		},
		{
			name: "base missing map merge identical branches | no conflict",
			base: ``,
			ours: `{"a": 1}`,
			theirs: `{"a": 1}`,
			expected: `{"a":1}`,
			expectedConflict: false,
		},
		{
			name: "base missing array identical branches | conflict",
			base: ``,
			ours: `[1, 2]`,
			theirs: `[1, 2]`,
			expected: `[1,2]`,
			expectedConflict: true,
		},
		{
			name: "both branches empty top-level | no conflict",
			base: `{"a": 1}`,
			ours: ``,
			theirs: ``,
			expected: `null`,
			expectedConflict: false,
		},
		{
			name: "one branch empty top-level | no conflict",
			base: `{"a": 1}`,
			ours: ``,
			theirs: `{"a": 1}`,
			expected: `null`,
			expectedConflict: false,
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

func TestReadJSON(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "empty.json")
		if err := os.WriteFile(path, nil, 0644); err != nil {
			t.Fatal(err)
		}

		value, err := readJSON(path)
		if err != nil {
			t.Fatal(err)
		}
		if value != nil {
			t.Fatalf("expected nil, got %#v", value)
		}
	})

	t.Run("valid JSON", func(t *testing.T) {
		path := filepath.Join(tmpDir, "valid.json")
		if err := os.WriteFile(path, []byte(`{"x": 1}`), 0644); err != nil {
			t.Fatal(err)
		}

		value, err := readJSON(path)
		if err != nil {
			t.Fatal(err)
		}
		expected := map[string]interface{}{"x": float64(1)}
		if !reflect.DeepEqual(value, expected) {
			t.Fatalf("expected %#v, got %#v", expected, value)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		path := filepath.Join(tmpDir, "invalid.json")
		if err := os.WriteFile(path, []byte(`{"x":`), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := readJSON(path)
		if err == nil {
			t.Fatal("expected an error for invalid JSON")
		}
	})
}

func TestMainHelper(t *testing.T) {
	if os.Getenv("TEST_MAIN_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
			break
		}
	}
	main()
	os.Exit(0)
}

func runMainHelper(t *testing.T, args []string) (stdout, stderr string, exitCode int, err error) {
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=TestMainHelper", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "TEST_MAIN_HELPER=1")
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err = cmd.Run()
	stdout = outb.String()
	stderr = errb.String()
	if err == nil {
		exitCode = 0
		return
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
		err = nil
		return
	}
	return
}

func TestMainUsage(t *testing.T) {
	_, stderr, exitCode, err := runMainHelper(t, []string{"one", "two"})
	if err != nil {
		t.Fatal(err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "Usage: json-merge-driver") {
		t.Fatalf("expected usage output, got %q", stderr)
	}
}

func TestMainReadJSONError(t *testing.T) {
	_, stderr, exitCode, err := runMainHelper(t, []string{"nonexistent.json", "current.json", "other.json"})
	if err != nil {
		t.Fatal(err)
	}
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if !strings.Contains(stderr, "Error reading ancestor") {
		t.Fatalf("expected ancestor read error, got %q", stderr)
	}
}

func TestMainConflictExit(t *testing.T) {
	tmpDir := t.TempDir()
	ancestorPath := filepath.Join(tmpDir, "ancestor.json")
	currentPath := filepath.Join(tmpDir, "current.json")
	otherPath := filepath.Join(tmpDir, "other.json")
	if err := os.WriteFile(ancestorPath, []byte(`{"a": 1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(currentPath, []byte(`{"a": 2}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(otherPath, []byte(`{"a": 3}`), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, exitCode, err := runMainHelper(t, []string{ancestorPath, currentPath, otherPath})
	if err != nil {
		t.Fatal(err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "JSON merge conflicts detected") {
		t.Fatalf("expected conflict message, got %q", stderr)
	}
}

func TestMainSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	ancestorPath := filepath.Join(tmpDir, "ancestor.json")
	currentPath := filepath.Join(tmpDir, "current.json")
	otherPath := filepath.Join(tmpDir, "other.json")
	if err := os.WriteFile(ancestorPath, []byte(`{"a": 1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(currentPath, []byte(`{"a": 1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(otherPath, []byte(`{"a": 1}`), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, exitCode, err := runMainHelper(t, []string{ancestorPath, currentPath, otherPath})
	if err != nil {
		t.Fatal(err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr output, got %q", stderr)
	}
	content, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"a": 1`) {
		t.Fatalf("expected merged file to contain a=1, got %q", string(content))
	}
}
