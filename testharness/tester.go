package testharness

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/tfdiags"
	lua "github.com/yuin/gopher-lua"
)

// Test applies the given tester to the given subject and returns a checklist
// of results. A Spec is a tester.
//
// This function blocks until all of the tests have completed, and so it is
// not possible to receive progress logs. To run tests asynchronously with
// progress logs, use TestStream.
func Test(subject *Subject, tester Tester) Checklist {
	var ret Checklist
	itemCh := make(chan CheckItem)
	cs := NewCheckStream(itemCh, nil)
	TestStream(subject, tester, cs)
	for {
		item, more := <-itemCh
		if !more {
			break
		}
		ret = append(ret, item)
	}
	return ret
}

// TestStream is like Test except that it appends its results to the given
// CheckStream rather than returning a Checklist.
//
// This function returns immediately and then runs its tests in a separate
// goroutine. The CheckStream is closed once the tests are complete.
func TestStream(subject *Subject, tester Tester, cs CheckStream) {
	go func() {
		subCs, closed := cs.Substream()
		tester.test(subject, subCs)
		closed.Wait()
		cs.Close()
	}()
}

// Tester is an interface implemented by objects that can run tests.
type Tester interface {
	test(subject *Subject, cs CheckStream)
}

// Testers is a slice of Tester.
type Testers []Tester

// Tester implementation
func (ts Testers) test(subject *Subject, cs CheckStream) {
	for _, tester := range ts {
		subCs, closed := cs.Substream()
		tester.test(subject, subCs)
		closed.Wait()
	}
	cs.Close()
}

// describe represents a single "describe" call in a test specification.
//
// describe implements Tester.
type describe struct {
	Described contextSetter
	BodyFn    *lua.LFunction
	Context   *Context

	DefRange tfdiags.SourceRange
}

// Tester implementation
func (t *describe) test(subject *Subject, cs CheckStream) {
	defer cs.Close()

	childContexts, diags := t.Described.AppendContexts(t.Context, subject, nil)
	if diags.HasErrors() {
		cs.Write(CheckItem{
			Result:  Error,
			Caption: t.Context.Name(),
			Diags:   diags,
		})
		return
	}

Contexts:
	for _, childContext := range childContexts {
		L := childContext.lstate
		var diags tfdiags.Diagnostics

		var fn *lua.LFunction
		{
			// Copy our function so we can modify its environment
			// without affecting global state.
			fnV := *t.BodyFn
			fn = &fnV
		}

		topEnv := L.NewTable()
		L.SetFEnv(fn, topEnv)

		builderDiags := &Diagnostics{}
		testersB := testersBuilder{
			Context: childContext,
			Diags:   builderDiags,
		}
		for k, v := range testersB.luaTesterDecls(L) {
			topEnv.RawSet(k, v)
		}
		for k, v := range testersB.luaContextSetters(L) {
			topEnv.RawSet(k, v)
		}

		L.Push(fn)
		err := L.PCall(0, 0, nil)
		if err != nil {
			diags = diags.Append(err)
		}
		diags = diags.Append(builderDiags.Diags)

		if diags.HasErrors() {
			cs.Write(CheckItem{
				Result:  Error,
				Caption: childContext.Name(),
				Diags:   diags,
			})
			continue
		}

		for _, requirement := range testersB.Requirements {
			switch requirement.Result() {
			case Skipped:
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Requirement created without assertion",
					Detail:   "An assertion method must be called on the result of each \"require\" call.",
					Subject:  requirement.defRange.ToHCL().Ptr(),
				})
				cs.Write(CheckItem{
					Result:  Error,
					Caption: childContext.Name(),
					Diags:   diags,
				})
				continue Contexts
			case Error:
				diags = diags.Append(requirement.diags)
				cs.Write(CheckItem{
					Result:  Error,
					Caption: childContext.Name(),
					Diags:   diags,
				})
				continue Contexts
			case Failure:
				// TODO: Do something with the detail message from the requirement, if any.
				cs.Write(CheckItem{
					Result:  Skipped,
					Caption: childContext.Name(),
					Diags:   diags,
				})
				continue Contexts
			}
		}

		for _, tester := range testersB.Testers {
			subCs, close := cs.Substream()
			tester.test(subject, subCs)
			close.Wait()
		}
	}
}

// it represents a single "it" call in a test specification.
//
// it implements Tester.
type it struct {
	Does    string
	BodyFn  *lua.LFunction
	Context *Context

	DefRange tfdiags.SourceRange
}

// Tester implementation
func (t *it) test(subject *Subject, cs CheckStream) {
	var diags tfdiags.Diagnostics
	defer cs.Close()

	// TODO: implement
	cs.Write(CheckItem{
		Result:  Skipped,
		Caption: t.Context.NameWithSuffix(t.Does),
		Diags:   diags,
	})
}
