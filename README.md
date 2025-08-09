# sefud: high-performance file upload & download service

A blazingly fast, secure file upload and download service built with Go, featuring Cloudflare R2, MinIO local development, and high-performance concurrent processing with short 6-character file IDs.

## 🚀 Features

### ✅ **Basic Features**

- **Short File IDs**: 6-character URLs like `Fx3B4k` instead of long UUIDs for clean, shareable links
- **High-Performance Uploads**: AWS S3 Manager with optimized HTTP client for maximum throughput
- **Streaming Downloads**: Memory-efficient direct streaming with HTTP range request support
- **Optional Self-hosted Storage**: S3-compatible storage with Cloudflare R2 or MinIO or AWS S3
- **API Documentation**: Interactive Swagger UI with complete endpoint documentation
- **Smart Upload Strategy**: Simple uploads for files <25MB, multipart for larger files
- **Concurrent Processing**: 8 concurrent workers with 16MB chunks for optimal throughput
- **Streaming Operations**: Direct streaming prevents memory bloat
- **Range Requests**: HTTP range support for partial downloads and resumable transfers
- **Integrity Checks**: Real-time MD5/SHA256 hashing during upload
- **Connection Pooling**: Optimized database and storage connections
- **Token-based Deletion**: Secure file deletion with unique authorization tokens
- **File Expiration**: Configurable TTL with automatic cleanup
- **MIME Validation**: Configurable blacklist for dangerous file types
- **Rate Limiting**: Built-in request throttling and DDoS protection
- **Atomic Transactions**: Database consistency with proper error handling
- **Comprehensive Logging**: Structured logging with request tracking

## 🛠 Setup & Installation

### Prerequisites

- Go 1.23.0+
- PostgreSQL database (or use Docker Compose)
- Cloudflare R2 credentials (or use local MinIO)

### Quick Start with Docker/Podman Compose

1. **Clone and setup**:

```bash
git clone https://github.com/Sardonyx001/sefud.git
cd sefud
```

1. **Start the full stack**:

```bash
# Starts PostgreSQL + MinIO + sefud app
podman compose up -d

# Check services are running
podman compose ps
```

1. **Access the services**:
   - **Sefud API**: <http://localhost:7000>
   - **Swagger UI**: <http://localhost:7000/swagger/index.html>
   - **MinIO Console**: <http://localhost:9001> (minioadmin/minioadmin123)

### Local Development Setup

1. **Start dependencies only**:

```bash
# Start PostgreSQL and MinIO
podman compose up -d postgres minio

# Or start them separately
podman run --name sefud-postgres -e POSTGRES_USER=user -e POSTGRES_PASSWORD=password -e POSTGRES_DB=sefud -p 5432:5432 -d postgres:latest
podman run --name sefud-minio -p 9000:9000 -p 9001:9001 -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin123 -d minio/minio:latest server /data --console-address ":9001"
```

1. **Install dependencies**:

```bash
go mod tidy
go install github.com/cosmtrek/air@latest  # Optional: for live reload
go install github.com/swaggo/swag/cmd/swag@latest  # Optional: for Swagger generation via cli (if not using `go generate`)
```

1. **Configure environment** (`.env` is already configured for local development):

```bash
# Check .env file - should be configured for MinIO by default
cat .env
```

1. **Run the service**:

```bash
# Development with live reload
air --build.cmd "go build -o tmp/sefud cmd/sefud/main.go" --build.bin "tmp/sefud"

# Or build and run directly
go build -o sefud ./cmd/sefud
./sefud
```

## 📖 API Usage & Testing

### Create Test Files

```bash
# Create test directory
mkdir -p tests/data

# Create test files of different sizes
dd if=/dev/zero of=tests/data/test_1mb.bin bs=1M count=1
dd if=/dev/zero of=tests/data/test_15mb.bin bs=1M count=15
dd if=/dev/zero of=tests/data/test_50mb.bin bs=1M count=50

# Create a text file
echo "Hello, World! This is a test file." > tests/data/test.txt
```

