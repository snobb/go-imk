package command

import (
	"context"
	"strings"
)

type Command struct {
	Command string
	Args    []string

	PID int
}

func New(command string) *Command {
	tokens := strings.Fields(command)

	return &Command{
		Command: tokens[0],
		Args:    tokens[1:],
	}
}

func (c *Command) Execute(ctx context.Context) error {
	return nil
}

func (c *Command) IsRunning() bool {
	return c.PID > 0
}

func (c *Command) String() string {
	return c.Command + " " + strings.Join(c.Args, " ")
}
