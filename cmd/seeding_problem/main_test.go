package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestProblemDefinitionsAndGeneratedTestCases(t *testing.T) {
	data, err := os.ReadFile("problem.json")
	if err != nil {
		t.Fatal(err)
	}
	var definitions []problemDefinition
	if err := json.Unmarshal(data, &definitions); err != nil {
		t.Fatal(err)
	}
	if len(definitions) == 0 {
		t.Fatal("problem.json must contain at least one problem")
	}
	for _, definition := range definitions {
		tests := generateTestCases(definition.Title)
		if len(tests) != 30 {
			t.Fatalf("%s: got %d testcases, want 30", definition.Title, len(tests))
		}
		for i, test := range tests {
			if strings.TrimSpace(test.Input) == "" || strings.TrimSpace(test.Output) == "" {
				t.Fatalf("%s testcase %d has empty input/output", definition.Title, i+1)
			}
		}
	}
}
