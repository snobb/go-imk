package command

import (
	"context"
	"time"
)

type CommandRunner struct {
	PrimaryCmd   *Command
	SecondaryCmd *Command
	TeardownCmd  *Command

	tearDownTimeout time.Duration
}

func NewCommandRunner(
	primaryCmd, secondaryCmd, tearDownCmd string,
	tearDownTimeout time.Duration,
) *CommandRunner {
	pCmd := NewCommand(primaryCmd)
	if pCmd != nil {
		pCmd = pCmd.WithTimeout(tearDownTimeout)
	}

	return &CommandRunner{
		PrimaryCmd:      pCmd,
		SecondaryCmd:    NewCommand(secondaryCmd),
		TeardownCmd:     NewCommand(tearDownCmd),
		tearDownTimeout: tearDownTimeout,
	}
}

// Run the primary command. If the primary command have succeeded, it will execute the secondary
// command. The command is run in a separate go routine and can be long running. In case it's
// running, the command is killed and restarted.
func (cr *CommandRunner) Run(ctx context.Context) error {
	if err := cr.runPrimary(ctx); err != nil {
		return err
	}

	cr.runSecondary(ctx)

	return nil
}

func (cr *CommandRunner) runPrimary(ctx context.Context) error {
	if cr.PrimaryCmd == nil {
		return nil
	}

	return cr.PrimaryCmd.Execute(ctx)
}

func (cr *CommandRunner) runSecondary(ctx context.Context) {
	if cr.SecondaryCmd != nil {
		go func() {
			cr.SecondaryCmd.Execute(ctx)
		}()
	}
}
