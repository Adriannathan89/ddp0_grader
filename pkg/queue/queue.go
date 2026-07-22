package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"ddp0_grader/app/models"
	"github.com/redis/go-redis/v9"
)

type Job struct {
	ID         string            `json:"id"`
	Attempt    int               `json:"attempt"`
	Submission models.Submission `json:"submission"`
	Problem    models.Problem    `json:"problem"`
	TestCases  []models.TestCase `json:"test_cases"`
}

type Config struct {
	Addr, Password, Stream, Group, Consumer, DeadLetterStream, RetryStream string
	DB                                                                     int
	MaxAttempts                                                            int
	ClaimIdle, RetryDelay                                                  time.Duration
	DeadLetterMaxLen                                                       int64
}
type Queue struct {
	client                                                 *redis.Client
	stream, group, consumer, deadLetterStream, retryStream string
	maxAttempts                                            int
	claimIdle, retryDelay                                  time.Duration
	deadLetterMaxLen                                       int64
}

func New(config Config) *Queue {
	if config.Stream == "" {
		config.Stream = "grader:jobs"
	}
	if config.Group == "" {
		config.Group = "grader-workers"
	}
	if config.Consumer == "" {
		config.Consumer = fmt.Sprintf("worker-%d", time.Now().UnixNano())
	}
	if config.DeadLetterStream == "" {
		config.DeadLetterStream = config.Stream + ":dead"
	}
	if config.RetryStream == "" {
		config.RetryStream = config.Stream + ":retry"
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	if config.ClaimIdle <= 0 {
		// This must exceed the longest expected grading job so an active worker
		// is not claimed by another worker. It is only a crash-recovery guard.
		config.ClaimIdle = 15 * time.Minute
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = 10 * time.Second
	}
	if config.DeadLetterMaxLen <= 0 {
		config.DeadLetterMaxLen = 10_000
	}
	return &Queue{client: redis.NewClient(&redis.Options{Addr: config.Addr, Password: config.Password, DB: config.DB}), stream: config.Stream, group: config.Group, consumer: config.Consumer, deadLetterStream: config.DeadLetterStream, retryStream: config.RetryStream, maxAttempts: config.MaxAttempts, claimIdle: config.ClaimIdle, retryDelay: config.RetryDelay, deadLetterMaxLen: config.DeadLetterMaxLen}
}

// NewWithClient reuses the Redis client initialized by app/config.
func NewWithClient(client *redis.Client, config Config) (*Queue, error) {
	if client == nil {
		return nil, errors.New("redis client is nil")
	}
	if config.Stream == "" {
		config.Stream = "grader:jobs"
	}
	if config.Group == "" {
		config.Group = "grader-workers"
	}
	if config.Consumer == "" {
		config.Consumer = fmt.Sprintf("worker-%d", time.Now().UnixNano())
	}
	if config.DeadLetterStream == "" {
		config.DeadLetterStream = config.Stream + ":dead"
	}
	if config.RetryStream == "" {
		config.RetryStream = config.Stream + ":retry"
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	if config.ClaimIdle <= 0 {
		config.ClaimIdle = 15 * time.Minute
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = 10 * time.Second
	}
	if config.DeadLetterMaxLen <= 0 {
		config.DeadLetterMaxLen = 10_000
	}
	return &Queue{client: client, stream: config.Stream, group: config.Group, consumer: config.Consumer, deadLetterStream: config.DeadLetterStream, retryStream: config.RetryStream, maxAttempts: config.MaxAttempts, claimIdle: config.ClaimIdle, retryDelay: config.RetryDelay, deadLetterMaxLen: config.DeadLetterMaxLen}, nil
}

func (q *Queue) Close() error { return q.client.Close() }

func (q *Queue) ensureGroup(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.stream, q.group, "0").Err()
	if err != nil && !stringsContains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (q *Queue) Enqueue(ctx context.Context, job Job) (string, error) {
	if job.ID == "" {
		return "", errors.New("job ID is required")
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return "", err
	}
	if err = q.ensureGroup(ctx); err != nil {
		return "", err
	}
	return q.client.XAdd(ctx, &redis.XAddArgs{Stream: q.stream, Values: map[string]any{"job": payload}}).Result()
}

type Handler func(context.Context, Job) error
type ExhaustedHandler func(context.Context, Job, error) error

// WorkN starts n concurrent workers. Redis failures are retried with bounded
// exponential backoff so a transient outage cannot permanently stop grading.
func (q *Queue) WorkN(ctx context.Context, n int, handler Handler) error {
	return q.WorkNWithExhaustedHandler(ctx, n, handler, nil)
}

// WorkNWithExhaustedHandler invokes exhausted after the final failed attempt,
// before the job is acknowledged and moved to the dead-letter stream.
func (q *Queue) WorkNWithExhaustedHandler(ctx context.Context, n int, handler Handler, exhausted ExhaustedHandler) error {
	if n <= 0 {
		return errors.New("worker count must be greater than zero")
	}
	if handler == nil {
		return errors.New("handler is nil")
	}
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		worker := *q
		worker.consumer = fmt.Sprintf("%s-%d", q.consumer, i)
		go func() { errs <- worker.workWithReconnect(workerCtx, handler, exhausted) }()
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil && !errors.Is(err, context.Canceled) {
			cancel()
			return err
		}
	}
	return ctx.Err()
}

func (q *Queue) workWithReconnect(ctx context.Context, handler Handler, exhausted ExhaustedHandler) error {
	backoff := time.Second
	for ctx.Err() == nil {
		err := q.work(ctx, handler, exhausted)
		if err == nil || errors.Is(err, context.Canceled) || ctx.Err() != nil {
			return ctx.Err()
		}
		log.Printf("grader queue worker %s disconnected: %v; retrying in %s", q.consumer, err, backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
	return ctx.Err()
}

// Work blocks until ctx is cancelled. Failed jobs are re-enqueued with a
// bounded attempt counter; crashed workers are recovered via XAUTOCLAIM.
func (q *Queue) Work(ctx context.Context, handler Handler) error {
	return q.work(ctx, handler, nil)
}

func (q *Queue) work(ctx context.Context, handler Handler, exhausted ExhaustedHandler) error {
	if handler == nil {
		return errors.New("handler is nil")
	}
	if err := q.ensureGroup(ctx); err != nil {
		return err
	}
	// Recover this worker's own pending messages immediately after reconnect.
	// Other consumers are recovered by XAUTOCLAIM after ClaimIdle.
	for {
		messages, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: q.group, Consumer: q.consumer, Streams: []string{q.stream, "0"}, Count: 1, Block: -1}).Result()
		if err == redis.Nil {
			break
		}
		if err != nil {
			return err
		}
		hasMessages := false
		for _, stream := range messages {
			for _, message := range stream.Messages {
				hasMessages = true
				if err := q.handleMessage(ctx, handler, exhausted, message); err != nil {
					return err
				}
			}
		}
		if !hasMessages {
			break
		}
	}
	for ctx.Err() == nil {
		if err := q.promoteDueRetries(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		claimed, _, err := q.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream: q.stream, Group: q.group, Consumer: q.consumer, MinIdle: q.claimIdle, Start: "0-0", Count: 1,
		}).Result()
		if err != nil && err != redis.Nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if len(claimed) > 0 {
			for _, message := range claimed {
				if err := q.handleMessage(ctx, handler, exhausted, message); err != nil {
					return err
				}
			}
			continue
		}
		messages, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: q.group, Consumer: q.consumer, Streams: []string{q.stream, ">"}, Count: 1, Block: time.Second}).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		for _, stream := range messages {
			for _, message := range stream.Messages {
				if err := q.handleMessage(ctx, handler, exhausted, message); err != nil {
					return err
				}
			}
		}
	}
	return ctx.Err()
}

