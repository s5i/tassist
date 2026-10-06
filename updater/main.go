//go:build windows

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

const listenAddr = "127.0.0.1:2138"

func main() {
	var logs updateLog
	log.SetFlags(0)
	log.SetOutput(&logs)

	var ready atomic.Bool

	log.Printf("Starting updater...")

	srv := &http.Server{
		Addr:     listenAddr,
		ErrorLog: log.Default(),
		Handler:  newMux(&logs, &ready),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Updater server failed: %v", err)
		}
	}()

	if !applyUpdate() {
		select {}
	}

	log.Printf("Update successful, restarting...")
	cmd := exec.Command(os.Args[1], "-open_browser=false")
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to restart %s: %v", os.Args[1], err)
		select {}
	}
	ready.Store(true)

	waitForApp()
	time.Sleep(5 * time.Second)
	srv.Close()
}

func newMux(logs *updateLog, ready *atomic.Bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/updater/log", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(logs.snapshot())
	})
	mux.HandleFunc("/updater/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Ready bool `json:"ready"`
		}{Ready: ready.Load()})
	})
	return mux
}

func applyUpdate() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if len(os.Args) < 3 {
		log.Printf("Usage: %s [target_exe] [new_exe]", os.Args[0])
		return false
	}

	dest := os.Args[1]
	src := os.Args[2]
	log.Printf("%s %q %q", os.Args[0], dest, src)
	log.Printf("Waiting for %s to become writeable...", dest)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Failed to overwrite %s.", dest)
			return false
		case <-ticker.C:
			if err := os.Rename(src, dest); err == nil {
				log.Printf("Replaced %s with %s.", dest, src)
				return true
			}
		}
	}
}

func waitForApp() {
	client := &http.Client{Timeout: time.Second}
	for {
		resp, err := client.Get("http://127.0.0.1:2137/api/version")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

type updateLog struct {
	mu sync.Mutex
	b  []byte
}

func (l *updateLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.b = append(l.b, p...)
	return len(p), nil
}

func (l *updateLog) snapshot() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]byte, len(l.b))
	copy(out, l.b)
	return out
}
