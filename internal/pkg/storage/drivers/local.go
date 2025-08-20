package drivers

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/VeRJiL/go-template/internal/pkg/storage"
)

// LocalDriver implements the Storage interface for local file system
type LocalDriver struct {
	rootPath  string
	baseURL   string
	urlPrefix string
}

// NewLocalDriver creates a new local storage driver
func NewLocalDriver(rootPath, baseURL, urlPrefix string) *LocalDriver {
	return &LocalDriver{
		rootPath:  rootPath,
		baseURL:   baseURL,
		urlPrefix: urlPrefix,
	}
}

// Put stores content at the given path
func (d *LocalDriver) Put(ctx context.Context, path string, content io.Reader) error {
	fullPath := d.getFullPath(path)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return storage.NewStorageError("put", path, err)
	}

	// Create or truncate the file
	file, err := os.Create(fullPath)
	if err != nil {
		return storage.NewStorageError("put", path, err)
	}
	defer file.Close()

	// Copy content to file
	_, err = io.Copy(file, content)
	if err != nil {
		return storage.NewStorageError("put", path, err)
	}

	return nil
}

// PutFile stores an uploaded file at the given path
func (d *LocalDriver) PutFile(ctx context.Context, path string, fileHeader *multipart.FileHeader) error {
	fullPath := d.getFullPath(path)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return storage.NewStorageError("putFile", path, err)
	}

	// Open the uploaded file
	src, err := fileHeader.Open()
	if err != nil {
		return storage.NewStorageError("putFile", path, err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return storage.NewStorageError("putFile", path, err)
	}
	defer dst.Close()

	// Copy file content
	_, err = io.Copy(dst, src)
	if err != nil {
		return storage.NewStorageError("putFile", path, err)
	}

	return nil
}

// Get retrieves content from the given path
func (d *LocalDriver) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := d.getFullPath(path)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.NewStorageError("get", path, fmt.Errorf("file not found"))
		}
		return nil, storage.NewStorageError("get", path, err)
	}

	return file, nil
}

// Delete removes the file at the given path
func (d *LocalDriver) Delete(ctx context.Context, path string) error {
	fullPath := d.getFullPath(path)

	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return storage.NewStorageError("delete", path, err)
	}

	return nil
}

// Exists checks if file exists at the given path
func (d *LocalDriver) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := d.getFullPath(path)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, storage.NewStorageError("exists", path, err)
	}

	return true, nil
}

// Size returns the size of the file at the given path
func (d *LocalDriver) Size(ctx context.Context, path string) (int64, error) {
	fullPath := d.getFullPath(path)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, storage.NewStorageError("size", path, fmt.Errorf("file not found"))
		}
		return 0, storage.NewStorageError("size", path, err)
	}

	return info.Size(), nil
}

// LastModified returns the last modification time of the file
func (d *LocalDriver) LastModified(ctx context.Context, path string) (time.Time, error) {
	fullPath := d.getFullPath(path)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, storage.NewStorageError("lastModified", path, fmt.Errorf("file not found"))
		}
		return time.Time{}, storage.NewStorageError("lastModified", path, err)
	}

	return info.ModTime(), nil
}

// MimeType returns the MIME type of the file
func (d *LocalDriver) MimeType(ctx context.Context, path string) (string, error) {
	// Detect MIME type by extension
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		// Fallback to reading file header
		fullPath := d.getFullPath(path)
		file, err := os.Open(fullPath)
		if err != nil {
			return "", storage.NewStorageError("mimeType", path, err)
		}
		defer file.Close()

		// Read first 512 bytes to detect content type
		buffer := make([]byte, 512)
		_, err = file.Read(buffer)
		if err != nil && err != io.EOF {
			return "", storage.NewStorageError("mimeType", path, err)
		}

		mimeType = http.DetectContentType(buffer)
	}

	return mimeType, nil
}

