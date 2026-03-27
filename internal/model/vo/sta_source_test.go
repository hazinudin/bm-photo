package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSTASource_ValidInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected STASource
	}{
		{
			name:     "user_provided lowercase",
			input:    "user_provided",
			expected: STASourceUserProvided,
		},
		{
			name:     "user_provided uppercase",
			input:    "USER_PROVIDED",
			expected: STASourceUserProvided,
		},
		{
			name:     "user_provided mixed case",
			input:    "User_Provided",
			expected: STASourceUserProvided,
		},
		{
			name:     "lrs_interpolated lowercase",
			input:    "lrs_interpolated",
			expected: STASourceLRSInterpolated,
		},
		{
			name:     "lrs_interpolated uppercase",
			input:    "LRS_INTERPOLATED",
			expected: STASourceLRSInterpolated,
		},
		{
			name:     "lrs_interpolated mixed case",
			input:    "Lrs_Interpolated",
			expected: STASourceLRSInterpolated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := ParseSTASource(tt.input)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, source)
		})
	}
}

func TestParseSTASource_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "invalid value",
			input: "invalid_source",
		},
		{
			name:  "numeric value",
			input: "123",
		},
		{
			name:  "partial match",
			input: "user",
		},
		{
			name:  "whitespace",
			input: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := ParseSTASource(tt.input)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidSTASource, err)
			assert.True(t, source == "")
		})
	}
}

func TestSTASource_String(t *testing.T) {
	tests := []struct {
		name     string
		source   STASource
		expected string
	}{
		{
			name:     "user_provided",
			source:   STASourceUserProvided,
			expected: "user_provided",
		},
		{
			name:     "lrs_interpolated",
			source:   STASourceLRSInterpolated,
			expected: "lrs_interpolated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.source.String())
		})
	}
}

func TestSTASource_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		source STASource
		valid  bool
	}{
		{
			name:   "user_provided",
			source: STASourceUserProvided,
			valid:  true,
		},
		{
			name:   "lrs_interpolated",
			source: STASourceLRSInterpolated,
			valid:  true,
		},
		{
			name:   "empty",
			source: STASource(""),
			valid:  false,
		},
		{
			name:   "invalid",
			source: STASource("invalid"),
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.source.IsValid())
		})
	}
}
