package vo

import (
	"errors"
	"strings"
)

type STASource string

const (
	STASourceUserProvided    STASource = "user_provided"
	STASourceLRSInterpolated STASource = "lrs_interpolated"
)

var (
	ErrInvalidSTASource = errors.New("invalid STA source")
)

func ParseSTASource(s string) (STASource, error) {
	source := STASource(strings.ToLower(s))
	switch source {
	case STASourceUserProvided, STASourceLRSInterpolated:
		return source, nil
	default:
		return "", ErrInvalidSTASource
	}
}

func (s STASource) String() string {
	return string(s)
}

func (s STASource) IsValid() bool {
	return s == STASourceUserProvided || s == STASourceLRSInterpolated
}
