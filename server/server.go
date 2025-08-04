// Package server initializes and runs the HTTP server for file upload/download.
package server

import (
	"github.com/Sardonyx001/sefud/config"
	"github.com/Sardonyx001/sefud/db"
	"github.com/Sardonyx001/sefud/handlers"
	"github.com/Sardonyx001/sefud/logger"
	"github.com/Sardonyx001/sefud/models"
	"github.com/Sardonyx001/sefud/storage"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	"gorm.io/gorm"
)

type Server struct {
	echo        *echo.Echo
	db          *gorm.DB
	config      *config.Config
	r2Client    *storage.R2Client
	fileHandler *handlers.FileHandler
}

func New(cfg *config.Config) (*Server, error) {
	e := echo.New()

	// Initialize database
	database := db.Init(cfg)
	
	// Auto-migrate database tables
	if err := database.AutoMigrate(&models.File{}); err != nil {
		return nil, err
	}

	// Initialize R2 client
	r2Client, err := storage.NewR2Client(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize file handler with dependencies
	fileHandler := handlers.NewFileHandler(database, r2Client, cfg)

	// Middleware
	e.Use(logger.InitLoggerMiddleware())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	
	// Security middleware
	e.Use(middleware.Secure())
	e.Use(middleware.RequestID())
	
	// Rate limiting for uploads (adjust as needed)
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20))) // 20 requests per second

	// Swagger documentation route
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// API Routes with the file handler methods
	e.POST("/up", fileHandler.UploadFile)
	e.GET("/:id", fileHandler.DownloadFile)
	e.DELETE("/:id", fileHandler.DeleteFile)
	e.HEAD("/:id", fileHandler.GetFileInfo) // For checking file existence

	return &Server{
		echo:        e,
		db:          database,
		config:      cfg,
		r2Client:    r2Client,
		fileHandler: fileHandler,
	}, nil
}

func (s *Server) Start(addr string) error {
	return s.echo.Start(":" + addr)
}
