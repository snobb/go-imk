package command

import "context"

//go:generate moq -rm -fmt goimports -out runner_mock.go . Runner

type Runner interface {
	// Run the primary command. If the primary command have succeeded, it will execute the secondary
	// command. The command is run in a separate go routine and can be long running. In case it's
	// running, the command is killed and restarted.
	Run(context.Context) error
}
