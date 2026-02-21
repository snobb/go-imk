// Package config provides configuration for go-imk
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/snobb/go-imk/internal/fsops"
)

var (
	ErrNoPrimaryCommand   = errors.New("no primary command specified")
	ErrNoSecondaryCommand = errors.New("no secondary command specified")
)

const (
	defaultRateLimitBucketCapacity = 1
	defaultRateLimitInterval       = 1500 * time.Millisecond
)

type Config struct {
	Files []string

	Commands []string

	TearDownTimeout time.Duration

	Recurse   bool
	OneRun    bool
	RunNow    bool
	WrapShell bool

	OutFilePfx string

	RateLimitBucketCapacity int
	RateLimitInterval       time.Duration

	version    string
	fileWalker fsops.Walker
}

func New(version string, fileWalker fsops.Walker) *Config {
	return &Config{
		RateLimitBucketCapacity: defaultRateLimitBucketCapacity,
		fileWalker:              fileWalker,
		version:                 version,
	}
}

func (c *Config) ParseCmdArgs() error {
	var version bool
	pflag.BoolVarP(&version, "version", "v", false,
		fmt.Sprintf("print version and exit. [%s]", c.version))

	pflag.BoolVarP(&c.Recurse, "recurse", "r", false,
		"if a directory is supplied, add all its sub-directories as well.")

	pflag.BoolVarP(&c.OneRun, "once", "n", false,
		"run primary command once and exit on event.")

	pflag.StringVarP(&c.OutFilePfx, "output", "o", "",
		"send the stdout of secondary command to a file with given prefix.")

	pflag.BoolVarP(&c.RunNow, "immediate", "i", false,
		"run commands immediately before watching for events.")

	pflag.StringArrayVarP(&c.Commands, "command", "c", []string{},
		"command to execute on change (can be specified multiple times)")

	pflag.BoolVarP(&c.WrapShell, "wrapshell", "w", false,
		"wrap command with shell (default - false).")

	pflag.DurationVarP(&c.TearDownTimeout, "timeout", "k", 0,
		"timeout after which to kill the command subprocess (default - do not kill).")

	pflag.DurationVarP(&c.RateLimitInterval, "rate-limit-interval", "",
		defaultRateLimitInterval, "Limit execution to one command per the given <interval>.")

	pflag.Usage = usage

	if len(os.Args) < 2 { //nolint:mnd
		pflag.Usage()
		os.Exit(0)
	}

	pflag.Parse()

	if version {
		fmt.Println(c.version)
		os.Exit(0)
	}

	if len(c.Commands) == 0 {
		return fmt.Errorf("at least one command must be specified")
	}

	if c.OneRun && len(c.Commands) > 1 {
		return fmt.Errorf("secondary commands is not supported with -o flag")
	}

	c.Files = pflag.Args()

	if c.Recurse {
		if err := c.EnrichFiles(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) String() string {
	tokens := make([]string, 0)

	for _, cmd := range c.Commands {
		tokens = append(tokens, fmt.Sprintf("command[%s]", cmd))
	}

	if c.TearDownTimeout != 0 {
		tokens = append(tokens, fmt.Sprintf("timeout[%s]", c.TearDownTimeout.String()))
	}

	if c.Recurse {
		tokens = append(tokens, "recurse")
	}

	if c.OneRun {
		tokens = append(tokens, "one-run")
	}

	if c.RunNow {
		tokens = append(tokens, "immediate")
	}

	if c.WrapShell {
		tokens = append(tokens, "wrapshell")
	}

	if c.Files != nil {
		tokens = append(tokens, fmt.Sprintf("files[%s]", strings.Join(c.Files, ",")))
	}

	return strings.Join(tokens, " ")
}

func (c *Config) EnrichFiles() error {
	withChildren := make([]string, 0, len(c.Files))
	for _, file := range c.Files {
		files, err := c.fileWalker.Walk(file)
		if err != nil {
			return err
		}

		withChildren = append(withChildren, files...)
	}

	c.Files = withChildren

	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	pflag.PrintDefaults()
	fmt.Println("\nIf multiple commands are specified, the first runs in the foreground as the primary " +
		"command.\nSubsequent commands run in the background and may be long-running.")
	fmt.Println("\nExamples:")
	fmt.Println("  imk -rc 'go build ./...' src/")
	fmt.Println("  imk -rc 'go build ./...' src/ -k 5m")
	fmt.Println("  imk -ric 'go build ./...' -c 'go run ./...' src/")
	fmt.Println()
}
