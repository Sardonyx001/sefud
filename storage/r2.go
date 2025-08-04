// Package storage provides high-performance storage operations using Cloudflare R2.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	sefudConfig "github.com/Sardonyx001/sefud/config"
)

// R2Client provides high-performance R2 storage operations
type R2Client struct {
	client     *s3.Client
	bucketName string
	mu         sync.RWMutex
}

// ChunkUploadResult represents the result of a chunk upload
type ChunkUploadResult struct {
	ETag     string
	PartNumber int32
	Error    error
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
	// Create AWS config for Cloudflare R2
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.R2.AccessKeyID,
			cfg.R2.SecretAccessKey,
			"",
		)),
		config.WithRegion(cfg.R2.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with custom endpoint for R2
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.R2.Endpoint)
		o.UsePathStyle = true // Required for R2
	})

	return &R2Client{
		client:     client,
		bucketName: cfg.R2.BucketName,
	}, nil
}

// Upload performs high-performance file upload with automatic chunking
func (r *R2Client) Upload(ctx context.Context, key string, reader io.Reader, opts UploadOptions) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// For small files or when multipart is disabled, use simple upload
	if !opts.EnableMultipart {
		return r.simpleUpload(ctx, key, reader, opts)
	}

	// Use multipart upload for large files
	return r.multipartUpload(ctx, key, reader, opts)
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

// multipartUpload performs chunked upload for large files with high concurrency
func (r *R2Client) multipartUpload(ctx context.Context, key string, reader io.Reader, opts UploadOptions) error {
	// Create multipart upload
	createInput := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		ContentType: aws.String(opts.ContentType),
	}

	if len(opts.Metadata) > 0 {
		createInput.Metadata = opts.Metadata
	}

	createResp, err := r.client.CreateMultipartUpload(ctx, createInput)
	if err != nil {
		return fmt.Errorf("failed to create multipart upload: %w", err)
	}

	uploadID := createResp.UploadId
	
	// Setup channels for concurrent chunk processing
	chunkChan := make(chan []byte, opts.MaxConcurrency)
	resultChan := make(chan ChunkUploadResult, opts.MaxConcurrency)
	
	// Start worker goroutines for concurrent uploads
	var wg sync.WaitGroup
	for i := 0; i < opts.MaxConcurrency; i++ {
		wg.Add(1)
		go r.uploadWorker(ctx, &wg, key, uploadID, chunkChan, resultChan)
	}

	// Read and send chunks
	go func() {
		defer close(chunkChan)
		buffer := make([]byte, opts.ChunkSize)
		
		for {
			n, err := reader.Read(buffer)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buffer[:n])
				chunkChan <- chunk
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				// Handle read error - for now just log and break
				break
			}
		}
	}()

	// Close result channel when all workers are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var completedParts []types.CompletedPart
	var uploadError error

	for result := range resultChan {
		if result.Error != nil {
			uploadError = result.Error
			break
		}
		
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       aws.String(result.ETag),
			PartNumber: aws.Int32(result.PartNumber),
		})
	}

	// Handle upload completion or abortion
	if uploadError != nil {
		// Abort multipart upload on error
		_, abortErr := r.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(r.bucketName),
			Key:      aws.String(key),
			UploadId: uploadID,
		})
		if abortErr != nil {
			return fmt.Errorf("upload failed and abort failed: %v, %v", uploadError, abortErr)
		}
		return uploadError
	}

	// Complete multipart upload
	_, err = r.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(r.bucketName),
		Key:      aws.String(key),
		UploadId: uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

// uploadWorker processes chunks concurrently
func (r *R2Client) uploadWorker(ctx context.Context, wg *sync.WaitGroup, key string, uploadID *string, chunkChan <-chan []byte, resultChan chan<- ChunkUploadResult) {
	defer wg.Done()
	
	partNumber := int32(1)
	
	for chunk := range chunkChan {
		result := ChunkUploadResult{PartNumber: partNumber}
		
		resp, err := r.client.UploadPart(ctx, &s3.UploadPartInput{
			Bucket:     aws.String(r.bucketName),
			Key:        aws.String(key),
			PartNumber: aws.Int32(partNumber),
			UploadId:   uploadID,
			Body:       bytes.NewReader(chunk),
		})
		
		if err != nil {
			result.Error = err
		} else {
			result.ETag = *resp.ETag
		}
		
		resultChan <- result
		partNumber++
	}
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
		ChunkSize:       5 * 1024 * 1024, // 5MB chunks
		MaxConcurrency:  4,                // 4 concurrent uploads
		EnableMultipart: true,
		Metadata:        make(map[string]string),
	}
}