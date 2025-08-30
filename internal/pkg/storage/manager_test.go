package storage

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/VeRJiL/go-template/internal/config"
)

// MockStorage implements the Storage interface for testing
type MockStorage struct {
	driver     string
	files      map[string][]byte
	lastAccess map[string]time.Time
}

func NewMockStorage(driver string) *MockStorage {
	return &MockStorage{
		driver:     driver,
		files:      make(map[string][]byte),
		lastAccess: make(map[string]time.Time),
	}
}

func (m *MockStorage) Put(ctx context.Context, path string, content io.Reader) error {
	data, err := io.ReadAll(content)
	if err != nil {
		return err
	}
	m.files[path] = data
	m.lastAccess[path] = time.Now()
	return nil
}

func (m *MockStorage) PutFile(ctx context.Context, path string, file *multipart.FileHeader) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	return m.Put(ctx, path, src)
}

func (m *MockStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	data, exists := m.files[path]
	if !exists {
		return nil, NewStorageError("get", path, os.ErrNotExist)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *MockStorage) Delete(ctx context.Context, path string) error {
	delete(m.files, path)
	delete(m.lastAccess, path)
	return nil
}

func (m *MockStorage) Exists(ctx context.Context, path string) (bool, error) {
	_, exists := m.files[path]
	return exists, nil
}

func (m *MockStorage) Size(ctx context.Context, path string) (int64, error) {
	data, exists := m.files[path]
	if !exists {
		return 0, NewStorageError("size", path, os.ErrNotExist)
	}
	return int64(len(data)), nil
}

func (m *MockStorage) LastModified(ctx context.Context, path string) (time.Time, error) {
	lastMod, exists := m.lastAccess[path]
	if !exists {
		return time.Time{}, NewStorageError("lastModified", path, os.ErrNotExist)
	}
	return lastMod, nil
}

func (m *MockStorage) MimeType(ctx context.Context, path string) (string, error) {
	if !m.fileExists(path) {
		return "", NewStorageError("mimeType", path, os.ErrNotExist)
	}
	if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") {
		return "image/jpeg", nil
	}
	if strings.HasSuffix(path, ".png") {
		return "image/png", nil
	}
	return "application/octet-stream", nil
}

func (m *MockStorage) Files(ctx context.Context, directory string) ([]string, error) {
	var files []string
	for path := range m.files {
		if strings.HasPrefix(path, directory) {
			files = append(files, path)
		}
	}
	return files, nil
}

func (m *MockStorage) AllFiles(ctx context.Context, directory string) ([]string, error) {
	return m.Files(ctx, directory)
}

func (m *MockStorage) Directories(ctx context.Context, directory string) ([]string, error) {
	dirs := make(map[string]bool)
	for path := range m.files {
		if strings.HasPrefix(path, directory) {
			dir := filepath.Dir(path)
			dirs[dir] = true
		}
	}

	var result []string
	for dir := range dirs {
		result = append(result, dir)
	}
	return result, nil
}

func (m *MockStorage) MakeDirectory(ctx context.Context, path string) error {
	// Mock doesn't need to explicitly create directories
	return nil
}

func (m *MockStorage) DeleteDirectory(ctx context.Context, directory string) error {
	for path := range m.files {
		if strings.HasPrefix(path, directory) {
			delete(m.files, path)
			delete(m.lastAccess, path)
		}
	}
	return nil
}

func (m *MockStorage) URL(ctx context.Context, path string) (string, error) {
	if !m.fileExists(path) {
		return "", NewStorageError("url", path, os.ErrNotExist)
	}
	return "http://example.com/" + path, nil
}

func (m *MockStorage) TemporaryURL(ctx context.Context, path string, expiration time.Duration) (string, error) {
	if !m.fileExists(path) {
		return "", NewStorageError("temporaryURL", path, os.ErrNotExist)
	}
	return "http://example.com/" + path + "?expires=" + time.Now().Add(expiration).Format("2006-01-02"), nil
}

func (m *MockStorage) Copy(ctx context.Context, from, to string) error {
	data, exists := m.files[from]
	if !exists {
		return NewStorageError("copy", from, os.ErrNotExist)
	}
	m.files[to] = data
	m.lastAccess[to] = time.Now()
	return nil
}

