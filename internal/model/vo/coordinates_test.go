package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCoordinates_ValidInput(t *testing.T) {
	tests := []struct {
		name      string
		lat       float64
		lon       float64
		expectLat float64
		expectLon float64
	}{
		{
			name:      "Jakarta coordinates",
			lat:       -6.2088,
			lon:       106.8456,
			expectLat: -6.2088,
			expectLon: 106.8456,
		},
		{
			name:      "zero coordinates",
			lat:       0,
			lon:       0,
			expectLat: 0,
			expectLon: 0,
		},
		{
			name:      "maximum latitude",
			lat:       90,
			lon:       0,
			expectLat: 90,
			expectLon: 0,
		},
		{
			name:      "minimum latitude",
			lat:       -90,
			lon:       0,
			expectLat: -90,
			expectLon: 0,
		},
		{
			name:      "maximum longitude",
			lat:       0,
			lon:       180,
			expectLat: 0,
			expectLon: 180,
		},
		{
			name:      "minimum longitude",
			lat:       0,
			lon:       -180,
			expectLat: 0,
			expectLon: -180,
		},
		{
			name:      "equator and prime meridian",
			lat:       0,
			lon:       0,
			expectLat: 0,
			expectLon: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coords, err := NewCoordinates(tt.lat, tt.lon)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectLat, coords.Latitude())
			assert.Equal(t, tt.expectLon, coords.Longitude())
		})
	}
}

func TestNewCoordinates_InvalidLatitude(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lon  float64
	}{
		{
			name: "latitude above maximum",
			lat:  90.0001,
			lon:  0,
		},
		{
			name: "latitude below minimum",
			lat:  -90.0001,
			lon:  0,
		},
		{
			name: "latitude way above maximum",
			lat:  100,
			lon:  0,
		},
		{
			name: "latitude way below minimum",
			lat:  -100,
			lon:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coords, err := NewCoordinates(tt.lat, tt.lon)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidLatitude, err)
			assert.Equal(t, Coordinates{}, coords)
		})
	}
}

func TestNewCoordinates_InvalidLongitude(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lon  float64
	}{
		{
			name: "longitude above maximum",
			lat:  0,
			lon:  180.0001,
		},
		{
			name: "longitude below minimum",
			lat:  0,
			lon:  -180.0001,
		},
		{
			name: "longitude way above maximum",
			lat:  0,
			lon:  200,
		},
		{
			name: "longitude way below minimum",
			lat:  0,
			lon:  -200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coords, err := NewCoordinates(tt.lat, tt.lon)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidLongitude, err)
			assert.Equal(t, Coordinates{}, coords)
		})
	}
}

func TestCoordinates_Latitude(t *testing.T) {
	coords, _ := NewCoordinates(-6.2088, 106.8456)

	assert.Equal(t, -6.2088, coords.Latitude())
}

func TestCoordinates_Longitude(t *testing.T) {
	coords, _ := NewCoordinates(-6.2088, 106.8456)

	assert.Equal(t, 106.8456, coords.Longitude())
}

func TestCoordinates_IsZero(t *testing.T) {
	tests := []struct {
		name   string
		lat    float64
		lon    float64
		isZero bool
	}{
		{
			name:   "zero coordinates",
			lat:    0,
			lon:    0,
			isZero: true,
		},
		{
			name:   "non-zero latitude",
			lat:    -6.2088,
			lon:    0,
			isZero: false,
		},
		{
			name:   "non-zero longitude",
			lat:    0,
			lon:    106.8456,
			isZero: false,
		},
		{
			name:   "both non-zero",
			lat:    -6.2088,
			lon:    106.8456,
			isZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coords, _ := NewCoordinates(tt.lat, tt.lon)

			assert.Equal(t, tt.isZero, coords.IsZero())
		})
	}
}

func TestCoordinates_EdgeCases(t *testing.T) {
	t.Run("valid coordinates at boundaries", func(t *testing.T) {
		coords, err := NewCoordinates(90, 180)
		assert.NoError(t, err)
		assert.Equal(t, 90.0, coords.Latitude())
		assert.Equal(t, 180.0, coords.Longitude())
	})

	t.Run("valid negative coordinates at boundaries", func(t *testing.T) {
		coords, err := NewCoordinates(-90, -180)
		assert.NoError(t, err)
		assert.Equal(t, -90.0, coords.Latitude())
		assert.Equal(t, -180.0, coords.Longitude())
	})
}
