package complete

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/posener/complete/cmd"
)

const (
	envLine  = "COMP_LINE"
	envPoint = "COMP_POINT"
	envDebug = "COMP_DEBUG"
)

// Complete structs define completion for a command with CLI options
type Complete struct {
	Command Command
	cmd.CLI
	Out io.Writer
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
		Out:     os.Stdout,
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
	line, point, ok := getEnv()
	if !ok {
		// make sure flags parsed,
		// in case they were not added in the main program
		return c.CLI.Run()
	}

	if point >= 0 && point < len(line) {
		line = line[:point]
	}

	Log("Completing phrase: %s", line)
	a := newArgs(line)
	Log("Completing last field: %s", a.Last)
	options := c.Command.Predict(a)
	Log("Options: %s", options)

	// filter only options that match the last argument
	matches := []string{}
	for _, option := range options {
		if strings.HasPrefix(option, a.Last) {
			matches = append(matches, option)
		}
	}
	Log("Matches: %s", matches)
	c.output(matches)
	return true
}

func getEnv() (line string, point int, ok bool) {
	line = os.Getenv(envLine)
	if line == "" {
		return
	}
	point, err := strconv.Atoi(os.Getenv(envPoint))
	if err != nil {
		// If failed parsing point for some reason, set it to point
		// on the end of the line.
		Log("Failed parsing point %s: %v", os.Getenv(envPoint), err)
		point = len(line)
	}
	return line, point, true
}

func (c *Complete) output(options []string) {
	// stdout of program defines the complete options
	for _, option := range options {
		fmt.Fprintln(c.Out, option)
	}
}
