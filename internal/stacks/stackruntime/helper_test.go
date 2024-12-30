// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// This file has helper functions used by other tests. It doesn't contain any
// test cases of its own.

// TestContext contains all the information shared across multiple operations
// in a single test.
type TestContext struct {
	// timestamp is the timestamp that should be applied for this test.
	timestamp *time.Time

	// config is the config to use for this test.
	config *stackconfig.Config

	// providers are the providers that should be available within this test.
	providers map[addrs.Provider]providers.Factory

	// dependencyLocks is the locks file that should be used for this test.
	dependencyLocks depsfile.Locks
}

// TestCycle defines a single plan / apply cycle that should be performed within
// a test.
type TestCycle struct {

	// Validate options

	wantValidateDiags tfdiags.Diagnostics

	// Plan options

	planMode           plans.Mode
	planInputs         map[string]cty.Value
	wantPlannedChanges []stackplan.PlannedChange
	wantPlannedDiags   tfdiags.Diagnostics

	// Apply options

	applyInputs        map[string]cty.Value
	wantAppliedChanges []stackstate.AppliedChange
	wantAppliedDiags   tfdiags.Diagnostics
}

func (tc TestContext) Validate(t *testing.T, ctx context.Context, cycle TestCycle) {
	t.Helper()

	gotDiags := Validate(ctx, &ValidateRequest{
		Config:             tc.config,
		ProviderFactories:  tc.providers,
		DependencyLocks:    tc.dependencyLocks,
		ExperimentsAllowed: true,
	})
	validateDiags(t, cycle.wantValidateDiags, gotDiags)
}

