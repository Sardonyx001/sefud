package server

import (
	"github.com/Sardonyx001/sefud/handlers"
	"github.com/Sardonyx001/sefud/logger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	echo *echo.Echo
}

func New() *Server {
	e := echo.New()

	e.Use(logger.InitLoggerMiddleware())
	e.Use(middleware.Recover())

	// Routes
	e.POST("/up", handlers.UploadFile)
	e.GET("/:id", handlers.DownloadFile)
	e.DELETE("/:id", handlers.DeleteFile)

	return &Server{
		echo: e,
	}
}

func (s *Server) Start() error {
	return s.echo.Start(":7000")
}
