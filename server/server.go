// Package server initializes and runs the HTTP server for file upload/download.
package server

import (
	"github.com/Sardonyx001/sefud/config"
	"github.com/Sardonyx001/sefud/db"
	"github.com/Sardonyx001/sefud/handlers"
	"github.com/Sardonyx001/sefud/logger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	"gorm.io/gorm"
)

type Server struct {
	echo   *echo.Echo
	db     *gorm.DB
	config *config.Config
}

func New(cfg *config.Config) *Server {
	e := echo.New()

	e.Use(logger.InitLoggerMiddleware())
	e.Use(middleware.Recover())

	// API Routes
	e.POST("/up", handlers.UploadFile)
	e.GET("/:id", handlers.DownloadFile)
	e.DELETE("/:id", handlers.DeleteFile)

	// Swagger documentation route
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	return &Server{
		echo:   e,
		db:     db.Init(cfg),
		config: cfg,
	}
}

func (s *Server) Start(addr string) error {
	return s.echo.Start(":" + addr)
}
