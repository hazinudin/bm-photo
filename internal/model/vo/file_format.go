package vo

import (
	"errors"
	"strings"
)

type FileFormat string

const (
	FileFormatJPEG FileFormat = "JPEG"
	FileFormatPNG  FileFormat = "PNG"
)

var (
	ErrInvalidFileFormat = errors.New("invalid file format")
)

func ParseFileFormat(s string) (FileFormat, error) {
	format := FileFormat(strings.ToUpper(s))
	switch format {
	case FileFormatJPEG, FileFormatPNG:
		return format, nil
	default:
		return "", ErrInvalidFileFormat
	}
}

func ParseFileFormatFromContentType(contentType string) (FileFormat, error) {
	switch strings.ToLower(contentType) {
	case "image/jpeg", "image/jpg":
		return FileFormatJPEG, nil
	case "image/png":
		return FileFormatPNG, nil
	default:
		return "", ErrInvalidFileFormat
	}
}

func (f FileFormat) String() string {
	return string(f)
}

func (f FileFormat) ContentType() string {
	switch f {
	case FileFormatJPEG:
		return "image/jpeg"
	case FileFormatPNG:
		return "image/png"
	default:
		return ""
	}
}

func (f FileFormat) IsValid() bool {
	return f == FileFormatJPEG || f == FileFormatPNG
}
