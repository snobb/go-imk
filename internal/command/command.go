// Package command provides commands execution interface..
package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/snobb/go-imk/internal/logger"
)

const (
	StatusExit = iota
	StatusKill
	StatusError
)

type Command struct {
	Command string
	Args    []string

	TearDownTimeout time.Duration

	cmd  *exec.Cmd
	pgid int
	out  io.Writer

	exitChan   chan struct{}
	cancelFunc context.CancelFunc
}

func NewCommand(command string) *Command {
	if command == "" {
		return nil
	}

	tokens := strings.Fields(command)

	cmd := &Command{
		Command: tokens[0],
		out:     os.Stdout,
	}

	if len(tokens) > 1 {
		cmd.Args = tokens[1:]
	}

	return cmd
}

func (c *Command) WithTimeout(timeout time.Duration) *Command {
	c.TearDownTimeout = timeout
	return c
}

func (c *Command) WithOutput(out io.Writer) *Command {
	c.out = out
	return c
}

func (c *Command) WithShell() *Command {
	args := append([]string{c.Command}, c.Args...)
	c.Args = append([]string{"-c"}, strings.Join(args, " "))
	c.Command = "/bin/sh"
	return c
}

func (c *Command) Execute(ctx context.Context) error {
	if c.cancelFunc != nil {
		c.cancelFunc()
		<-c.exitChan
	}

	c.exitChan = make(chan struct{})

	ctx, c.cancelFunc = c.cancelContext(ctx)
	defer func() {
		c.cancelFunc()
		c.cancelFunc = nil
		close(c.exitChan) // broadcast the command has exited.
	}()

	//nolint:gosec // G204 - need to run the command.
	c.cmd = exec.CommandContext(ctx, c.Command, c.Args...)
	c.cmd.Stderr = os.Stderr
	c.cmd.Stdout = c.out

	// Run command in its own process group.
	c.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0, // make child process owner of the group
	}

	if err := c.cmd.Start(); err != nil {
		return err
	}

	c.pgid = c.cmd.Process.Pid

	if err := c.cmd.Wait(); err != nil {
		status, err := exitInfo(err)
		if err != nil {
			if status == StatusKill {
				logger.Shoutf("process killed by signal [%s %s]: %s",
					c.Command, strings.Join(c.Args, " "), err)
				return err
			}

			if status == StatusError {
				logger.Shoutf("error [%s %s]: %s", c.Command, strings.Join(c.Args, " "), err)
				return err
			}
		}

		if status == StatusKill {
			logger.Shoutf("process terminated [%s %s]",
				c.Command, strings.Join(c.Args, " "))
			return nil
		}
	}

	logger.Shoutf("exit code %d [%s %s]", c.cmd.ProcessState.ExitCode(),
		c.Command, strings.Join(c.Args, " "))

	return nil
}

func (c *Command) String() string {
	return c.Command + " " + strings.Join(c.Args, " ")
}

func (c *Command) cancelContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.TearDownTimeout > 0 {
		return context.WithTimeout(ctx, c.TearDownTimeout)
	}

	return context.WithCancel(ctx)
}

func exitInfo(err error) (int, error) {
	exitErr, ok := errors.AsType[*exec.ExitError](err)
	if !ok {
		return StatusError, fmt.Errorf("unexpected error > %w", err) // other error
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return StatusError, fmt.Errorf("no wait status > %w", err) // unknown error
	}

	if status.Exited() {
		return StatusExit, nil
	}

	switch status.Signal() {
	case syscall.SIGKILL:
		return StatusKill, nil // normal kill
	default:
		logger.Shoutf("unexpected signal [%d]", status.Signal())
		return StatusKill, fmt.Errorf("unexpected signal %s > %w", status.Signal(), err) // abnormal kill
	}
}
