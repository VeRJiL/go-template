package drivers

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
)

func TestNewLocalDriver(t *testing.T) {
	t.Run("should create local driver with correct configuration", func(t *testing.T) {
		rootPath := "/tmp/storage"
		baseURL := "http://example.com"
		urlPrefix := "/files"

		driver := NewLocalDriver(rootPath, baseURL, urlPrefix)

		assert.Equal(t, rootPath, driver.rootPath)
		assert.Equal(t, baseURL, driver.baseURL)
		assert.Equal(t, urlPrefix, driver.urlPrefix)
		assert.Equal(t, "local", driver.Driver())
	})
}

func TestLocalDriverPut(t *testing.T) {
	tempDir := t.TempDir()
	driver := NewLocalDriver(tempDir, "http://localhost", "/storage")
	ctx := context.Background()

	t.Run("should store file successfully", func(t *testing.T) {
		content := "test file content"
		path := "test/file.txt"

		err := driver.Put(ctx, path, strings.NewReader(content))

		assert.NoError(t, err)

		// Verify file exists
		fullPath := filepath.Join(tempDir, path)
		assert.FileExists(t, fullPath)

		// Verify content
		data, err := os.ReadFile(fullPath)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("should create directories if they don't exist", func(t *testing.T) {
		content := "nested content"
		path := "deeply/nested/directory/file.txt"

		err := driver.Put(ctx, path, strings.NewReader(content))

		assert.NoError(t, err)

		// Verify directory structure was created
		fullPath := filepath.Join(tempDir, path)
		assert.FileExists(t, fullPath)

		// Verify content
		data, err := os.ReadFile(fullPath)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("should overwrite existing file", func(t *testing.T) {
		path := "test/overwrite.txt"
		originalContent := "original content"
		newContent := "new content"

		// Put original file
		err := driver.Put(ctx, path, strings.NewReader(originalContent))
		require.NoError(t, err)

		// Overwrite with new content
		err = driver.Put(ctx, path, strings.NewReader(newContent))
		assert.NoError(t, err)

		// Verify new content
		fullPath := filepath.Join(tempDir, path)
		data, err := os.ReadFile(fullPath)
		assert.NoError(t, err)
		assert.Equal(t, newContent, string(data))
	})

	t.Run("should handle empty content", func(t *testing.T) {
		path := "test/empty.txt"

		err := driver.Put(ctx, path, strings.NewReader(""))

		assert.NoError(t, err)

		fullPath := filepath.Join(tempDir, path)
		assert.FileExists(t, fullPath)

		data, err := os.ReadFile(fullPath)
		assert.NoError(t, err)
		assert.Empty(t, data)
	})
}

func TestLocalDriverPutFile(t *testing.T) {
	tempDir := t.TempDir()
	driver := NewLocalDriver(tempDir, "http://localhost", "/storage")
	ctx := context.Background()

	t.Run("should store uploaded file successfully", func(t *testing.T) {
		// Create a mock multipart file
		content := "uploaded file content"
		fileName := "upload.txt"

		// Create multipart file header
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", fileName)
		require.NoError(t, err)

		_, err = part.Write([]byte(content))
		require.NoError(t, err)
		writer.Close()

		// Parse the multipart form
		reader := multipart.NewReader(body, writer.Boundary())
		form, err := reader.ReadForm(10 << 20) // 10MB max
		require.NoError(t, err)
		defer form.RemoveAll()

		fileHeaders := form.File["file"]
		require.Len(t, fileHeaders, 1)

		path := "uploads/test-upload.txt"
		err = driver.PutFile(ctx, path, fileHeaders[0])

		assert.NoError(t, err)

		// Verify file exists
		fullPath := filepath.Join(tempDir, path)
		assert.FileExists(t, fullPath)

		// Verify content
		data, err := os.ReadFile(fullPath)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})
}

func TestLocalDriverGet(t *testing.T) {
	tempDir := t.TempDir()
	driver := NewLocalDriver(tempDir, "http://localhost", "/storage")
	ctx := context.Background()

	t.Run("should retrieve file successfully", func(t *testing.T) {
		content := "file to retrieve"
		path := "test/get.txt"

		// Put file first
		err := driver.Put(ctx, path, strings.NewReader(content))
		require.NoError(t, err)

		// Get file
		reader, err := driver.Get(ctx, path)
		require.NoError(t, err)
		defer reader.Close()

		// Verify content
		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("should return error for non-existent file", func(t *testing.T) {
		path := "nonexistent/file.txt"

		reader, err := driver.Get(ctx, path)

		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), "file not found")
	})
}

func TestLocalDriverDelete(t *testing.T) {
	tempDir := t.TempDir()
	driver := NewLocalDriver(tempDir, "http://localhost", "/storage")
	ctx := context.Background()

	t.Run("should delete file successfully", func(t *testing.T) {
		content := "file to delete"
		path := "test/delete.txt"

		// Put file first
		err := driver.Put(ctx, path, strings.NewReader(content))
		require.NoError(t, err)

		// Verify file exists
		fullPath := filepath.Join(tempDir, path)
		assert.FileExists(t, fullPath)

		// Delete file
		err = driver.Delete(ctx, path)
		assert.NoError(t, err)

		// Verify file is deleted
		assert.NoFileExists(t, fullPath)
	})

	t.Run("should return error for non-existent file", func(t *testing.T) {
		path := "nonexistent/file.txt"

		err := driver.Delete(ctx, path)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})
}

func TestLocalDriverExists(t *testing.T) {
	tempDir := t.TempDir()
	driver := NewLocalDriver(tempDir, "http://localhost", "/storage")
	ctx := context.Background()

	t.Run("should return true for existing file", func(t *testing.T) {
		content := "existing file"
		path := "test/exists.txt"

		// Put file first
		err := driver.Put(ctx, path, strings.NewReader(content))
		require.NoError(t, err)

		exists, err := driver.Exists(ctx, path)

		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should return false for non-existent file", func(t *testing.T) {
		path := "nonexistent/file.txt"

		exists, err := driver.Exists(ctx, path)

		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestLocalDriverSize(t *testing.T) {
	tempDir := t.TempDir()
	driver := NewLocalDriver(tempDir, "http://localhost", "/storage")
	ctx := context.Background()

	t.Run("should return correct file size", func(t *testing.T) {
		content := "file content for size test"
		path := "test/size.txt"

		// Put file first
		err := driver.Put(ctx, path, strings.NewReader(content))
		require.NoError(t, err)

		size, err := driver.Size(ctx, path)

		assert.NoError(t, err)
		assert.Equal(t, int64(len(content)), size)
	})

	t.Run("should return zero for empty file", func(t *testing.T) {
		path := "test/empty-size.txt"

		// Put empty file
		err := driver.Put(ctx, path, strings.NewReader(""))
		require.NoError(t, err)

		size, err := driver.Size(ctx, path)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), size)
	})

	t.Run("should return error for non-existent file", func(t *testing.T) {
		path := "nonexistent/file.txt"

		size, err := driver.Size(ctx, path)

		assert.Error(t, err)
		assert.Equal(t, int64(0), size)
		assert.Contains(t, err.Error(), "file not found")
	})
}

func TestLocalDriverLastModified(t *testing.T) {
	tempDir := t.TempDir()
	driver := NewLocalDriver(tempDir, "http://localhost", "/storage")
	ctx := context.Background()

	t.Run("should return correct last modified time", func(t *testing.T) {
		content := "file for last modified test"
		path := "test/lastmod.txt"

		beforeTime := time.Now()

		// Put file
		err := driver.Put(ctx, path, strings.NewReader(content))
		require.NoError(t, err)

		afterTime := time.Now()

		lastMod, err := driver.LastModified(ctx, path)

		assert.NoError(t, err)
		assert.True(t, lastMod.After(beforeTime.Add(-time.Second))) // Allow 1 second tolerance
		assert.True(t, lastMod.Before(afterTime.Add(time.Second)))  // Allow 1 second tolerance
	})

	t.Run("should return error for non-existent file", func(t *testing.T) {
		path := "nonexistent/file.txt"

		lastMod, err := driver.LastModified(ctx, path)

		assert.Error(t, err)
		assert.True(t, lastMod.IsZero())
		assert.Contains(t, err.Error(), "file not found")
	})
}

func TestLocalDriverMimeType(t *testing.T) {
	tempDir := t.TempDir()
	driver := NewLocalDriver(tempDir, "http://localhost", "/storage")
	ctx := context.Background()

	tests := []struct {
		name         string
		filename     string
		expectedMime string
	}{
		{"JPEG image", "image.jpg", "image/jpeg"},
		{"PNG image", "photo.png", "image/png"},
		{"Text file", "document.txt", "text/plain"},
		{"PDF file", "document.pdf", "application/pdf"},
		{"JSON file", "data.json", "application/json"},
		{"HTML file", "page.html", "text/html"},
		{"CSS file", "style.css", "text/css"},
		{"JavaScript file", "script.js", "text/javascript"},
		{"Unknown extension", "file.unknown", "application/octet-stream"},
		{"No extension", "filename", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "test content"
			path := "test/" + tt.filename

			// Put file first
			err := driver.Put(ctx, path, strings.NewReader(content))
			require.NoError(t, err)

			mimeType, err := driver.MimeType(ctx, path)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedMime, mimeType)
		})
	}

	t.Run("should return error for non-existent file", func(t *testing.T) {
		path := "nonexistent/file.txt"

		mimeType, err := driver.MimeType(ctx, path)

		assert.Error(t, err)
		assert.Empty(t, mimeType)
		assert.Contains(t, err.Error(), "file not found")
	})
}

func TestLocalDriverDirectoryOperations(t *testing.T) {
	tempDir := t.TempDir()
	driver := NewLocalDriver(tempDir, "http://localhost", "/storage")
	ctx := context.Background()

	t.Run("should list files in directory", func(t *testing.T) {
		// Create multiple files
		files := []string{
			"dir/file1.txt",
			"dir/file2.txt",
			"dir/subdir/file3.txt",
			"other/file4.txt",
		}

		for _, file := range files {
			err := driver.Put(ctx, file, strings.NewReader("content"))
			require.NoError(t, err)
		}

		// List files in 'dir'
		fileList, err := driver.Files(ctx, "dir")

		assert.NoError(t, err)
		assert.Len(t, fileList, 3) // Should include files in subdirectories
		assert.Contains(t, fileList, "dir/file1.txt")
		assert.Contains(t, fileList, "dir/file2.txt")
		assert.Contains(t, fileList, "dir/subdir/file3.txt")
		assert.NotContains(t, fileList, "other/file4.txt")
	})

	t.Run("should list all files recursively", func(t *testing.T) {
		// Clear and recreate files
		os.RemoveAll(tempDir)
		tempDir = t.TempDir()
		driver = NewLocalDriver(tempDir, "http://localhost", "/storage")

		files := []string{
			"recursive/file1.txt",
			"recursive/sub1/file2.txt",
			"recursive/sub2/file3.txt",
			"recursive/sub1/nested/file4.txt",
		}

		for _, file := range files {
			err := driver.Put(ctx, file, strings.NewReader("content"))
			require.NoError(t, err)
		}

		allFiles, err := driver.AllFiles(ctx, "recursive")

		assert.NoError(t, err)
		assert.Len(t, allFiles, 4)
		for _, file := range files {
			assert.Contains(t, allFiles, file)
		}
	})

	t.Run("should list directories", func(t *testing.T) {
		// Clear and recreate structure
		os.RemoveAll(tempDir)
		tempDir = t.TempDir()
		driver = NewLocalDriver(tempDir, "http://localhost", "/storage")

		files := []string{
			"root/dir1/file1.txt",
			"root/dir2/file2.txt",
			"root/dir3/subdir/file3.txt",
		}

		for _, file := range files {
			err := driver.Put(ctx, file, strings.NewReader("content"))
			require.NoError(t, err)
		}

		dirs, err := driver.Directories(ctx, "root")

		assert.NoError(t, err)
		assert.Contains(t, dirs, "root/dir1")
		assert.Contains(t, dirs, "root/dir2")
		assert.Contains(t, dirs, "root/dir3")
	})

	t.Run("should create directory", func(t *testing.T) {
		path := "created/directory/structure"

		err := driver.MakeDirectory(ctx, path)

		assert.NoError(t, err)

		// Verify directory exists
		fullPath := filepath.Join(tempDir, path)
		info, err := os.Stat(fullPath)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("should delete directory and its contents", func(t *testing.T) {
		// Create directory with files
		files := []string{
			"delete-dir/file1.txt",
			"delete-dir/subdir/file2.txt",
		}

		for _, file := range files {
			err := driver.Put(ctx, file, strings.NewReader("content"))
			require.NoError(t, err)
		}

		// Verify directory exists
		dirPath := filepath.Join(tempDir, "delete-dir")
		_, err := os.Stat(dirPath)
		require.NoError(t, err)

		// Delete directory
		err = driver.DeleteDirectory(ctx, "delete-dir")
		assert.NoError(t, err)

		// Verify directory is deleted
		_, err = os.Stat(dirPath)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestLocalDriverURL(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("should generate URL with base URL", func(t *testing.T) {
		driver := NewLocalDriver(tempDir, "http://example.com", "/storage")
		ctx := context.Background()
		path := "test/file.txt"

		// Put file first
		err := driver.Put(ctx, path, strings.NewReader("content"))
		require.NoError(t, err)

		url, err := driver.URL(ctx, path)

		assert.NoError(t, err)
		assert.Equal(t, "http://example.com/storage/test/file.txt", url)
	})

	t.Run("should generate URL without base URL", func(t *testing.T) {
		driver := NewLocalDriver(tempDir, "", "/storage")
		ctx := context.Background()
		path := "test/file.txt"

		// Put file first
		err := driver.Put(ctx, path, strings.NewReader("content"))
		require.NoError(t, err)

		url, err := driver.URL(ctx, path)

		assert.NoError(t, err)
		assert.Equal(t, "/storage/test/file.txt", url)
	})

	t.Run("should return error for non-existent file", func(t *testing.T) {
		driver := NewLocalDriver(tempDir, "http://example.com", "/storage")
		ctx := context.Background()
		path := "nonexistent/file.txt"

		url, err := driver.URL(ctx, path)

		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "file not found")
	})
}

func TestLocalDriverTemporaryURL(t *testing.T) {
	tempDir := t.TempDir()
	driver := NewLocalDriver(tempDir, "http://example.com", "/storage")
	ctx := context.Background()

	t.Run("should generate temporary URL", func(t *testing.T) {
		path := "test/temp-file.txt"

		// Put file first
		err := driver.Put(ctx, path, strings.NewReader("content"))
		require.NoError(t, err)

		expiration := time.Hour
		url, err := driver.TemporaryURL(ctx, path, expiration)

		assert.NoError(t, err)
		assert.Contains(t, url, "http://example.com/storage/test/temp-file.txt")
		assert.Contains(t, url, "expires=")

		// Note: LocalDriver doesn't implement true temporary URLs with validation,
		// it just appends an expiration parameter for compatibility
	})

	t.Run("should return error for non-existent file", func(t *testing.T) {
		path := "nonexistent/file.txt"
		expiration := time.Hour

		url, err := driver.TemporaryURL(ctx, path, expiration)

		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "file not found")
	})
}

func TestLocalDriverCopyAndMove(t *testing.T) {
	tempDir := t.TempDir()
	driver := NewLocalDriver(tempDir, "http://localhost", "/storage")
	ctx := context.Background()

	t.Run("should copy file successfully", func(t *testing.T) {
		content := "file to copy"
		fromPath := "test/source.txt"
		toPath := "test/copy.txt"

		// Put source file
		err := driver.Put(ctx, fromPath, strings.NewReader(content))
		require.NoError(t, err)

		// Copy file
		err = driver.Copy(ctx, fromPath, toPath)
		assert.NoError(t, err)

		// Verify both files exist with same content
		sourceExists, err := driver.Exists(ctx, fromPath)
		assert.NoError(t, err)
		assert.True(t, sourceExists)

		copyExists, err := driver.Exists(ctx, toPath)
		assert.NoError(t, err)
		assert.True(t, copyExists)

		// Verify content
		reader, err := driver.Get(ctx, toPath)
		require.NoError(t, err)
		defer reader.Close()

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("should move file successfully", func(t *testing.T) {
		content := "file to move"
		fromPath := "test/move-source.txt"
		toPath := "test/move-dest.txt"

		// Put source file
		err := driver.Put(ctx, fromPath, strings.NewReader(content))
		require.NoError(t, err)

		// Move file
		err = driver.Move(ctx, fromPath, toPath)
		assert.NoError(t, err)

		// Verify source doesn't exist
		sourceExists, err := driver.Exists(ctx, fromPath)
		assert.NoError(t, err)
		assert.False(t, sourceExists)

		// Verify destination exists with correct content
		destExists, err := driver.Exists(ctx, toPath)
		assert.NoError(t, err)
		assert.True(t, destExists)

		reader, err := driver.Get(ctx, toPath)
		require.NoError(t, err)
		defer reader.Close()

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("should return error when copying non-existent file", func(t *testing.T) {
		fromPath := "nonexistent/source.txt"
		toPath := "test/dest.txt"

		err := driver.Copy(ctx, fromPath, toPath)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("should return error when moving non-existent file", func(t *testing.T) {
		fromPath := "nonexistent/source.txt"
		toPath := "test/dest.txt"

		err := driver.Move(ctx, fromPath, toPath)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})
}