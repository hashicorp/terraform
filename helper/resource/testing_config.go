package resource

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/states"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// testStepConfig runs a config-mode test step
func testStepConfig(
	opts terraform.ContextOpts,
	state *terraform.State,
	step TestStep) (*terraform.State, error) {
	return testStep(opts, state, step)
}

func testStep(opts terraform.ContextOpts, state *terraform.State, step TestStep) (*terraform.State, error) {
	if !step.Destroy {
		if err := testStepTaint(state, step); err != nil {
			return state, err
		}
	}

	cfg, err := testConfig(opts, step)
	if err != nil {
		return state, err
	}

	var stepDiags tfdiags.Diagnostics

	// Build the context
	opts.Config = cfg
	opts.State, err = terraform.ShimLegacyState(state)
	if err != nil {
		return nil, err
	}

	opts.Destroy = step.Destroy
	ctx, stepDiags := terraform.NewContext(&opts)
	if stepDiags.HasErrors() {
		return state, fmt.Errorf("Error initializing context: %s", stepDiags.Err())
	}
	if stepDiags := ctx.Validate(); len(stepDiags) > 0 {
		if stepDiags.HasErrors() {
			return state, errwrap.Wrapf("config is invalid: {{err}}", stepDiags.Err())
		}

		log.Printf("[WARN] Config warnings:\n%s", stepDiags)
	}

	// Refresh!
	newState, stepDiags := ctx.Refresh()
	// shim the state first so the test can check the state on errors

	state, err = shimNewState(newState, step.providers)
	if err != nil {
		return nil, err
	}
	if stepDiags.HasErrors() {
		return state, newOperationError("refresh", stepDiags)
	}

	// If this step is a PlanOnly step, skip over this first Plan and subsequent
	// Apply, and use the follow up Plan that checks for perpetual diffs
	if !step.PlanOnly {
		// Plan!
		if p, stepDiags := ctx.Plan(); stepDiags.HasErrors() {
			return state, newOperationError("plan", stepDiags)
		} else {
			log.Printf("[WARN] Test: Step plan: %s", legacyPlanComparisonString(newState, p.Changes))
		}

		// We need to keep a copy of the state prior to destroying
		// such that destroy steps can verify their behavior in the check
		// function
		stateBeforeApplication := state.DeepCopy()

		// Apply the diff, creating real resources.
		newState, stepDiags = ctx.Apply()
		// shim the state first so the test can check the state on errors
		state, err = shimNewState(newState, step.providers)
		if err != nil {
			return nil, err
		}
		if stepDiags.HasErrors() {
			return state, newOperationError("apply", stepDiags)
		}

		// Run any configured checks
		if step.Check != nil {
			if step.Destroy {
				if err := step.Check(stateBeforeApplication); err != nil {
					return state, fmt.Errorf("Check failed: %s", err)
				}
			} else {
				if err := step.Check(state); err != nil {
					return state, fmt.Errorf("Check failed: %s", err)
				}
			}
		}
	}

	// Now, verify that Plan is now empty and we don't have a perpetual diff issue
	// We do this with TWO plans. One without a refresh.
	var p *plans.Plan
	if p, stepDiags = ctx.Plan(); stepDiags.HasErrors() {
		return state, newOperationError("follow-up plan", stepDiags)
	}
	if !p.Changes.Empty() {
		if step.ExpectNonEmptyPlan {
			log.Printf("[INFO] Got non-empty plan, as expected:\n\n%s", legacyPlanComparisonString(newState, p.Changes))
		} else {
			return state, fmt.Errorf(
				"After applying this step, the plan was not empty:\n\n%s", legacyPlanComparisonString(newState, p.Changes))
		}
	}

	// And another after a Refresh.
	if !step.Destroy || (step.Destroy && !step.PreventPostDestroyRefresh) {
		newState, stepDiags = ctx.Refresh()
		if stepDiags.HasErrors() {
			return state, newOperationError("follow-up refresh", stepDiags)
		}

		state, err = shimNewState(newState, step.providers)
		if err != nil {
			return nil, err
		}
	}
	if p, stepDiags = ctx.Plan(); stepDiags.HasErrors() {
		return state, newOperationError("second follow-up refresh", stepDiags)
	}
	empty := p.Changes.Empty()

	// Data resources are tricky because they legitimately get instantiated
	// during refresh so that they will be already populated during the
	// plan walk. Because of this, if we have any data resources in the
	// config we'll end up wanting to destroy them again here. This is
	// acceptable and expected, and we'll treat it as "empty" for the
	// sake of this testing.
	if step.Destroy && !empty {
		empty = true
		for _, change := range p.Changes.Resources {
			if change.Addr.Resource.Resource.Mode != addrs.DataResourceMode {
				empty = false
				break
			}
		}
	}

	if !empty {
		if step.ExpectNonEmptyPlan {
			log.Printf("[INFO] Got non-empty plan, as expected:\n\n%s", legacyPlanComparisonString(newState, p.Changes))
		} else {
			return state, fmt.Errorf(
				"After applying this step and refreshing, "+
					"the plan was not empty:\n\n%s", legacyPlanComparisonString(newState, p.Changes))
		}
	}

	// Made it here, but expected a non-empty plan, fail!
	if step.ExpectNonEmptyPlan && empty {
		return state, fmt.Errorf("Expected a non-empty plan, but got an empty plan!")
	}

	// Made it here? Good job test step!
	return state, nil
}

