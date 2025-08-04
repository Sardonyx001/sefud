// Package main is the entry point for the sefud file upload/download server.
package main

import (
	"github.com/Sardonyx001/sefud/server"
)

func main() {
	s := server.New()
	if err := s.Start(); err != nil {
		panic(err)
	}
}
