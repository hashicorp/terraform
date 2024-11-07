// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"

	_ "github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// This file contains some general test utilities that many of our other
// _test.go files rely on. It doesn't actually contain any tests itself.

// testStackConfig loads a stack configuration from the source bundle in this
// package's testdata directory.
//
// "collection" is the name of one of the synthetic source packages that's
// declared in the source bundle, and "subPath" is the path within that
// package.
func testStackConfig(t *testing.T, collection string, subPath string) *stackconfig.Config {
	t.Helper()

	// Our collection of test configurations is laid out like a source
	// bundle that was installed from some source addresses that don't
	// really exist, and so we'll construct a suitable fake source
	// address following that scheme.
	fakeSrcStr := fmt.Sprintf("https://testing.invalid/%s.tar.gz//%s", collection, subPath)
	fakeSrc, err := sourceaddrs.ParseRemoteSource(fakeSrcStr)
	if err != nil {
		t.Fatalf("artificial source address string %q is invalid: %s", fakeSrcStr, err)
	}

	sources := testSourceBundle(t)
	ret, diags := stackconfig.LoadConfigDir(fakeSrc, sources)
	if diags.HasErrors() {
		diags.Sort()
		t.Fatalf("configuration is invalid\n%s", testFormatDiagnostics(t, diags))
	}
	return ret
}

func testStackConfigEmpty(t *testing.T) *stackconfig.Config {
	t.Helper()
	sources := testSourceBundle(t)
	fakeAddr := sourceaddrs.MustParseSource("https://testing.invalid/nonexist.tar.gz").(sourceaddrs.RemoteSource)
	return stackconfig.NewEmptyConfig(fakeAddr, sources)
}

func testSourceBundle(t *testing.T) *sourcebundle.Bundle {
	t.Helper()
	sources, err := sourcebundle.OpenDir("testdata/sourcebundle")
	if err != nil {
		t.Fatalf("cannot open source bundle: %s", err)
	}
	return sources
}

func testPriorState(t *testing.T, msgs map[string]protoreflect.ProtoMessage) *stackstate.State {
	t.Helper()
	ret, err := stackstate.LoadFromDirectProto(msgs)
	if err != nil {
		t.Fatal(err)
	}
	return ret
}

func testPlan(t *testing.T, main *Main) (*stackplan.Plan, tfdiags.Diagnostics) {
	t.Helper()
	outp, outpTest := testPlanOutput(t)
	main.PlanAll(context.Background(), outp)
	return outpTest.Close(t)
}

func testPlanOutput(t *testing.T) (PlanOutput, *planOutputTester) {
	t.Helper()
	tester := &planOutputTester{}
	outp := PlanOutput{
		AnnouncePlannedChange: func(ctx context.Context, pc stackplan.PlannedChange) {
			tester.mu.Lock()
			tester.planned = append(tester.planned, pc)
			tester.mu.Unlock()
		},
		AnnounceDiagnostics: func(ctx context.Context, d tfdiags.Diagnostics) {
			tester.mu.Lock()
			tester.diags = tester.diags.Append(d)
			tester.mu.Unlock()
		},
	}
	return outp, tester
}

type planOutputTester struct {
	planned []stackplan.PlannedChange
	diags   tfdiags.Diagnostics
	mu      sync.Mutex
}

// PlannedChanges returns the planned changes that have been accumulated in the
// receiver.
//
// It isn't safe to access the returned slice concurrently with a planning
// operation. Use this method only once the plan operation is complete and
// thus the changes are finalized.
func (pot *planOutputTester) PlannedChanges() []stackplan.PlannedChange {
	return pot.planned
}