// legacyPlanComparisonString produces a string representation of the changes
// from a plan and a given state togther, as was formerly produced by the
// String method of terraform.Plan.
//
// This is here only for compatibility with existing tests that predate our
// new plan and state types, and should not be used in new tests. Instead, use
// a library like "cmp" to do a deep equality  and diff on the two
// data structures.
func legacyPlanComparisonString(state *states.State, changes *plans.Changes) string {
	return fmt.Sprintf(
		"DIFF:\n\n%s\n\nSTATE:\n\n%s",
		legacyDiffComparisonString(changes),
		state.String(),
	)
}

// legacyDiffComparisonString produces a string representation of the changes
// from a planned changes object, as was formerly produced by the String method
// of terraform.Diff.
//
// This is here only for compatibility with existing tests that predate our
// new plan types, and should not be used in new tests. Instead, use a library
// like "cmp" to do a deep equality check and diff on the two data structures.
func legacyDiffComparisonString(changes *plans.Changes) string {
	// The old string representation of a plan was grouped by module, but
	// our new plan structure is not grouped in that way and so we'll need
	// to preprocess it in order to produce that grouping.
	type ResourceChanges struct {
		Current *plans.ResourceInstanceChangeSrc
		Deposed map[states.DeposedKey]*plans.ResourceInstanceChangeSrc
	}
	byModule := map[string]map[string]*ResourceChanges{}
	resourceKeys := map[string][]string{}
	requiresReplace := map[string][]string{}
	var moduleKeys []string
	for _, rc := range changes.Resources {
		if rc.Action == plans.NoOp {
			// We won't mention no-op changes here at all, since the old plan
			// model we are emulating here didn't have such a concept.
			continue
		}
		moduleKey := rc.Addr.Module.String()
		if _, exists := byModule[moduleKey]; !exists {
			moduleKeys = append(moduleKeys, moduleKey)
			byModule[moduleKey] = make(map[string]*ResourceChanges)
		}
		resourceKey := rc.Addr.Resource.String()
		if _, exists := byModule[moduleKey][resourceKey]; !exists {
			resourceKeys[moduleKey] = append(resourceKeys[moduleKey], resourceKey)
			byModule[moduleKey][resourceKey] = &ResourceChanges{
				Deposed: make(map[states.DeposedKey]*plans.ResourceInstanceChangeSrc),
			}
		}

		if rc.DeposedKey == states.NotDeposed {
			byModule[moduleKey][resourceKey].Current = rc
		} else {
			byModule[moduleKey][resourceKey].Deposed[rc.DeposedKey] = rc
		}

		rr := []string{}
		for _, p := range rc.RequiredReplace.List() {
			rr = append(rr, hcl2shim.FlatmapKeyFromPath(p))
		}
		requiresReplace[resourceKey] = rr
	}
	sort.Strings(moduleKeys)
	for _, ks := range resourceKeys {
		sort.Strings(ks)
	}

	var buf bytes.Buffer

	for _, moduleKey := range moduleKeys {
		rcs := byModule[moduleKey]
		var mBuf bytes.Buffer

		for _, resourceKey := range resourceKeys[moduleKey] {
			rc := rcs[resourceKey]

			forceNewAttrs := requiresReplace[resourceKey]

			crud := "UPDATE"
			if rc.Current != nil {
				switch rc.Current.Action {
				case plans.DeleteThenCreate:
					crud = "DESTROY/CREATE"
				case plans.CreateThenDelete:
					crud = "CREATE/DESTROY"
				case plans.Delete:
					crud = "DESTROY"
				case plans.Create:
					crud = "CREATE"
				}
			} else {
				// We must be working on a deposed object then, in which
				// case destroying is the only possible action.
				crud = "DESTROY"
			}

			extra := ""
			if rc.Current == nil && len(rc.Deposed) > 0 {
				extra = " (deposed only)"
			}

			fmt.Fprintf(
				&mBuf, "%s: %s%s\n",
				crud, resourceKey, extra,
			)

			attrNames := map[string]bool{}
			var oldAttrs map[string]string
			var newAttrs map[string]string
			if rc.Current != nil {
				if before := rc.Current.Before; before != nil {
					ty, err := before.ImpliedType()
					if err == nil {
						val, err := before.Decode(ty)
						if err == nil {
							oldAttrs = hcl2shim.FlatmapValueFromHCL2(val)
							for k := range oldAttrs {
								attrNames[k] = true
							}
						}
					}
				}
				if after := rc.Current.After; after != nil {
					ty, err := after.ImpliedType()
					if err == nil {
						val, err := after.Decode(ty)
						if err == nil {
							newAttrs = hcl2shim.FlatmapValueFromHCL2(val)
							for k := range newAttrs {
								attrNames[k] = true
							}
						}
					}
				}
			}
			if oldAttrs == nil {
				oldAttrs = make(map[string]string)
			}
			if newAttrs == nil {
				newAttrs = make(map[string]string)
			}

			attrNamesOrder := make([]string, 0, len(attrNames))
			keyLen := 0
			for n := range attrNames {
				attrNamesOrder = append(attrNamesOrder, n)
				if len(n) > keyLen {
					keyLen = len(n)
				}
			}
			sort.Strings(attrNamesOrder)

			for _, attrK := range attrNamesOrder {
				v := newAttrs[attrK]
				u := oldAttrs[attrK]

				if v == hcl2shim.UnknownVariableValue {
					v = "<computed>"
				}
				// NOTE: we don't support <sensitive> here because we would
				// need schema to do that. Excluding sensitive values
				// is now done at the UI layer, and so should not be tested
				// at the core layer.

				updateMsg := ""

				// This may not be as precise as in the old diff, as it matches
				// everything under the attribute that was originally marked as
				// ForceNew, but should help make it easier to determine what
				// caused replacement here.
				for _, k := range forceNewAttrs {
					if strings.HasPrefix(attrK, k) {
						updateMsg = " (forces new resource)"
						break
					}
				}

				fmt.Fprintf(
					&mBuf, "  %s:%s %#v => %#v%s\n",
					attrK,
					strings.Repeat(" ", keyLen-len(attrK)),
					u, v,
					updateMsg,
				)
			}
		}

		if moduleKey == "" { // root module
			buf.Write(mBuf.Bytes())
			buf.WriteByte('\n')
			continue
		}

		fmt.Fprintf(&buf, "%s:\n", moduleKey)
		s := bufio.NewScanner(&mBuf)
		for s.Scan() {
			buf.WriteString(fmt.Sprintf("  %s\n", s.Text()))
		}
	}

	return buf.String()
}

func testStepTaint(state *terraform.State, step TestStep) error {
	for _, p := range step.Taint {
		m := state.RootModule()
		if m == nil {
			return errors.New("no state")
		}
		rs, ok := m.Resources[p]
		if !ok {
			return fmt.Errorf("resource %q not found in state", p)
		}
		log.Printf("[WARN] Test: Explicitly tainting resource %q", p)
		rs.Taint()
	}
	return nil
}
