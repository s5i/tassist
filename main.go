//go:build windows

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/s5i/taccount/server"
	"github.com/s5i/taccount/storage"
	"github.com/s5i/taccount/tray"
)

func main() {
	yamlPath := yamlFilePath()

	entries, err := storage.Load(yamlPath)
	if err != nil {
		log.Fatalf("Failed to load accounts: %v", err)
	}

	srv := server.New(&entries, yamlPath)

	url, err := srv.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	log.Printf("Listening on %s", url)

	tray.Run(url)
}

func yamlFilePath() string {
	return filepath.Join(os.Getenv("AppData"), "TAccount", "accounts.yaml")
}
