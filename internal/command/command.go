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
	pgid int
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

	if c.TearDownTimeout > 0 {
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

	if err := c.cmd.Start(); err != nil {
		return err
	}

	// Record PGID once, while we know the process exists
	pgid, err := syscall.Getpgid(c.cmd.Process.Pid)
	if err == nil && pgid > 0 {
		c.pgid = pgid
	}

	if err := c.cmd.Wait(); err != nil {
		if isExpectedSignal(err) {
			logger.Shoutf("process killed by signal [%s %s]",
				c.Command, strings.Join(c.Args, " "))
			return nil
		}

		return err
	}

	logger.Shoutf("exit code %d [%s %s]", c.cmd.ProcessState.ExitCode(),
		c.Command, strings.Join(c.Args, " "))

	return nil
}

func (c *Command) Kill() {
	if c.cmd == nil || c.pgid == 0 {
		return
	}

	selfPGID, _ := syscall.Getpgid(0)
	if selfPGID == c.pgid {
		logger.Shout("refusing to commit suicide - attempting to kill own process group")
		return
	}

	_ = syscall.Kill(-c.pgid, syscall.SIGTERM)
}

func (c *Command) String() string {
	return c.Command + " " + strings.Join(c.Args, " ")
}

func isExpectedSignal(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return false
	}

	switch status.Signal() {
	case syscall.SIGTERM, syscall.SIGKILL:
		return true
	default:
		return false
	}
}
