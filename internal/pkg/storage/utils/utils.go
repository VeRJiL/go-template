package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Common errors
type StorageError struct {
	Operation string
	Path      string
	Err       error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage %s operation failed for path %s: %v", e.Operation, e.Path, e.Err)
}

func NewStorageError(operation, path string, err error) *StorageError {
	return &StorageError{
		Operation: operation,
		Path:      path,
		Err:       err,
	}
}

// Helper functions
func IsImage(mimeType string) bool {
	imageTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
		"image/svg+xml",
		"image/bmp",
		"image/tiff",
	}

	for _, t := range imageTypes {
		if t == mimeType {
			return true
		}
	}
	return false
}

func GetFileExtension(filename string) string {
	if dot := strings.LastIndex(filename, "."); dot > 0 && dot < len(filename)-1 {
		return strings.ToLower(filename[dot+1:])
	}
	return ""
}

func GenerateFilePath(directory, filename string) string {
	// Generate a path like: uploads/avatars/2024/01/15/uuid-filename.jpg
	now := time.Now()
	year := fmt.Sprintf("%d", now.Year())
	month := fmt.Sprintf("%02d", now.Month())
	day := fmt.Sprintf("%02d", now.Day())

	// Generate UUID for filename
	id := uuid.New().String()
	ext := GetFileExtension(filename)

	var newFilename string
	if ext != "" {
		newFilename = fmt.Sprintf("%s-%s.%s", id, SanitizeFilename(filename), ext)
	} else {
		newFilename = fmt.Sprintf("%s-%s", id, SanitizeFilename(filename))
	}

	return fmt.Sprintf("%s/%s/%s/%s/%s", directory, year, month, day, newFilename)
}

func SanitizeFilename(filename string) string {
	// Remove extension
	if dot := strings.LastIndex(filename, "."); dot >= 0 {
		filename = filename[:dot]
	}

	// Replace invalid characters (keep only alphanumeric, dash, underscore)
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_]`)
	filename = reg.ReplaceAllString(filename, "-")

	// Remove consecutive dashes
	reg = regexp.MustCompile(`-+`)
	filename = reg.ReplaceAllString(filename, "-")

	// Remove leading/trailing dashes
	filename = strings.Trim(filename, "-")

	// Limit length
	if len(filename) > 50 {
		filename = filename[:50]
		// Trim again in case we cut in the middle of consecutive dashes
		filename = strings.TrimRight(filename, "-")
	}

	return filename
}