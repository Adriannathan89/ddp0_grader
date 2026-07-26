package controller

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	gradeRequestLimit  = 5
	gradeRequestWindow = 30 * time.Second
)

// GradeRateLimiter limits grade requests for one authenticated user.
type GradeRateLimiter interface {
	Allow(ctx context.Context, userID string) (allowed bool, retryAfter time.Duration, err error)
}

type redisGradeRateLimiter struct {
	client *redis.Client
}

func NewRedisGradeRateLimiter(client *redis.Client) GradeRateLimiter {
	return &redisGradeRateLimiter{client: client}
}

var gradeRateLimitScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("PTTL", KEYS[1])
if count > tonumber(ARGV[2]) then
  return {0, ttl}
end
return {1, ttl}
`)

func (l *redisGradeRateLimiter) Allow(ctx context.Context, userID string) (bool, time.Duration, error) {
	if l == nil || l.client == nil {
		return false, 0, errors.New("grade rate limiter is not configured")
	}
	result, err := gradeRateLimitScript.Run(ctx, l.client, []string{"grader:rate:submissions:grade:" + userID}, gradeRequestWindow.Milliseconds(), gradeRequestLimit).Slice()
	if err != nil {
		return false, 0, err
	}
	if len(result) != 2 {
		return false, 0, errors.New("invalid grade rate limiter response")
	}
	allowed, ok := result[0].(int64)
	if !ok {
		return false, 0, errors.New("invalid grade rate limiter allowance")
	}
	ttl, ok := result[1].(int64)
	if !ok {
		return false, 0, errors.New("invalid grade rate limiter ttl")
	}
	if ttl < 0 {
		ttl = 0
	}
	return allowed == 1, time.Duration(ttl) * time.Millisecond, nil
}
