package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"go-imk/internal/command"
	"go-imk/internal/fsops"
)

var (
	ErrNoPrimaryCommand   = errors.New("no primary command specified")
	ErrNoSecondaryCommand = errors.New("no secondary command specified")
)

type Config struct {
	Files []string

	PrimaryCmd   *command.Command
	SecondaryCmd *command.Command
	TearDownCmd  *command.Command

	CommandDelay time.Duration
	KillTimeout  time.Duration

	Recurse bool
	OneRun  bool
	RunNow  bool

	version    string
	fileWalker fsops.Walker
}

func New(version string, fileWalker fsops.Walker) *Config {
	return &Config{
		fileWalker: fileWalker,
		version:    version,
	}
}

func (c *Config) ParseCmdArgs() error {
	var version bool
	pflag.BoolVarP(&version, "version", "v", false,
		"print version and exit.")

	pflag.BoolVarP(&c.Recurse, "recurse", "r", false,
		"if a directory is supplied, add all its sub-directories as well.")

	pflag.BoolVarP(&c.OneRun, "once", "o", false,
		"run command once and exit on event.")

	pflag.BoolVarP(&c.RunNow, "immediate", "i", false,
		"run command immediately before watching for events.")

	pflag.DurationVarP(&c.CommandDelay, "threshold", "t", 0,
		"number of seconds to skip after the last executed command (default: 0).")

	pflag.DurationVarP(&c.KillTimeout, "timeout", "k", 0,
		"timeout after which to kill the command subproces (default - do not kill).")

	var primaryCmd string
	var secondaryCmd string
	var tearDownCmd string

	pflag.StringVarP(&primaryCmd, "command", "c", "",
		"command to execute when file is modified.")

	pflag.StringVarP(&secondaryCmd, "run", "u", "",
		"command to execute if primary command succeeded.")

	pflag.StringVarP(&tearDownCmd, "teardown", "d", "",
		"teardown command to execute when -k timeout occurs (assumes -w). "+
			"The PID is available in CMD_PID environment variable.")

	pflag.Parse()

	if version {
		fmt.Println(c.version)
		return nil
	}

	if primaryCmd == "" && secondaryCmd == "" {
		return fmt.Errorf("either primary or secondary command must be specified")
	}

	c.Files = pflag.Args()

	if c.Recurse {
		if err := c.EnrichFiles(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) EnrichFiles() error {
	for _, file := range c.Files {
		files, err := c.fileWalker.Walk(file)
		if err != nil {
			return err
		}

		c.Files = append(c.Files, files...)
	}

	return nil
}