func (q *Queue) handleMessage(ctx context.Context, handler Handler, exhausted ExhaustedHandler, message redis.XMessage) error {
	value, ok := message.Values["job"].(string)
	if !ok {
		// A poison message must not stay in the PEL forever.
		return q.ack(ctx, message.ID)
	}
	var job Job
	if err := json.Unmarshal([]byte(value), &job); err != nil || job.ID == "" {
		return q.ack(ctx, message.ID)
	}
	if err := handler(ctx, job); err != nil {
		return q.retryOrDeadLetter(ctx, message.ID, job, err, exhausted)
	}
	return q.ack(ctx, message.ID)
}

func (q *Queue) retryOrDeadLetter(ctx context.Context, messageID string, job Job, handlerErr error, exhausted ExhaustedHandler) error {
	job.Attempt++
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	pipeline := q.client.TxPipeline()
	if job.Attempt >= q.maxAttempts {
		if exhausted != nil {
			if err := exhausted(ctx, job, handlerErr); err != nil {
				return err
			}
		}
		pipeline.XAdd(ctx, &redis.XAddArgs{Stream: q.deadLetterStream, MaxLen: q.deadLetterMaxLen, Approx: true, Values: map[string]any{"job": payload, "error": truncateError(handlerErr)}})
	} else {
		pipeline.ZAdd(ctx, q.retryStream, redis.Z{Score: float64(time.Now().Add(q.retryDelay).UnixMilli()), Member: string(payload)})
	}
	pipeline.XAck(ctx, q.stream, q.group, messageID)
	pipeline.XDel(ctx, q.stream, messageID)
	_, err = pipeline.Exec(ctx)
	return err
}

const promoteRetriesScript = `
local jobs = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'LIMIT', 0, ARGV[2])
for _, job in ipairs(jobs) do
  if redis.call('ZREM', KEYS[1], job) == 1 then
    redis.call('XADD', KEYS[2], '*', 'job', job)
  end
end
return #jobs
`

func (q *Queue) promoteDueRetries(ctx context.Context) error {
	_, err := q.client.Eval(ctx, promoteRetriesScript, []string{q.retryStream, q.stream}, time.Now().UnixMilli(), 100).Result()
	return err
}

func (q *Queue) ack(ctx context.Context, messageID string) error {
	pipeline := q.client.TxPipeline()
	pipeline.XAck(ctx, q.stream, q.group, messageID)
	pipeline.XDel(ctx, q.stream, messageID)
	_, err := pipeline.Exec(ctx)
	return err
}

func truncateError(err error) string {
	const maxLength = 1024
	message := err.Error()
	if len(message) > maxLength {
		return message[:maxLength]
	}
	return message
}

func stringsContains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
