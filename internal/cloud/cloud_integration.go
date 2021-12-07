package cloud

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/backend"
)

type IntegrationOutputWriter interface {
	End()
	OutputElapsed(message string, maxMessage int)
	Output(str string)
	SubOutput(str string)
}

type IntegrationContext struct {
	started       time.Time
	B             *Cloud
	StopContext   context.Context
	CancelContext context.Context
	Op            *backend.Operation
	Run           *tfe.Run
}

type integrationCLIOutput struct {
	ctx *IntegrationContext
}

var _ IntegrationOutputWriter = (*integrationCLIOutput)(nil) // Compile time check

func NewIntegrationContext(stopCtx, cancelCtx context.Context, b *Cloud, op *backend.Operation, r *tfe.Run) *IntegrationContext {
	return &IntegrationContext{
		B:             b,
		StopContext:   stopCtx,
		CancelContext: cancelCtx,
		Op:            op,
		Run:           r,
	}
}

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

func (s *IntegrationContext) BeginOutput(name string) IntegrationOutputWriter {
	var result IntegrationOutputWriter = &integrationCLIOutput{
		ctx: s,
	}

	s.started = time.Now()

	if s.HasCLI() {
		s.B.CLI.Output("\n------------------------------------------------------------------------\n")
	}

	result.Output("[bold]" + name + ":\n")

	return result
}

func (s *IntegrationContext) HasCLI() bool {
	return s.B.CLI != nil
}

func (s *integrationCLIOutput) End() {
	if !s.ctx.HasCLI() {
		return
	}

	s.ctx.B.CLI.Output("\n------------------------------------------------------------------------\n")
}

func (s *integrationCLIOutput) Output(str string) {
	if !s.ctx.HasCLI() {
		return
	}
	s.ctx.B.CLI.Output(s.ctx.B.Colorize().Color(str))
}

func (s *integrationCLIOutput) SubOutput(str string) {
	if !s.ctx.HasCLI() {
		return
	}
	s.ctx.B.CLI.Output(s.ctx.B.Colorize().Color(fmt.Sprintf("[reset]│ %s", str)))
}

// Example pending output; the variable spacing (50 chars) allows up to 99 tasks (two digits) in each category:
// ---------------
// 13 tasks still pending, 0 passed, 0 failed ...
// 13 tasks still pending, 0 passed, 0 failed ...       (8s elapsed)
// 13 tasks still pending, 0 passed, 0 failed ...       (19s elapsed)
// 13 tasks still pending, 0 passed, 0 failed ...       (33s elapsed)
func (s *integrationCLIOutput) OutputElapsed(message string, maxMessage int) {
	if !s.ctx.HasCLI() {
		return
	}
	elapsed := time.Since(s.ctx.started).Truncate(1 * time.Second)
	s.ctx.B.CLI.Output(fmt.Sprintf("%-"+strconv.FormatInt(int64(maxMessage), 10)+"s", message) + s.ctx.B.Colorize().Color(fmt.Sprintf("[dim](%s elapsed)", elapsed)))
}
