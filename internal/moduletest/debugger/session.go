package debugger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-dap"
	"github.com/hashicorp/terraform/internal/moduletest/graph"
)

var (
	singletonThreadID = 1
)

type DebugState struct {
	Run     string
	State   map[string]any
	Outputs map[string]any
}

type DebugSession struct {
	RootDir         string // root directory where the terraform test debug server is running
	Source          string
	Step            int
	Debugger        *Debugger
	WriteCh         chan DapMsg
	LaunchArgs      *LaunchArgs
	Conn            io.ReadWriteCloser
	DebugState      DebugState
	VariablesStore  map[int]any
	variableCounter int

	currentFrame int

	Context *graph.DebugContext
}

func NewSession(ctx *graph.DebugContext) *DebugSession {
	return &DebugSession{
		Step:           1,
		Context:        ctx,
		VariablesStore: make(map[int]any),
		WriteCh:        make(chan DapMsg),
	}
}

// storeVariable stores a composite variable (map or slice) and returns a reference id for it.
func (h *DebugSession) storeVariable(value any) int {
	h.variableCounter++
	if h.VariablesStore == nil {
		h.VariablesStore = make(map[int]any)
	}
	h.VariablesStore[h.variableCounter] = value
	return h.variableCounter
}

func (h *DebugSession) Initialize(ctx context.Context, req *dap.InitializeRequest) (*dap.InitializeResponse, error) {
	resp := &dap.InitializeResponse{}

	// Set capabilities.
	resp.Body.SupportTerminateDebuggee = true
	resp.Body.SupportsConfigurationDoneRequest = true
	resp.Body.SupportsFunctionBreakpoints = true
	resp.Body.SupportsConditionalBreakpoints = true
	resp.Body.SupportsHitConditionalBreakpoints = true
	// resp.Body.SupportsEvaluateForHovers = true
	// resp.Body.SupportsStepBack = true
	resp.Body.SupportsSetVariable = true
	resp.Body.SupportsRestartFrame = true
	// resp.Body.SupportsGotoTargetsRequest = true
	// resp.Body.SupportsStepInTargetsRequest = true
	resp.Body.SupportsCompletionsRequest = true
	resp.Body.SupportsModulesRequest = true
	resp.Body.SupportsRestartRequest = true
	resp.Body.SupportsExceptionOptions = true
	// resp.Body.SupportsValueFormattingOptions = true
	resp.Body.SupportsExceptionInfoRequest = true
	// resp.Body.SupportsDelayedStackTraceLoading = true
	resp.Body.SupportsLoadedSourcesRequest = true
	resp.Body.SupportsLogPoints = true
	resp.Body.SupportsTerminateThreadsRequest = true
	resp.Body.SupportsSetExpression = true
	resp.Body.SupportsTerminateRequest = true
	resp.Body.SupportsDataBreakpoints = true
	// resp.Body.SupportsReadMemoryRequest = true
	// resp.Body.SupportsWriteMemoryRequest = true
	// resp.Body.SupportsDisassembleRequest = true
	resp.Body.SupportsCancelRequest = true
	resp.Body.SupportsBreakpointLocationsRequest = true
	// resp.Body.SupportsClipboardContext = true
	// resp.Body.SupportsSteppingGranularity = true
	resp.Body.SupportsInstructionBreakpoints = true
	resp.Body.SupportsExceptionFilterOptions = true
	// resp.Body.SupportsSingleThreadExecutionRequests = true

	h.send(req, resp)
	time.Sleep(1000 * time.Millisecond)
	h.sendEvent(&dap.InitializedEvent{Event: newEvent("initialized")})
	return resp, nil
}

type LaunchArgs struct {
	DebugServer int    `json:"debugServer"`
	Name        string `json:"name"`
	Program     string `json:"program"`
	Request     string
	Type        string
}

func (h *DebugSession) Launch(ctx context.Context, req *dap.LaunchRequest) (*dap.LaunchResponse, error) {
	args := LaunchArgs{}
	if err := json.Unmarshal(req.Arguments, &args); err != nil {
		fmt.Println(err, string(req.Arguments))
		return nil, err
	}

	h.LaunchArgs = &args
	if !strings.HasPrefix(args.Program, h.RootDir) {
		return nil, fmt.Errorf("program not in debug server directory: %s", h.RootDir)
	}
	h.Source = args.Program
	h.variableCounter = 1
	h.currentFrame = 1
	h.Debugger = &Debugger{
		Breakpoints: make([]dap.Breakpoint, 0),
		sess:        h,
		breakMap:    make(map[int]dap.Breakpoint),
		stateMu:     sync.Mutex{},
		Context:     h.Context,
	}

	// start a goroutine that prepares to receive debug commands
	go h.Debugger.Work()
	resp := &dap.LaunchResponse{}
	h.send(req, resp)
	go func() {
		// wait a bit for the debug client to receive the launch response
		time.Sleep(4 * time.Second)

		// resume until the first breakpoint
		h.Debugger.Resume()
	}()
	return resp, nil
}

