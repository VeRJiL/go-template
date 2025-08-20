package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/VeRJiL/go-template/internal/config"
	"github.com/VeRJiL/go-template/internal/pkg/storage/drivers"
)

// Manager manages multiple storage drivers similar to Laravel's Storage facade
type Manager struct {
	drivers    map[string]Storage
	defaultDisk string
}

// NewManager creates a new storage manager
func NewManager(cfg *config.StorageConfig) (*Manager, error) {
	manager := &Manager{
		drivers:     make(map[string]Storage),
		defaultDisk: cfg.Provider,
	}

	// Initialize local driver
	if cfg.Provider == "local" || cfg.Local.Path != "" {
		localDriver := drivers.NewLocalDriver(
			cfg.Local.Path,
			"", // Base URL (will be set from server config)
			cfg.Local.URLPrefix,
		)
		manager.drivers["local"] = localDriver
	}

	// Initialize S3 driver
	if cfg.Provider == "s3" || (cfg.S3.Bucket != "" && cfg.S3.AccessKey != "") {
		s3Config := drivers.S3Config{
			Region:         cfg.S3.Region,
			Bucket:         cfg.S3.Bucket,
			AccessKey:      cfg.S3.AccessKey,
			SecretKey:      cfg.S3.SecretKey,
			UseSSL:         cfg.S3.UseSSL,
			ForcePathStyle: cfg.S3.ForcePathStyle,
			PublicURL:      "", // Can be configured if needed
		}

		s3Driver, err := drivers.NewS3Driver(s3Config)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize S3 driver: %w", err)
		}
		manager.drivers["s3"] = s3Driver
	}

	// Initialize MinIO driver
	if cfg.Provider == "minio" || (cfg.MinIO.Endpoint != "" && cfg.MinIO.AccessKey != "") {
		minioConfig := drivers.MinIOConfig{
			Endpoint:  cfg.MinIO.Endpoint,
			AccessKey: cfg.MinIO.AccessKey,
			SecretKey: cfg.MinIO.SecretKey,
			Bucket:    cfg.MinIO.Bucket,
			UseSSL:    cfg.MinIO.UseSSL,
			PublicURL: cfg.MinIO.PublicURL,
		}

		minioDriver, err := drivers.NewMinIODriver(minioConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize MinIO driver: %w", err)
		}
		manager.drivers["minio"] = minioDriver
	}

	// Initialize Cloudflare R2 driver
	if cfg.Provider == "cloudflare_r2" || (cfg.CloudflareR2.AccountID != "" && cfg.CloudflareR2.AccessKey != "") {
		r2Config := drivers.CloudflareR2Config{
			AccountID: cfg.CloudflareR2.AccountID,
			AccessKey: cfg.CloudflareR2.AccessKey,
			SecretKey: cfg.CloudflareR2.SecretKey,
			Bucket:    cfg.CloudflareR2.Bucket,
			PublicURL: cfg.CloudflareR2.PublicURL,
		}

		r2Driver, err := drivers.NewCloudflareR2Driver(r2Config)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Cloudflare R2 driver: %w", err)
		}
		manager.drivers["cloudflare_r2"] = r2Driver
	}

	// Initialize Backblaze B2 driver
	if cfg.Provider == "backblaze_b2" || (cfg.BackblazeB2.Region != "" && cfg.BackblazeB2.KeyID != "") {
		b2Config := drivers.BackblazeB2Config{
			Region:    cfg.BackblazeB2.Region,
			KeyID:     cfg.BackblazeB2.KeyID,
			KeySecret: cfg.BackblazeB2.KeySecret,
			Bucket:    cfg.BackblazeB2.Bucket,
			PublicURL: cfg.BackblazeB2.PublicURL,
		}

		b2Driver, err := drivers.NewBackblazeB2Driver(b2Config)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Backblaze B2 driver: %w", err)
		}
		manager.drivers["backblaze_b2"] = b2Driver
	}

	// Validate default driver exists
	if _, exists := manager.drivers[manager.defaultDisk]; !exists {
		return nil, fmt.Errorf("default storage driver '%s' not configured", manager.defaultDisk)
	}

	return manager, nil
}

// Disk returns a storage driver by name (similar to Laravel's Storage::disk())
func (m *Manager) Disk(name string) Storage {
	if driver, exists := m.drivers[name]; exists {
		return driver
	}
	return m.drivers[m.defaultDisk] // Fallback to default
}

// Default returns the default storage driver
func (m *Manager) Default() Storage {
	return m.drivers[m.defaultDisk]
}

// Laravel-style facade methods that delegate to the default driver
func (m *Manager) Put(ctx context.Context, path string, content io.Reader) error {
	return m.Default().Put(ctx, path, content)
}

func (m *Manager) PutFile(ctx context.Context, path string, file *multipart.FileHeader) error {
	return m.Default().PutFile(ctx, path, file)
}

func (m *Manager) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	return m.Default().Get(ctx, path)
}

func (m *Manager) Delete(ctx context.Context, path string) error {
	return m.Default().Delete(ctx, path)
}

func (m *Manager) Exists(ctx context.Context, path string) (bool, error) {
	return m.Default().Exists(ctx, path)
}

func (m *Manager) Size(ctx context.Context, path string) (int64, error) {
	return m.Default().Size(ctx, path)
}

func (m *Manager) LastModified(ctx context.Context, path string) (time.Time, error) {
	return m.Default().LastModified(ctx, path)
}

func (m *Manager) MimeType(ctx context.Context, path string) (string, error) {
	return m.Default().MimeType(ctx, path)
}

func (m *Manager) URL(ctx context.Context, path string) (string, error) {
	return m.Default().URL(ctx, path)
}

