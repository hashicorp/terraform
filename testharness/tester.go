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
		// TODO: run our BodyFn to collect our child testers
		// TODO: run child testers, each in a sub-stream of cs
		// TODO: wait for sub-stream to close before returning
		cs.Write(CheckItem{
			Result:  Skipped,
			Caption: childContext.Name(),
		})
	}
}
