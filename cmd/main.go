package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	secondaryOutput := os.Stdout
	if cfg.SecondaryCmd != "" && cfg.OutFile != "" {
		secondaryOutput, err = os.OpenFile(cfg.OutFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer secondaryOutput.Close()

		logger.Shoutf("redirecting secondary command output to file: %s", cfg.OutFile)
	}

	commandRunner := command.NewCommandRunner(
		cfg.PrimaryCmd,
		cfg.SecondaryCmd,
		cfg.TearDownTimeout,
		secondaryOutput,
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
	rlimit := ratelimit.New(1, time.Second) // one command per second

	for event := range events {
		if !isInterestingOp(event.Op) {
			continue
		}

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

func isInterestingOp(op string) bool {
	return op == "CREATE" || op == "RENAME" || op == "WRITE"
}