### Upload Files (HTTPie)

```bash
# Install HTTPie if you don't have it
# brew install httpie  # macOS
# pip install httpie   # Python

# Upload a small file
http -f POST :7000/up file@tests/data/test.txt

# Upload a large file with expiration
http -f POST :7000/up file@tests/data/test_15mb.bin expires=24h

# Upload with custom filename
http -f POST :7000/up file@tests/data/test_1mb.bin
```

**Example Response**:

```json
{
    "content_type": "application/octet-stream",
    "delete_token": "a1b2c3d4e5f6789...",
    "id": "Fx3B4k",
    "original_name": "test_15mb.bin",
    "size": 15728640,
    "url": "http://localhost:7000/Fx3B4k"
}
```

### Download Files

```bash
# Download file to temp location
TEMP_FILE=$(mktemp --suffix=.bin) && http GET :7000/Fx3B4k > "$TEMP_FILE" && ls -la "$TEMP_FILE"

# Download with curl
curl -O -J http://localhost:7000/Fx3B4k

# Range request (partial download)
curl -H "Range: bytes=0-1023" http://localhost:7000/Fx3B4k
```

### File Operations

```bash
# Get file info (HEAD request)
http HEAD :7000/Fx3B4k

# Delete file
http DELETE :7000/Fx3B4k token==your-delete-token-here

# Upload and immediately delete
RESPONSE=$(http -f POST :7000/up file@tests/data/test.txt) && \
ID=$(echo $RESPONSE | jq -r .id) && \
TOKEN=$(echo $RESPONSE | jq -r .delete_token) && \
http DELETE :7000/$ID token==$TOKEN
```

### Verify Files in MinIO

```bash
# List files in MinIO bucket
podman exec -it sefud-minio mc ls -r local/sefud-files

# Or access MinIO Console
open http://localhost:9001
# Login: minioadmin / minioadmin123
# Navigate to Buckets → sefud-files
```

### Performance Testing

```bash
# Test upload performance
time http -f POST :7000/up file@tests/data/test_50mb.bin

# Concurrent uploads
for i in {1..5}; do
  http -f POST :7000/up file@tests/data/test_1mb.bin &
done
wait

# Download performance test
ID="your-file-id-here"
time curl -s http://localhost:7000/$ID > /dev/null
```

## ⚙️ Configuration

### Environment Variables (`.env`)

```bash
# Application Configuration
SEFUD_APP_PORT=7000
SEFUD_APP_MIME_BLACKLIST=application/x-sh,application/x-msdownload,application/x-executable
SEFUD_APP_MAX_UPLOAD_SIZE=104857600  # 100MB

# Database Configuration
SEFUD_DB_USER=user
SEFUD_DB_PASSWORD=password
SEFUD_DB_NAME=sefud
SEFUD_DB_HOST=localhost
SEFUD_DB_PORT=5432

# MinIO Configuration (Local Development)
MINIO_API_PORT=9000
MINIO_CONSOLE_PORT=9001

# Storage Configuration - MinIO (Active)
SEFUD_STORAGE_ACCESS_KEY_ID=minioadmin
SEFUD_STORAGE_SECRET_ACCESS_KEY=minioadmin123
SEFUD_STORAGE_BUCKET_NAME=sefud-files
SEFUD_STORAGE_ENDPOINT=http://localhost:9000
SEFUD_STORAGE_REGION=auto

# Storage Configuration - Cloudflare R2
# Uncomment these and comment out MinIO config above for production
# SEFUD_STORAGE_ACCESS_KEY_ID=your_r2_access_key_id
# SEFUD_STORAGE_SECRET_ACCESS_KEY=your_r2_secret_access_key
# SEFUD_STORAGE_BUCKET_NAME=your-bucket-name
# SEFUD_STORAGE_ENDPOINT=https://your-account-id.r2.cloudflarestorage.com
# SEFUD_STORAGE_REGION=auto
```