func (m *MockStorage) Move(ctx context.Context, from, to string) error {
	if err := m.Copy(ctx, from, to); err != nil {
		return err
	}
	return m.Delete(ctx, from)
}

func (m *MockStorage) Driver() string {
	return m.driver
}

func (m *MockStorage) fileExists(path string) bool {
	_, exists := m.files[path]
	return exists
}

func TestNewManager(t *testing.T) {
	t.Run("should create manager with local driver", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := &config.StorageConfig{
			Provider: "local",
			Local: config.LocalStorageConfig{
				Path:      tempDir,
				URLPrefix: "/storage",
			},
		}

		manager, err := NewManager(cfg)

		require.NoError(t, err)
		assert.NotNil(t, manager)
		assert.Equal(t, "local", manager.GetDefaultDriver())
		assert.Contains(t, manager.GetAvailableDrivers(), "local")
	})

	t.Run("should return error if default driver not configured", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Provider: "nonexistent",
		}

		manager, err := NewManager(cfg)

		assert.Error(t, err)
		assert.Nil(t, manager)
		assert.Contains(t, err.Error(), "default storage driver 'nonexistent' not configured")
	})

	t.Run("should skip S3 driver if not configured", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := &config.StorageConfig{
			Provider: "local",
			Local: config.LocalStorageConfig{
				Path: tempDir,
			},
			// S3 not configured (empty bucket/access key)
			S3: config.S3StorageConfig{},
		}

		manager, err := NewManager(cfg)

		require.NoError(t, err)
		drivers := manager.GetAvailableDrivers()
		assert.Contains(t, drivers, "local")
		assert.NotContains(t, drivers, "s3")
	})

	t.Run("should skip MinIO driver if not configured", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := &config.StorageConfig{
			Provider: "local",
			Local: config.LocalStorageConfig{
				Path: tempDir,
			},
			// MinIO not configured
			MinIO: config.MinIOStorageConfig{},
		}

		manager, err := NewManager(cfg)

		require.NoError(t, err)
		drivers := manager.GetAvailableDrivers()
		assert.Contains(t, drivers, "local")
		assert.NotContains(t, drivers, "minio")
	})
}