func (tc TestContext) Plan(t *testing.T, ctx context.Context, state *stackstate.State, cycle TestCycle) *stackplan.Plan {
	t.Helper()

	request := PlanRequest{
		PlanMode:  cycle.planMode,
		Config:    tc.config,
		PrevState: state,
		InputValues: func() map[stackaddrs.InputVariable]ExternalInputValue {
			inputs := make(map[stackaddrs.InputVariable]ExternalInputValue, len(cycle.planInputs))
			for k, v := range cycle.planInputs {
				inputs[stackaddrs.InputVariable{Name: k}] = ExternalInputValue{Value: v}
			}
			return inputs
		}(),
		ProviderFactories:  tc.providers,
		DependencyLocks:    tc.dependencyLocks,
		ForcePlanTimestamp: tc.timestamp,
		ExperimentsAllowed: true,
	}

	changesCh := make(chan stackplan.PlannedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	response := PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}

	go Plan(ctx, &request, &response)
	changes, diags := collectPlanOutput(changesCh, diagsCh)
	validateDiags(t, cycle.wantPlannedDiags, diags)

	if cycle.wantPlannedChanges != nil {

		var filteredChanges []stackplan.PlannedChange
		for _, change := range changes {
			if _, ok := change.(*stackplan.PlannedChangePriorStateElement); ok {
				// Remove the prior state elements from the analysis for tests
				// using this framework. They're really difficult to properly
				// compare as they use the raw state and raw key, and they're
				// actually ignored by most of the stacks runtime and so aren't
				// useful to be included in these kind of tests.
				continue
			}
			filteredChanges = append(filteredChanges, change)
		}

		// if this is nil (as opposed to empty) then we don't validate the
		// returned changes.

		sort.SliceStable(filteredChanges, func(i, j int) bool {
			return plannedChangeSortKey(filteredChanges[i]) < plannedChangeSortKey(filteredChanges[j])
		})
		if diff := cmp.Diff(cycle.wantPlannedChanges, filteredChanges, changesCmpOpts); len(diff) > 0 {
			t.Errorf("wrong planned changes\n%s", diff)
		}
	}

	planLoader := stackplan.NewLoader()
	for _, change := range changes {
		proto, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		for _, rawMsg := range proto.Raw {
			err = planLoader.AddRaw(rawMsg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	plan, err := planLoader.Plan()
	if err != nil {
		t.Fatal(err)
	}

	return plan
}

func (tc TestContext) Apply(t *testing.T, ctx context.Context, plan *stackplan.Plan, cycle TestCycle) *stackstate.State {
	t.Helper()

	request := ApplyRequest{
		Config: tc.config,
		Plan:   plan,
		InputValues: func() map[stackaddrs.InputVariable]ExternalInputValue {
			inputs := make(map[stackaddrs.InputVariable]ExternalInputValue, len(cycle.applyInputs))
			for k, v := range cycle.applyInputs {
				inputs[stackaddrs.InputVariable{Name: k}] = ExternalInputValue{Value: v}
			}
			return inputs
		}(),
		ProviderFactories:  tc.providers,
		ExperimentsAllowed: true,
		DependencyLocks:    tc.dependencyLocks,
	}

	changesCh := make(chan stackstate.AppliedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	response := ApplyResponse{
		AppliedChanges: changesCh,
		Diagnostics:    diagsCh,
	}

	go Apply(ctx, &request, &response)
	changes, diags := collectApplyOutput(changesCh, diagsCh)
	validateDiags(t, cycle.wantAppliedDiags, diags)

	if cycle.wantAppliedChanges != nil {
		// nil indicates skip this check, empty slice indicates no changes expected.
		sort.SliceStable(changes, func(i, j int) bool {
			return appliedChangeSortKey(changes[i]) < appliedChangeSortKey(changes[j])
		})
		if diff := cmp.Diff(cycle.wantAppliedChanges, changes, changesCmpOpts); diff != "" {
			t.Errorf("wrong applied changes\n%s", diff)
		}
	}

	stateLoader := stackstate.NewLoader()
	for _, change := range changes {
		proto, err := change.AppliedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		for _, rawMsg := range proto.Raw {
			if rawMsg.Value == nil {
				// This is a removal notice, so we don't need to add it to the
				// state.
				continue
			}
			err = stateLoader.AddRaw(rawMsg.Key, rawMsg.Value)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	return stateLoader.State()
}

func initDiags(cb func(diags tfdiags.Diagnostics) tfdiags.Diagnostics) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	return cb(diags)
}

func validateDiags(t *testing.T, wantDiags, gotDiags tfdiags.Diagnostics) {
	t.Helper()

	sort.SliceStable(gotDiags, diagnosticSortFunc(gotDiags))
	sort.SliceStable(wantDiags, diagnosticSortFunc(wantDiags))

	gotDiags = gotDiags.ForRPC()
	wantDiags = wantDiags.ForRPC()
	if diff := cmp.Diff(wantDiags, gotDiags); len(diff) > 0 {
		t.Errorf("wrong diagnostics\n%s", diff)
	}
}

// loadConfigForTest is a test helper that tries to open bundleRoot as a
// source bundle, and then if successful tries to load the given source address
// from it as a stack configuration. If any part of the operation fails then
// it halts execution of the test and doesn't return.
func loadConfigForTest(t *testing.T, bundleRoot string, configSourceAddr string) *stackconfig.Config {
	t.Helper()
	sources, err := sourcebundle.OpenDir(bundleRoot)
	if err != nil {
		t.Fatalf("cannot load source bundle: %s", err)
	}

	// We force using remote source addresses here because that avoids
	// us having to deal with the extra version constraints argument
	// that registry sources require. Exactly what source address type
	// we use isn't relevant for tests in this package, since it's
	// the sourcebundle package's responsibility to make sure its
	// abstraction works for all of the source types.
	sourceAddr, err := sourceaddrs.ParseRemoteSource(configSourceAddr)
	if err != nil {
		t.Fatalf("invalid config source address: %s", err)
	}

	cfg, diags := stackconfig.LoadConfigDir(sourceAddr, sources)
	reportDiagnosticsForTest(t, diags)
	return cfg
}

func mainBundleSourceAddrStr(dirName string) string {
	return "git::https://example.com/test.git//" + dirName
}

// loadMainBundleConfigForTest is a convenience wrapper around
// loadConfigForTest that knows the location and package address of our
// "main" source bundle, in ./testdata/mainbundle, so that we can use that
// conveniently without duplicating its location and synthetic package address
// in every single test function.
//
// dirName should begin with the name of a subdirectory that's present in
// ./testdata/mainbundle/test . It can optionally refer to subdirectories
// thereof, using forward slashes as the path separator just as we'd do
// in the subdirectory portion of a remote source address (which is exactly
// what we're using this as.)
func loadMainBundleConfigForTest(t *testing.T, dirName string) *stackconfig.Config {
	t.Helper()
	fullSourceAddr := mainBundleSourceAddrStr(dirName)
	return loadConfigForTest(t, "./testdata/mainbundle", fullSourceAddr)
}

type expectedDiagnostic struct {
	severity tfdiags.Severity
	summary  string
	detail   string
}

func expectDiagnostic(severity tfdiags.Severity, summary, detail string) expectedDiagnostic {
	return expectedDiagnostic{
		severity: severity,
		summary:  summary,
		detail:   detail,
	}
}

func expectDiagnosticsForTest(t *testing.T, actual tfdiags.Diagnostics, expected ...expectedDiagnostic) {
	t.Helper()

	max := len(expected)
	if len(actual) > max {
		max = len(actual)
	}

	for ix := 0; ix < max; ix++ {
		if ix >= len(expected) {
			t.Errorf("unexpected diagnostic [%d]: %s - %s", ix, actual[ix].Description().Summary, actual[ix].Description().Detail)
			continue
		}

		if ix >= len(actual) {
			t.Errorf("missing diagnostic [%d]: %s - %s", ix, expected[ix].summary, expected[ix].detail)
			continue
		}

		if actual[ix].Severity() != expected[ix].severity {
			t.Errorf("diagnostic [%d] has wrong severity: %s (expected %s)", ix, actual[ix].Severity(), expected[ix].severity)
		}

		if diff := cmp.Diff(actual[ix].Description().Summary, expected[ix].summary); len(diff) > 0 {
			t.Errorf("diagnostic [%d] has wrong summary: %s", ix, diff)
		}

		if diff := cmp.Diff(actual[ix].Description().Detail, expected[ix].detail); len(diff) > 0 {
			t.Errorf("diagnostic [%d] has wrong detail: %s", ix, diff)
		}
	}
}

// reportDiagnosticsForTest creates a test log entry for every diagnostic in
// the given diags, and halts the test if any of them are error diagnostics.
func reportDiagnosticsForTest(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()
	for _, diag := range diags {
		var b strings.Builder
		desc := diag.Description()
		locs := diag.Source()

		switch sev := diag.Severity(); sev {
		case tfdiags.Error:
			b.WriteString("Error: ")
		case tfdiags.Warning:
			b.WriteString("Warning: ")
		default:
			t.Errorf("unsupported diagnostic type %s", sev)
		}
		b.WriteString(desc.Summary)
		if desc.Address != "" {
			b.WriteString("\nwith ")
			b.WriteString(desc.Summary)
		}
		if locs.Subject != nil {
			b.WriteString("\nat ")
			b.WriteString(locs.Subject.StartString())
		}
		if desc.Detail != "" {
			b.WriteString("\n\n")
			b.WriteString(desc.Detail)
		}
		t.Log(b.String())
	}
	if diags.HasErrors() {
		t.FailNow()
	}
}

// appliedChangeSortKey returns a string that can be used to sort applied
// changes in a predictable order for testing purposes. This is used to
// ensure that we can compare applied changes in a consistent way across
// different test runs.
func appliedChangeSortKey(change stackstate.AppliedChange) string {
	switch change := change.(type) {
	case *stackstate.AppliedChangeResourceInstanceObject:
		return change.ResourceInstanceObjectAddr.String()
	case *stackstate.AppliedChangeComponentInstance:
		return change.ComponentInstanceAddr.String()
	case *stackstate.AppliedChangeComponentInstanceRemoved:
		return change.ComponentInstanceAddr.String()
	case *stackstate.AppliedChangeOutputValue:
		return change.Addr.String()
	case *stackstate.AppliedChangeInputVariable:
		return change.Addr.String()
	case *stackstate.AppliedChangeDiscardKeys:
		// There should only be a single discard keys in a plan, so we can just
		// return a static string here.
		return "discard"
	default:
		// This is only going to happen during tests, so we can panic here.
		panic(fmt.Errorf("unrecognized applied change type: %T", change))
	}
}

// plannedChangeSortKey returns a string that can be used to sort planned
// changes in a predictable order for testing purposes. This is used to
// ensure that we can compare planned changes in a consistent way across
// different test runs.
func plannedChangeSortKey(change stackplan.PlannedChange) string {
	switch change := change.(type) {
	case *stackplan.PlannedChangeRootInputValue:
		return change.Addr.String()
	case *stackplan.PlannedChangeComponentInstance:
		return change.Addr.String()
	case *stackplan.PlannedChangeComponentInstanceRemoved:
		return change.Addr.String()
	case *stackplan.PlannedChangeResourceInstancePlanned:
		return change.ResourceInstanceObjectAddr.String()
	case *stackplan.PlannedChangeDeferredResourceInstancePlanned:
		return change.ResourceInstancePlanned.ResourceInstanceObjectAddr.String()
	case *stackplan.PlannedChangeOutputValue:
		return change.Addr.String()
	case *stackplan.PlannedChangeHeader:
		// There should only be a single header in a plan, so we can just return
		// a static string here.
		return "header"
	case *stackplan.PlannedChangeApplyable:
		// There should only be a single applyable marker in a plan, so we can
		// just return a static string here.
		return "applyable"
	case *stackplan.PlannedChangePlannedTimestamp:
		// There should only be a single timestamp in a plan, so we can
		// just return a static string here.
		return "planned-timestamp"
	case *stackplan.PlannedChangeProviderFunctionResults:
		// There should only be a single timestamp in a plan, so we can just
		// return a simple string.
		return "function-results"
	default:
		// This is only going to happen during tests, so we can panic here.
		panic(fmt.Errorf("unrecognized planned change type: %T", change))
	}
}

func diagnosticSortFunc(diags tfdiags.Diagnostics) func(i, j int) bool {
	sortDescription := func(i, j tfdiags.Description) bool {
		if i.Summary != j.Summary {
			return i.Summary < j.Summary
		}
		return i.Detail < j.Detail
	}

	sortPos := func(i, j tfdiags.SourcePos) bool {
		if i.Line != j.Line {
			return i.Line < j.Line
		}
		return i.Column < j.Column
	}

	sortRange := func(i, j *tfdiags.SourceRange) bool {
		if i.Filename != j.Filename {
			return i.Filename < j.Filename
		}
		if !cmp.Equal(i.Start, j.Start) {
			return sortPos(i.Start, j.Start)
		}
		return sortPos(i.End, j.End)
	}

	return func(i, j int) bool {
		id, jd := diags[i], diags[j]
		if id.Severity() != jd.Severity() {
			return id.Severity() == tfdiags.Error
		}
		if !cmp.Equal(id.Description(), jd.Description()) {
			return sortDescription(id.Description(), jd.Description())
		}
		if id.Source().Subject != nil && jd.Source().Subject != nil {

			return sortRange(id.Source().Subject, jd.Source().Subject)
		}

		return false
	}
}

func mustDefaultRootProvider(provider string) addrs.AbsProviderConfig {
	return addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider(provider),
	}
}

func mustAbsResourceInstance(addr string) addrs.AbsResourceInstance {
	ret, diags := addrs.ParseAbsResourceInstanceStr(addr)
	if len(diags) > 0 {
		panic(fmt.Sprintf("failed to parse resource instance address %q: %s", addr, diags))
	}
	return ret
}

func mustAbsResourceInstanceObject(addr string) stackaddrs.AbsResourceInstanceObject {
	ret, diags := stackaddrs.ParseAbsResourceInstanceObjectStr(addr)
	if len(diags) > 0 {
		panic(fmt.Sprintf("failed to parse resource instance object address %q: %s", addr, diags))
	}
	return ret
}

func mustAbsResourceInstanceObjectPtr(addr string) *stackaddrs.AbsResourceInstanceObject {
	ret := mustAbsResourceInstanceObject(addr)
	return &ret
}

func mustAbsComponentInstance(addr string) stackaddrs.AbsComponentInstance {
	ret, diags := stackaddrs.ParsePartialComponentInstanceStr(addr)
	if len(diags) > 0 {
		panic(fmt.Sprintf("failed to parse component instance address %q: %s", addr, diags))
	}
	return ret
}

func mustAbsComponent(addr string) stackaddrs.AbsComponent {
	ret, diags := stackaddrs.ParsePartialComponentInstanceStr(addr)
	if len(diags) > 0 {
		panic(fmt.Sprintf("failed to parse component instance address %q: %s", addr, diags))
	}
	return stackaddrs.AbsComponent{
		Stack: ret.Stack,
		Item:  ret.Item.Component,
	}
}

// mustPlanDynamicValue is a helper function that constructs a
// plans.DynamicValue from the given cty.Value, panicking if the construction
// fails.
func mustPlanDynamicValue(v cty.Value) plans.DynamicValue {
	ret, err := plans.NewDynamicValue(v, v.Type())
	if err != nil {
		panic(err)
	}
	return ret
}

// mustPlanDynamicValueDynamicType is a helper function that constructs a
// plans.DynamicValue from the given cty.Value, using cty.DynamicPseudoType as
// the type, and panicking if the construction fails.
func mustPlanDynamicValueDynamicType(v cty.Value) plans.DynamicValue {
	ret, err := plans.NewDynamicValue(v, cty.DynamicPseudoType)
	if err != nil {
		panic(err)
	}
	return ret
}

// mustPlanDynamicValueSchema is a helper function that constructs a
// plans.DynamicValue from the given cty.Value and configschema.Block, panicking
// if the construction fails.
func mustPlanDynamicValueSchema(v cty.Value, block *configschema.Block) plans.DynamicValue {
	ty := block.ImpliedType()
	ret, err := plans.NewDynamicValue(v, ty)
	if err != nil {
		panic(err)
	}
	return ret
}

func mustInputVariable(name string) addrs.InputVariable {
	return addrs.InputVariable{Name: name}
}

func mustStackInputVariable(name string) stackaddrs.InputVariable {
	return stackaddrs.InputVariable{Name: name}
}

func mustStackOutputValue(name string) stackaddrs.OutputValue {
	return stackaddrs.OutputValue{Name: name}
}

func mustMarshalJSONAttrs(attrs map[string]interface{}) []byte {
	jsonAttrs, err := json.Marshal(attrs)
	if err != nil {
		panic(err)
	}
	return jsonAttrs
}

func providerFunctionHashArgs(provider addrs.Provider, name string, args ...cty.Value) []byte {
	sum := sha256.New()

	sum.Write([]byte(provider.String()))
	sum.Write([]byte("|"))
	sum.Write([]byte(name))
	for _, arg := range args {
		sum.Write([]byte("|"))
		sum.Write([]byte(arg.GoString()))
	}

	return sum.Sum(nil)
}

func providerFunctionHashResult(value cty.Value) []byte {
	bytes := sha256.Sum256([]byte(value.GoString()))
	return bytes[:]
}
