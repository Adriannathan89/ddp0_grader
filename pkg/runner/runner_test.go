package runner

import (
	"context"
	"testing"
	"time"

	"ddp0_grader/app/models"
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

func TestRunReturnsErrorWhenRunnerCannotExecuteAnyTestCase(t *testing.T) {
	r := New(Config{DockerBinary: "definitely-not-a-docker-binary"})
	_, err := r.Run(context.Background(), &models.Submission{SourceCode: "print(1)"}, &models.Problem{TimeLimit: 10, MemoryLimit: 16}, []models.TestCase{{ID: "case-1"}})
	if err == nil {
		t.Fatal("expected an infrastructure error when docker cannot be started")
	}
}

func TestNewCapsRunnerLimits(t *testing.T) {
	r := New(Config{DefaultTime: 3 * time.Second, MaxTime: time.Second, DefaultMemoryMB: 128, MaxMemoryMB: 64})
	if r.config.DefaultTime != time.Second || r.config.MaxTime != time.Second {
		t.Fatalf("unexpected time limits: default=%s max=%s", r.config.DefaultTime, r.config.MaxTime)
	}
	if r.config.DefaultMemoryMB != 64 || r.config.MaxMemoryMB != 64 {
		t.Fatalf("unexpected memory limits: default=%d max=%d", r.config.DefaultMemoryMB, r.config.MaxMemoryMB)
	}
}
