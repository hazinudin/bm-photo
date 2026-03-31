package vo

import "errors"

type UploadStatus string

const (
	UploadStatusPending   UploadStatus = "pending"
	UploadStatusCompleted UploadStatus = "completed"
	UploadStatusExpired   UploadStatus = "expired"
)

var ErrInvalidUploadStatus = errors.New("invalid upload status")

func ParseUploadStatus(s string) (UploadStatus, error) {
	status := UploadStatus(s)
	if !status.IsValid() {
		return "", ErrInvalidUploadStatus
	}
	return status, nil
}

func (s UploadStatus) String() string {
	return string(s)
}

func (s UploadStatus) IsValid() bool {
	switch s {
	case UploadStatusPending, UploadStatusCompleted, UploadStatusExpired:
		return true
	default:
		return false
	}
}
