package command

import (
	"context"
	"io"
	"sync"
	"time"
)

type CommandRunner struct {
	commands []*Command

	tearDownTimeout time.Duration
	outFile         string
}

func NewCommandRunner(
	commands []string,
	tearDownTimeout time.Duration,
	outFiles []io.Writer,
) *CommandRunner {
	cmds := make([]*Command, 0, len(commands))

	cmd := NewCommand(commands[0]).
		WithTimeout(tearDownTimeout)
	cmds = append(cmds, cmd)

	for i, cmdStr := range commands[1:] {
		cmd = NewCommand(cmdStr).
			WithTimeout(tearDownTimeout)

		if len(outFiles) > 0 {
			cmd.WithOutput(outFiles[i])
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

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			cmd.Execute(ctx)
		}()

		wg.Wait()

		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}

	return nil
}
