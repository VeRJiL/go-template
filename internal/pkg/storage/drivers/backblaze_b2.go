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
)

// BackblazeB2Driver implements the Storage interface for Backblaze B2
// Backblaze B2 offers 10GB of free storage + 1GB free download per day
// Compatible with S3 API through their S3-compatible endpoint
type BackblazeB2Driver struct {
	client     *s3.S3
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	bucket     string
	region     string
	publicURL  string
}

// BackblazeB2Config holds the configuration for Backblaze B2 driver
type BackblazeB2Config struct {
	Region    string // B2 region (e.g., "us-west-000", "eu-central-003")
	KeyID     string // Application Key ID
	KeySecret string // Application Key
	Bucket    string // Bucket name
	PublicURL string // Custom domain for B2 bucket (optional)
}

// NewBackblazeB2Driver creates a new Backblaze B2 storage driver
func NewBackblazeB2Driver(config BackblazeB2Config) (*BackblazeB2Driver, error) {
	if config.Region == "" {
		return nil, fmt.Errorf("region is required for Backblaze B2")
	}

	// Backblaze B2 S3-compatible endpoint format
	endpoint := fmt.Sprintf("https://s3.%s.backblazeb2.com", config.Region)

	// Create AWS session configured for Backblaze B2
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(config.Region),
		Credentials:      credentials.NewStaticCredentials(config.KeyID, config.KeySecret, ""),
		Endpoint:         aws.String(endpoint),
		S3ForcePathStyle: aws.Bool(false), // B2 uses virtual-hosted style
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Backblaze B2 session: %w", err)
	}

	// Create S3 client
	client := s3.New(sess)

	// Generate public URL
	publicURL := config.PublicURL
	if publicURL == "" {
		// Default B2 public URL format (when bucket is set to public)
		publicURL = fmt.Sprintf("https://f000.backblazeb2.com/file/%s", config.Bucket)
	}

	driver := &BackblazeB2Driver{
		client:     client,
		uploader:   s3manager.NewUploader(sess),
		downloader: s3manager.NewDownloader(sess),
		bucket:     config.Bucket,
		region:     config.Region,
		publicURL:  publicURL,
	}

	return driver, nil
}

// Put stores content at the given path
func (d *BackblazeB2Driver) Put(ctx context.Context, path string, content io.Reader) error {
	var contentType string
	if seeker, ok := content.(io.ReadSeeker); ok {
		buffer := make([]byte, 512)
		n, _ := seeker.Read(buffer)
		contentType = http.DetectContentType(buffer[:n])
		seeker.Seek(0, io.SeekStart)
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
		return fmt.Errorf("failed to put file %s: %w", path, err)
	}

	return nil
}

// PutFile stores an uploaded file at the given path
func (d *BackblazeB2Driver) PutFile(ctx context.Context, path string, fileHeader *multipart.FileHeader) error {
	src, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("failed to putFile %s: %w", path, err)
	}
	defer src.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		buffer := make([]byte, 512)
		n, _ := src.Read(buffer)
		contentType = http.DetectContentType(buffer[:n])
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
		return fmt.Errorf("failed to putFile %s: %w", path, err)
	}

	return nil
}

// Get retrieves content from the given path
func (d *BackblazeB2Driver) Get(ctx context.Context, path string) (io.ReadCloser, error) {
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
func (d *BackblazeB2Driver) Delete(ctx context.Context, path string) error {
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
func (d *BackblazeB2Driver) Exists(ctx context.Context, path string) (bool, error) {
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
func (d *BackblazeB2Driver) Size(ctx context.Context, path string) (int64, error) {
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
func (d *BackblazeB2Driver) LastModified(ctx context.Context, path string) (time.Time, error) {
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
func (d *BackblazeB2Driver) MimeType(ctx context.Context, path string) (string, error) {
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
func (d *BackblazeB2Driver) Files(ctx context.Context, directory string) ([]string, error) {
	prefix := strings.TrimSuffix(directory, "/") + "/"
	if directory == "" || directory == "." {
		prefix = ""
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(d.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	var files []string
	err := d.client.ListObjectsV2PagesWithContext(ctx, input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			if obj.Key != nil && *obj.Key != prefix {
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
func (d *BackblazeB2Driver) AllFiles(ctx context.Context, directory string) ([]string, error) {
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
			if obj.Key != nil && *obj.Key != prefix {
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
func (d *BackblazeB2Driver) Directories(ctx context.Context, directory string) ([]string, error) {
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

// MakeDirectory creates a directory at the given path
func (d *BackblazeB2Driver) MakeDirectory(ctx context.Context, path string) error {
	return nil // B2 creates directories implicitly
}

// DeleteDirectory removes the directory at the given path
func (d *BackblazeB2Driver) DeleteDirectory(ctx context.Context, directory string) error {
	files, err := d.AllFiles(ctx, directory)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := d.Delete(ctx, file); err != nil {
			return err
		}
	}

	return nil
}

// URL returns the public URL for the given path
func (d *BackblazeB2Driver) URL(ctx context.Context, path string) (string, error) {
	cleanPath := strings.TrimPrefix(path, "/")
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(d.publicURL, "/"), cleanPath), nil
}

// TemporaryURL returns a temporary signed URL
func (d *BackblazeB2Driver) TemporaryURL(ctx context.Context, path string, expiration time.Duration) (string, error) {
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
func (d *BackblazeB2Driver) Copy(ctx context.Context, from, to string) error {
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
func (d *BackblazeB2Driver) Move(ctx context.Context, from, to string) error {
	if err := d.Copy(ctx, from, to); err != nil {
		return err
	}

	if err := d.Delete(ctx, from); err != nil {
		d.Delete(ctx, to) // Cleanup on failure
		return err
	}

	return nil
}

// Driver returns the driver name
func (d *BackblazeB2Driver) Driver() string {
	return "backblaze_b2"
}
