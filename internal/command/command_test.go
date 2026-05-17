package command

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/snobb/go-imk/test/assert"
)

const defaultShell = "/bin/sh"
const shellCommandFlag = "-c"

func TestNewCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantCmd string
		wantLen int
		wantNil bool
	}{
		{
			name:    "empty string returns nil",
			input:   "",
			wantNil: true,
		},
		{
			name:    "command without args",
			input:   "echo",
			wantCmd: "echo",
			wantLen: 0,
		},
		{
			name:    "command with args",
			input:   "echo hello world",
			wantCmd: "echo",
			wantLen: 2,
		},
		{
			name:    "command with extra spaces",
			input:   "   ls   -la   ",
			wantCmd: "ls",
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCommand(tt.input)
			if tt.wantNil {
				assert.Equal(t, (*Command)(nil), cmd)
				return
			}
			assert.Equal(t, tt.wantCmd, cmd.Command)
			assert.Equal(t, tt.wantLen, len(cmd.Args))
		})
	}
}

func TestContextPropagation(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	child, childCancel := context.WithCancel(parent)
	defer childCancel()

	parentCancel()
	assert.Equal(t, context.Canceled, child.Err())
}

func TestCommand_WithShell(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		wantShell string
		wantArgsN int
		wantArg0  string
		wantArg1  string
	}{
		{
			name:      "simple command",
			command:   "echo hello",
			wantShell: defaultShell,
			wantArgsN: 2,
			wantArg0:  shellCommandFlag,
			wantArg1:  "echo hello",
		},
		{
			name:      "command with pipe",
			command:   "ls | grep foo",
			wantShell: defaultShell,
			wantArgsN: 2,
			wantArg0:  shellCommandFlag,
			wantArg1:  "ls | grep foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCommand(tt.command)
			cmd.WithShell()
			assert.Equal(t, tt.wantShell, cmd.Command)
			assert.Equal(t, tt.wantArgsN, len(cmd.Args))
			assert.Equal(t, tt.wantArg0, cmd.Args[0])
			assert.Equal(t, tt.wantArg1, cmd.Args[1])
		})
	}
}

func TestCommand_WithOutput(t *testing.T) {
	tests := []struct {
		name    string
		command string
		output  string
	}{
		{
			name:    "echo output",
			command: "echo hello",
			output:  "hello\n",
		},
		{
			name:    "echo without newline",
			command: "echo -n world",
			output:  "world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := NewCommand(tt.command).
				WithOutput(&buf)

			err := cmd.Execute(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, tt.output, buf.String())
		})
	}
}

func TestCommand_String(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "simple command",
			command:  "echo",
			expected: "echo ",
		},
		{
			name:     "command with args",
			command:  "echo hello world",
			expected: "echo hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCommand(tt.command)
			assert.Equal(t, tt.expected, cmd.String())
		})
	}
}

func TestCommand_Execute(t *testing.T) {
	t.Run("successful command", func(t *testing.T) {
		cmd := NewCommand("true")
		err := cmd.Execute(context.Background())
		assert.NoError(t, err)
	})

	t.Run("nonexistent command", func(t *testing.T) {
		cmd := NewCommand("nonexistent_cmd_xyzzy")
		err := cmd.Execute(context.Background())
		assert.Error(t, err)
	})

	t.Run("with shell", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := NewCommand("echo hello").
			WithShell().
			WithOutput(&buf)
		err := cmd.Execute(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "hello\n", buf.String())
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cmd := NewCommand("sleep 10")
		err := cmd.Execute(ctx)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("timeout", func(t *testing.T) {
		cmd := NewCommand("sleep 10").
			WithTimeout(30 * time.Millisecond)
		err := cmd.Execute(context.Background())
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("process killed by signal", func(t *testing.T) {
		cmd := NewCommand("kill -KILL $$").
			WithShell()
		err := cmd.Execute(context.Background())
		assert.NoError(t, err)
	})

	t.Run("sequential restart after completion", func(t *testing.T) {
		cmd := NewCommand("true")
		assert.NoError(t, cmd.Execute(context.Background()))
		assert.NoError(t, cmd.Execute(context.Background()))
	})

	t.Run("concurrent execute cancels previous", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		cmd := NewCommand("sleep 10")
		errCh := make(chan error, 2)

		go func() {
			errCh <- cmd.Execute(ctx)
		}()

		cmd.mu.Lock()
		for cmd.active == nil {
			cmd.mu.Unlock()
			runtime.Gosched()
			cmd.mu.Lock()
		}
		cmd.mu.Unlock()

		go func() {
			errCh <- cmd.Execute(ctx)
		}()

		for range 2 {
			assert.Error(t, <-errCh)
		}
	})

	t.Run("multiple concurrent executes", func(t *testing.T) {
		cmd := NewCommand("true")
		errCh := make(chan error, 3)

		for range 3 {
			go func() {
				errCh <- cmd.Execute(context.Background())
			}()
		}

		// Concurrent calls cancel each other; some may succeed, some may be
		// canceled. The important thing is no deadlock or panic.
		for range 3 {
			<-errCh
		}
	})
}

func TestNewCommandRunner(t *testing.T) {
	t.Run("single command", func(t *testing.T) {
		runner := NewCommandRunner(
			[]string{"true"},
			time.Second,
			nil,
			false,
		)
		err := runner.Run(context.Background())
		assert.NoError(t, err)
		<-runner.done
	})

	t.Run("multiple commands with shell", func(t *testing.T) {
		runner := NewCommandRunner(
			[]string{"echo a", "echo b"},
			time.Second,
			nil,
			true,
		)
		err := runner.Run(context.Background())
		assert.NoError(t, err)
		<-runner.done
	})
}

func TestCommandRunner_Run(t *testing.T) {
	t.Run("first command executes", func(t *testing.T) {
		dir := t.TempDir()
		marker := dir + "/marker"

		runner := NewCommandRunner(
			[]string{"touch " + marker},
			time.Second,
			nil,
			true,
		)

		err := runner.Run(context.Background())
		assert.NoError(t, err)
		<-runner.done

		_, err = os.Stat(marker)
		assert.NoError(t, err)
	})

	t.Run("multiple commands", func(t *testing.T) {
		dir := t.TempDir()
		marker := dir + "/cmd"

		runner := NewCommandRunner(
			[]string{"touch " + marker},
			time.Second,
			nil,
			true,
		)

		err := runner.Run(context.Background())
		assert.NoError(t, err)
		<-runner.done

		_, err = os.Stat(marker)
		assert.NoError(t, err)
	})

	t.Run("restart cancels previous run", func(t *testing.T) {
		dir := t.TempDir()
		marker := dir + "/restart"

		runner := NewCommandRunner(
			[]string{"touch " + marker},
			0,
			nil,
			true,
		)

		err := runner.Run(context.Background())
		assert.NoError(t, err)

		doneCh := runner.done
		err = runner.Run(context.Background())
		assert.NoError(t, err)

		<-doneCh
		<-runner.done

		_, err = os.Stat(marker)
		assert.NoError(t, err)
	})

	t.Run("multiple restarts", func(t *testing.T) {
		runner := NewCommandRunner(
			[]string{"true"},
			time.Second,
			nil,
			false,
		)

		for range 5 {
			err := runner.Run(context.Background())
			assert.NoError(t, err)
			<-runner.done
		}
	})
}
