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
	Breakpoints []dap.Breakpoint // Todo; map by source
	breakMap    map[int]dap.Breakpoint
	sess        *DebugSession
	activate    chan bool
	breakIndex  int

	// stateMu must be held when changing the state
	stateMu sync.Mutex

	Context *graph.DebugContext

	notFirst bool
}

func (e *Debugger) Work() {
	errgh := errgroup.Group{}
	errgh.Go(func() error {
		for range e.activate {
			e.Break()
		}
		return nil
	})

	errgh.Go(func() error {
		for run := range e.Context.RunCh {
			if e.breakIndex > len(e.Breakpoints)-1 {
				fmt.Println("No more breakpoints")
				continue
			}
			e.Break()
			e.sess.State["run"] = run.Name
			e.sess.State["state"] = e.extractStateMap(run)

			ctyjsonBytes, err := ctyjson.Marshal(run.Outputs, run.Outputs.Type())
			if err != nil {
				fmt.Println("error marshaling outputs:", err)
				continue
			}
			e.sess.State["outputs"] = string(ctyjsonBytes)

			//---------------------
			e.sess.State2.Run = run.Name
			err = json.Unmarshal(ctyjsonBytes, &e.sess.State2.Outputs)
			if err != nil {
				fmt.Println("error unmarshaling outputs:", err)
			}
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

func (e *Debugger) extractStateMap(run *moduletest.Run) map[string]any {
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
				instanceMap[keyStr] = string(instance.Current.AttrsJSON)
			}
			nestedMap[resource.Addr.String()] = instanceMap
		}
		mp[module.Addr.String()] = nestedMap
	}
	return mp
}

// func (e *Debugger) convertValueToDAPVariable(name string, value cadence.Value) dap.Variable {
// 	referenceHandle := 0
// 	switch value.(type) {
// 	case cadence.Dictionary, cadence.Array, cadence.Struct, cadence.Resource:
// 		referenceHandle = e.storeVariable(value)
// 	}
// 	return dap.Variable{
// 		Name:  name,
// 		Value: value.String(),
// 		Type:  value.Type().ID(),
// 		PresentationHint: &dap.VariablePresentationHint{
// 			Kind:       "property",
// 			Visibility: "private",
// 		},
// 		VariablesReference: referenceHandle,
// 	}
// }

// func (e *Debugger) storeVariable(value any) int {
// 	e.variableHandleCounter++
// 	e.variables[e.variableHandleCounter] = value
// 	return e.variableHandleCounter
// }
