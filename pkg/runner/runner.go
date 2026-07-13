package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ddp0_grader/app/models"
)

const (
	VerdictAccepted          = "accepted"
	VerdictWrongAnswer       = "wrong_answer"
	VerdictRuntimeError      = "runtime_error"
	VerdictTimeLimitExceeded = "time_limit_exceeded"
	VerdictOutputLimit       = "output_limit_exceeded"
	VerdictSystemError       = "system_error"
	runtimeMarker            = "__GRADER_PYTHON_RUNTIME_NS__="
	timeoutMarker            = "__GRADER_PYTHON_TIMEOUT__=1"
)

const pythonEntrypoint = `import subprocess
import sys
import time

timeout_seconds = TIMEOUT_NS / 1_000_000_000
started = time.perf_counter_ns()
timed_out = False
process = subprocess.Popen(
    [sys.executable, "/runner/main.py"],
    stdin=sys.stdin,
    stdout=sys.stdout,
    stderr=sys.stderr,
)
try:
    return_code = process.wait(timeout=timeout_seconds)
except subprocess.TimeoutExpired:
    timed_out = True
    process.kill()
    process.wait()
    return_code = 124
finally:
    elapsed = time.perf_counter_ns() - started
    if timed_out:
        sys.stderr.write("\n` + timeoutMarker + `\n")
    sys.stderr.write("\n` + runtimeMarker + `" + str(elapsed) + "\n")
    sys.stderr.flush()
sys.exit(return_code)
`

func pythonCommand(timeLimit time.Duration) string {
	return strings.Replace(pythonEntrypoint, "TIMEOUT_NS", strconv.FormatInt(timeLimit.Nanoseconds(), 10), 1)
}

type Config struct {
	DockerBinary    string
	Image           string
	OutputLimit     int64
	DefaultTime     time.Duration
	DefaultMemoryMB int
}

type Runner struct{ config Config }

func New(config Config) *Runner {
	if config.DockerBinary == "" {
		config.DockerBinary = "docker"
	}
	if config.Image == "" {
		config.Image = "python:3.12-slim"
	}
	if config.OutputLimit <= 0 {
		config.OutputLimit = 1 << 20
	}
	if config.DefaultTime <= 0 {
		config.DefaultTime = 1 * time.Second
	}
	if config.DefaultMemoryMB <= 0 {
		config.DefaultMemoryMB = 256
	}
	return &Runner{config: config}
}

type TestResult struct {
	TestCaseID string
	Passed     bool
	Verdict    string
	RunTime    time.Duration
	Stdout     string
	Stderr     string
	Error      error
}

// Run executes every testcase in its own short-lived, network-disabled container.
func (r *Runner) Run(ctx context.Context, submission *models.Submission, problem *models.Problem, testCases []models.TestCase) ([]TestResult, error) {
	if submission == nil {
		return nil, errors.New("submission is nil")
	}
	if problem == nil {
		return nil, errors.New("problem is nil")
	}
	results := make([]TestResult, 0, len(testCases))
	for _, tc := range testCases {
		result := r.RunTestCase(ctx, submission.SourceCode, tc, problem.TimeLimit, problem.MemoryLimit)
		result.TestCaseID = tc.ID
		results = append(results, result)
	}
	return results, nil
}

