package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestParseExpiresIn(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNil   bool
		wantHours float64 // approximate expected hours from now
	}{
		{"nil JSON", "null", true, 0},
		{"empty raw", "", true, 0},
		{"empty string", `""`, true, 0},
		{"duration 24h", `"24h"`, false, 24},
		{"duration 30m", `"30m"`, false, 0.5},
		{"hours number", `168`, false, 168},
		{"hours zero", `0`, true, 0},
		{"hours negative", `-5`, true, 0},
		{"invalid string", `"not-a-duration"`, false, -1}, // will error
		{"invalid type", `true`, false, -1},                // will error
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var raw json.RawMessage
			if tc.input == "" {
				raw = nil
			} else {
				raw = json.RawMessage(tc.input)
			}

			result, err := parseExpiresIn(raw)

			if tc.wantNil {
				require.NoError(t, err)
				assert.Nil(t, result)
				return
			}

			if tc.wantHours < 0 {
				// Expect an error
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// Check the expiry is approximately correct (within 1 minute)
			expected := time.Now().Add(time.Duration(tc.wantHours * float64(time.Hour)))
			assert.WithinDuration(t, expected, *result, time.Minute)
		})
	}
}

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			"unique violation",
			&pgconn.PgError{Code: "23505"},
			true,
		},
		{
			"foreign key violation",
			&pgconn.PgError{Code: "23503"},
			false,
		},
		{
			"not a pg error",
			assert.AnError,
			false,
		},
		{
			"nil error",
			nil,
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isUniqueViolation(tc.err))
		})
	}
}
