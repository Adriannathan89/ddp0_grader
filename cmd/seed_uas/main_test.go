package main

import "testing"

func TestSeedTestsContainOnePublicCase(t *testing.T) {
	tests := seedTests()
	if len(tests) != 30 {
		t.Fatalf("seedTests() count = %d, want 30", len(tests))
	}

	publicCount := 0
	for _, test := range tests {
		if !test.isHidden {
			publicCount++
		}
		if expected(test.input) == "" {
			t.Fatalf("testcase %q has an empty expected output", test.id)
		}
	}
	if publicCount != 1 {
		t.Fatalf("public testcase count = %d, want 1", publicCount)
	}
	if tests[0].isHidden {
		t.Fatal("the first testcase must be public")
	}
}
