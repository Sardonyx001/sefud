// Package main is the entry point for the sefud file upload/download server.
package main

import (
	"log"

	"github.com/Sardonyx001/sefud/config"
	"github.com/Sardonyx001/sefud/server"
)

func main() {
	cfg := config.NewConfig()

	s := server.New(cfg)
	if err := s.Start(cfg.App.Port); err != nil {
		log.Fatalf("Failed to start server, port already used?: %v", err)
	}
}
