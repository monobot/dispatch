package models

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"testing"
)

func captureCommandOutput(f func()) string {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	return buf.String()
}

func TestCommand_Help(t *testing.T) {
	tests := []struct {
		name     string
		command  Command
		indent   int
		expected string
	}{
		{
			name: "No conditions",
			command: Command{
				Command: "echo 'Hello, world!'",
			},
			indent:   1,
			expected: "        - \"echo 'Hello, world!'\"\n",
		},
		{
			name: "With conditions",
			command: Command{
				Command: "echo 'Hello, world!'",
				Conditions: []Condition{
					{
						Variable:  "USER",
						Value:     "admin",
						Allowance: true,
					},
				},
			},
			indent:   1,
			expected: "        - \"echo 'Hello, world!'\"\n            Conditions:\n                When variable \"USER\" equals \"admin\" then Allow\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureCommandOutput(func() {
				tt.command.Help(tt.indent)
			})

			if output != tt.expected {
				t.Errorf("Help() output = %v, want %v", output, tt.expected)
			}
		})
	}
}

func TestCommand_IsAllowed(t *testing.T) {
	tests := []struct {
		name           string
		command        Command
		configuration  *Configuration
		expectedAllow  bool
		expectedOutput string // Expected output if verbose flag is set
	}{
		{
			name: "Allowed condition",
			command: Command{
				Command: "test command",
				Conditions: []Condition{
					{
						Variable:  "USER",
						Value:     "admin",
						Allowance: true,
					},
				},
			},
			configuration: &Configuration{
				ContextData: ContextData{
					Data: map[string]string{
						"USER": "admin",
					},
				},
			},
			expectedAllow: true,
		},
		{
			name: "Not allowed condition",
			command: Command{
				Command: "test command",
				Conditions: []Condition{
					{
						Variable:  "USER",
						Value:     "guest",
						Allowance: true,
					},
				},
			},
			configuration: &Configuration{
				ContextData: ContextData{
					Data: map[string]string{
						"USER": "admin",
					},
				},
			},
			expectedAllow:  false,
			expectedOutput: "    condition: When variable \"USER\" equals \"guest\" then Allow not met\n",
		},
		{
			name: "Allowed condition not allowance",
			command: Command{
				Command: "test command",
				Conditions: []Condition{
					{
						Variable:  "USER",
						Value:     "admin",
						Allowance: false,
					},
				},
			},
			configuration: &Configuration{
				ContextData: ContextData{
					Data: map[string]string{
						"USER": "guest",
					},
				},
			},
			expectedAllow: true,
		},
		{
			name: "Not allowed condition not allowance",
			command: Command{
				Command: "test command",
				Conditions: []Condition{
					{
						Variable:  "USER",
						Value:     "guest",
						Allowance: false,
					},
				},
			},
			configuration: &Configuration{
				ContextData: ContextData{
					Data: map[string]string{
						"USER": "guest",
					},
				},
			},
			expectedAllow:  false,
			expectedOutput: "    condition: When variable \"USER\" equals \"guest\" then Allow not met\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowance := tt.command.IsAllowed(tt.configuration)

			if allowance != tt.expectedAllow {
				t.Errorf("IsAllowed() = %v, want %v", allowance, tt.expectedAllow)
			}
		})
	}
}

// MockExecCommand is a function that simulates exec.Command for testing.
// It returns a *exec.Cmd that when run, will return the specified output and error.
func MockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}