func (h *DebugSession) Disconnect(ctx context.Context, req *dap.DisconnectRequest) (*dap.DisconnectResponse, error) {
	resp := &dap.DisconnectResponse{}
	if h.WriteCh != nil {
		h.send(req, resp)
	}
	return resp, nil
}

func (h *DebugSession) Terminate(ctx context.Context, req *dap.TerminateRequest) (*dap.TerminateResponse, error) {
	resp := &dap.TerminateResponse{}
	h.send(req, resp)
	close(h.WriteCh)
	return resp, nil
}

func (h *DebugSession) SetBreakpoints(ctx context.Context, req *dap.SetBreakpointsRequest) (*dap.SetBreakpointsResponse, error) {
	// h.Debugger.stateMu.Lock()
	// defer h.Debugger.stateMu.Unlock()
	resp := &dap.SetBreakpointsResponse{}

	if !strings.HasSuffix(h.Source, req.Arguments.Source.Name) {
		h.send(req, resp)
		return resp, nil
	}
	runs, diags := h.Debugger.fileRuns()
	if diags.HasErrors() {
		return resp, diags.Err()
	}

	bps := make([]dap.Breakpoint, 0)
	for _, sourceBp := range req.Arguments.Breakpoints {
		bp := dap.Breakpoint{
			Id:        sourceBp.Line,
			Line:      sourceBp.Line,
			EndLine:   sourceBp.Line + 1,
			Column:    sourceBp.Column,
			EndColumn: sourceBp.Column + 1,
			Source:    &req.Arguments.Source,
			Offset:    sourceBp.Line,
		}
		if _, foundLine := runs[sourceBp.Line]; foundLine {
			bp.Verified = true
			bps = append(bps, bp)
		}
		resp.Body.Breakpoints = append(resp.Body.Breakpoints, bp)
	}
	go func() {
		for _, bp := range bps {
			h.Step = bps[0].Line
			h.Debugger.addBreakPoint(bp)
		}

	}()
	h.send(req, resp)
	return resp, nil
}

func (h *DebugSession) BreakpointLocations(ctx context.Context, req *dap.BreakpointLocationsRequest) (*dap.BreakpointLocationsResponse, error) {
	resp := &dap.BreakpointLocationsResponse{}
	// TODO: Go through the code-written breakpoints and add them to the response
	resp.Body.Breakpoints = make([]dap.BreakpointLocation, 0)
	for _, b := range h.Debugger.Breakpoints {
		resp.Body.Breakpoints = append(resp.Body.Breakpoints, dap.BreakpointLocation{
			Line:      b.Line,
			EndLine:   b.EndLine,
			Column:    b.Column,
			EndColumn: b.EndColumn,
		})
	}
	h.send(req, resp)
	return resp, nil
}

func (h *DebugSession) Threads(ctx context.Context, req *dap.ThreadsRequest) (*dap.ThreadsResponse, error) {
	resp := &dap.ThreadsResponse{
		Body: dap.ThreadsResponseBody{
			Threads: []dap.Thread{
				{
					Id:   singletonThreadID,
					Name: "main",
				},
			},
		},
	}
	h.send(req, resp)
	return resp, nil
}

// This request indicates that the client has finished initialization of the debug adapter.
// So it is the last request in the sequence of configuration requests (which was started by the initialized event).
func (h *DebugSession) ConfigurationDone(ctx context.Context, req *dap.ConfigurationDoneRequest) (*dap.ConfigurationDoneResponse, error) {
	resp := &dap.ConfigurationDoneResponse{}
	h.send(req, resp)
	return resp, nil
}

func (h *DebugSession) StackTrace(ctx context.Context, req *dap.StackTraceRequest) (*dap.StackTraceResponse, error) {
	resp := &dap.StackTraceResponse{
		Body: dap.StackTraceResponseBody{
			TotalFrames: 1,
			StackFrames: []dap.StackFrame{
				{
					Id:     h.currentFrame,
					Source: &dap.Source{Name: path.Base(h.Source), Path: h.Source, SourceReference: 0},
					Line:   h.Step,
					Column: 0,
					Name:   "main.main",
				},
			},
		},
	}

	// New frame!. Increment the frame counter to be after all current variables
	h.currentFrame = h.currentFrame + h.variableCounter
	h.variableCounter = h.currentFrame
	h.send(req, resp)
	return resp, nil
}

func (h *DebugSession) Next(ctx context.Context, req *dap.NextRequest) (*dap.NextResponse, error) {
	h.Step++
	for _, breakpoint := range h.Debugger.Breakpoints {
		if breakpoint.Line == h.Step {
			h.Debugger.Resume()
			break
		}
	}
	h.WriteCh <- DapMsg{&dap.StoppedEvent{
		Event: newEvent("stopped"),
		Body: dap.StoppedEventBody{
			Reason:            "step",
			ThreadId:          singletonThreadID,
			AllThreadsStopped: true,
		},
	}}
	resp := &dap.NextResponse{}
	h.send(req, resp)
	return resp, nil
}

