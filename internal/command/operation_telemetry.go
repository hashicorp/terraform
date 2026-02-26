// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	commandRunSpanAttrCommandName = "terraform.command.name"

	commandRunEventUserInterrupt = "user interrupt"
	commandRunEventStopRequested = "graceful stop requested"
	commandRunEventForcedCancel  = "forced cancel requested"
)

type commandRunSpanStateKey struct{}

type commandRunSpanState struct {
	span         trace.Span
	interrupted  bool
	forcedCancel bool
}

type commandRunSpanHandle struct {
	meta        *Meta
	prevContext context.Context
	state       *commandRunSpanState
}

func (m *Meta) beginCommandRunSpan(commandName string) *commandRunSpanHandle {
	parentCtx := m.CommandContext()
	spanCtx, span := tracer.Start(parentCtx, commandRunSpanName(commandName), trace.WithAttributes(
		attribute.String(commandRunSpanAttrCommandName, commandName),
	))

	state := &commandRunSpanState{
		span: span,
	}

	spanCtx = context.WithValue(spanCtx, commandRunSpanStateKey{}, state)

	handle := &commandRunSpanHandle{
		meta:        m,
		prevContext: m.CallerContext,
		state:       state,
	}
	m.CallerContext = spanCtx
	return handle
}

func (h *commandRunSpanHandle) end(op *backendrun.RunningOperation, err error) {
	if h == nil {
		return
	}

	defer func() {
		if h.meta != nil {
			h.meta.CallerContext = h.prevContext
		}
		h.state.span.End()
	}()

	switch {
	case h.state.forcedCancel:
		h.state.span.SetStatus(codes.Error, "user cancel")
	case h.state.interrupted:
		h.state.span.SetStatus(codes.Error, "user interrupt")
	case err != nil:
		h.state.span.SetStatus(codes.Error, "command error")
	case op != nil && op.Result != backendrun.OperationSuccess:
		h.state.span.SetStatus(codes.Error, "operation unsuccessful")
	}
}

func commandRunSpanName(commandName string) string {
	return fmt.Sprintf("command run (%s)", commandName)
}

func recordCommandRunInterrupt(ctx context.Context) {
	state := commandRunSpanStateFromContext(ctx)
	if state == nil {
		return
	}
	state.interrupted = true
	state.span.AddEvent(commandRunEventUserInterrupt)
}

func recordCommandRunStopRequested(ctx context.Context) {
	state := commandRunSpanStateFromContext(ctx)
	if state == nil {
		return
	}
	state.span.AddEvent(commandRunEventStopRequested)
}

func recordCommandRunForcedCancel(ctx context.Context) {
	state := commandRunSpanStateFromContext(ctx)
	if state == nil {
		return
	}
	state.interrupted = true
	state.forcedCancel = true
	state.span.AddEvent(commandRunEventForcedCancel)
}

func commandRunSpanStateFromContext(ctx context.Context) *commandRunSpanState {
	if ctx == nil {
		return nil
	}
	state, _ := ctx.Value(commandRunSpanStateKey{}).(*commandRunSpanState)
	return state
}
