// Package storage provides high-performance storage operations using Cloudflare R2.
package storage

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	sefudConfig "github.com/Sardonyx001/sefud/config"
)

// R2Client provides high-performance R2 storage operations
type R2Client struct {
	client     *s3.Client
	uploader   *manager.Uploader
	bucketName string
	mu         sync.RWMutex
}


// UploadOptions configures upload behavior for performance optimization
type UploadOptions struct {
	ContentType     string
	ChunkSize       int64
	MaxConcurrency  int
	EnableMultipart bool
	Metadata        map[string]string
}

// NewR2Client creates a new high-performance R2 client
func NewR2Client(cfg *sefudConfig.Config) (*R2Client, error) {
	// Create optimized HTTP client for R2
	httpClient := &http.Client{
		Timeout: 10 * time.Minute,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
			IdleConnTimeout:     90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableCompression:    true, // Disable compression for better upload performance
		},
	}

	// Create AWS config for Cloudflare R2 with optimizations
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.R2.AccessKeyID,
			cfg.R2.SecretAccessKey,
			"",
		)),
		config.WithRegion(cfg.R2.Region),
		config.WithHTTPClient(httpClient),
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 2) // Reduce retries for speed
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with custom endpoint for R2
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.R2.Endpoint)
		o.UsePathStyle = true // Required for R2
	})

	// Create uploader with disabled checksums for R2 performance
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = 16 * 1024 * 1024 // 16MB parts
		u.Concurrency = 8             // 8 concurrent uploads
		u.LeavePartsOnError = false   // Clean up failed uploads
	})

	return &R2Client{
		client:     client,
		uploader:   uploader,
		bucketName: cfg.R2.BucketName,
	}, nil
}

// Upload performs high-performance file upload with automatic chunking
func (r *R2Client) Upload(ctx context.Context, key string, reader io.Reader, opts UploadOptions) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// For small files, use simple upload
	if !opts.EnableMultipart {
		return r.simpleUpload(ctx, key, reader, opts)
	}

	// Use S3 manager for multipart uploads (optimized for R2)
	return r.managerUpload(ctx, key, reader, opts)
}

// simpleUpload performs a direct upload for smaller files
func (r *R2Client) simpleUpload(ctx context.Context, key string, reader io.Reader, opts UploadOptions) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(opts.ContentType),
	}

	// Add metadata if provided
	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}

	_, err := r.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// managerUpload uses AWS S3 manager for optimized R2 uploads
func (r *R2Client) managerUpload(ctx context.Context, key string, reader io.Reader, opts UploadOptions) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(opts.ContentType),
	}

	// Add metadata if provided
	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}

	_, err := r.uploader.Upload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload file with manager: %w", err)
	}

	return nil
}

// Download streams a file from R2 with optimized performance
func (r *R2Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	input := &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	}

	resp, err := r.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	return resp.Body, nil
}

// Delete removes a file from R2
func (r *R2Client) Delete(ctx context.Context, key string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	}

	_, err := r.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetObjectInfo retrieves file metadata from R2
func (r *R2Client) GetObjectInfo(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	input := &s3.HeadObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	}

	resp, err := r.client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return resp, nil
}

// DefaultUploadOptions provides sensible defaults for high-performance uploads
func DefaultUploadOptions() UploadOptions {
	return UploadOptions{
		ContentType:     "application/octet-stream",
		ChunkSize:       16 * 1024 * 1024, // 16MB chunks for better throughput
		MaxConcurrency:  8,                // 8 concurrent uploads for maximum performance
		EnableMultipart: true,
		Metadata:        make(map[string]string),
	}
}