func (h *DebugSession) Continue(ctx context.Context, req *dap.ContinueRequest) (*dap.ContinueResponse, error) {
	// process until next breakpoint
	h.Debugger.Resume()

	resp := &dap.ContinueResponse{}
	resp.Body.AllThreadsContinued = true
	if h.Debugger.breakIndex > len(h.Debugger.Breakpoints)-1 {
		h.WriteCh <- DapMsg{&dap.ContinuedEvent{
			Event: newEvent("continued"),
			Body: dap.ContinuedEventBody{
				ThreadId:            singletonThreadID,
				AllThreadsContinued: true,
			},
		}}
		h.send(req, resp)
		return resp, nil
	}
	next := h.Debugger.Breakpoints[h.Debugger.breakIndex]
	h.Step = next.Line
	h.send(req, resp)
	return resp, nil
}

func (h *DebugSession) Evaluate(ctx context.Context, req *dap.EvaluateRequest) (*dap.EvaluateResponse, error) {
	resp := &dap.EvaluateResponse{}
	h.send(req, resp)
	return resp, nil
}

func (h *DebugSession) SourceReq(ctx context.Context, req *dap.SourceRequest) (*dap.SourceResponse, error) {
	resp := &dap.SourceResponse{}
	h.send(req, resp)
	return resp, nil
}

func (h *DebugSession) Variables(ctx context.Context, req *dap.VariablesRequest) (*dap.VariablesResponse, error) {
	var vars []dap.Variable
	// If the reference is the first variable in this frame, return Top-level variables: Run, State, Outputs from DebugState
	if req.Arguments.VariablesReference == h.currentFrame {
		vars = append(vars, dap.Variable{
			Name:               "Run",
			Value:              h.DebugState.Run,
			VariablesReference: 0,
		})
		stateRef := h.storeVariable(h.DebugState.State)

		vars = append(vars, dap.Variable{
			Name:               "State",
			Value:              spew.Sprintf("<object>: %v", h.DebugState.State),
			VariablesReference: stateRef,
		})
		outputsRef := h.storeVariable(h.DebugState.Outputs)
		vars = append(vars, dap.Variable{
			Name:               "Outputs",
			Value:              spew.Sprintf("<object>: %v", h.DebugState.Outputs),
			VariablesReference: outputsRef,
		})
	} else {
		composite, found := h.VariablesStore[req.Arguments.VariablesReference]
		if !found {
			return nil, fmt.Errorf("no variables found for reference %d", req.Arguments.VariablesReference)
		}
		switch comp := composite.(type) {
		case map[string]any:
			for key, val := range comp {
				switch v := val.(type) {
				case map[string]any, []any:
					ref := h.storeVariable(v)
					vars = append(vars, dap.Variable{
						Name:               key,
						Value:              spew.Sprintf("<object>: %v", v),
						VariablesReference: ref,
					})
				default:
					vars = append(vars, dap.Variable{
						Name:               key,
						Value:              fmt.Sprintf("%v", v),
						VariablesReference: 0,
					})
				}
			}
		case []any:
			for idx, val := range comp {
				switch v := val.(type) {
				case map[string]any, []any:
					ref := h.storeVariable(v)
					vars = append(vars, dap.Variable{
						Name:               fmt.Sprintf("[%d]", idx),
						Value:              spew.Sprintf("<object>: %v", v),
						VariablesReference: ref,
					})
				default:
					vars = append(vars, dap.Variable{
						Name:               fmt.Sprintf("[%d]", idx),
						Value:              fmt.Sprintf("%v", v),
						VariablesReference: 0,
					})
				}
			}
		default:
			return nil, fmt.Errorf("variable reference %d is not composite", req.Arguments.VariablesReference)
		}
	}
	resp := &dap.VariablesResponse{
		Body: dap.VariablesResponseBody{Variables: vars},
	}
	h.send(req, resp)
	return resp, nil
}

func (h *DebugSession) Scopes(ctx context.Context, req *dap.ScopesRequest) (*dap.ScopesResponse, error) {
	resp := &dap.ScopesResponse{}
	resp.Body = dap.ScopesResponseBody{
		Scopes: []dap.Scope{
			{Name: "Local", VariablesReference: h.variableCounter},
		},
	}
	h.send(req, resp)
	return resp, nil
}

func (h *DebugSession) send(req dap.RequestMessage, msg dap.ResponseMessage) {
	h.WriteCh <- DapMsg{newResponse(req, msg, nil)}
}
func (h *DebugSession) sendEvent(msg dap.EventMessage) {
	h.WriteCh <- DapMsg{msg}
}

func newResponse(req dap.RequestMessage, msg dap.ResponseMessage, err error) dap.ResponseMessage {
	if err != nil {
		msg = &dap.Response{}
		msg.GetResponse().Message = err.Error()
	}
	msg.GetResponse().RequestSeq = req.GetSeq()
	msg.GetResponse().Command = req.GetRequest().Command
	msg.GetResponse().Success = err == nil
	return msg
}

func newEvent(evt string) dap.Event {
	return dap.Event{
		ProtocolMessage: dap.ProtocolMessage{
			Type: "event",
		},
		Event: evt,
	}
}
