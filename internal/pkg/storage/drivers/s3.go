package drivers

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/VeRJiL/go-template/internal/pkg/storage"
)

// S3Driver implements the Storage interface for Amazon S3
type S3Driver struct {
	client     *s3.S3
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	bucket     string
	region     string
	baseURL    string
	publicURL  string
}

// S3Config holds the configuration for S3 driver
type S3Config struct {
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	UseSSL         bool
	ForcePathStyle bool
	Endpoint       string // For S3-compatible services like MinIO
	PublicURL      string // Custom public URL for files
}

// NewS3Driver creates a new S3 storage driver
func NewS3Driver(config S3Config) (*S3Driver, error) {
	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(config.Region),
		Credentials:      credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
		DisableSSL:       aws.Bool(!config.UseSSL),
		S3ForcePathStyle: aws.Bool(config.ForcePathStyle),
		Endpoint:         aws.String(config.Endpoint),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Create S3 client
	client := s3.New(sess)

	// Generate base URL
	var baseURL string
	if config.PublicURL != "" {
		baseURL = config.PublicURL
	} else if config.Endpoint != "" {
		protocol := "http"
		if config.UseSSL {
			protocol = "https"
		}
		baseURL = fmt.Sprintf("%s://%s", protocol, config.Endpoint)
	} else {
		baseURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", config.Bucket, config.Region)
	}

	return &S3Driver{
		client:     client,
		uploader:   s3manager.NewUploader(sess),
		downloader: s3manager.NewDownloader(sess),
		bucket:     config.Bucket,
		region:     config.Region,
		baseURL:    baseURL,
		publicURL:  config.PublicURL,
	}, nil
}

// Put stores content at the given path
func (d *S3Driver) Put(ctx context.Context, path string, content io.Reader) error {
	// Detect content type
	var contentType string
	if seeker, ok := content.(io.ReadSeeker); ok {
		buffer := make([]byte, 512)
		n, _ := seeker.Read(buffer)
		contentType = http.DetectContentType(buffer[:n])
		seeker.Seek(0, io.SeekStart) // Reset to beginning
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	input := &s3manager.UploadInput{
		Bucket:      aws.String(d.bucket),
		Key:         aws.String(path),
		Body:        content,
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"), // Make files publicly accessible
	}

	_, err := d.uploader.UploadWithContext(ctx, input)
	if err != nil {
		return storage.NewStorageError("put", path, err)
	}

	return nil
}

// PutFile stores an uploaded file at the given path
func (d *S3Driver) PutFile(ctx context.Context, path string, fileHeader *multipart.FileHeader) error {
	// Open the uploaded file
	src, err := fileHeader.Open()
	if err != nil {
		return storage.NewStorageError("putFile", path, err)
	}
	defer src.Close()

	// Detect content type from file header
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		// Fallback to detecting from content
		buffer := make([]byte, 512)
		n, _ := src.Read(buffer)
		contentType = http.DetectContentType(buffer[:n])

		// Reset to beginning of file
		src.Close()
		src, _ = fileHeader.Open()
	}

	input := &s3manager.UploadInput{
		Bucket:      aws.String(d.bucket),
		Key:         aws.String(path),
		Body:        src,
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"),
	}

	_, err = d.uploader.UploadWithContext(ctx, input)
	if err != nil {
		return storage.NewStorageError("putFile", path, err)
	}

	return nil
}

// Get retrieves content from the given path
func (d *S3Driver) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
	}

	result, err := d.client.GetObjectWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			return nil, storage.NewStorageError("get", path, fmt.Errorf("file not found"))
		}
		return nil, storage.NewStorageError("get", path, err)
	}

	return result.Body, nil
}

// Delete removes the file at the given path
func (d *S3Driver) Delete(ctx context.Context, path string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
	}

	_, err := d.client.DeleteObjectWithContext(ctx, input)
	if err != nil {
		return storage.NewStorageError("delete", path, err)
	}

	return nil
}

// Exists checks if file exists at the given path
func (d *S3Driver) Exists(ctx context.Context, path string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
	}

	_, err := d.client.HeadObjectWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
			return false, nil
		}
		return false, storage.NewStorageError("exists", path, err)
	}

	return true, nil
}

// Size returns the size of the file at the given path
func (d *S3Driver) Size(ctx context.Context, path string) (int64, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
	}

	result, err := d.client.HeadObjectWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
			return 0, storage.NewStorageError("size", path, fmt.Errorf("file not found"))
		}
		return 0, storage.NewStorageError("size", path, err)
	}

	if result.ContentLength != nil {
		return *result.ContentLength, nil
	}

	return 0, nil
}

// LastModified returns the last modification time of the file
func (d *S3Driver) LastModified(ctx context.Context, path string) (time.Time, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
	}

	result, err := d.client.HeadObjectWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
			return time.Time{}, storage.NewStorageError("lastModified", path, fmt.Errorf("file not found"))
		}
		return time.Time{}, storage.NewStorageError("lastModified", path, err)
	}

	if result.LastModified != nil {
		return *result.LastModified, nil
	}

	return time.Time{}, nil
}

