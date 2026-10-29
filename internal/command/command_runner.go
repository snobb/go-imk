package command

import (
	"context"
	"io"
	"sync"
	"time"
)

// CommandRequestInfo represents information passed to the child process.
type CommandRequestInfo struct {
	FilePath  string
	Operation string
}

// CommandRunner manages the execution of a sequence of commands, ensuring that only one command is
// active at a time and handling cancellation and teardown.
type CommandRunner struct {
	commands []*Command

	tearDownTimeout time.Duration
	cancelFunc      context.CancelFunc
	done            chan struct{}
	mu              sync.Mutex
}

// NewCommandRunner creates a new CommandRunner instance with the provided commands, teardown timeout,
func NewCommandRunner(
	commands []string,
	tearDownTimeout time.Duration,
	outFiles []io.WriteCloser,
	wrapShell bool,
) *CommandRunner {
	cmds := make([]*Command, 0, len(commands))

	cmd := NewCommand(commands[0]).
		WithTimeout(tearDownTimeout)

	if wrapShell {
		cmd = cmd.WithShell()
	}

	cmds = append(cmds, cmd)

	for i, cmdStr := range commands[1:] {
		cmd = NewCommand(cmdStr).
			WithTimeout(tearDownTimeout)

		if len(outFiles) > 0 {
			cmd.WithOutput(outFiles[i])
		}

		if wrapShell {
			cmd = cmd.WithShell()
		}

		cmds = append(cmds, cmd)
	}

	return &CommandRunner{
		tearDownTimeout: tearDownTimeout,
		commands:        cmds,
	}
}

// Run the primary command. If the primary command have succeeded, it will execute the secondary
// command. The command is run in a separate go routine and can be long running. In case it's
// running, the command is killed and restarted.
// path can be empty string when the command is run durign the imk startup, but otherwise the
// changed file/dir should be passed.
func (cr *CommandRunner) Run(ctx context.Context, reqInfo *CommandRequestInfo) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.cancelFunc != nil {
		cr.cancelFunc()
		cr.cancelFunc = nil

		doneCh := cr.done
		cr.mu.Unlock()
		<-doneCh // unlock while waiting.
		cr.mu.Lock()
		cr.done = nil
	}

	var cancelCtx context.Context
	cancelCtx, cr.cancelFunc = context.WithCancel(ctx)

	cr.done = make(chan struct{})
	doneCh := cr.done // capture global channel to avoid it being overridden.

	go func() {
		defer close(doneCh)
		cr.runOnce(cancelCtx, reqInfo)
	}()

	return nil
}

func (cr *CommandRunner) runOnce(ctx context.Context, reqInfo *CommandRequestInfo) {
	for i, cmd := range cr.commands {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if i == 0 {
			_ = cmd.Execute(ctx, reqInfo)
			continue
		}

		go func() {
			_ = cmd.Execute(ctx, reqInfo)
		}()
	}
}
