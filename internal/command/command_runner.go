package command

import (
	"context"
	"io"
	"time"
)

type CommandRunner struct {
	commands []*Command

	tearDownTimeout time.Duration
}

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
func (cr *CommandRunner) Run(ctx context.Context) error {
	for i, cmd := range cr.commands {
		if i == 0 {
			if err := cmd.Execute(ctx); err != nil {
				return err
			}
			continue
		}

		go func() {
			_ = cmd.Execute(ctx)
		}()
	}

	return nil
}
