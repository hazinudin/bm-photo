package vo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFileFormat_ValidInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected FileFormat
	}{
		{
			name:     "JPEG uppercase",
			input:    "JPEG",
			expected: FileFormatJPEG,
		},
		{
			name:     "jpeg lowercase",
			input:    "jpeg",
			expected: FileFormatJPEG,
		},
		{
			name:     "jpeg mixed case",
			input:    "Jpeg",
			expected: FileFormatJPEG,
		},
		{
			name:     "PNG uppercase",
			input:    "PNG",
			expected: FileFormatPNG,
		},
		{
			name:     "png lowercase",
			input:    "png",
			expected: FileFormatPNG,
		},
		{
			name:     "png mixed case",
			input:    "Png",
			expected: FileFormatPNG,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := ParseFileFormat(tt.input)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, format)
		})
	}
}

func TestParseFileFormat_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "invalid format",
			input: "GIF",
		},
		{
			name:  "invalid format bmp",
			input: "BMP",
		},
		{
			name:  "numeric value",
			input: "123",
		},
		{
			name:  "whitespace",
			input: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := ParseFileFormat(tt.input)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidFileFormat, err)
			assert.True(t, format == "")
		})
	}
}

func TestParseFileFormatFromContentType_ValidInput(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    FileFormat
	}{
		{
			name:        "image/jpeg",
			contentType: "image/jpeg",
			expected:    FileFormatJPEG,
		},
		{
			name:        "image/jpg",
			contentType: "image/jpg",
			expected:    FileFormatJPEG,
		},
		{
			name:        "image/JPEG uppercase",
			contentType: "image/JPEG",
			expected:    FileFormatJPEG,
		},
		{
			name:        "image/png",
			contentType: "image/png",
			expected:    FileFormatPNG,
		},
		{
			name:        "image/PNG uppercase",
			contentType: "image/PNG",
			expected:    FileFormatPNG,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := ParseFileFormatFromContentType(tt.contentType)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, format)
		})
	}
}

func TestParseFileFormatFromContentType_InvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{
			name:        "empty string",
			contentType: "",
		},
		{
			name:        "unsupported format gif",
			contentType: "image/gif",
		},
		{
			name:        "unsupported format webp",
			contentType: "image/webp",
		},
		{
			name:        "invalid content type",
			contentType: "text/html",
		},
		{
			name:        "partial content type",
			contentType: "jpeg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := ParseFileFormatFromContentType(tt.contentType)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidFileFormat, err)
			assert.True(t, format == "")
		})
	}
}

func TestFileFormat_String(t *testing.T) {
	tests := []struct {
		name     string
		format   FileFormat
		expected string
	}{
		{
			name:     "JPEG",
			format:   FileFormatJPEG,
			expected: "JPEG",
		},
		{
			name:     "PNG",
			format:   FileFormatPNG,
			expected: "PNG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.format.String())
		})
	}
}

func TestFileFormat_ContentType(t *testing.T) {
	tests := []struct {
		name        string
		format      FileFormat
		contentType string
	}{
		{
			name:        "JPEG content type",
			format:      FileFormatJPEG,
			contentType: "image/jpeg",
		},
		{
			name:        "PNG content type",
			format:      FileFormatPNG,
			contentType: "image/png",
		},
		{
			name:        "invalid format",
			format:      FileFormat(""),
			contentType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.contentType, tt.format.ContentType())
		})
	}
}

func TestFileFormat_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		format FileFormat
		valid  bool
	}{
		{
			name:   "JPEG",
			format: FileFormatJPEG,
			valid:  true,
		},
		{
			name:   "PNG",
			format: FileFormatPNG,
			valid:  true,
		},
		{
			name:   "empty",
			format: FileFormat(""),
			valid:  false,
		},
		{
			name:   "invalid",
			format: FileFormat("GIF"),
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.format.IsValid())
		})
	}
}
