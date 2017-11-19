package testharness

import (
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

		if testersB.Skip {
			// testersB.Skip is set if there's a call to require() in
			// the body and the given condition didn't hold.
			cs.Write(CheckItem{
				Result:  Skipped,
				Caption: childContext.Name(),
				Diags:   diags,
			})
			continue
		}

		for _, tester := range testersB.Testers {
			subCs, close := cs.Substream()
			tester.test(subject, subCs)
			close.Wait()
		}
	}
}