// RawChanges returns the protobuf representation changes that have been
// accumulated in the receiver.
//
// It isn't safe to call this method concurrently with a planning
// operation. Use this method only once the plan operation is complete and
// thus the raw changes are finalized.
func (pot *planOutputTester) RawChanges(t *testing.T) []*anypb.Any {
	t.Helper()

	var msgs []*anypb.Any
	for _, change := range pot.planned {
		protoChange, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatalf("failed to encode %T: %s", change, err)
		}
		msgs = append(msgs, protoChange.Raw...)
	}

	// Normally it's the stackeval caller (in stackruntime) that marks a
	// plan as "applyable", but since we're calling into the stackeval functions
	// directly here we'll need to add that extra item ourselves.
	if !pot.diags.HasErrors() {
		change := stackplan.PlannedChangeApplyable{
			Applyable: true,
		}
		protoChange, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatalf("failed to encode %T: %s", change, err)
		}
		msgs = append(msgs, protoChange.Raw...)
	}

	return msgs
}

// Diags returns the diagnostics that have been accumulated in the
// receiver.
//
// It isn't safe to access the returned slice concurrently with a planning
// operation. Use this method only once the plan operation is complete and
// thus the diagnostics are finalized.
func (pot *planOutputTester) Diags() tfdiags.Diagnostics {
	return pot.diags
}

func (pot *planOutputTester) Close(t *testing.T) (*stackplan.Plan, tfdiags.Diagnostics) {
	t.Helper()

	// Caller shouldn't close concurrently with other work anyway, but we'll
	// include this just to help make things behave more consistently even when
	// the caller is buggy.
	pot.mu.Lock()
	defer pot.mu.Unlock()

	// We'll now round-trip all of the planned changes through the serialize
	// and deserialize logic to approximate the effect of this plan having been
	// saved and then reloaded during a subsequent apply phase, since
	// the reloaded plan is a more convenient artifact to inspect in tests.
	msgs := pot.RawChanges(t)
	plan, err := stackplan.LoadFromProto(msgs)
	if err != nil {
		t.Fatalf("failed to reload saved plan: %s", err)
	}
	return plan, pot.diags
}

func testApplyOutput(t *testing.T, priorStateRaw map[string]*anypb.Any) (ApplyOutput, *applyOutputTester) {
	t.Helper()
	tester := &applyOutputTester{}
	outp := ApplyOutput{
		AnnounceAppliedChange: func(ctx context.Context, ac stackstate.AppliedChange) {
			tester.mu.Lock()
			tester.applied = append(tester.applied, ac)
			tester.mu.Unlock()
		},
		AnnounceDiagnostics: func(ctx context.Context, d tfdiags.Diagnostics) {
			tester.mu.Lock()
			tester.diags = tester.diags.Append(d)
			tester.mu.Unlock()
		},
	}
	return outp, tester
}

type applyOutputTester struct {
	prior   map[string]*anypb.Any
	applied []stackstate.AppliedChange
	diags   tfdiags.Diagnostics
	mu      sync.Mutex
}

// AppliedChanges returns the applied change objects that have been accumulated
// in the receiver.
//
// It isn't safe to access the returned slice concurrently with an apply
// operation. Use this method only once the apply operation is complete and
// thus the changes are finalized.
func (aot *applyOutputTester) AppliedChanges() []stackstate.AppliedChange {
	return aot.applied
}

// RawUpdatedState returns the protobuf representation of the state with the
// accumulated changes merged into it.
//
// It isn't safe to call this method concurrently with an apply
// operation. Use this method only once the apply operation is complete and
// thus the changes are finalized.
func (aot *applyOutputTester) RawUpdatedState(t *testing.T) map[string]*anypb.Any {
	t.Helper()

	msgs := make(map[string]*anypb.Any)
	for k, v := range aot.prior {
		msgs[k] = v
	}
	for _, change := range aot.applied {
		protoChange, err := change.AppliedChangeProto()
		if err != nil {
			t.Fatalf("failed to encode %T: %s", change, err)
		}
		for _, protoRaw := range protoChange.Raw {
			if protoRaw.Value != nil {
				msgs[protoRaw.Key] = protoRaw.Value
			} else {
				delete(msgs, protoRaw.Key)
			}
		}
	}

	return msgs
}

