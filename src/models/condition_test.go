package models

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestConditionHelpString(t *testing.T) {
	// Disable color output for testing
	color.NoColor = true

	tests := []struct {
		name      string
		condition Condition
		expected  string
	}{
		{
			name: "Allow condition",
			condition: Condition{
				Variable:  "USER",
				Value:     "admin",
				Allowance: true,
			},
			expected: `variable "USER" equals "admin" then Allow`,
		},
		{
			name: "Deny condition",
			condition: Condition{
				Variable:  "USER",
				Value:     "guest",
				Allowance: false,
			},
			expected: `variable "USER" equals "guest" then Deny`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.condition.HelpString()
			if !strings.EqualFold(result, tt.expected) {
				t.Errorf("HelpString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func captureConditionOutput(f func()) string {
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

func TestConditionHelp(t *testing.T) {
	// Disable color output for testing
	color.NoColor = true

	tests := []struct {
		name     string
		cond     Condition
		expected string
	}{
		{
			name: "Allow condition with indent",
			cond: Condition{
				Variable:  "USER",
				Value:     "admin",
				Allowance: true,
			},
			expected: "            When variable \"USER\" equals \"admin\" then Allow\n",
		},
		{
			name: "Deny condition with indent",
			cond: Condition{
				Variable:  "USER",
				Value:     "guest",
				Allowance: false,
			},
			expected: "            When variable \"USER\" equals \"guest\" then Deny\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureConditionOutput(func() {
				tt.cond.Help(2) // Assuming 4 spaces per indent level
			})

			if !strings.EqualFold(output, tt.expected) {
				t.Errorf("Help() output = %v, want %v", output, tt.expected)
			}
		})
	}
}