func (r *Runner) RunTestCase(parent context.Context, source string, tc models.TestCase, timeLimitMS, memoryLimitMB int) TestResult {
	hostStarted := time.Now()
	result := TestResult{TestCaseID: tc.ID, Verdict: VerdictSystemError}
	limit := r.config.DefaultTime
	if timeLimitMS > 0 {
		limit = time.Duration(timeLimitMS) * time.Millisecond
	}
	memory := r.config.DefaultMemoryMB
	if memoryLimitMB > 0 {
		memory = memoryLimitMB
	}
	// Docker startup is outside the contestant's time budget. Keep a generous
	// outer guard only to prevent a stuck Docker command from hanging forever;
	// the Python wrapper enforces the actual limit inside the container.
	ctx, cancel := context.WithTimeout(parent, limit+10*time.Second)
	defer cancel()

	dir, err := os.MkdirTemp("", "grader-run-")
	if err != nil {
		result.Error = err
		return result
	}
	defer os.RemoveAll(dir)
	if err = os.Chmod(dir, 0755); err != nil {
		result.Error = err
		return result
	}
	sourcePath := filepath.Join(dir, "main.py")
	if err = os.WriteFile(sourcePath, []byte(source), 0444); err != nil {
		result.Error = err
		return result
	}

	stdout := &limitedBuffer{limit: r.config.OutputLimit}
	stderr := &limitedBuffer{limit: r.config.OutputLimit}
	args := []string{"run", "--rm", "-i", "--network", "none", "--read-only", "--init",
		"--memory", strconv.Itoa(memory) + "m", "--memory-swap", strconv.Itoa(memory) + "m",
		"--cpus", "1", "--pids-limit", "64", "--cap-drop", "ALL", "--security-opt", "no-new-privileges",
		"--user", "65534:65534", "--tmpfs", "/tmp:rw,noexec,nosuid,size=64m",
		"--mount", "type=bind,src=" + dir + ",dst=/runner,readonly",
		r.config.Image, "python", "-c", pythonCommand(limit)}
	cmd := exec.CommandContext(ctx, r.config.DockerBinary, args...)
	cmd.Stdin = strings.NewReader(tc.Input)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err = cmd.Run()
	rawStderr := stderr.String()
	var pythonTimedOut bool
	result.RunTime, rawStderr, pythonTimedOut = pythonRuntime(rawStderr, time.Since(hostStarted))
	result.Stdout, result.Stderr = stdout.String(), rawStderr
	if stdout.exceeded {
		result.Verdict = VerdictOutputLimit
		result.Error = errors.New("stdout exceeded configured limit")
		return result
	}
	if pythonTimedOut {
		result.Verdict = VerdictTimeLimitExceeded
		result.Error = context.DeadlineExceeded
		return result
	}
	if ctx.Err() == context.DeadlineExceeded {
		result.Verdict = VerdictSystemError
		result.Error = ctx.Err()
		return result
	}
	if err != nil {
		result.Verdict = VerdictRuntimeError
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() >= 125 {
			result.Verdict = VerdictSystemError
		}
		detail := strings.TrimSpace(result.Stderr)
		if detail != "" {
			result.Error = fmt.Errorf("docker run: %w: %s", err, detail)
		} else {
			result.Error = fmt.Errorf("docker run: %w", err)
		}
		return result
	}
	if equalTokens(result.Stdout, tc.Output) {
		result.Verdict, result.Passed = VerdictAccepted, true
	} else {
		result.Verdict = VerdictWrongAnswer
	}
	return result
}

func pythonRuntime(stderr string, fallback time.Duration) (time.Duration, string, bool) {
	marker := strings.LastIndex(stderr, runtimeMarker)
	if marker < 0 {
		return fallback, stderr, false
	}

	valueStart := marker + len(runtimeMarker)
	valueEnd := valueStart
	for valueEnd < len(stderr) && stderr[valueEnd] >= '0' && stderr[valueEnd] <= '9' {
		valueEnd++
	}
	ns, err := strconv.ParseInt(stderr[valueStart:valueEnd], 10, 64)
	if err != nil {
		return fallback, stderr, false
	}

	lineStart := marker
	lineEnd := valueEnd
	if lineEnd < len(stderr) && stderr[lineEnd] == '\r' {
		lineEnd++
	}
	if lineEnd < len(stderr) && stderr[lineEnd] == '\n' {
		lineEnd++
	}
	cleaned := stderr[:lineStart] + stderr[lineEnd:]
	timedOut := strings.Contains(cleaned, timeoutMarker)
	cleaned = strings.ReplaceAll(cleaned, timeoutMarker+"\n", "")
	return time.Duration(ns), cleaned, timedOut
}

func equalTokens(actual, expected string) bool {
	return strings.Join(strings.Fields(actual), " ") == strings.Join(strings.Fields(expected), " ")
}

type limitedBuffer struct {
	bytes.Buffer
	limit    int64
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - int64(b.Len())
	if remaining <= 0 {
		b.exceeded = true
		return len(p), nil
	}
	if int64(len(p)) > remaining {
		_, _ = b.Buffer.Write(p[:remaining])
		b.exceeded = true
		return len(p), nil
	}
	return b.Buffer.Write(p)
}
