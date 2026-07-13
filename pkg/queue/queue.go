package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ddp0_grader/app/models"
	"github.com/redis/go-redis/v9"
)

type Job struct {
	ID         string            `json:"id"`
	Submission models.Submission `json:"submission"`
	Problem    models.Problem    `json:"problem"`
	TestCases  []models.TestCase `json:"test_cases"`
}

type Config struct {
	Addr, Password, Stream, Group, Consumer string
	DB                                      int
}
type Queue struct {
	client                  *redis.Client
	stream, group, consumer string
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
	return &Queue{redis.NewClient(&redis.Options{Addr: config.Addr, Password: config.Password, DB: config.DB}), config.Stream, config.Group, config.Consumer}
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
	return &Queue{client: client, stream: config.Stream, group: config.Group, consumer: config.Consumer}, nil
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

// WorkN starts n concurrent workers and returns when the context is cancelled
// or one worker encounters a Redis error.
func (q *Queue) WorkN(ctx context.Context, n int, handler Handler) error {
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
		go func() { errs <- worker.Work(workerCtx, handler) }()
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil && !errors.Is(err, context.Canceled) {
			cancel()
			return err
		}
	}
	return ctx.Err()
}

// Work blocks until ctx is cancelled. A message is acknowledged only after handler succeeds.
func (q *Queue) Work(ctx context.Context, handler Handler) error {
	if handler == nil {
		return errors.New("handler is nil")
	}
	if err := q.ensureGroup(ctx); err != nil {
		return err
	}
	for ctx.Err() == nil {
		messages, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: q.group, Consumer: q.consumer, Streams: []string{q.stream, ">"}, Count: 1, Block: 5 * time.Second}).Result()
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
				var job Job
				value, ok := message.Values["job"].(string)
				if !ok {
					continue
				}
				if err := json.Unmarshal([]byte(value), &job); err != nil {
					continue
				}
				if err := handler(ctx, job); err != nil {
					continue
				}
				if err := q.client.XAck(ctx, q.stream, q.group, message.ID).Err(); err != nil {
					return err
				}
			}
		}
	}
	return ctx.Err()
}

func stringsContains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
