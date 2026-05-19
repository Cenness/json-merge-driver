/*
Copyright 2026 Cenness.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Usage: json-merge-driver <ancestor> <current> <other>")
		os.Exit(1)
	}

	ancestorPath, currentPath, otherPath := os.Args[1], os.Args[2], os.Args[3]

	ancestor, err := readJSON(ancestorPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading ancestor (%s): %v\n", ancestorPath, err)
		os.Exit(2)
	}
	ours, err := readJSON(currentPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading current (%s): %v\n", currentPath, err)
		os.Exit(2)
	}
	theirs, err := readJSON(otherPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading other (%s): %v\n", otherPath, err)
		os.Exit(2)
	}

	merged, conflict := merge(ancestor, ours, theirs)

	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling merged JSON: %v\n", err)
		os.Exit(2)
	}
	data = append(data, '\n')

	if err := os.WriteFile(currentPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing merged file: %v\n", err)
		os.Exit(2)
	}

	if conflict {
		fmt.Fprintln(os.Stderr, "JSON merge conflicts detected. File contains best-effort merge. Please resolve manually.")
		os.Exit(1)
	}
}

func readJSON(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return v, nil
}

// merge performs a 3-way recursive deep merge.
// Returns the merged value and a boolean indicating if conflicts occurred.
func merge(base, ours, theirs interface{}) (interface{}, bool) {
	// No changes on either side
	if reflect.DeepEqual(ours, base) && reflect.DeepEqual(theirs, base) {
		return base, false
	}
	// Only theirs changed
	if reflect.DeepEqual(ours, base) {
		return theirs, false
	}
	// Only ours changed
	if reflect.DeepEqual(theirs, base) {
		return ours, false
	}

	// Both sides changed: check type compatibility
	if reflect.TypeOf(ours) != reflect.TypeOf(theirs) {
		return ours, true // Type mismatch → conflict
	}

	switch ours.(type) {
	case map[string]interface{}:
		theirsMap := theirs.(map[string]interface{})
		var baseMap map[string]interface{}
		if base != nil {
				baseMap = base.(map[string]interface{})
		}
		return mergeMaps(baseMap, ours.(map[string]interface{}), theirsMap)
	case []interface{}:
		theirsArr := theirs.([]interface{})
		var baseArr []interface{}
		if base != nil {
				baseArr = base.([]interface{})
		}
		return mergeArrays(baseArr, ours.([]interface{}), theirsArr)
	default:
		// Scalars (string, number, bool, null)
		if reflect.DeepEqual(ours, theirs) {
				return ours, false
		}
		return ours, true // Different values → conflict
	}
}

func mergeMaps(base, ours, theirs map[string]interface{}) (interface{}, bool) {
	if base == nil {
		base = make(map[string]interface{})
	}
	if ours == nil {
		ours = make(map[string]interface{})
	}
	if theirs == nil {
		theirs = make(map[string]interface{})
	}

	result := make(map[string]interface{})
	conflict := false

	// Collect all keys from base, ours, and theirs
	keys := make(map[string]bool)
	for k := range base {
		keys[k] = true
	}
	for k := range ours {
		keys[k] = true
	}
	for k := range theirs {
		keys[k] = true
	}

	for k := range keys {
		bv, bOk := base[k]
		ov, oOk := ours[k]
		tv, tOk := theirs[k]

		// Missing keys are treated as nil for merge logic
		if !bOk {
			bv = nil
		}
		if !oOk {
			ov = nil
		}
		if !tOk {
			tv = nil
		}

		merged, isConflict := merge(bv, ov, tv)
		result[k] = merged
		if isConflict {
			conflict = true
		}
	}
	return result, conflict
}

func mergeArrays(base, ours, theirs []interface{}) (interface{}, bool) {
	if base == nil {
		base = []interface{}{}
	}
	if ours == nil {
		ours = []interface{}{}
	}
	if theirs == nil {
		theirs = []interface{}{}
	}

	// Array length changes in different branches are treated as conflicts
	if len(ours) != len(theirs) || len(base) != len(ours) {
		return ours, true
	}

	result := make([]interface{}, len(ours))
	conflict := false
	for i := range ours {
		merged, isConflict := merge(base[i], ours[i], theirs[i])
		result[i] = merged
		if isConflict {
			conflict = true
		}
	}
	return result, conflict
}
