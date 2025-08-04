//go:generate swag init -g cmd/sefud/main.go

// Package main is the entry point for the sefud file upload/download server.
//
// @title sefud API
// @description Simple, encrypted file upload & download service
// @version 1.0
// @host localhost:7000
// @BasePath /
// @schemes http https
//
// @contact.name API Support
// @contact.url https://github.com/Sardonyx001/sefud
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
package main

import (
	"log"

	"github.com/Sardonyx001/sefud/config"
	_ "github.com/Sardonyx001/sefud/docs"
	"github.com/Sardonyx001/sefud/server"
)

func main() {
	cfg := config.NewConfig()

	s := server.New(cfg)
	if err := s.Start(cfg.App.Port); err != nil {
		log.Fatalf("Failed to start server, port already used?: %v", err)
	}
}
