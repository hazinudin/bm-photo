package vo

import (
	"errors"
	"strings"
)

type PhotoStatus string

const (
	PhotoStatusProcessing PhotoStatus = "processing"
	PhotoStatusReady      PhotoStatus = "ready"
	PhotoStatusFailed     PhotoStatus = "failed"
)

var (
	ErrInvalidPhotoStatus = errors.New("invalid photo status")
)

func ParsePhotoStatus(s string) (PhotoStatus, error) {
	status := PhotoStatus(strings.ToLower(s))
	switch status {
	case PhotoStatusProcessing, PhotoStatusReady, PhotoStatusFailed:
		return status, nil
	default:
		return "", ErrInvalidPhotoStatus
	}
}

func (s PhotoStatus) String() string {
	return string(s)
}

func (s PhotoStatus) IsValid() bool {
	switch s {
	case PhotoStatusProcessing, PhotoStatusReady, PhotoStatusFailed:
		return true
	default:
		return false
	}
}
