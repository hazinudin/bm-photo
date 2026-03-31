package gcs

import "errors"

var (
	ErrBucketNotFound     = errors.New("GCS bucket not found")
	ErrObjectNotFound     = errors.New("GCS object not found")
	ErrPermissionDenied   = errors.New("permission denied accessing GCS")
	ErrInvalidCredentials = errors.New("invalid service account credentials")
	ErrSignedURLFailed    = errors.New("failed to generate signed URL")
)
