package cloud

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/mitchellh/cli"
)

// IntegrationOutputWriter is an interface used to to write output tailored for
// Terraform Cloud integrations
type IntegrationOutputWriter interface {
	End()
	OutputElapsed(message string, maxMessage int)
	Output(str string)
	SubOutput(str string)
}

// IntegrationContext is a set of data that is useful when performing Terraform Cloud integration operations
type IntegrationContext struct {
	B             *Cloud
	StopContext   context.Context
	CancelContext context.Context
	Op            *backend.Operation
	Run           *tfe.Run
}

// integrationCLIOutput implements IntegrationOutputWriter
type integrationCLIOutput struct {
	CLI       cli.Ui
	Colorizer Colorer
	started   time.Time
}

var _ IntegrationOutputWriter = (*integrationCLIOutput)(nil) // Compile time check

func (s *IntegrationContext) Poll(every func(i int) (bool, error)) error {
	for i := 0; ; i++ {
		select {
		case <-s.StopContext.Done():
			return s.StopContext.Err()
		case <-s.CancelContext.Done():
			return s.CancelContext.Err()
		case <-time.After(backoff(backoffMin, backoffMax, i)):
			// blocks for a time between min and max
		}

		cont, err := every(i)
		if !cont {
			return err
		}
	}
}

// BeginOutput writes a preamble to the CLI and creates a new IntegrationOutputWriter interface
// to write the remaining CLI output to. Use IntegrationOutputWriter.End() to complete integration
// output
func (s *IntegrationContext) BeginOutput(name string) IntegrationOutputWriter {
	var result IntegrationOutputWriter = &integrationCLIOutput{
		CLI:       s.B.CLI,
		Colorizer: s.B.Colorize(),
		started:   time.Now(),
	}

	result.Output("\n[bold]" + name + ":\n")

	return result
}

// End writes the termination output for the integration
func (s *integrationCLIOutput) End() {
	if s.CLI == nil {
		return
	}

	s.CLI.Output("\n------------------------------------------------------------------------\n")
}

// Output writes a string after colorizing it using any [colorstrings](https://github.com/mitchellh/colorstring) it contains
func (s *integrationCLIOutput) Output(str string) {
	if s.CLI == nil {
		return
	}
	s.CLI.Output(s.Colorizer.Color(str))
}

// SubOutput writes a string prefixed by a "│ " after colorizing it using any [colorstrings](https://github.com/mitchellh/colorstring) it contains
func (s *integrationCLIOutput) SubOutput(str string) {
	if s.CLI == nil {
		return
	}
	s.CLI.Output(s.Colorizer.Color(fmt.Sprintf("[reset]│ %s", str)))
}

// OutputElapsed writes a string followed by the amount of time that has elapsed since calling BeginOutput.
// Example pending output; the variable spacing (50 chars) allows up to 99 tasks (two digits) in each category:
// ---------------
// 13 tasks still pending, 0 passed, 0 failed ...
// 13 tasks still pending, 0 passed, 0 failed ...       (8s elapsed)
// 13 tasks still pending, 0 passed, 0 failed ...       (19s elapsed)
// 13 tasks still pending, 0 passed, 0 failed ...       (33s elapsed)
func (s *integrationCLIOutput) OutputElapsed(message string, maxMessage int) {
	if s.CLI == nil {
		return
	}
	elapsed := time.Since(s.started).Truncate(1 * time.Second)
	s.CLI.Output(fmt.Sprintf("%-"+strconv.FormatInt(int64(maxMessage), 10)+"s", message) + s.Colorizer.Color(fmt.Sprintf("[dim](%s elapsed)", elapsed)))
}
