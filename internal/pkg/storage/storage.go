package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Storage defines the interface that all storage drivers must implement
// Similar to Laravel's Storage facade
type Storage interface {
	// Core operations
	Put(ctx context.Context, path string, content io.Reader) error
	PutFile(ctx context.Context, path string, file *multipart.FileHeader) error
	Get(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
	
	// File information
	Size(ctx context.Context, path string) (int64, error)
	LastModified(ctx context.Context, path string) (time.Time, error)
	MimeType(ctx context.Context, path string) (string, error)
	
	// Directory operations
	Files(ctx context.Context, directory string) ([]string, error)
	AllFiles(ctx context.Context, directory string) ([]string, error)
	Directories(ctx context.Context, directory string) ([]string, error)
	MakeDirectory(ctx context.Context, path string) error
	DeleteDirectory(ctx context.Context, directory string) error
	
	// URL generation
	URL(ctx context.Context, path string) (string, error)
	TemporaryURL(ctx context.Context, path string, expiration time.Duration) (string, error)
	
	// Utility methods
	Copy(ctx context.Context, from, to string) error
	Move(ctx context.Context, from, to string) error
	
	// Driver information
	Driver() string
}

// FileInfo represents information about a stored file
type FileInfo struct {
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	MimeType     string    `json:"mime_type"`
	Extension    string    `json:"extension"`
	LastModified time.Time `json:"last_modified"`
	URL          string    `json:"url,omitempty"`
	Driver       string    `json:"driver"`
}

// UploadedFile represents an uploaded file with metadata
type UploadedFile struct {
	FileInfo
	OriginalName string            `json:"original_name"`
	Hash         string            `json:"hash,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	UploadedAt   time.Time         `json:"uploaded_at"`
}

// ImageVariant represents different sizes/versions of an image
type ImageVariant struct {
	Name   string `json:"name"`   // thumbnail, medium, large, original
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   int64  `json:"size"`
	URL    string `json:"url"`
}

// ImageUpload represents an uploaded image with its variants
type ImageUpload struct {
	UploadedFile
	Variants []ImageVariant `json:"variants,omitempty"`
	Width    int            `json:"width"`
	Height   int            `json:"height"`
	IsImage  bool           `json:"is_image"`
}

// StorageConfig holds configuration for different storage drivers
type StorageConfig struct {
	Default string                 `json:"default"`
	Disks   map[string]DiskConfig  `json:"disks"`
}

type DiskConfig struct {
	Driver string            `json:"driver"`
	Config map[string]string `json:"config"`
}

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

// Storage operation options
type PutOptions struct {
	MimeType    string
	Metadata    map[string]string
	Permissions string
	CacheControl string
}

type GetOptions struct {
	Range string // For partial content requests
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
	if dot := strings.LastIndex(filename, "."); dot >= 0 {
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
		newFilename = fmt.Sprintf("%s-%s.%s", id, sanitizeFilename(filename), ext)
	} else {
		newFilename = fmt.Sprintf("%s-%s", id, sanitizeFilename(filename))
	}
	
	return fmt.Sprintf("%s/%s/%s/%s/%s", directory, year, month, day, newFilename)
}

func sanitizeFilename(filename string) string {
	// Remove extension and invalid characters
	if dot := strings.LastIndex(filename, "."); dot >= 0 {
		filename = filename[:dot]
	}
	
	// Replace invalid characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_]`)
	filename = reg.ReplaceAllString(filename, "-")
	
	// Remove consecutive dashes and limit length
	reg = regexp.MustCompile(`-+`)
	filename = reg.ReplaceAllString(filename, "-")
	filename = strings.Trim(filename, "-")
	
	if len(filename) > 50 {
		filename = filename[:50]
	}
	
	return filename
}