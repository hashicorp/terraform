package debug

import (
	"context"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

// Interface serves as the bridge between Terraform Core and the debugger,
// with methods that mark different events that the debugger needs to know
// about and react to.
type Interface interface {
	// BeginDebuggable marks the event of beginning work on a particular
	// debuggable object. If there is a breakpoint on that object, or if
	// the debugger is operating in single-step mode, then this method
	// will block until the corresponding operation ought to continue.
	//
	// BeginDebuggable will also return if the given context becomes "done",
	// in the hope of then allowing the language runtime to then respect
	// the cancellation/deadline.
	//
	// The debugger will use the given RuntimeContext only until BeginDebuggable
	// returns, to service any debugger-user requests made while the debugger
	// is halted at this particular event.
	BeginDebuggable(ctx context.Context, addr addrs.Debuggable, runtime RuntimeContext)

	// EndDebuggable marks the event of completing work on a particular
	// debuggable object. If there is a breakpoint on that object, or if
	// the debugger is operating in single-step mode, then this method
	// will block until the corresponding operation ought to continue.
	//
	// EndDebuggable will also return if the given context becomes "done",
	// in the hope of then allowing the language runtime to then respect
	// the cancellation/deadline.
	//
	// The debugger will use the given RuntimeContext only until EndDebuggable
	// returns, to service any debugger-user requests made while the debugger
	// is halted at this particular event.
	EndDebuggable(ctx context.Context, addr addrs.Debuggable, runtime RuntimeContext)

	// Failure reports that the debuggable object represented by the given
	// address has encountered an error condition.
	//
	// If the debugger is configured to break on error then this method
	// will block until the debugger user resumes execution.
	//
	// Failure will also return if the given context becomes "done",
	// in the hope of then allowing the language runtime to then respect
	// the cancellation/deadline.
	//
	// The given diagnostics should contain at least one error, but may also
	// optionally contain non-error diagnostics alongside it.
	//
	// The debugger will use the given RuntimeContext only until Failure
	// returns, to service any debugger-user requests made while the debugger
	// is halted at this particular event.
	Failure(ctx context.Context, addr addrs.Debuggable, runtime RuntimeContext, diags tfdiags.Diagnostics)
}

// mainInterface is the implementation of Interface we use with a real
// debugging session. Its methods just proxy through to the enclosed
// Debugger object.
type mainInterface struct {
	d *Debugger
}

func (i mainInterface) BeginDebuggable(ctx context.Context, addr addrs.Debuggable, runtime RuntimeContext) {
	i.d.beginDebuggable(ctx, addr, runtime)
}

func (i mainInterface) EndDebuggable(ctx context.Context, addr addrs.Debuggable, runtime RuntimeContext) {
	i.d.endDebuggable(ctx, addr, runtime)
}

func (i mainInterface) Failure(ctx context.Context, addr addrs.Debuggable, runtime RuntimeContext, diags tfdiags.Diagnostics) {
	i.d.failure(ctx, addr, runtime, diags)
}

// noOpInterface is an alternative implementation of Interface that does
// nothing at all, returning immediately from all events.
type noOpInterface struct {
}

// NewNoOpInterface returns an implementation of Interface that does nothing
// at all, always just allowing execution to immediately proceed after any
// event that may be marked.
func NewNoOpInterface() Interface {
	return noOpInterface{}
}

func (i noOpInterface) BeginDebuggable(ctx context.Context, addr addrs.Debuggable, runtime RuntimeContext) {
}

func (i noOpInterface) EndDebuggable(ctx context.Context, addr addrs.Debuggable, runtime RuntimeContext) {
}

func (i noOpInterface) Failure(ctx context.Context, addr addrs.Debuggable, runtime RuntimeContext, diags tfdiags.Diagnostics) {
}
