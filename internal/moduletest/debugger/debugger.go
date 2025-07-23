package debugger

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/go-dap"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/moduletest/graph"
	"github.com/hashicorp/terraform/internal/tfdiags"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"golang.org/x/sync/errgroup"
)

type Debugger struct {
	Breakpoints []dap.Breakpoint
	breakMap    map[int]dap.Breakpoint
	sess        *DebugSession
	breakIndex  int

	// stateMu must be held when changing the state
	stateMu sync.Mutex

	Context *graph.DebugContext

	notFirst bool
}

func (e *Debugger) Work() {
	errgh := errgroup.Group{}

	errgh.Go(func() error {
		for run := range e.Context.RunCh {
			// all breakpoints processed
			if e.breakIndex > len(e.Breakpoints)-1 {
				continue
			}

			e.Break()
			e.sess.DebugState.Run = run.Name
			e.sess.DebugState.State = e.runStateMap(run)

			ctyjsonBytes, err := ctyjson.Marshal(run.Outputs, run.Outputs.Type())
			if err != nil {
				fmt.Println("error marshaling outputs:", err)
				continue
			}
			var outputs map[string]any
			err = json.Unmarshal(ctyjsonBytes, &outputs)
			if err != nil {
				fmt.Println("error unmarshaling outputs:", err)
				continue
			}
			e.sess.DebugState.Outputs = outputs
		}
		return nil
	})

	err := errgh.Wait()
	if err != nil {
		fmt.Println("error waiting for debugger goroutines:", err)
	}
}

func (e *Debugger) addBreakPoint(b dap.Breakpoint) {
	if _, found := e.breakMap[b.Id]; found {
		return
	}

	e.breakMap[b.Id] = b
	e.Breakpoints = append(e.Breakpoints, b)
	e.Context.AddBreakpoint(b)
}

func (e *Debugger) Break() {
	e.breakIndex++ // Todo: Fix sync issues
	e.sess.WriteCh <- DapMsg{&dap.StoppedEvent{
		Event: newEvent("stopped"),
		Body: dap.StoppedEventBody{
			Reason:            "breakpoint",
			ThreadId:          singletonThreadID,
			Text:              "Paused on breakpoint",
			Description:       "Paused on breakpoint",
			AllThreadsStopped: true,
		},
	}}
}

func (e *Debugger) Resume() {
	e.Context.Resume()
}

func (e *Debugger) fileRuns() (runs map[int]hcl.Pos, diags tfdiags.Diagnostics) {
	runs = make(map[int]hcl.Pos)
	file := e.Context.ActiveEvalContext.File
	for _, run := range file.Runs {
		runs[run.Config.DeclRange.Start.Line] = run.Config.DeclRange.Start
	}

	return runs, diags
}

func (e *Debugger) runStateMap(run *moduletest.Run) map[string]any {
	mp := map[string]any{}
	for _, module := range e.Context.ActiveEvalContext.GetFileState(run.Config.StateKey).State.Modules {
		nestedMap := map[string]any{}
		for _, resource := range module.Resources {
			instanceMap := map[string]any{}
			for key, instance := range resource.Instances {
				keyStr := ""
				if key != addrs.NoKey {
					keyStr = key.String()
				}
				var attrs map[string]any
				if err := json.Unmarshal(instance.Current.AttrsJSON, &attrs); err != nil {
					instanceMap[keyStr] = string(instance.Current.AttrsJSON)
				} else {
					instanceMap[keyStr] = attrs
				}
			}
			nestedMap[resource.Addr.String()] = instanceMap
		}
		mp[module.Addr.String()] = nestedMap
	}
	return mp
}