// Diags returns the diagnostics that have been accumulated in the
// receiver.
//
// It isn't safe to access the returned slice concurrently with an apply
// operation. Use this method only once the apply operation is complete and
// thus the diagnostics are finalized.
func (aot *applyOutputTester) Diags() tfdiags.Diagnostics {
	return aot.diags
}

func (aot *applyOutputTester) Close(t *testing.T) (*stackstate.State, tfdiags.Diagnostics) {
	t.Helper()

	// Caller shouldn't close concurrently with other work anyway, but we'll
	// include this just to help make things behave more consistently even when
	// the caller is buggy.
	aot.mu.Lock()
	defer aot.mu.Unlock()

	// We'll now round-trip all of the applied changes through the serialize
	// and deserialize logic to approximate the effect of this having having been
	// saved and then reloaded during a subsequent planning phase.
	msgs := aot.RawUpdatedState(t)
	state, err := stackstate.LoadFromProto(msgs)
	if err != nil {
		t.Fatalf("failed to reload saved state: %s", err)
	}
	return state, aot.diags
}

func testFormatDiagnostics(t *testing.T, diags tfdiags.Diagnostics) string {
	t.Helper()
	var buf strings.Builder
	for _, diag := range diags {
		buf.WriteString(testFormatDiagnostic(t, diag))
		buf.WriteByte('\n')
	}
	return buf.String()
}

func testFormatDiagnostic(t *testing.T, diag tfdiags.Diagnostic) string {
	t.Helper()

	var buf strings.Builder
	switch diag.Severity() {
	case tfdiags.Error:
		buf.WriteString("[ERROR] ")
	case tfdiags.Warning:
		buf.WriteString("[WARNING] ")
	default:
		buf.WriteString("[PROBLEM] ")
	}
	desc := diag.Description()
	buf.WriteString(desc.Summary)
	buf.WriteByte('\n')
	if subj := diag.Source().Subject; subj != nil {
		buf.WriteString("at " + subj.StartString() + "\n")
	}
	if desc.Detail != "" {
		buf.WriteByte('\n')
		buf.WriteString(desc.Detail)
		buf.WriteByte('\n')
	}
	return buf.String()
}

func assertNoDiagnostics(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()
	if len(diags) != 0 {
		diags.Sort()
		t.Fatalf("unexpected diagnostics\n\n%s", testFormatDiagnostics(t, diags))
	}
}

func assertNoErrors(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()
	if diags.HasErrors() {
		diags.Sort()
		t.Fatalf("unexpected errors\n\n%s", testFormatDiagnostics(t, diags))
	}
}

// testEvaluator constructs a [Main] that's configured for [InspectPhase] using
// the given configuration, state, and other options.
//
// This evaluator is suitable for tests that focus only on evaluation logic
// within this package, but will not be suitable for all situations. Some
// tests should instantiate [Main] directly, particularly if they intend to
// exercise phase-specific functionality like planning or applying component
// instances.
func testEvaluator(t *testing.T, opts testEvaluatorOpts) *Main {
	t.Helper()
	if opts.Config == nil {
		t.Fatal("Config field must not be nil")
	}
	if opts.State == nil {
		opts.State = stackstate.NewState()
	}

	inputVals := make(map[stackaddrs.InputVariable]ExternalInputValue, len(opts.InputVariableValues))
	for name, val := range opts.InputVariableValues {
		inputVals[stackaddrs.InputVariable{Name: name}] = ExternalInputValue{
			Value: val,
			DefRange: tfdiags.SourceRange{
				Filename: "<test-input>",
				Start: tfdiags.SourcePos{
					Line:   1,
					Column: 1,
					Byte:   0,
				},
				End: tfdiags.SourcePos{
					Line:   1,
					Column: 1,
					Byte:   0,
				},
			},
		}
	}

	main := NewForInspecting(opts.Config, opts.State, InspectOpts{
		InputVariableValues: inputVals,
		ProviderFactories:   opts.ProviderFactories,
		TestOnlyGlobals:     opts.TestOnlyGlobals,
	})
	t.Cleanup(func() {
		main.DoCleanup(context.Background())
	})
	return main
}

