package command

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"go-imk/internal/logger"
)

type Command struct {
	Command string
	Args    []string

	TearDownTimeout time.Duration

	cmd *exec.Cmd
	wg  sync.WaitGroup
}

func NewCommand(command string) *Command {
	if command == "" {
		return nil
	}

	tokens := strings.Fields(command)

	cmd := &Command{
		Command: tokens[0],
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

func (c *Command) Execute(ctx context.Context) error {
	c.Kill()
	c.wg.Wait()

	if c.TearDownTimeout > 1 {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, c.TearDownTimeout)
		defer timeoutCancel()
	}

	//nolint:gosec // G204 - need to run the command.
	c.cmd = exec.Command(c.Command, c.Args...)
	c.cmd.Stderr = os.Stderr
	c.cmd.Stdout = os.Stdout

	// Run command in its own process group.
	c.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0, // make child process owner of the group
	}

	go func() {
		<-ctx.Done()
		c.Kill() // handle context cancellation.
	}()

	c.wg.Add(1)
	defer c.wg.Done()

	if err := c.cmd.Run(); err != nil {
		if isSigKill(err) {
			logger.Shout("process killed by signal")
			return nil
		}

		return err
	}

	logger.Shoutf("exit code %d", c.cmd.ProcessState.ExitCode())

	return nil
}

func (c *Command) Kill() {
	if c.cmd != nil {
		pid := c.cmd.Process.Pid
		if pid <= 0 {
			return
		}

		pgid, err := syscall.Getpgid(pid)
		if err != nil || pgid <= 0 {
			return
		}
		syscall.Kill(-pgid, syscall.SIGTERM)
	}
}

func (c *Command) String() string {
	return c.Command + " " + strings.Join(c.Args, " ")
}

func isSigKill(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return false
	}

	return status.Signaled() && status.Signal() == syscall.SIGKILL
}