func TestManagerWithMockDrivers(t *testing.T) {
	// Create a manager with mock drivers for testing
	manager := &Manager{
		drivers: map[string]Storage{
			"local": NewMockStorage("local"),
			"s3":    NewMockStorage("s3"),
		},
		defaultDisk: "local",
	}

	t.Run("should return correct default driver", func(t *testing.T) {
		driver := manager.Default()
		assert.Equal(t, "local", driver.Driver())
	})

	t.Run("should return specific disk", func(t *testing.T) {
		s3Driver := manager.Disk("s3")
		assert.Equal(t, "s3", s3Driver.Driver())
	})

	t.Run("should return default disk for unknown disk name", func(t *testing.T) {
		unknownDriver := manager.Disk("unknown")
		assert.Equal(t, "local", unknownDriver.Driver())
	})

	t.Run("should delegate Put to default driver", func(t *testing.T) {
		ctx := context.Background()
		content := strings.NewReader("test content")

		err := manager.Put(ctx, "test/file.txt", content)

		assert.NoError(t, err)

		// Verify file was stored in default driver
		exists, err := manager.Default().Exists(ctx, "test/file.txt")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should delegate Get to default driver", func(t *testing.T) {
		ctx := context.Background()
		content := "test content"

		// Put file first
		err := manager.Put(ctx, "test/file.txt", strings.NewReader(content))
		require.NoError(t, err)

		// Get file
		reader, err := manager.Get(ctx, "test/file.txt")
		require.NoError(t, err)
		defer reader.Close()

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("should delegate Delete to default driver", func(t *testing.T) {
		ctx := context.Background()

		// Put file first
		err := manager.Put(ctx, "test/delete.txt", strings.NewReader("content"))
		require.NoError(t, err)

		// Verify exists
		exists, err := manager.Exists(ctx, "test/delete.txt")
		require.NoError(t, err)
		assert.True(t, exists)

		// Delete file
		err = manager.Delete(ctx, "test/delete.txt")
		assert.NoError(t, err)

		// Verify deleted
		exists, err = manager.Exists(ctx, "test/delete.txt")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("should delegate Size to default driver", func(t *testing.T) {
		ctx := context.Background()
		content := "test content"

		err := manager.Put(ctx, "test/size.txt", strings.NewReader(content))
		require.NoError(t, err)

		size, err := manager.Size(ctx, "test/size.txt")
		assert.NoError(t, err)
		assert.Equal(t, int64(len(content)), size)
	})

	t.Run("should delegate URL generation to default driver", func(t *testing.T) {
		ctx := context.Background()

		err := manager.Put(ctx, "test/url.txt", strings.NewReader("content"))
		require.NoError(t, err)

		url, err := manager.URL(ctx, "test/url.txt")
		assert.NoError(t, err)
		assert.Equal(t, "http://example.com/test/url.txt", url)
	})

	t.Run("should delegate TemporaryURL to default driver", func(t *testing.T) {
		ctx := context.Background()
		expiration := time.Hour

		err := manager.Put(ctx, "test/temp.txt", strings.NewReader("content"))
		require.NoError(t, err)

		url, err := manager.TemporaryURL(ctx, "test/temp.txt", expiration)
		assert.NoError(t, err)
		assert.Contains(t, url, "http://example.com/test/temp.txt")
		assert.Contains(t, url, "expires=")
	})

	t.Run("should delegate Copy to default driver", func(t *testing.T) {
		ctx := context.Background()
		content := "test content"

		// Put source file
		err := manager.Put(ctx, "test/source.txt", strings.NewReader(content))
		require.NoError(t, err)

		// Copy file
		err = manager.Copy(ctx, "test/source.txt", "test/copy.txt")
		assert.NoError(t, err)

		// Verify both files exist
		sourceExists, err := manager.Exists(ctx, "test/source.txt")
		assert.NoError(t, err)
		assert.True(t, sourceExists)

		copyExists, err := manager.Exists(ctx, "test/copy.txt")
		assert.NoError(t, err)
		assert.True(t, copyExists)

		// Verify content is the same
		reader, err := manager.Get(ctx, "test/copy.txt")
		require.NoError(t, err)
		defer reader.Close()

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("should delegate Move to default driver", func(t *testing.T) {
		ctx := context.Background()
		content := "test content"

		// Put source file
		err := manager.Put(ctx, "test/move-source.txt", strings.NewReader(content))
		require.NoError(t, err)

		// Move file
		err = manager.Move(ctx, "test/move-source.txt", "test/move-dest.txt")
		assert.NoError(t, err)

		// Verify source doesn't exist
		sourceExists, err := manager.Exists(ctx, "test/move-source.txt")
		assert.NoError(t, err)
		assert.False(t, sourceExists)

		// Verify destination exists with correct content
		destExists, err := manager.Exists(ctx, "test/move-dest.txt")
		assert.NoError(t, err)
		assert.True(t, destExists)

		reader, err := manager.Get(ctx, "test/move-dest.txt")
		require.NoError(t, err)
		defer reader.Close()

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})
}

func TestManagerCrossDiskOperations(t *testing.T) {
	manager := &Manager{
		drivers: map[string]Storage{
			"local": NewMockStorage("local"),
			"s3":    NewMockStorage("s3"),
		},
		defaultDisk: "local",
	}

	t.Run("should copy between different disks", func(t *testing.T) {
		ctx := context.Background()
		content := "cross-disk content"

		// Put file on local disk
		localDriver := manager.Disk("local")
		err := localDriver.Put(ctx, "test/source.txt", strings.NewReader(content))
		require.NoError(t, err)

		// Copy from local to S3
		err = manager.CopyBetweenDisks(ctx, "local", "s3", "test/source.txt", "test/s3-copy.txt")
		assert.NoError(t, err)

		// Verify file exists on both disks
		localExists, err := manager.Disk("local").Exists(ctx, "test/source.txt")
		assert.NoError(t, err)
		assert.True(t, localExists)

		s3Exists, err := manager.Disk("s3").Exists(ctx, "test/s3-copy.txt")
		assert.NoError(t, err)
		assert.True(t, s3Exists)

		// Verify content is correct on S3
		reader, err := manager.Disk("s3").Get(ctx, "test/s3-copy.txt")
		require.NoError(t, err)
		defer reader.Close()

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("should move between different disks", func(t *testing.T) {
		ctx := context.Background()
		content := "cross-disk move content"

		// Put file on local disk
		localDriver := manager.Disk("local")
		err := localDriver.Put(ctx, "test/move-source.txt", strings.NewReader(content))
		require.NoError(t, err)

		// Move from local to S3
		err = manager.MoveBetweenDisks(ctx, "local", "s3", "test/move-source.txt", "test/s3-moved.txt")
		assert.NoError(t, err)

		// Verify source is deleted from local
		localExists, err := manager.Disk("local").Exists(ctx, "test/move-source.txt")
		assert.NoError(t, err)
		assert.False(t, localExists)

		// Verify file exists on S3 with correct content
		s3Exists, err := manager.Disk("s3").Exists(ctx, "test/s3-moved.txt")
		assert.NoError(t, err)
		assert.True(t, s3Exists)

		reader, err := manager.Disk("s3").Get(ctx, "test/s3-moved.txt")
		require.NoError(t, err)
		defer reader.Close()

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("should return error when copying non-existent file", func(t *testing.T) {
		ctx := context.Background()

		err := manager.CopyBetweenDisks(ctx, "local", "s3", "nonexistent.txt", "dest.txt")
		assert.Error(t, err)
	})
}

func TestManagerFileOperations(t *testing.T) {
	manager := &Manager{
		drivers: map[string]Storage{
			"local": NewMockStorage("local"),
		},
		defaultDisk: "local",
	}

	t.Run("should get file info", func(t *testing.T) {
		ctx := context.Background()
		content := "file info content"

		// Put file
		err := manager.Put(ctx, "test/info.txt", strings.NewReader(content))
		require.NoError(t, err)

		// Get file info
		info, err := manager.GetFileInfo(ctx, "test/info.txt")
		require.NoError(t, err)

		assert.Equal(t, "test/info.txt", info.Path)
		assert.Equal(t, "info.txt", info.Name)
		assert.Equal(t, int64(len(content)), info.Size)
		assert.Equal(t, "application/octet-stream", info.MimeType)
		assert.Equal(t, "txt", info.Extension)
		assert.Equal(t, "local", info.Driver)
		assert.Contains(t, info.URL, "test/info.txt")
	})

	t.Run("should return error for non-existent file info", func(t *testing.T) {
		ctx := context.Background()

		info, err := manager.GetFileInfo(ctx, "nonexistent.txt")
		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("should list files in directory", func(t *testing.T) {
		ctx := context.Background()

		// Put multiple files
		files := []string{"dir/file1.txt", "dir/file2.jpg", "dir/subdir/file3.png"}
		for _, file := range files {
			err := manager.Put(ctx, file, strings.NewReader("content"))
			require.NoError(t, err)
		}

		// List files
		fileInfos, err := manager.ListFiles(ctx, "dir")
		assert.NoError(t, err)
		assert.Len(t, fileInfos, 3)

		// Verify file info
		for _, info := range fileInfos {
			assert.Contains(t, []string{"dir/file1.txt", "dir/file2.jpg", "dir/subdir/file3.png"}, info.Path)
			assert.Equal(t, "local", info.Driver)
		}
	})
}

func TestManagerAvailableDrivers(t *testing.T) {
	manager := &Manager{
		drivers: map[string]Storage{
			"local":        NewMockStorage("local"),
			"s3":           NewMockStorage("s3"),
			"cloudflare":   NewMockStorage("cloudflare"),
		},
		defaultDisk: "local",
	}

	t.Run("should return all available drivers", func(t *testing.T) {
		drivers := manager.GetAvailableDrivers()

		assert.Len(t, drivers, 3)
		assert.Contains(t, drivers, "local")
		assert.Contains(t, drivers, "s3")
		assert.Contains(t, drivers, "cloudflare")
	})

	t.Run("should return correct default driver", func(t *testing.T) {
		defaultDriver := manager.GetDefaultDriver()
		assert.Equal(t, "local", defaultDriver)
	})
}