// Files returns all files in the given directory
func (d *LocalDriver) Files(ctx context.Context, directory string) ([]string, error) {
	fullPath := d.getFullPath(directory)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, storage.NewStorageError("files", directory, err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(directory, entry.Name()))
		}
	}

	return files, nil
}

// AllFiles returns all files in the directory recursively
func (d *LocalDriver) AllFiles(ctx context.Context, directory string) ([]string, error) {
	fullPath := d.getFullPath(directory)
	var files []string

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Convert absolute path to relative path
			relPath, err := filepath.Rel(d.rootPath, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(relPath))
		}

		return nil
	})

	if err != nil {
		return nil, storage.NewStorageError("allFiles", directory, err)
	}

	return files, nil
}

// Directories returns all directories in the given path
func (d *LocalDriver) Directories(ctx context.Context, directory string) ([]string, error) {
	fullPath := d.getFullPath(directory)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, storage.NewStorageError("directories", directory, err)
	}

	var directories []string
	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, filepath.Join(directory, entry.Name()))
		}
	}

	return directories, nil
}

// MakeDirectory creates a directory at the given path
func (d *LocalDriver) MakeDirectory(ctx context.Context, path string) error {
	fullPath := d.getFullPath(path)

	err := os.MkdirAll(fullPath, 0755)
	if err != nil {
		return storage.NewStorageError("makeDirectory", path, err)
	}

	return nil
}

// DeleteDirectory removes the directory at the given path
func (d *LocalDriver) DeleteDirectory(ctx context.Context, directory string) error {
	fullPath := d.getFullPath(directory)

	err := os.RemoveAll(fullPath)
	if err != nil {
		return storage.NewStorageError("deleteDirectory", directory, err)
	}

	return nil
}

// URL returns the public URL for the given path
func (d *LocalDriver) URL(ctx context.Context, path string) (string, error) {
	// Clean the path and ensure it doesn't start with /
	cleanPath := strings.TrimPrefix(filepath.ToSlash(path), "/")

	if d.baseURL != "" {
		return fmt.Sprintf("%s/%s/%s", d.baseURL, strings.Trim(d.urlPrefix, "/"), cleanPath), nil
	}

	return fmt.Sprintf("/%s/%s", strings.Trim(d.urlPrefix, "/"), cleanPath), nil
}

// TemporaryURL returns a temporary URL (not supported for local storage)
func (d *LocalDriver) TemporaryURL(ctx context.Context, path string, expiration time.Duration) (string, error) {
	// Local storage doesn't support temporary URLs, return regular URL
	return d.URL(ctx, path)
}

// Copy copies a file from source to destination
func (d *LocalDriver) Copy(ctx context.Context, from, to string) error {
	srcPath := d.getFullPath(from)
	dstPath := d.getFullPath(to)

	// Create destination directory
	dir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return storage.NewStorageError("copy", from, err)
	}

	// Open source file
	src, err := os.Open(srcPath)
	if err != nil {
		return storage.NewStorageError("copy", from, err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(dstPath)
	if err != nil {
		return storage.NewStorageError("copy", from, err)
	}
	defer dst.Close()

	// Copy content
	_, err = io.Copy(dst, src)
	if err != nil {
		return storage.NewStorageError("copy", from, err)
	}

	return nil
}

// Move moves a file from source to destination
func (d *LocalDriver) Move(ctx context.Context, from, to string) error {
	srcPath := d.getFullPath(from)
	dstPath := d.getFullPath(to)

	// Create destination directory
	dir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return storage.NewStorageError("move", from, err)
	}

	// Move file
	err := os.Rename(srcPath, dstPath)
	if err != nil {
		return storage.NewStorageError("move", from, err)
	}

	return nil
}

// Driver returns the driver name
func (d *LocalDriver) Driver() string {
	return "local"
}

// Private helper methods
func (d *LocalDriver) getFullPath(path string) string {
	return filepath.Join(d.rootPath, path)
}
