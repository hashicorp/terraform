// Package complete provides a tool for bash writing bash completion in go.
//
// Writing bash completion scripts is a hard work. This package provides an easy way
// to create bash completion scripts for any command, and also an easy way to install/uninstall
// the completion of the command.
package complete

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/posener/complete/cmd"
)

const (
	envComplete = "COMP_LINE"
	envDebug    = "COMP_DEBUG"
)

// Complete structs define completion for a command with CLI options
type Complete struct {
	Command Command
	cmd.CLI
}

// New creates a new complete command.
// name is the name of command we want to auto complete.
// IMPORTANT: it must be the same name - if the auto complete
// completes the 'go' command, name must be equal to "go".
// command is the struct of the command completion.
func New(name string, command Command) *Complete {
	return &Complete{
		Command: command,
		CLI:     cmd.CLI{Name: name},
	}
}

// Run runs the completion and add installation flags beforehand.
// The flags are added to the main flag CommandLine variable.
func (c *Complete) Run() bool {
	c.AddFlags(nil)
	flag.Parse()
	return c.Complete()
}

// Complete a command from completion line in environment variable,
// and print out the complete options.
// returns success if the completion ran or if the cli matched
// any of the given flags, false otherwise
// For installation: it assumes that flags were added and parsed before
// it was called.
func (c *Complete) Complete() bool {
	line, ok := getLine()
	if !ok {
		// make sure flags parsed,
		// in case they were not added in the main program
		return c.CLI.Run()
	}
	Log("Completing line: %s", line)

	a := newArgs(line)

	options := c.Command.Predict(a)

	Log("Completion: %s", options)
	output(options)
	return true
}

func getLine() ([]string, bool) {
	line := os.Getenv(envComplete)
	if line == "" {
		return nil, false
	}
	return strings.Split(line, " "), true
}

func output(options []string) {
	Log("")
	// stdout of program defines the complete options
	for _, option := range options {
		fmt.Println(option)
	}
}
