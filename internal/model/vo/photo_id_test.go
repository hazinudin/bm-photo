package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPhotoID_ValidUUID(t *testing.T) {
	id := NewPhotoID()

	assert.NotEmpty(t, id)
	assert.True(t, id.IsValid())
	assert.False(t, id.IsZero())
}

func TestParsePhotoID_ValidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "valid UUID",
			input: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:  "valid UUID uppercase",
			input: "550E8400-E29B-41D4-A716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParsePhotoID(tt.input)

			assert.NoError(t, err)
			assert.Equal(t, tt.input, id.String())
			assert.True(t, id.IsValid())
		})
	}
}

func TestParsePhotoID_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "invalid UUID format",
			input: "not-a-uuid",
		},
		{
			name:  "partial UUID",
			input: "550e8400-e29b",
		},
		{
			name:  "UUID with special chars",
			input: "550e8400-e29b-41d4-a716-446655440000!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParsePhotoID(tt.input)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidPhotoID, err)
			assert.True(t, id.IsZero())
		})
	}
}

func TestMustParsePhotoID_ValidInput(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	id := MustParsePhotoID(validUUID)

	assert.Equal(t, validUUID, id.String())
}

func TestMustParsePhotoID_InvalidInput_Panics(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "invalid UUID", input: "not-a-uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				assert.NotNil(t, r)
			}()

			MustParsePhotoID(tt.input)
		})
	}
}

func TestPhotoID_String(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	id := PhotoID(validUUID)

	assert.Equal(t, validUUID, id.String())
}

func TestPhotoID_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		id    PhotoID
		valid bool
	}{
		{
			name:  "valid UUID",
			id:    PhotoID("550e8400-e29b-41d4-a716-446655440000"),
			valid: true,
		},
		{
			name:  "empty ID",
			id:    PhotoID(""),
			valid: false,
		},
		{
			name:  "invalid UUID",
			id:    PhotoID("invalid"),
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.id.IsValid())
		})
	}
}

func TestPhotoID_IsZero(t *testing.T) {
	tests := []struct {
		name   string
		id     PhotoID
		isZero bool
	}{
		{
			name:   "empty ID",
			id:     PhotoID(""),
			isZero: true,
		},
		{
			name:   "valid UUID",
			id:     PhotoID("550e8400-e29b-41d4-a716-446655440000"),
			isZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isZero, tt.id.IsZero())
		})
	}
}
