package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			name:   "uploaded (removed state)",
			status: UploadStatus("uploaded"),
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.status.IsValid())
		})
	}
}
