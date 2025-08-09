// Package server initializes and runs the HTTP server for file upload/download.
package server

import (
	"fmt"
	"math/rand/v2"

	"github.com/Sardonyx001/sefud/config"
	"github.com/Sardonyx001/sefud/db"
	"github.com/Sardonyx001/sefud/handlers"
	"github.com/Sardonyx001/sefud/logger"
	"github.com/Sardonyx001/sefud/models"
	"github.com/Sardonyx001/sefud/storage"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sqids/sqids-go"
	echoSwagger "github.com/swaggo/echo-swagger"
	"gorm.io/gorm"
)

type Server struct {
	echo          *echo.Echo
	db            *gorm.DB
	config        *config.Config
	storageClient *storage.StorageClient
	fileHandler   *handlers.FileHandler
}

func New(cfg *config.Config) (*Server, error) {
	e := echo.New()

	// Initialize database
	database := db.Init(cfg)

	// Auto-migrate database tables
	if err := database.AutoMigrate(&models.File{}); err != nil {
		return nil, err
	}

	// Migrate existing records to have short_id
	if err := migrateShortIDs(database); err != nil {
		return nil, fmt.Errorf("failed to migrate short IDs: %w", err)
	}

	// Initialize storage client
	storageClient, err := storage.NewStorageClient(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize file handler with dependencies
	fileHandler := handlers.NewFileHandler(database, storageClient, cfg)

	// Middleware
	e.Use(logger.InitLoggerMiddleware())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Security middleware
	e.Use(middleware.Secure())
	e.Use(middleware.RequestID())

	// Rate limiting for uploads (adjust as needed)
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20))) // 20 requests per second

	// API Routes with the file handler methods
	e.POST("/up", fileHandler.UploadFile)
	e.GET("/:id", fileHandler.DownloadFile)
	e.DELETE("/:id", fileHandler.DeleteFile)
	e.HEAD("/:id", fileHandler.GetFileInfo) // For checking file existence

	// Swagger documentation route
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	return &Server{
		echo:          e,
		db:            database,
		config:        cfg,
		storageClient: storageClient,
		fileHandler:   fileHandler,
	}, nil
}

func (s *Server) Start(addr string) error {
	return s.echo.Start(":" + addr)
}

// migrateShortIDs generates short_id for existing records that don't have one
func migrateShortIDs(db *gorm.DB) error {
	// Create sqids instance with same config as handlers
	s, _ := sqids.New(sqids.Options{
		MinLength: 6,
		Alphabet:  "FxnXM1kBN6cuhsAvjW3Co7l2RePyY8DwaU04Tzt9fHQrqSVKdpimLGIJOgb5ZE",
	})

	// Find records without short_id
	var files []models.File
	if err := db.Where("short_id = '' OR short_id IS NULL").Find(&files).Error; err != nil {
		return err
	}

	// Generate short_id for each record
	for _, file := range files {
		var shortID string
		for {
			// Generate short ID
			randomNum := uint64(rand.Uint64() % 56_800_000_000)
			shortID, _ = s.Encode([]uint64{randomNum})
			// Ensure exactly 6 chars
			for len(shortID) < 6 {
				shortID = "F" + shortID
			}
			shortID = shortID[:6]

			// Check uniqueness
			var existing models.File
			err := db.Where("short_id = ?", shortID).First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				break // Unique ID found
			}
		}

		// Update the record
		if err := db.Model(&file).Update("short_id", shortID).Error; err != nil {
			return fmt.Errorf("failed to update short_id for file %s: %w", file.ID, err)
		}
	}

	return nil
}