func (m *Manager) TemporaryURL(ctx context.Context, path string, expiration time.Duration) (string, error) {
	return m.Default().TemporaryURL(ctx, path, expiration)
}

func (m *Manager) Copy(ctx context.Context, from, to string) error {
	return m.Default().Copy(ctx, from, to)
}

func (m *Manager) Move(ctx context.Context, from, to string) error {
	return m.Default().Move(ctx, from, to)
}

// Advanced methods for file uploads and management

// StoreUploadedFile stores an uploaded file with automatic path generation
func (m *Manager) StoreUploadedFile(ctx context.Context, file *multipart.FileHeader, directory string) (*UploadedFile, error) {
	// Generate unique path
	path := GenerateFilePath(directory, file.Filename)
	
	// Store the file
	if err := m.PutFile(ctx, path, file); err != nil {
		return nil, err
	}
	
	// Get file information
	size, _ := m.Size(ctx, path)
	mimeType, _ := m.MimeType(ctx, path)
	url, _ := m.URL(ctx, path)
	lastModified, _ := m.LastModified(ctx, path)
	
	uploadedFile := &UploadedFile{
		FileInfo: FileInfo{
			Path:         path,
			Name:         file.Filename,
			Size:         size,
			MimeType:     mimeType,
			Extension:    GetFileExtension(file.Filename),
			LastModified: lastModified,
			URL:          url,
			Driver:       m.Default().Driver(),
		},
		OriginalName: file.Filename,
		UploadedAt:   time.Now(),
	}
	
	return uploadedFile, nil
}

// StoreUploadedImage stores an uploaded image and creates variants
func (m *Manager) StoreUploadedImage(ctx context.Context, file *multipart.FileHeader, directory string) (*ImageUpload, error) {
	// First, store the original file
	uploadedFile, err := m.StoreUploadedFile(ctx, file, directory)
	if err != nil {
		return nil, err
	}
	
	// Check if it's actually an image
	isImageFile := IsImage(uploadedFile.MimeType)
	
	imageUpload := &ImageUpload{
		UploadedFile: *uploadedFile,
		IsImage:      isImageFile,
		Variants:     []ImageVariant{},
	}
	
	if isImageFile {
		// TODO: Add image processing to create variants (thumbnails, etc.)
		// This would require an image processing library like imaging or vips
		imageUpload.Variants = append(imageUpload.Variants, ImageVariant{
			Name: "original",
			Path: uploadedFile.Path,
			URL:  uploadedFile.URL,
			Size: uploadedFile.Size,
		})
	}
	
	return imageUpload, nil
}

// DeleteFile removes a file and all its variants (for images)
func (m *Manager) DeleteFile(ctx context.Context, uploadedFile *UploadedFile) error {
	// Delete main file
	if err := m.Delete(ctx, uploadedFile.Path); err != nil {
		return err
	}
	
	// If it's an image, delete variants
	if imageUpload, ok := interface{}(uploadedFile).(*ImageUpload); ok {
		for _, variant := range imageUpload.Variants {
			if variant.Path != uploadedFile.Path { // Don't delete original twice
				m.Delete(ctx, variant.Path) // Ignore errors for variants
			}
		}
	}
	
	return nil
}

// GetFileInfo returns detailed information about a file
func (m *Manager) GetFileInfo(ctx context.Context, path string) (*FileInfo, error) {
	exists, err := m.Exists(ctx, path)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	
	size, _ := m.Size(ctx, path)
	mimeType, _ := m.MimeType(ctx, path)
	lastModified, _ := m.LastModified(ctx, path)
	url, _ := m.URL(ctx, path)
	
	// Extract filename from path
	parts := strings.Split(path, "/")
	filename := parts[len(parts)-1]
	
	return &FileInfo{
		Path:         path,
		Name:         filename,
		Size:         size,
		MimeType:     mimeType,
		Extension:    GetFileExtension(filename),
		LastModified: lastModified,
		URL:          url,
		Driver:       m.Default().Driver(),
	}, nil
}

// ListFiles returns all files in a directory with their information
func (m *Manager) ListFiles(ctx context.Context, directory string) ([]*FileInfo, error) {
	files, err := m.Default().Files(ctx, directory)
	if err != nil {
		return nil, err
	}
	
	var fileInfos []*FileInfo
	for _, filePath := range files {
		if info, err := m.GetFileInfo(ctx, filePath); err == nil {
			fileInfos = append(fileInfos, info)
		}
	}
	
	return fileInfos, nil
}

// Cross-driver operations

// CopyBetweenDisks copies a file from one disk to another
func (m *Manager) CopyBetweenDisks(ctx context.Context, fromDisk, toDisk, fromPath, toPath string) error {
	sourceDriver := m.Disk(fromDisk)
	targetDriver := m.Disk(toDisk)
	
	// Get file from source
	reader, err := sourceDriver.Get(ctx, fromPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	
	// Put file to target
	return targetDriver.Put(ctx, toPath, reader)
}

// MoveBetweenDisks moves a file from one disk to another
func (m *Manager) MoveBetweenDisks(ctx context.Context, fromDisk, toDisk, fromPath, toPath string) error {
	// Copy file
	if err := m.CopyBetweenDisks(ctx, fromDisk, toDisk, fromPath, toPath); err != nil {
		return err
	}
	
	// Delete from source
	return m.Disk(fromDisk).Delete(ctx, fromPath)
}

// GetAvailableDrivers returns list of configured drivers
func (m *Manager) GetAvailableDrivers() []string {
	var drivers []string
	for name := range m.drivers {
		drivers = append(drivers, name)
	}
	return drivers
}

// GetDefaultDriver returns the name of the default driver
func (m *Manager) GetDefaultDriver() string {
	return m.defaultDisk
}