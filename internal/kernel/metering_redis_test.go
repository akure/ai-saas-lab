package kernel

import (
	"errors"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
)

type mockRedisError string

func (e mockRedisError) Error() string { return string(e) }
func (e mockRedisError) RedisError()  {}

func TestIsRedisConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "redis.Nil error",
			err:      redis.Nil,
			expected: false,
		},
		{
			name:     "redis server response error (WRONGTYPE key collision)",
			err:      mockRedisError("WRONGTYPE Operation against a key holding the wrong kind of value"),
			expected: false,
		},
		{
			name:     "wrapped redis server response error",
			err:      fmt.Errorf("HSET failed: %w", mockRedisError("ERR invalid expire time")),
			expected: false,
		},
		{
			name:     "network connection refused error",
			err:      errors.New("dial tcp 127.0.0.1:6379: connect: connection refused"),
			expected: true,
		},
		{
			name:     "wrapped read timeout error",
			err:      fmt.Errorf("redis: exec: %w", errors.New("i/o timeout")),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRedisConnectionError(tt.err)
			if got != tt.expected {
				t.Errorf("isRedisConnectionError(%v) = %v; want %v", tt.err, got, tt.expected)
			}
		})
	}
}
