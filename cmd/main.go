package main

import (
	"os"

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

	return nil
}
