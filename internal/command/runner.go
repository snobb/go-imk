package command

import "context"

//go:generate moq -rm -fmt goimports -out runner_mock.go . Runner

type Runner interface {
	// Build runs the build command on a change event.
	Build(context.Context) error

	// Execute runs the execute command in case the build command has succeeded. The command is run
	// in a separate go routine and can be long running.
	// In case it's running, the command is killed and restarted.
	Execute(context.Context) error
}