// MimeType returns the MIME type of the file
func (d *S3Driver) MimeType(ctx context.Context, path string) (string, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
	}

	result, err := d.client.HeadObjectWithContext(ctx, input)
	if err != nil {
		return "", storage.NewStorageError("mimeType", path, err)
	}

	if result.ContentType != nil {
		return *result.ContentType, nil
	}

	return "application/octet-stream", nil
}

// Files returns all files in the given directory
func (d *S3Driver) Files(ctx context.Context, directory string) ([]string, error) {
	prefix := strings.TrimSuffix(directory, "/") + "/"
	if directory == "" || directory == "." {
		prefix = ""
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(d.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"), // Only get files in current directory
	}

	var files []string
	err := d.client.ListObjectsV2PagesWithContext(ctx, input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			if obj.Key != nil && *obj.Key != prefix { // Exclude directory itself
				files = append(files, *obj.Key)
			}
		}
		return !lastPage
	})

	if err != nil {
		return nil, storage.NewStorageError("files", directory, err)
	}

	return files, nil
}

// AllFiles returns all files in the directory recursively
func (d *S3Driver) AllFiles(ctx context.Context, directory string) ([]string, error) {
	prefix := strings.TrimSuffix(directory, "/") + "/"
	if directory == "" || directory == "." {
		prefix = ""
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(d.bucket),
		Prefix: aws.String(prefix),
	}

	var files []string
	err := d.client.ListObjectsV2PagesWithContext(ctx, input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			if obj.Key != nil && *obj.Key != prefix { // Exclude directory itself
				files = append(files, *obj.Key)
			}
		}
		return !lastPage
	})

	if err != nil {
		return nil, storage.NewStorageError("allFiles", directory, err)
	}

	return files, nil
}

// Directories returns all directories in the given path
func (d *S3Driver) Directories(ctx context.Context, directory string) ([]string, error) {
	prefix := strings.TrimSuffix(directory, "/") + "/"
	if directory == "" || directory == "." {
		prefix = ""
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(d.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	var directories []string
	err := d.client.ListObjectsV2PagesWithContext(ctx, input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, commonPrefix := range page.CommonPrefixes {
			if commonPrefix.Prefix != nil {
				// Remove trailing slash and convert to relative path
				dir := strings.TrimSuffix(*commonPrefix.Prefix, "/")
				directories = append(directories, dir)
			}
		}
		return !lastPage
	})

	if err != nil {
		return nil, storage.NewStorageError("directories", directory, err)
	}

	return directories, nil
}

// MakeDirectory creates a directory at the given path (S3 doesn't have real directories)
func (d *S3Driver) MakeDirectory(ctx context.Context, path string) error {
	// S3 doesn't have real directories, they're created implicitly when files are uploaded
	return nil
}

// DeleteDirectory removes the directory at the given path
func (d *S3Driver) DeleteDirectory(ctx context.Context, directory string) error {
	// Get all files in directory
	files, err := d.AllFiles(ctx, directory)
	if err != nil {
		return err
	}

	// Delete all files
	for _, file := range files {
		if err := d.Delete(ctx, file); err != nil {
			return err
		}
	}

	return nil
}

// URL returns the public URL for the given path
func (d *S3Driver) URL(ctx context.Context, path string) (string, error) {
	cleanPath := strings.TrimPrefix(path, "/")
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(d.baseURL, "/"), cleanPath), nil
}

// TemporaryURL returns a temporary signed URL
func (d *S3Driver) TemporaryURL(ctx context.Context, path string, expiration time.Duration) (string, error) {
	req, _ := d.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(path),
	})

	url, err := req.Presign(expiration)
	if err != nil {
		return "", storage.NewStorageError("temporaryURL", path, err)
	}

	return url, nil
}

// Copy copies a file from source to destination
func (d *S3Driver) Copy(ctx context.Context, from, to string) error {
	source := fmt.Sprintf("%s/%s", d.bucket, from)

	input := &s3.CopyObjectInput{
		Bucket:     aws.String(d.bucket),
		CopySource: aws.String(source),
		Key:        aws.String(to),
		ACL:        aws.String("public-read"),
	}

	_, err := d.client.CopyObjectWithContext(ctx, input)
	if err != nil {
		return storage.NewStorageError("copy", from, err)
	}

	return nil
}

// Move moves a file from source to destination
func (d *S3Driver) Move(ctx context.Context, from, to string) error {
	// Copy file to new location
	if err := d.Copy(ctx, from, to); err != nil {
		return err
	}

	// Delete original file
	if err := d.Delete(ctx, from); err != nil {
		// If delete fails, try to clean up the copied file
		d.Delete(ctx, to)
		return err
	}

	return nil
}

// Driver returns the driver name
func (d *S3Driver) Driver() string {
	return "s3"
}
