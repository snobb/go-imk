package main

import (
	"fmt"
	"os"

	"go-imk/internal/config"
	"go-imk/internal/fsops"
)

var version string

func main() {
	cfg := config.New(version, fsops.DefaultWalker)

	if err := cfg.ParseCmdArgs(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err := run(cfg); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	fmt.Printf("%+v\n", cfg)
	return nil
}