type testEvaluatorOpts struct {
	// Config is required.
	Config *stackconfig.Config

	// State is optional; testEvaluator will use an empty state if this is nil.
	State *stackstate.State

	// InputVariableValues is optional and if set will provide the values
	// for the root stack input variables. Any variable not defined here
	// will evaluate to an unknown value of the configured type.
	InputVariableValues map[string]cty.Value

	// ProviderFactories is optional and if set provides factory functions
	// for provider types that the test can use. If not set then any attempt
	// to use provider configurations will lead to some sort of error.
	ProviderFactories ProviderFactories

	// TestOnlyGlobals is optional and if set makes it possible to use
	// references like _test_only_global.name to refer to values from this
	// map from anywhere in the entire stack configuration.
	//
	// This is intended as a kind of "test double" so that we can write more
	// minimal unit tests that can avoid relying on too many language features
	// all at once, so that hopefully future maintenance will not require
	// making broad changes across many different tests at once, which would
	// then risk inadvertently treating a regression as expected behavior.
	//
	// Configurations that refer to test-only globals are not valid for use
	// outside of the test suite of this package.
	TestOnlyGlobals map[string]cty.Value
}

// SetTestOnlyGlobals assigns the test-only globals map for the receiving
// main evaluator.
//
// This may be used only from unit tests in this package and must be called
// before performing any other operations against the reciever. It's invalid
// to change the test-only globals after some evaluation has already been
// performed, because the evaluator expects its input to be immutable and
// caches values derived from that input, and there's no mechanism to
// invalidate those caches.
//
// This is intentionally defined in a _test.go file to prevent it from
// being used from non-test code, despite being named as if it's exported.
// It's named as if exported to help differentiate it from unexported
// methods that are intended only as internal API, since it's a public API
// from the perspective of a test caller even though it's not public to
// other callers.
func (m *Main) SetTestOnlyGlobals(t *testing.T, vals map[string]cty.Value) {
	m.testOnlyGlobals = vals
}

func assertFalse(t *testing.T, value bool) {
	t.Helper()
	if value {
		t.Fatalf("expected false but got true")
	}
}

func assertTrue(t *testing.T, value bool) {
	t.Helper()
	if !value {
		t.Fatalf("expected true but got false")
	}
}

func assertNoDiags(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics\n%s", diags.Err())
	}
}

func assertMatchingDiag(t *testing.T, diags tfdiags.Diagnostics, check func(diag tfdiags.Diagnostic) bool) {
	t.Helper()
	for _, diag := range diags {
		if check(diag) {
			return
		}
	}
	t.Fatalf("none of the diagnostics is the one we are expecting\n%s", diags.Err())
}

// inPromisingTask is a helper for conveniently running some code in the context
// of a [promising.MainTask], with automatic promise error checking. This
// makes it valid to call functions that expect to run only as part of a
// promising task, which is true of essentially every method in this package
// that takes a [context.Context] as its first argument.
//
// Specifically, if the function encounters any direct promise-related failures,
// such as failure to resolve a promise before returning, this function will
// halt the test with an error message.
func inPromisingTask(t *testing.T, f func(ctx context.Context, t *testing.T)) {
	t.Helper()

	// We'll introduce an extra cancellable context here just to make
	// sure everything descending from this task gets terminated promptly
	// after the test is complete.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
	})

	_, err := promising.MainTask(ctx, func(ctx context.Context) (struct{}, error) {
		t.Helper()

		f(ctx, t)
		return struct{}{}, nil
	})
	if err != nil {
		// We could get here if the test produces any self-references or
		// if it creates any promises that are left unresolved once it exits.
		t.Fatalf("promise resolution failure: %s", err)
	}
}

// subtestInPromisingTask compiles [testing.T.Run] with [inPromisingTask] as
// a convenience wrapper for running an entire subtest as a [promising.MainTask].
func subtestInPromisingTask(t *testing.T, name string, f func(ctx context.Context, t *testing.T)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Helper()
		inPromisingTask(t, f)
	})
}
