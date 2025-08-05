# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**sefud** is a simple, encrypted file upload & download service written in Go, inspired by 0x0.st and waifuvault.moe. The project is in early development - handlers are currently stub implementations that need actual file handling logic.

## Development Commands

### Building and Running

```bash
# Development with live reload (recommended)
air --build.cmd "go build -o tmp/sefud cmd/sefud/main.go" --build.bin "tmp/sefud"

# Standard build
go build -o sefud ./cmd/sefud

# Docker development
docker-compose up
```

### Go Module Management

```bash
go mod download    # Install dependencies
go mod tidy       # Clean up dependencies
```

### Swagger Documentation

```bash
go generate ./...                    # Regenerate Swagger docs automatically
swag init -g cmd/sefud/main.go     # Manual regeneration
```

## Architecture

### Tech Stack

- **Web Framework**: Echo v4 with high-performance middleware
- **Storage**: Cloudflare R2 (S3-compatible) with AWS SDK v2
- **Database**: PostgreSQL with GORM ORM for metadata
- **Logging**: Charmbracelet Log with structured logging middleware
- **Config**: Environment variables via godotenv
- **Performance**: Concurrent chunked uploads, streaming downloads, range requests

### Key Components

- `cmd/sefud/main.go` - Application entry point with dependency injection
- `server/server.go` - HTTP server with middleware stack and routing
- `handlers/` - High-performance file operation handlers with R2 integration
- `storage/r2.go` - Cloudflare R2 client with concurrent multipart uploads
- `models/file.go` - File metadata model with comprehensive tracking
- `config/` - Environment-based configuration with R2 settings
- `db/connection.go` - PostgreSQL connection via GORM
- `logger/logger.go` - Request logging middleware

### API Routes

- `POST /up` - High-performance file upload with chunking and concurrent processing
- `GET /:id` - Streaming file download with range request support
- `DELETE /:id` - Secure file deletion with token-based authorization
- `HEAD /:id` - File metadata retrieval without download
- `GET /swagger/*` - Swagger UI documentation

### Performance Features

- **Concurrent Uploads**: Configurable chunk size and worker pools for large files
- **Streaming Downloads**: Memory-efficient streaming with range request support
- **Async Cleanup**: Non-blocking R2 deletion with retry capabilities
- **Rate Limiting**: Built-in request rate limiting and security middleware
- **Integrity Checks**: MD5 and SHA256 hashing during upload
- **Connection Pooling**: Optimized database and R2 connection management

## Configuration

### Required Environment Variables

```bash
# Application
SEFUD_APP_PORT=7000
SEFUD_STORAGE_PATH=./uploads
SEFUD_MIME_BLACKLIST=application/x-sh,application/x-msdownload,application/x-executable
SEFUD_MAX_UPLOAD_SIZE=104857600  # 100MB default

# Database
SEFUD_DB_USER=user
SEFUD_DB_PASSWORD=password
SEFUD_DB_NAME=sefud
SEFUD_DB_HOST=localhost
SEFUD_DB_PORT=5432

# Cloudflare R2 (Required for file storage)
SEFUD_R2_ACCESS_KEY_ID=your_r2_access_key_id
SEFUD_R2_SECRET_ACCESS_KEY=your_r2_secret_access_key
SEFUD_R2_BUCKET_NAME=sefud-files
SEFUD_R2_ENDPOINT=https://your-account-id.r2.cloudflarestorage.com
SEFUD_R2_REGION=auto
```

### Performance Tuning

The system is designed for high-performance file operations:

#### Upload Performance

- **Chunk Size**: 8MB chunks for optimal throughput (configurable)
- **Concurrency**: 6 concurrent uploads for large files (configurable)
- **Threshold**: Multipart uploads triggered for files >10MB
- **Hashing**: Concurrent MD5/SHA256 calculation during upload
- **Memory**: Streaming processing to minimize memory usage

#### Download Performance

- **Streaming**: Direct streaming from R2 to client (no buffering)
- **Range Requests**: Full HTTP range request support for partial downloads
- **Caching**: Aggressive caching headers with ETag support
- **Timeouts**: Optimized timeouts for different operations

#### Database Optimization

- **Indexes**: Optimized indexes on frequently queried fields
- **Transactions**: Atomic operations with proper rollback handling
- **Connection Pooling**: GORM connection pooling for concurrent requests

## Current State & Next Steps

The project has solid foundation but needs core implementation:

### ✅ Completed

- High-performance HTTP server with Echo v4 and middleware stack
- Cloudflare R2 integration with AWS SDK v2
- Advanced file upload with concurrent chunked processing
- Streaming file downloads with range request support
- Secure file deletion with token-based authorization
- Comprehensive file metadata tracking and database models
- Request logging middleware with handler identification
- Docker containerization with PostgreSQL
- Swagger UI with interactive API documentation
- Auto-regenerating docs via go generate
- Performance optimizations (connection pooling, async operations)
- Security features (rate limiting, CORS, input validation)

### ❌ Future Enhancements

- File encryption/decryption functionality
- Background job system for expired file cleanup
- File deduplication based on content hashes
- Upload progress tracking via WebSocket
- File compression/optimization
- Advanced monitoring and metrics
- CDN integration for faster downloads
- Multi-region R2 support

### Performance Notes

- **Upload Throughput**: Optimized for 100MB+ files with 6-worker concurrent processing
- **Memory Efficiency**: Streaming operations prevent memory bloat
- **Database Performance**: Indexed queries with connection pooling
- **R2 Integration**: Direct streaming with minimal latency
- **Error Handling**: Comprehensive error codes and cleanup procedures
- **Security**: Token-based deletion, MIME type validation, rate limiting
