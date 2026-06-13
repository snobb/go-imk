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
	"sync"
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
	out             io.Writer

	mu     sync.Mutex
	active *activeExecution
}

type activeExecution struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	doneCh     chan struct{}
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
	// Atomically swap out or cancel all running instances.
	// (there shouldn't be more than one though).
	c.mu.Lock()
	for c.active != nil {
		c.active.cancelFunc()
		doneCh := c.active.doneCh
		c.mu.Unlock() // unlock while waiting.
		<-doneCh
		c.mu.Lock()
	}

	cancelCtx, cancelFunc := c.cancelContext(ctx)
	worker := &activeExecution{
		ctx:        cancelCtx,
		cancelFunc: cancelFunc,
		doneCh:     make(chan struct{}),
	}
	c.active = worker
	c.mu.Unlock()

	defer func() {
		worker.cancelFunc()
		c.mu.Lock()
		if c.active == worker {
			c.active = nil // only clear the pointer if it's still this instance.
		}
		c.mu.Unlock()
		close(worker.doneCh)
	}()

	return worker.run(cancelCtx, c)
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

func (ae *activeExecution) run(ctx context.Context, cfg *Command) error {
	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...) //nolint:gosec
	cmd.Stderr = os.Stderr
	cmd.Stdout = cfg.out

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	cmd.Cancel = func() error {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ae.doneCh:
			return nil // SIGTERM done it.
		case <-time.After(time.Second): // Wait for 1 sec before hard killing.
		}

		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			logger.Shoutf("process timed out [%s]", cfg)
			return ctx.Err()
		}

		if errors.Is(ctx.Err(), context.Canceled) {
			logger.Shoutf("process terminated [%s]", cfg)
			return ctx.Err()
		}

		status, err := exitInfo(err)
		if err != nil {
			if status == StatusKill {
				logger.Shoutf("process killed by signal [%s]: %s", cfg, err)
				return err
			}
			if status == StatusError {
				logger.Shoutf("error [%s]: %s", cfg, err)
				return err
			}
		}

		if status == StatusKill {
			logger.Shoutf("process terminated [%s]", cfg)
			return nil
		}
	}

	logger.Shoutf("exit code %d [%s]", cmd.ProcessState.ExitCode(), cfg)
	return nil
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
