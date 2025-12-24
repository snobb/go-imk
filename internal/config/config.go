package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"

	"go-imk/internal/fsops"
)

var (
	ErrNoPrimaryCommand   = errors.New("no primary command specified")
	ErrNoSecondaryCommand = errors.New("no secondary command specified")
)

type Config struct {
	Files []string

	PrimaryCmd   string
	SecondaryCmd string

	TearDownTimeout time.Duration

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
		fmt.Sprintf("print version and exit. [%s]", c.version))

	pflag.BoolVarP(&c.Recurse, "recurse", "r", false,
		"if a directory is supplied, add all its sub-directories as well.")

	pflag.BoolVarP(&c.OneRun, "once", "o", false,
		"run command once and exit on event.")

	pflag.BoolVarP(&c.RunNow, "immediate", "i", false,
		"run command immediately before watching for events.")

	pflag.StringVarP(&c.PrimaryCmd, "command", "c", "",
		"command to execute when file is modified.")

	pflag.StringVarP(&c.SecondaryCmd, "run", "u", "",
		"command to execute if primary command succeeded - runs in background. ")

	pflag.DurationVarP(&c.TearDownTimeout, "timeout", "k", 0,
		"timeout after which to kill the command subproces (default - do not kill).")

	pflag.Usage = usage

	if len(os.Args) < 2 {
		pflag.Usage()
		os.Exit(0)
	}

	pflag.Parse()

	if version {
		fmt.Println(c.version)
		os.Exit(0)
	}

	if c.PrimaryCmd == "" && c.SecondaryCmd == "" {
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

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	pflag.PrintDefaults()
	fmt.Println("\nThe secondary command will run in the background and will be " +
		"killed if the primary command fails.")
	fmt.Println("\nExamples:")
	fmt.Println("  imk -rc 'go build ./...' src/")
	fmt.Println("  imk -rc 'go build ./...' src/ -k 5m")
	fmt.Println("  imk -ric 'go build ./...' -u 'go run ./...' src/ -o")
	fmt.Println()
}
