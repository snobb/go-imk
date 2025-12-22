package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go-imk/internal/config"
	"go-imk/internal/fsops"
	"go-imk/internal/logger"
)

var version string

func main() {
	cfg := config.New(version, fsops.DefaultWalker)

	if err := cfg.ParseCmdArgs(); err != nil {
		logger.Shoutf("error :: %s", err.Error())
		os.Exit(1)
	}

	if err := run(cfg); err != nil {
		logger.Shoutf("error :: %s", err.Error())
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	logger.Shoutf("%+v", cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		osSignalCh := make(chan os.Signal, 1)
		signal.Notify(osSignalCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

		select {
		case <-ctx.Done():
			return
		case sig := <-osSignalCh:
			logger.Shoutf("received sys signal %s", sig.String())
			cancel()
		}
	}()

	logger.Shoutf("watching files: %+v", cfg.Files)

	watcher := fsops.NewFileWatcher(cfg.Files)

	events, err := watcher.Watch(ctx)
	if err != nil {
		return err
	}

	for event := range events {
		logger.Shoutf("%s :: %s", event.Op, event.Path)
	}

	return nil
}
