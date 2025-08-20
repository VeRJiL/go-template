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

// CloudflareR2Driver implements the Storage interface for Cloudflare R2
// Cloudflare R2 offers 10GB of free storage with no egress fees
// Compatible with S3 API but has some differences in endpoint structure
type CloudflareR2Driver struct {
	client     *s3.S3
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	bucket     string
	accountID  string
	publicURL  string
}

// CloudflareR2Config holds the configuration for Cloudflare R2 driver
type CloudflareR2Config struct {
	AccountID string // Cloudflare Account ID (required)
	AccessKey string // R2 Token ID
	SecretKey string // R2 Token Secret
	Bucket    string // Bucket name
	PublicURL string // Custom domain for R2 bucket (optional)
}

// NewCloudflareR2Driver creates a new Cloudflare R2 storage driver
func NewCloudflareR2Driver(config CloudflareR2Config) (*CloudflareR2Driver, error) {
	if config.AccountID == "" {
		return nil, fmt.Errorf("account ID is required for Cloudflare R2")
	}

	// Cloudflare R2 endpoint format
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", config.AccountID)

	// Create AWS session configured for Cloudflare R2
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String("auto"), // R2 uses "auto" region
		Credentials:      credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
		Endpoint:         aws.String(endpoint),
		S3ForcePathStyle: aws.Bool(false), // R2 uses virtual-hosted style
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare R2 session: %w", err)
	}

	// Create S3 client
	client := s3.New(sess)

	// Generate public URL
	publicURL := config.PublicURL
	if publicURL == "" {
		// Default R2 public URL format
		publicURL = fmt.Sprintf("https://pub-%s.r2.dev", config.Bucket)
	}

	driver := &CloudflareR2Driver{
		client:     client,
		uploader:   s3manager.NewUploader(sess),
		downloader: s3manager.NewDownloader(sess),
		bucket:     config.Bucket,
		accountID:  config.AccountID,
		publicURL:  publicURL,
	}

	return driver, nil
}

// Put stores content at the given path
func (d *CloudflareR2Driver) Put(ctx context.Context, path string, content io.Reader) error {
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
		return storage.NewStorageError("put", path, err)
	}

	return nil
}

// PutFile stores an uploaded file at the given path
func (d *CloudflareR2Driver) PutFile(ctx context.Context, path string, fileHeader *multipart.FileHeader) error {
	src, err := fileHeader.Open()
	if err != nil {
		return storage.NewStorageError("putFile", path, err)
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
		return storage.NewStorageError("putFile", path, err)
	}

	return nil
}

// Get retrieves content from the given path
func (d *CloudflareR2Driver) Get(ctx context.Context, path string) (io.ReadCloser, error) {
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
func (d *CloudflareR2Driver) Delete(ctx context.Context, path string) error {
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
func (d *CloudflareR2Driver) Exists(ctx context.Context, path string) (bool, error) {
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
func (d *CloudflareR2Driver) Size(ctx context.Context, path string) (int64, error) {
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
func (d *CloudflareR2Driver) LastModified(ctx context.Context, path string) (time.Time, error) {
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
func (d *CloudflareR2Driver) MimeType(ctx context.Context, path string) (string, error) {
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
func (d *CloudflareR2Driver) Files(ctx context.Context, directory string) ([]string, error) {
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
func (d *CloudflareR2Driver) AllFiles(ctx context.Context, directory string) ([]string, error) {
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
func (d *CloudflareR2Driver) Directories(ctx context.Context, directory string) ([]string, error) {
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
func (d *CloudflareR2Driver) MakeDirectory(ctx context.Context, path string) error {
	return nil // R2 creates directories implicitly
}

// DeleteDirectory removes the directory at the given path
func (d *CloudflareR2Driver) DeleteDirectory(ctx context.Context, directory string) error {
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
func (d *CloudflareR2Driver) URL(ctx context.Context, path string) (string, error) {
	cleanPath := strings.TrimPrefix(path, "/")
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(d.publicURL, "/"), cleanPath), nil
}

// TemporaryURL returns a temporary signed URL
func (d *CloudflareR2Driver) TemporaryURL(ctx context.Context, path string, expiration time.Duration) (string, error) {
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
func (d *CloudflareR2Driver) Copy(ctx context.Context, from, to string) error {
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
func (d *CloudflareR2Driver) Move(ctx context.Context, from, to string) error {
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
func (d *CloudflareR2Driver) Driver() string {
	return "cloudflare_r2"
}
