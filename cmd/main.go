package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/snobb/go-imk/internal/command"
	"github.com/snobb/go-imk/internal/config"
	"github.com/snobb/go-imk/internal/fsops"
	"github.com/snobb/go-imk/internal/logger"
	"github.com/snobb/go-imk/internal/ratelimit"
)

const defaultPermissions = 0644

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

	logger.Shoutf("start monitoring: %s", cfg)

	var outFiles []io.WriteCloser
	if cfg.OutFilePfx != "" {
		for i := range len(cfg.Commands) - 1 {
			outFile := fmt.Sprintf("%s%d.out", cfg.OutFilePfx, i+1)

			cmdOut, err := os.OpenFile(outFile,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, defaultPermissions)
			if err != nil {
				return err
			}

			logger.Shoutf("redirecting %d command output to file: %s", i+1, outFile)
			outFiles = append(outFiles, cmdOut)
		}

		defer func() {
			for _, outFile := range outFiles {
				outFile.Close()
			}
		}()
	}

	commandRunner := command.NewCommandRunner(
		cfg.Commands,
		cfg.TearDownTimeout,
		outFiles,
		cfg.WrapShell,
	)

	if cfg.RunNow {
		if err := commandRunner.Run(ctx); err != nil {
			return err
		}
	}

	// often there is a burst of events that comes at about the same time. Eg. IDE saves file and
	// then runs formatting tool, which results in 2 writes and thus 2 events.
	rlimit := ratelimit.NewTokenBucket(
		cfg.RateLimitBucketCapacity,
		cfg.RateLimitBucketCapacity,
		cfg.RateLimitInterval)

	watcher := fsops.NewFileWatcher(cfg.Files)

	events, err := watcher.Watch(ctx)
	if err != nil {
		return err
	}

	for event := range events {
		if !isInterestingOp(event.Op) {
			continue
		}

		if err := rlimit.Lease(1); err != nil {
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
