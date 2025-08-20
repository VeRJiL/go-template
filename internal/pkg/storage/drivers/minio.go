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

// MinIODriver implements the Storage interface for MinIO (self-hosted S3-compatible storage)
// MinIO is completely free and can be run locally or on your own servers
type MinIODriver struct {
	client     *s3.S3
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	bucket     string
	endpoint   string
	publicURL  string
}

// MinIOConfig holds the configuration for MinIO driver
type MinIOConfig struct {
	Endpoint  string // MinIO server endpoint (e.g., "localhost:9000")
	AccessKey string // MinIO access key
	SecretKey string // MinIO secret key
	Bucket    string // Bucket name
	UseSSL    bool   // Whether to use HTTPS
	PublicURL string // Public URL for file access (optional)
}

// NewMinIODriver creates a new MinIO storage driver
func NewMinIODriver(config MinIOConfig) (*MinIODriver, error) {
	// Create AWS session configured for MinIO
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String("us-east-1"), // MinIO doesn't care about region
		Credentials:      credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
		Endpoint:         aws.String(config.Endpoint),
		DisableSSL:       aws.Bool(!config.UseSSL),
		S3ForcePathStyle: aws.Bool(true), // MinIO requires path-style URLs
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO session: %w", err)
	}

	// Create S3 client
	client := s3.New(sess)

	// Generate public URL
	publicURL := config.PublicURL
	if publicURL == "" {
		protocol := "http"
		if config.UseSSL {
			protocol = "https"
		}
		publicURL = fmt.Sprintf("%s://%s", protocol, config.Endpoint)
	}

	driver := &MinIODriver{
		client:     client,
		uploader:   s3manager.NewUploader(sess),
		downloader: s3manager.NewDownloader(sess),
		bucket:     config.Bucket,
		endpoint:   config.Endpoint,
		publicURL:  publicURL,
	}

	// Ensure bucket exists
	if err := driver.ensureBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return driver, nil
}

// ensureBucket creates the bucket if it doesn't exist
func (d *MinIODriver) ensureBucket(ctx context.Context) error {
	// Check if bucket exists
	_, err := d.client.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(d.bucket),
	})

	if err != nil {
		// If bucket doesn't exist, create it
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
			_, err = d.client.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
				Bucket: aws.String(d.bucket),
			})
			if err != nil {
				return fmt.Errorf("failed to create bucket: %w", err)
			}

			// Set bucket policy to allow public read access
			policy := fmt.Sprintf(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Principal": "*",
						"Action": "s3:GetObject",
						"Resource": "arn:aws:s3:::%s/*"
					}
				]
			}`, d.bucket)

			_, err = d.client.PutBucketPolicyWithContext(ctx, &s3.PutBucketPolicyInput{
				Bucket: aws.String(d.bucket),
				Policy: aws.String(policy),
			})
			// Ignore policy errors as some MinIO setups might not support policies
		} else {
			return fmt.Errorf("failed to check bucket: %w", err)
		}
	}

	return nil
}

// Put stores content at the given path
func (d *MinIODriver) Put(ctx context.Context, path string, content io.Reader) error {
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
	}

	_, err := d.uploader.UploadWithContext(ctx, input)
	if err != nil {
		return storage.NewStorageError("put", path, err)
	}

	return nil
}

// PutFile stores an uploaded file at the given path
func (d *MinIODriver) PutFile(ctx context.Context, path string, fileHeader *multipart.FileHeader) error {
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
	}

	_, err = d.uploader.UploadWithContext(ctx, input)
	if err != nil {
		return storage.NewStorageError("putFile", path, err)
	}

	return nil
}

// Get retrieves content from the given path
func (d *MinIODriver) Get(ctx context.Context, path string) (io.ReadCloser, error) {
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
func (d *MinIODriver) Delete(ctx context.Context, path string) error {
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
func (d *MinIODriver) Exists(ctx context.Context, path string) (bool, error) {
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
func (d *MinIODriver) Size(ctx context.Context, path string) (int64, error) {
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
func (d *MinIODriver) LastModified(ctx context.Context, path string) (time.Time, error) {
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
func (d *MinIODriver) MimeType(ctx context.Context, path string) (string, error) {
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
func (d *MinIODriver) Files(ctx context.Context, directory string) ([]string, error) {
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
func (d *MinIODriver) AllFiles(ctx context.Context, directory string) ([]string, error) {
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
func (d *MinIODriver) Directories(ctx context.Context, directory string) ([]string, error) {
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
func (d *MinIODriver) MakeDirectory(ctx context.Context, path string) error {
	return nil // MinIO creates directories implicitly
}

// DeleteDirectory removes the directory at the given path
func (d *MinIODriver) DeleteDirectory(ctx context.Context, directory string) error {
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
func (d *MinIODriver) URL(ctx context.Context, path string) (string, error) {
	cleanPath := strings.TrimPrefix(path, "/")
	return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(d.publicURL, "/"), d.bucket, cleanPath), nil
}

// TemporaryURL returns a temporary signed URL
func (d *MinIODriver) TemporaryURL(ctx context.Context, path string, expiration time.Duration) (string, error) {
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
func (d *MinIODriver) Copy(ctx context.Context, from, to string) error {
	source := fmt.Sprintf("%s/%s", d.bucket, from)

	input := &s3.CopyObjectInput{
		Bucket:     aws.String(d.bucket),
		CopySource: aws.String(source),
		Key:        aws.String(to),
	}

	_, err := d.client.CopyObjectWithContext(ctx, input)
	if err != nil {
		return storage.NewStorageError("copy", from, err)
	}

	return nil
}

// Move moves a file from source to destination
func (d *MinIODriver) Move(ctx context.Context, from, to string) error {
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
func (d *MinIODriver) Driver() string {
	return "minio"
}
