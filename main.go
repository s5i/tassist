//go:build windows

package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/s5i/goutil/version"
	"github.com/s5i/tassist/acc"
	"github.com/s5i/tassist/exp"
	"github.com/s5i/tassist/loot"
	"github.com/s5i/tassist/online"
	"github.com/s5i/tassist/ping"
	"github.com/s5i/tassist/server"
	"github.com/s5i/tassist/settings"
	"github.com/s5i/tassist/timer"
	"golang.org/x/sync/errgroup"
)

var (
	dir    = flag.String("dir", filepath.Join(os.Getenv("AppData"), "TAssistant"), "Path to persistent dir.")
	tmpDir = flag.String("tmp_dir", filepath.Join(os.Getenv("Temp"), "tassist"), "Path to temp dir.")
)

func main() {
	flag.Parse()

	for {
		var exit bool
		var exitCode int

		func() {
			if f, err := os.OpenFile(filepath.Join(*tmpDir, "tassist.log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644); err == nil {
				log.SetOutput(io.MultiWriter(f, os.Stderr))
				defer f.Close()
			}

			err := mainErr()
			switch {

			case errors.Is(err, server.ErrRestart):
				log.Printf("Restarting...")
				exit = false

			case err != nil:
				log.Printf("Quitting with error: %v", err)
				exit, exitCode = true, 1

			default:
				log.Printf("Quitting.")
				exit, exitCode = true, 0
			}
		}()

		if exit {
			os.Exit(exitCode)
		}
	}
}

func mainErr() (retErr error) {
	if err := os.MkdirAll(*tmpDir, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(*dir, 0755); err != nil {
		return err
	}

	defer logPanic()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ver := version.Get()
	log.Printf("Running Tibiantis Assist %s", ver)

	stStorage, err := settings.New(*dir)
	if err != nil {
		return err
	}

	accStorage, err := acc.New(*dir, stStorage)
	if err != nil {
		return err
	}

	expCache, err := exp.NewCache(*tmpDir, stStorage)
	if err != nil {
		log.Printf("exp.NewCache() failed: %v", err)
		return err
	}

	pinger, err := ping.New(stStorage)
	if err != nil {
		log.Printf("ping.New() failed: %v", err)
		return err
	}

	online, err := online.New(stStorage)
	if err != nil {
		log.Printf("online.New() failed: %v", err)
		return err
	}

	tmStorage, err := timer.NewStorage(*dir)
	if err != nil {
		log.Printf("timer.NewStorage() failed: %v", err)
		return err
	}

	lootStorage, err := loot.NewStorage(*dir)
	if err != nil {
		log.Printf("loot.NewStorage() failed: %v", err)
		return err
	}

	srv, err := server.New(*dir, *tmpDir, accStorage, expCache, pinger, online, ver, stStorage, tmStorage, loot.NewService(), lootStorage)
	if err != nil {
		log.Printf("server.New() failed: %v", err)
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer logPanic()
		return expCache.Run(ctx)
	})

	eg.Go(func() error {
		defer logPanic()
		return pinger.Run(ctx)
	})

	eg.Go(func() error {
		defer logPanic()
		return online.Run(ctx)
	})

	eg.Go(func() error {
		defer logPanic()
		return srv.Run(ctx)
	})

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

func logPanic() {
	if r := recover(); r != nil {
		log.Printf("Crash detected: %v", r)
		log.Printf("Stack trace:\n%s", string(debug.Stack()))
	}
}
