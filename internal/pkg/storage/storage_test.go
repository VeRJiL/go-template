package storage

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
		assert.True(t, len(parts) > 50) // Should be long due to UUID
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
		{"Unicode characters", "файл.jpg", "----"},
		{"Empty after sanitization", "@#$%.txt", ""},
		{"Very long filename", "this-is-a-very-long-filename-that-should-be-truncated-because-it-exceeds-fifty-characters.txt", "this-is-a-very-long-filename-that-should-be-truncat"},
		{"Mixed case", "MyFile.TXT", "MyFile"},
		{"Numbers and hyphens", "file-123_456.zip", "file-123-456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.filename)
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

func TestFileInfo(t *testing.T) {
	t.Run("should create file info struct correctly", func(t *testing.T) {
		now := time.Now()

		fileInfo := FileInfo{
			Path:         "/uploads/test.jpg",
			Name:         "test.jpg",
			Size:         1024,
			MimeType:     "image/jpeg",
			Extension:    "jpg",
			LastModified: now,
			URL:          "http://example.com/uploads/test.jpg",
			Driver:       "local",
		}

		assert.Equal(t, "/uploads/test.jpg", fileInfo.Path)
		assert.Equal(t, "test.jpg", fileInfo.Name)
		assert.Equal(t, int64(1024), fileInfo.Size)
		assert.Equal(t, "image/jpeg", fileInfo.MimeType)
		assert.Equal(t, "jpg", fileInfo.Extension)
		assert.Equal(t, now, fileInfo.LastModified)
		assert.Equal(t, "http://example.com/uploads/test.jpg", fileInfo.URL)
		assert.Equal(t, "local", fileInfo.Driver)
	})
}

func TestUploadedFile(t *testing.T) {
	t.Run("should create uploaded file struct correctly", func(t *testing.T) {
		now := time.Now()

		uploadedFile := UploadedFile{
			FileInfo: FileInfo{
				Path:      "/uploads/test.jpg",
				Name:      "test.jpg",
				Size:      1024,
				MimeType:  "image/jpeg",
				Extension: "jpg",
				Driver:    "local",
			},
			OriginalName: "my-photo.jpg",
			Hash:         "abc123",
			Metadata:     map[string]string{"description": "test image"},
			UploadedAt:   now,
		}

		assert.Equal(t, "my-photo.jpg", uploadedFile.OriginalName)
		assert.Equal(t, "abc123", uploadedFile.Hash)
		assert.Equal(t, "test image", uploadedFile.Metadata["description"])
		assert.Equal(t, now, uploadedFile.UploadedAt)

		// Check embedded FileInfo
		assert.Equal(t, "/uploads/test.jpg", uploadedFile.Path)
		assert.Equal(t, "image/jpeg", uploadedFile.MimeType)
	})
}

func TestImageVariant(t *testing.T) {
	t.Run("should create image variant correctly", func(t *testing.T) {
		variant := ImageVariant{
			Name:   "thumbnail",
			Path:   "/uploads/thumb_test.jpg",
			Width:  150,
			Height: 150,
			Size:   2048,
			URL:    "http://example.com/uploads/thumb_test.jpg",
		}

		assert.Equal(t, "thumbnail", variant.Name)
		assert.Equal(t, "/uploads/thumb_test.jpg", variant.Path)
		assert.Equal(t, 150, variant.Width)
		assert.Equal(t, 150, variant.Height)
		assert.Equal(t, int64(2048), variant.Size)
		assert.Equal(t, "http://example.com/uploads/thumb_test.jpg", variant.URL)
	})
}

func TestImageUpload(t *testing.T) {
	t.Run("should create image upload with variants", func(t *testing.T) {
		now := time.Now()

		variants := []ImageVariant{
			{
				Name:   "thumbnail",
				Path:   "/uploads/thumb_test.jpg",
				Width:  150,
				Height: 150,
				Size:   2048,
				URL:    "http://example.com/uploads/thumb_test.jpg",
			},
			{
				Name:   "medium",
				Path:   "/uploads/med_test.jpg",
				Width:  500,
				Height: 500,
				Size:   8192,
				URL:    "http://example.com/uploads/med_test.jpg",
			},
		}

		imageUpload := ImageUpload{
			UploadedFile: UploadedFile{
				FileInfo: FileInfo{
					Path:      "/uploads/test.jpg",
					Name:      "test.jpg",
					Size:      16384,
					MimeType:  "image/jpeg",
					Extension: "jpg",
					Driver:    "local",
				},
				OriginalName: "my-photo.jpg",
				UploadedAt:   now,
			},
			Variants: variants,
			Width:    800,
			Height:   600,
			IsImage:  true,
		}

		assert.True(t, imageUpload.IsImage)
		assert.Equal(t, 800, imageUpload.Width)
		assert.Equal(t, 600, imageUpload.Height)
		assert.Len(t, imageUpload.Variants, 2)
		assert.Equal(t, "thumbnail", imageUpload.Variants[0].Name)
		assert.Equal(t, "medium", imageUpload.Variants[1].Name)

		// Check embedded structs
		assert.Equal(t, "my-photo.jpg", imageUpload.OriginalName)
		assert.Equal(t, "/uploads/test.jpg", imageUpload.Path)
	})
}

func TestStorageConfig(t *testing.T) {
	t.Run("should create storage config correctly", func(t *testing.T) {
		diskConfigs := map[string]DiskConfig{
			"local": {
				Driver: "local",
				Config: map[string]string{
					"path": "/var/uploads",
				},
			},
			"s3": {
				Driver: "s3",
				Config: map[string]string{
					"bucket": "my-bucket",
					"region": "us-east-1",
				},
			},
		}

		config := StorageConfig{
			Default: "local",
			Disks:   diskConfigs,
		}

		assert.Equal(t, "local", config.Default)
		assert.Len(t, config.Disks, 2)
		assert.Equal(t, "local", config.Disks["local"].Driver)
		assert.Equal(t, "/var/uploads", config.Disks["local"].Config["path"])
		assert.Equal(t, "s3", config.Disks["s3"].Driver)
		assert.Equal(t, "my-bucket", config.Disks["s3"].Config["bucket"])
	})
}

func TestPutOptions(t *testing.T) {
	t.Run("should create put options correctly", func(t *testing.T) {
		metadata := map[string]string{
			"author": "test user",
			"tags":   "important,urgent",
		}

		options := PutOptions{
			MimeType:     "image/jpeg",
			Metadata:     metadata,
			Permissions:  "0644",
			CacheControl: "max-age=3600",
		}

		assert.Equal(t, "image/jpeg", options.MimeType)
		assert.Equal(t, "test user", options.Metadata["author"])
		assert.Equal(t, "0644", options.Permissions)
		assert.Equal(t, "max-age=3600", options.CacheControl)
	})
}

func TestGetOptions(t *testing.T) {
	t.Run("should create get options correctly", func(t *testing.T) {
		options := GetOptions{
			Range: "bytes=0-1023",
		}

		assert.Equal(t, "bytes=0-1023", options.Range)
	})
}