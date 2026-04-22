//go:build windows

package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/s5i/tassist/server"
	"github.com/s5i/tassist/tray"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx := context.Background()

	srv, err := server.New(storagePath())
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	eg, ctx := errgroup.WithContext(ctx)
	defer eg.Wait()

	eg.Go(func() error {
		return srv.Run(ctx)
	})

	<-srv.Ready()
	addr := "http://" + srv.Addr()

	tray.OpenBrowser(addr)
	tray.Run(addr)
}

func storagePath() string {
	return filepath.Join(os.Getenv("AppData"), "TAssistant", "accounts.yaml")
}
