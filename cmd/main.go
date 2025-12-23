package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go-imk/internal/command"
	"go-imk/internal/config"
	"go-imk/internal/fsops"
	"go-imk/internal/logger"
	"go-imk/internal/ratelimit"
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

	logger.Shoutf("watching files and folders: %+v", cfg.Files)

	watcher := fsops.NewFileWatcher(cfg.Files)

	events, err := watcher.Watch(ctx)
	if err != nil {
		return err
	}

	commandRunner := command.NewCommandRunner(
		cfg.PrimaryCmd,
		cfg.SecondaryCmd,
		cfg.TearDownCmd,
		cfg.TearDownTimeout,
	)

	if cfg.RunNow {
		if err := commandRunner.Run(ctx); err != nil {
			return err
		}
	}

	// often there is a burst of events that comes at about the same time. Eg. IDE saves file and
	// then runs formatting tool, which results in 2 writes and thus 2 events.
	// So I'm introducing a rate limiter that would only allow one command per second regardless of
	// how many events have actuall come.
	rlimit := ratelimit.New(1) // one command per second

	for event := range events {
		if _, err := rlimit.Lease(ctx, 1); err != nil {
			continue // ignore event per rate limit
		}

		logger.Shoutf("%s :: %s", event.Op, event.Path)

		if err := commandRunner.Run(ctx); err != nil {
			return err
		}

		if cfg.OneRun {
			break
		}
	}

	return nil
}
