//go:build windows

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/s5i/taccount/server"
	"github.com/s5i/taccount/tray"
)

func main() {
	storagePath := storagePath()

	srv, err := server.New(storagePath)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	url, err := srv.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Printf("Listening on %s", url)

	tray.Run(url)
}

func storagePath() string {
	return filepath.Join(os.Getenv("AppData"), "TAccount", "accounts.yaml")
}
