package vo

import (
	"errors"
)

var (
	ErrInvalidLatitude  = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude = errors.New("longitude must be between -180 and 180")
)

type Coordinates struct {
	latitude  float64
	longitude float64
}

func NewCoordinates(lat, lon float64) (Coordinates, error) {
	if lat < -90 || lat > 90 {
		return Coordinates{}, ErrInvalidLatitude
	}
	if lon < -180 || lon > 180 {
		return Coordinates{}, ErrInvalidLongitude
	}
	return Coordinates{latitude: lat, longitude: lon}, nil
}

func (c Coordinates) Latitude() float64 {
	return c.latitude
}

func (c Coordinates) Longitude() float64 {
	return c.longitude
}

func (c Coordinates) IsZero() bool {
	return c.latitude == 0 && c.longitude == 0
}