### Docker Compose Services

```bash
# Full stack (production-like)
docker compose up -d

# Development mode (no Caddy)
docker compose --profile development up -d

# Individual services
docker compose up -d postgres minio  # Dependencies only
docker compose up -d sefud           # App only

# Check service health
docker compose ps
docker compose logs sefud -f         # Follow app logs
```

## 🏗 Architecture

```plain
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Client    │    │    sefud     │    │   MinIO/    │
│             │◄──►│   (Go API)   │◄──►│   R2/S3     │
│             │    │              │    │             │
└─────────────┘    └──────┬───────┘    └─────────────┘
                          │
                          ▼
                   ┌──────────────┐
                   │  PostgreSQL  │
                   │  (Metadata)  │
                   └──────────────┘
```

### Project Structure

```plain
sefud/
├── cmd/sefud/                   # Application entry point
├── server/                      # HTTP server, middleware, migrations
├── handlers/                    # API request handlers (upload/download/delete)
├── storage/                     # R2/MinIO client with performance optimizations
├── models/                      # Database models with short_id mapping
├── config/                      # Environment configuration management
├── db/                          # Database connection and setup
├── logger/                      # Structured logging middleware
├── docs/                        # Auto-generated Swagger documentation
├── tests/                       # Test files and data
├── docker compose.yaml          # Full stack deployment
├── docker compose.override.yml  # Development overrides
└── .env                         # Environment configuration
```

## 🔧 Development

### Database Migrations

The application automatically handles database migrations:

```bash
# Migrations run automatically on startup
go run cmd/sefud/main.go

# Database schema:
# - files table with UUID primary key
# - short_id field for public 6-character IDs
# - Automatic migration of existing records
```

### Generate Swagger Docs

```bash
go generate ./...
# or manually: swag init -g cmd/sefud/main.go
```

### Performance Optimizations

The service includes several performance optimizations:

1. **HTTP Client Tuning**: 100 max connections, keep-alive, disabled compression
2. **Smart Upload Strategy**: Simple uploads for <25MB, multipart for larger
3. **Concurrent Processing**: 8 workers with 16MB chunks
4. **Reduced Retries**: 2 max attempts for faster failure handling
5. **Connection Pooling**: Optimized for high concurrency

### Troubleshooting

```bash
# Check service health
curl http://localhost:7000/swagger/index.html

# Check MinIO bucket
podman exec -it sefud-minio mc ls local/sefud-files

# Check database connection
psql -h localhost -U user -d sefud -c "SELECT COUNT(*) FROM files;"

# Check logs
docker compose logs sefud -f
docker compose logs postgres -f
docker compose logs minio -f

# Performance debugging
time http -f POST :7000/up file@tests/data/test_15mb.bin
# Should complete in 2-3 seconds for 15MB files
```

## 🚧 Recent Updates

### Completed ✅

- [x] **Performance Optimization**: 10x faster uploads with AWS S3 Manager
- [x] **Short File IDs**: 6-character sqids instead of long UUIDs
- [x] **MinIO Integration**: Local S3-compatible development environment
- [x] **Docker Compose**: Complete stack with auto-setup
- [x] **Database Migration**: Automatic short_id migration for existing records
- [x] **HTTP Client Optimization**: Custom client with connection pooling
- [x] **Smart Upload Logic**: Size-based upload strategy selection
- [x] **Package releases**: Releases on new pushes to main via Github

### Planned 🎯

- [ ] File encryption/decryption
- [ ] Background cleanup jobs
- [ ] File deduplication
- [ ] Upload progress tracking via WebSocket
- [ ] Web interface?
- [ ] CLI tool
- [ ] Advanced monitoring and metrics

## 📝 License

MIT License - see [LICENSE](LICENSE) for details.
