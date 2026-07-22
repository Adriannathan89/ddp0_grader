package queue

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewAppliesRetryDefaults(t *testing.T) {
	q := New(Config{Addr: "127.0.0.1:1", Stream: "jobs"})
	defer q.Close()
	if q.maxAttempts != 3 || q.claimIdle != 15*time.Minute || q.retryDelay != 10*time.Second || q.deadLetterStream != "jobs:dead" || q.retryStream != "jobs:retry" {
		t.Fatalf("unexpected retry defaults: attempts=%d idle=%s delay=%s dead-letter=%q retry=%q", q.maxAttempts, q.claimIdle, q.retryDelay, q.deadLetterStream, q.retryStream)
	}
}

func TestTruncateError(t *testing.T) {
	message := strings.Repeat("x", 2048)
	if got := truncateError(errors.New(message)); len(got) != 1024 {
		t.Fatalf("error length = %d, want 1024", len(got))
	}
}

func TestNewWithClientRejectsNilClient(t *testing.T) {
	if _, err := NewWithClient(nil, Config{}); err == nil {
		t.Fatal("expected nil Redis client to be rejected")
	}
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer client.Close()
	if _, err := NewWithClient(client, Config{}); err != nil {
		t.Fatalf("NewWithClient() error = %v", err)
	}
}

func TestFailedJobMovesToDeadLetterAfterMaxAttempts(t *testing.T) {
	redisBinary, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("redis-server is not installed")
	}
	socket := filepath.Join(t.TempDir(), "redis.sock")

	var redisOutput bytes.Buffer
	server := exec.Command(redisBinary, "--port", "0", "--unixsocket", socket, "--unixsocketperm", "700", "--save", "", "--appendonly", "no")
	server.Stdout, server.Stderr = &redisOutput, &redisOutput
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = server.Process.Kill()
		_ = server.Wait()
	}()

	client := redis.NewClient(&redis.Options{Network: "unix", Addr: socket})
	defer client.Close()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if err := client.Ping(context.Background()).Err(); err == nil {
			break
		}
		if time.Now().After(deadline) {
			if strings.Contains(redisOutput.String(), "Operation not permitted") {
				t.Skip("sandbox does not permit local Redis sockets")
			}
			t.Fatalf("redis-server did not start: %s", redisOutput.String())
		}
		time.Sleep(10 * time.Millisecond)
	}

	q, err := NewWithClient(client, Config{Stream: "jobs", Group: "workers", Consumer: "test", MaxAttempts: 3, ClaimIdle: time.Second, RetryDelay: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var attempts atomic.Int32
	exhausted := make(chan Job, 1)
	done := make(chan error, 1)
	go func() {
		done <- q.WorkNWithExhaustedHandler(ctx, 1, func(context.Context, Job) error {
			attempts.Add(1)
			return errors.New("temporary database failure")
		}, func(_ context.Context, job Job, _ error) error {
			exhausted <- job
			return nil
		})
	}()
	if _, err := q.Enqueue(context.Background(), Job{ID: "submission-1"}); err != nil {
		t.Fatal(err)
	}

	select {
	case job := <-exhausted:
		if job.Attempt != 3 {
			t.Fatalf("dead-letter attempt = %d, want 3", job.Attempt)
		}
	case <-time.After(5 * time.Second):
		pending, _ := client.XPending(context.Background(), "jobs", "workers").Result()
		retrying, _ := client.ZCard(context.Background(), "jobs:retry").Result()
		streamLength, _ := client.XLen(context.Background(), "jobs").Result()
		groups, _ := client.XInfoGroups(context.Background(), "jobs").Result()
		select {
		case err := <-done:
			t.Fatalf("job was not exhausted: worker stopped with %v; handler calls=%d pending=%d retry=%d stream=%d groups=%+v", err, attempts.Load(), pending.Count, retrying, streamLength, groups)
		default:
			t.Fatalf("job was not exhausted: handler calls=%d pending=%d retry=%d stream=%d groups=%+v", attempts.Load(), pending.Count, retrying, streamLength, groups)
		}
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("handler calls = %d, want 3", got)
	}
	if length, err := client.XLen(context.Background(), "jobs:dead").Result(); err != nil || length != 1 {
		t.Fatalf("dead-letter length = %d, error = %v", length, err)
	}
	if pending, err := client.XPending(context.Background(), "jobs", "workers").Result(); err != nil || pending.Count != 0 {
		t.Fatalf("pending count = %d, error = %v", pending.Count, err)
	}
	if length, err := client.XLen(context.Background(), "jobs").Result(); err != nil || length != 0 {
		t.Fatalf("active stream length = %d, error = %v", length, err)
	}
	if size, err := client.ZCard(context.Background(), "jobs:retry").Result(); err != nil || size != 0 {
		t.Fatalf("retry queue size = %d, error = %v", size, err)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not stop after cancellation")
	}
}
