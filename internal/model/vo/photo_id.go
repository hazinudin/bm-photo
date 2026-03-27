package vo

import (
	"errors"

	"github.com/google/uuid"
)

type PhotoID string

var (
	ErrInvalidPhotoID = errors.New("invalid photo ID format")
)

func NewPhotoID() PhotoID {
	return PhotoID(uuid.New().String())
}

func ParsePhotoID(s string) (PhotoID, error) {
	if s == "" {
		return "", ErrInvalidPhotoID
	}
	if _, err := uuid.Parse(s); err != nil {
		return "", ErrInvalidPhotoID
	}
	return PhotoID(s), nil
}

func MustParsePhotoID(s string) PhotoID {
	id, err := ParsePhotoID(s)
	if err != nil {
		panic(err)
	}
	return id
}

func (id PhotoID) String() string {
	return string(id)
}

func (id PhotoID) IsValid() bool {
	_, err := uuid.Parse(string(id))
	return err == nil
}

func (id PhotoID) IsZero() bool {
	return id == ""
}
