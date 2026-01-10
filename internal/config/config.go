package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
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

	OutFile string

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

	pflag.BoolVarP(&c.OneRun, "once", "n", false,
		"run primary command once and exit on event.")

	pflag.StringVarP(&c.OutFile, "output", "o", "",
		"send the stdout of secondary command to a file.")

	pflag.BoolVarP(&c.RunNow, "immediate", "i", false,
		"run commands immediately before watching for events.")

	pflag.StringVarP(&c.PrimaryCmd, "command", "c", "",
		"primary command to execute when a file or a folder is modified.")

	pflag.StringVarP(&c.SecondaryCmd, "run", "u", "",
		"secondary command to execute if primary command succeeded - runs in background.")

	pflag.DurationVarP(&c.TearDownTimeout, "timeout", "k", 0,
		"timeout after which to kill the command subprocess (default - do not kill).")

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

	if c.OneRun && c.SecondaryCmd != "" {
		return fmt.Errorf("secondary command is not supported with -o flag")
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

	if c.PrimaryCmd != "" {
		tokens = append(tokens, fmt.Sprintf("primary[%s]", c.PrimaryCmd))
	}

	if c.SecondaryCmd != "" {
		tokens = append(tokens, fmt.Sprintf("secondary[%s]", c.SecondaryCmd))
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
	fmt.Println("\nIt is required to specify either primary or secondary command (or both).")
	fmt.Println("\nThe secondary command will run in the background and will be restarted " +
		"immediately after the primary command is executed the next time.")
	fmt.Println("\nExamples:")
	fmt.Println("  imk -rc 'go build ./...' src/")
	fmt.Println("  imk -rc 'go build ./...' src/ -k 5m")
	fmt.Println("  imk -ric 'go build ./...' -u 'go run ./...' src/")
	fmt.Println()
}
