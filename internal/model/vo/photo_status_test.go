package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePhotoStatus_ValidInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected PhotoStatus
	}{
		{
			name:     "processing lowercase",
			input:    "processing",
			expected: PhotoStatusProcessing,
		},
		{
			name:     "PROCESSING uppercase",
			input:    "PROCESSING",
			expected: PhotoStatusProcessing,
		},
		{
			name:     "Processing mixed case",
			input:    "Processing",
			expected: PhotoStatusProcessing,
		},
		{
			name:     "ready lowercase",
			input:    "ready",
			expected: PhotoStatusReady,
		},
		{
			name:     "READY uppercase",
			input:    "READY",
			expected: PhotoStatusReady,
		},
		{
			name:     "Ready mixed case",
			input:    "Ready",
			expected: PhotoStatusReady,
		},
		{
			name:     "failed lowercase",
			input:    "failed",
			expected: PhotoStatusFailed,
		},
		{
			name:     "FAILED uppercase",
			input:    "FAILED",
			expected: PhotoStatusFailed,
		},
		{
			name:     "Failed mixed case",
			input:    "Failed",
			expected: PhotoStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := ParsePhotoStatus(tt.input)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestParsePhotoStatus_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "invalid status",
			input: "pending",
		},
		{
			name:  "invalid status uploaded",
			input: "uploaded",
		},
		{
			name:  "numeric value",
			input: "123",
		},
		{
			name:  "whitespace",
			input: "   ",
		},
		{
			name:  "partial match",
			input: "process",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := ParsePhotoStatus(tt.input)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidPhotoStatus, err)
			assert.True(t, status == "")
		})
	}
}

func TestPhotoStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   PhotoStatus
		expected string
	}{
		{
			name:     "processing",
			status:   PhotoStatusProcessing,
			expected: "processing",
		},
		{
			name:     "ready",
			status:   PhotoStatusReady,
			expected: "ready",
		},
		{
			name:     "failed",
			status:   PhotoStatusFailed,
			expected: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestPhotoStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status PhotoStatus
		valid  bool
	}{
		{
			name:   "processing",
			status: PhotoStatusProcessing,
			valid:  true,
		},
		{
			name:   "ready",
			status: PhotoStatusReady,
			valid:  true,
		},
		{
			name:   "failed",
			status: PhotoStatusFailed,
			valid:  true,
		},
		{
			name:   "empty",
			status: PhotoStatus(""),
			valid:  false,
		},
		{
			name:   "invalid",
			status: PhotoStatus("invalid"),
			valid:  false,
		},
		{
			name:   "pending (upload status, not photo status)",
			status: PhotoStatus("pending"),
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.status.IsValid())
		})
	}
}
