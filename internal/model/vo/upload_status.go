package vo

import (
	"errors"
	"strings"
)

type UploadStatus string

const (
	UploadStatusPending   UploadStatus = "pending"
	UploadStatusUploaded  UploadStatus = "uploaded"
	UploadStatusCompleted UploadStatus = "completed"
	UploadStatusExpired   UploadStatus = "expired"
)

var (
	ErrInvalidUploadStatus = errors.New("invalid upload status")
)

func ParseUploadStatus(s string) (UploadStatus, error) {
	status := UploadStatus(strings.ToLower(s))
	switch status {
	case UploadStatusPending, UploadStatusUploaded, UploadStatusCompleted, UploadStatusExpired:
		return status, nil
	default:
		return "", ErrInvalidUploadStatus
	}
}

func (s UploadStatus) String() string {
	return string(s)
}

func (s UploadStatus) IsValid() bool {
	switch s {
	case UploadStatusPending, UploadStatusUploaded, UploadStatusCompleted, UploadStatusExpired:
		return true
	default:
		return false
	}
}
