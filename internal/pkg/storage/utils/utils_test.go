package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStorageError(t *testing.T) {
	t.Run("should create storage error with correct message", func(t *testing.T) {
		originalErr := assert.AnError
		operation := "put"
		path := "/test/file.txt"

		storageErr := NewStorageError(operation, path, originalErr)

		assert.Equal(t, operation, storageErr.Operation)
		assert.Equal(t, path, storageErr.Path)
		assert.Equal(t, originalErr, storageErr.Err)

		expectedMsg := "storage put operation failed for path /test/file.txt: assert.AnError general error for testing"
		assert.Equal(t, expectedMsg, storageErr.Error())
	})

	t.Run("should handle nil error", func(t *testing.T) {
		storageErr := NewStorageError("get", "/path", nil)

		expectedMsg := "storage get operation failed for path /path: <nil>"
		assert.Equal(t, expectedMsg, storageErr.Error())
	})
}

func TestIsImage(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{"JPEG image", "image/jpeg", true},
		{"JPG image", "image/jpg", true},
		{"PNG image", "image/png", true},
		{"GIF image", "image/gif", true},
		{"WebP image", "image/webp", true},
		{"SVG image", "image/svg+xml", true},
		{"BMP image", "image/bmp", true},
		{"TIFF image", "image/tiff", true},
		{"PDF document", "application/pdf", false},
		{"Text file", "text/plain", false},
		{"Video file", "video/mp4", false},
		{"Empty mime type", "", false},
		{"Unknown mime type", "application/unknown", false},
		{"Case sensitive - should not match", "IMAGE/JPEG", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsImage(tt.mimeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"JPEG file", "image.jpg", "jpg"},
		{"PNG file", "photo.PNG", "png"},
		{"Multiple dots", "file.name.txt", "txt"},
		{"No extension", "filename", ""},
		{"Dot at start", ".hidden", ""},
		{"Just dot", ".", ""},
		{"Empty filename", "", ""},
		{"Complex filename", "my-file_2024.backup.zip", "zip"},
		{"Uppercase extension", "FILE.PDF", "pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFileExtension(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateFilePath(t *testing.T) {
	t.Run("should generate valid file path with date structure", func(t *testing.T) {
		directory := "uploads/avatars"
		filename := "profile.jpg"

		path := GenerateFilePath(directory, filename)

		// Should match pattern: uploads/avatars/YYYY/MM/DD/uuid-profile.jpg
		assert.Contains(t, path, directory)
		assert.Contains(t, path, "-profile.jpg")

		// Check date structure
		now := time.Now()
		year := now.Format("2006")
		month := now.Format("01")
		day := now.Format("02")

		assert.Contains(t, path, "/"+year+"/")
		assert.Contains(t, path, "/"+month+"/")
		assert.Contains(t, path, "/"+day+"/")

		// Should contain UUID (36 chars + hyphens)
		parts := path[len(directory)+1:] // Remove directory prefix
		assert.True(t, len(parts) > 50)  // Should be long due to UUID
	})

	t.Run("should handle filename without extension", func(t *testing.T) {
		directory := "docs"
		filename := "readme"

		path := GenerateFilePath(directory, filename)

		assert.Contains(t, path, directory)
		assert.Contains(t, path, "-readme")
		assert.NotContains(t, path, ".") // No extension should be added
	})

	t.Run("should sanitize special characters in filename", func(t *testing.T) {
		directory := "uploads"
		filename := "my file@#$%.jpg"

		path := GenerateFilePath(directory, filename)

		// Should sanitize special characters
		assert.Contains(t, path, "-my-file.jpg")
		assert.NotContains(t, path, "@")
		assert.NotContains(t, path, "#")
		assert.NotContains(t, path, "$")
		assert.NotContains(t, path, "%")
	})
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"Normal filename", "profile.jpg", "profile"},
		{"Special characters", "my@file#$%.txt", "my-file"},
		{"Multiple spaces", "my   file.pdf", "my-file"},
		{"Consecutive dashes", "my---file.doc", "my-file"},
		{"Leading/trailing dashes", "-file-.txt", "file"},
		{"Unicode characters", "файл.jpg", ""},
		{"Empty after sanitization", "@#$%.txt", ""},
		{"Very long filename", "this-is-a-very-long-filename-that-should-be-truncated-because-it-exceeds-fifty-characters.txt", "this-is-a-very-long-filename-that-should-be-trunca"},
		{"Mixed case", "MyFile.TXT", "MyFile"},
		{"Numbers and hyphens", "file-123_456.zip", "file-123_456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.filename)
			assert.Equal(t, tt.expected, result)

			// Ensure result doesn't exceed 50 characters
			assert.True(t, len(result) <= 50)

			// Ensure no consecutive dashes (unless empty)
			if result != "" {
				assert.NotContains(t, result, "--")
				// Ensure no leading/trailing dashes
				if len(result) > 0 {
					assert.NotEqual(t, "-", string(result[0]))
					assert.NotEqual(t, "-", string(result[len(result)-1]))
				}
			}
		})
	}
}