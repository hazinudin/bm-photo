package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUploadStatus_ValidInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected UploadStatus
	}{
		{
			name:     "pending lowercase",
			input:    "pending",
			expected: UploadStatusPending,
		},
		{
			name:     "PENDING uppercase",
			input:    "PENDING",
			expected: UploadStatusPending,
		},
		{
			name:     "Pending mixed case",
			input:    "Pending",
			expected: UploadStatusPending,
		},
		{
			name:     "uploaded lowercase",
			input:    "uploaded",
			expected: UploadStatusUploaded,
		},
		{
			name:     "UPLOADED uppercase",
			input:    "UPLOADED",
			expected: UploadStatusUploaded,
		},
		{
			name:     "Uploaded mixed case",
			input:    "Uploaded",
			expected: UploadStatusUploaded,
		},
		{
			name:     "completed lowercase",
			input:    "completed",
			expected: UploadStatusCompleted,
		},
		{
			name:     "COMPLETED uppercase",
			input:    "COMPLETED",
			expected: UploadStatusCompleted,
		},
		{
			name:     "Completed mixed case",
			input:    "Completed",
			expected: UploadStatusCompleted,
		},
		{
			name:     "expired lowercase",
			input:    "expired",
			expected: UploadStatusExpired,
		},
		{
			name:     "EXPIRED uppercase",
			input:    "EXPIRED",
			expected: UploadStatusExpired,
		},
		{
			name:     "Expired mixed case",
			input:    "Expired",
			expected: UploadStatusExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := ParseUploadStatus(tt.input)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestParseUploadStatus_InvalidInput(t *testing.T) {
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
			input: "processing",
		},
		{
			name:  "invalid status ready",
			input: "ready",
		},
		{
			name:  "invalid status failed",
			input: "failed",
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
			input: "pend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := ParseUploadStatus(tt.input)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidUploadStatus, err)
			assert.True(t, status == "")
		})
	}
}

func TestUploadStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   UploadStatus
		expected string
	}{
		{
			name:     "pending",
			status:   UploadStatusPending,
			expected: "pending",
		},
		{
			name:     "uploaded",
			status:   UploadStatusUploaded,
			expected: "uploaded",
		},
		{
			name:     "completed",
			status:   UploadStatusCompleted,
			expected: "completed",
		},
		{
			name:     "expired",
			status:   UploadStatusExpired,
			expected: "expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestUploadStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status UploadStatus
		valid  bool
	}{
		{
			name:   "pending",
			status: UploadStatusPending,
			valid:  true,
		},
		{
			name:   "uploaded",
			status: UploadStatusUploaded,
			valid:  true,
		},
		{
			name:   "completed",
			status: UploadStatusCompleted,
			valid:  true,
		},
		{
			name:   "expired",
			status: UploadStatusExpired,
			valid:  true,
		},
		{
			name:   "empty",
			status: UploadStatus(""),
			valid:  false,
		},
		{
			name:   "invalid",
			status: UploadStatus("invalid"),
			valid:  false,
		},
		{
			name:   "processing (photo status, not upload status)",
			status: UploadStatus("processing"),
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.status.IsValid())
		})
	}
}
