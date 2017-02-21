package repl

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

// ErrSessionExit is a special error result that should be checked for
// from Handle to signal a graceful exit.
var ErrSessionExit = errors.New("session exit")

// Session represents the state for a single REPL session.
type Session struct {
	// Interpolater is used for calculating interpolations
	Interpolater *terraform.Interpolater
}

// Handle handles a single line of input from the REPL.
//
// This is a stateful operation if a command is given (such as setting
// a variable). This function should not be called in parallel.
//
// The return value is the output and the error to show.
func (s *Session) Handle(line string) (string, error) {
	switch {
	case strings.TrimSpace(line) == "exit":
		return "", ErrSessionExit
	case strings.TrimSpace(line) == "help":
		return s.handleHelp()
	default:
		return s.handleEval(line)
	}
}

func (s *Session) handleEval(line string) (string, error) {
	// Wrap the line to make it an interpolation.
	line = fmt.Sprintf("${%s}", line)

	// Parse the line
	raw, err := config.NewRawConfig(map[string]interface{}{
		"value": line,
	})
	if err != nil {
		return "", err
	}

	// Set the value
	raw.Key = "value"

	// Get the values
	vars, err := s.Interpolater.Values(&terraform.InterpolationScope{
		Path: []string{"root"},
	}, raw.Variables)
	if err != nil {
		return "", err
	}

	// Interpolate
	if err := raw.Interpolate(vars); err != nil {
		return "", err
	}

	// If we have any unknown keys, let the user know.
	if ks := raw.UnknownKeys(); len(ks) > 0 {
		return "", fmt.Errorf("unknown values referenced, can't compute value")
	}

	// Read the value
	result, err := FormatResult(raw.Value())
	if err != nil {
		return "", err
	}

	return result, nil
}

func (s *Session) handleHelp() (string, error) {
	text := `
The Terraform console allows you to experiment with Terraform interpolations.
You may access resources in the state (if you have one) just as you would
from a configuration. For example: "aws_instance.foo.id" would evaluate
to the ID of "aws_instance.foo" if it exists in your state.

Type in the interpolation to test and hit <enter> to see the result.

To exit the console, type "exit" and hit <enter>, or use Control-C or
Control-D.
`

	return strings.TrimSpace(text), nil
}
