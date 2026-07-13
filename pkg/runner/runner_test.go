package runner

import (
	"testing"
	"time"
)

func TestEqualTokens(t *testing.T) {
	if !equalTokens("1  2\n", "1 2") {
		t.Fatal("expected whitespace-insensitive comparison")
	}
	if equalTokens("1 2", "1 3") {
		t.Fatal("different outputs must fail")
	}
}

func TestLimitedBuffer(t *testing.T) {
	b := &limitedBuffer{limit: 3}
	_, _ = b.Write([]byte("abcd"))
	if b.String() != "abc" || !b.exceeded {
		t.Fatalf("unexpected buffer: %q, exceeded=%v", b.String(), b.exceeded)
	}
}

func TestPythonRuntime(t *testing.T) {
	got, stderr, timedOut := pythonRuntime("trace\n"+runtimeMarker+"123456\n", 9*time.Second)
	if got != 123456*time.Nanosecond {
		t.Fatalf("unexpected runtime: %v", got)
	}
	if timedOut || stderr != "trace\n" {
		t.Fatalf("runtime marker was not removed: %q", stderr)
	}
}

func TestPythonRuntimeFallback(t *testing.T) {
	fallback := 2 * time.Second
	got, stderr, timedOut := pythonRuntime("python failed", fallback)
	if got != fallback || timedOut || stderr != "python failed" {
		t.Fatalf("unexpected fallback: runtime=%v stderr=%q", got, stderr)
	}
}
