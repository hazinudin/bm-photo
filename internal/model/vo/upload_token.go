package vo

import (
	"errors"

	"github.com/google/uuid"
)

type UploadToken string

var (
	ErrInvalidUploadToken = errors.New("invalid upload token format")
)

func NewUploadToken() UploadToken {
	return UploadToken(uuid.New().String())
}

func ParseUploadToken(s string) (UploadToken, error) {
	if s == "" {
		return "", ErrInvalidUploadToken
	}
	if _, err := uuid.Parse(s); err != nil {
		return "", ErrInvalidUploadToken
	}
	return UploadToken(s), nil
}

func MustParseUploadToken(s string) UploadToken {
	token, err := ParseUploadToken(s)
	if err != nil {
		panic(err)
	}
	return token
}

func (t UploadToken) String() string {
	return string(t)
}

func (t UploadToken) IsValid() bool {
	_, err := uuid.Parse(string(t))
	return err == nil
}
