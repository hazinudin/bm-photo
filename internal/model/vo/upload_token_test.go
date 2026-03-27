package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUploadToken_ValidUUID(t *testing.T) {
	token := NewUploadToken()

	assert.NotEmpty(t, token)
	assert.True(t, token.IsValid())
}

func TestParseUploadToken_ValidInput(t *testing.T) {
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
			token, err := ParseUploadToken(tt.input)

			assert.NoError(t, err)
			assert.Equal(t, tt.input, token.String())
			assert.True(t, token.IsValid())
		})
	}
}

func TestParseUploadToken_InvalidInput(t *testing.T) {
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
			token, err := ParseUploadToken(tt.input)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidUploadToken, err)
			assert.True(t, token == "")
		})
	}
}

func TestMustParseUploadToken_ValidInput(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	token := MustParseUploadToken(validUUID)

	assert.Equal(t, validUUID, token.String())
}

func TestMustParseUploadToken_InvalidInput_Panics(t *testing.T) {
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

			MustParseUploadToken(tt.input)
		})
	}
}

func TestUploadToken_String(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	token := UploadToken(validUUID)

	assert.Equal(t, validUUID, token.String())
}

func TestUploadToken_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		token UploadToken
		valid bool
	}{
		{
			name:  "valid UUID",
			token: UploadToken("550e8400-e29b-41d4-a716-446655440000"),
			valid: true,
		},
		{
			name:  "empty token",
			token: UploadToken(""),
			valid: false,
		},
		{
			name:  "invalid UUID",
			token: UploadToken("invalid"),
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.token.IsValid())
		})
	}
}
