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

	cmd  *exec.Cmd
	wg   sync.WaitGroup
	done chan struct{}
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

	if c.TearDownTimeout > 1 {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, c.TearDownTimeout)
		defer timeoutCancel()
	}

	c.done = make(chan struct{})
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-c.done
		cancel()
	}()

	//nolint:gosec // G204 - need to run the command.
	c.cmd = exec.CommandContext(ctx, c.Command, c.Args...)
	c.cmd.Stderr = os.Stderr
	c.cmd.Stdout = os.Stdout

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
		close(c.done)
		c.wg.Wait()
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
