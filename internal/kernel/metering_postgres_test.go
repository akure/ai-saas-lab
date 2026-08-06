package kernel

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsConnectionError(t *testing.T) {
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
			name:     "query-level error (PgError constraint violation)",
			err:      &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"},
			expected: false,
		},
		{
			name:     "wrapped query-level error",
			err:      fmt.Errorf("insert failed: %w", &pgconn.PgError{Code: "23502", Message: "null value in column violates not-null constraint"}),
			expected: false,
		},
		{
			name:     "generic network/connection error",
			err:      errors.New("dial tcp 127.0.0.1:5432: connect: connection refused"),
			expected: true,
		},
		{
			name:     "wrapped connection reset error",
			err:      fmt.Errorf("postgres: exec: %w", errors.New("read tcp: connection reset by peer")),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isConnectionError(tt.err)
			if got != tt.expected {
				t.Errorf("isConnectionError(%v) = %v; want %v", tt.err, got, tt.expected)
			}
		})
	}
}
