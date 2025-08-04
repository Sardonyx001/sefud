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

## Architecture

### Tech Stack
- **Web Framework**: Echo v4
- **Database**: PostgreSQL with GORM ORM
- **Logging**: Charmbracelet Log with structured logging middleware
- **Config**: Environment variables via godotenv

### Key Components
- `cmd/sefud/main.go` - Application entry point
- `server/server.go` - HTTP server setup and routing (Echo)
- `handlers/` - HTTP request handlers (currently stubs)
- `config/` - Environment-based configuration management
- `db/connection.go` - PostgreSQL connection via GORM
- `logger/logger.go` - Request logging middleware

### API Routes
- `POST /up` - File upload
- `GET /:id` - File download by ID  
- `DELETE /:id` - File deletion by ID

## Configuration

### Required Environment Variables
```bash
# Application
SEFUD_APP_PORT=7000
SEFUD_STORAGE_PATH=./uploads
SEFUD_MIME_BLACKLIST=application/x-sh
SEFUD_MAX_UPLOAD_SIZE=10485760

# Database
SEFUD_DB_USER=user
SEFUD_DB_PASSWORD=password
SEFUD_DB_NAME=sefud
SEFUD_DB_HOST=postgres
SEFUD_DB_PORT=5432
```

## Current State & Next Steps

The project has solid foundation but needs core implementation:

### ✅ Completed
- Basic HTTP server with Echo
- Configuration management 
- Database connection setup
- Request logging middleware
- Docker containerization
- Swagger API documentation annotations

### ❌ Needs Implementation  
- Actual file upload/download logic in handlers
- Database models and migrations
- File encryption functionality
- Automated file deletion
- Unit and integration tests
- Error handling and validation

### Development Notes
- All handlers currently return stub responses - they need actual file handling implementation
- Database connection is established but no models/tables are defined
- The project uses GORM for ORM but no database operations are implemented yet
- Configuration supports file storage path and MIME blacklisting but handlers don't use these yet