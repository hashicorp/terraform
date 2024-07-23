// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackplan

import (
	"fmt"
	"sync"

	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/plans/planproto"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/version"
)

// A helper for loading saved plans in a streaming manner.
type Loader struct {
	ret         *Plan
	foundHeader bool

	mu sync.Mutex
}

// Constructs a new [Loader], with an initial empty plan.
func NewLoader() *Loader {
	ret := &Plan{
		RootInputValues:         make(map[stackaddrs.InputVariable]cty.Value),
		ApplyTimeInputVariables: collections.NewSetCmp[stackaddrs.InputVariable](),
		Components:              collections.NewMap[stackaddrs.AbsComponentInstance, *Component](),
		PrevRunStateRaw:         make(map[string]*anypb.Any),
	}
	return &Loader{
		ret: ret,
	}
}

// AddRaw adds a single raw change object to the plan being loaded.
func (l *Loader) AddRaw(rawMsg *anypb.Any) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ret == nil {
		return fmt.Errorf("loader has been consumed")
	}

	msg, err := anypb.UnmarshalNew(rawMsg, proto.UnmarshalOptions{
		// Just the default unmarshalling options
	})
	if err != nil {
		return fmt.Errorf("invalid raw message: %w", err)
	}

	// The references to specific message types below ensure that
	// the protobuf descriptors for these types are included in the
	// compiled program, and thus available in the global protobuf
	// registry that anypb.UnmarshalNew relies on above.
	switch msg := msg.(type) {

	case *tfstackdata1.PlanHeader:
		wantVersion := version.SemVer.String()
		gotVersion := msg.TerraformVersion
		if gotVersion != wantVersion {
			return fmt.Errorf("plan was created by Terraform %s, but this is Terraform %s", gotVersion, wantVersion)
		}
		l.foundHeader = true

	case *tfstackdata1.PlanPriorStateElem:
		if _, exists := l.ret.PrevRunStateRaw[msg.Key]; exists {
			// Suggests a bug in the caller, because a valid prior state
			// can only have one object associated with each key.
			return fmt.Errorf("duplicate prior state key %q", msg.Key)
		}
		// NOTE: We intentionally don't actually decode and validate the
		// state elements here; we'll deal with that piecemeal as we make
		// further use of this data structure elsewhere. This avoids spending
		// time on decoding here if a caller is loading the plan only to
		// extract some metadata from it, and doesn't care about the prior
		// state.
		l.ret.PrevRunStateRaw[msg.Key] = msg.Raw

	case *tfstackdata1.PlanApplyable:
		l.ret.Applyable = msg.Applyable

	case *tfstackdata1.PlanTimestamp:
		err = l.ret.PlanTimestamp.UnmarshalText([]byte(msg.PlanTimestamp))
		if err != nil {
			return fmt.Errorf("invalid plan timestamp %q", msg.PlanTimestamp)
		}

	case *tfstackdata1.PlanRootInputValue:
		addr := stackaddrs.InputVariable{
			Name: msg.Name,
		}
		if msg.Value != nil {
			dv := plans.DynamicValue(msg.Value.Msgpack)
			val, err := dv.Decode(cty.DynamicPseudoType)
			if err != nil {
				return fmt.Errorf("invalid stored value for %s: %w", addr, err)
			}
			l.ret.RootInputValues[addr] = val
		}
		if msg.RequiredOnApply {
			if msg.Value != nil {
				// A variable can't be both persisted _and_ required on apply.
				return fmt.Errorf("plan has value for required-on-apply input variable %s", addr)
			}
			l.ret.ApplyTimeInputVariables.Add(addr)
		}

	case *tfstackdata1.PlanComponentInstance:
		addr, diags := stackaddrs.ParseAbsComponentInstanceStr(msg.ComponentInstanceAddr)
		if diags.HasErrors() {
			// Should not get here because the address we're parsing
			// should've been produced by this same version of Terraform.
			return fmt.Errorf("invalid component instance address syntax in %q", msg.ComponentInstanceAddr)
		}

		dependencies := collections.NewSet[stackaddrs.AbsComponent]()
		for _, rawAddr := range msg.DependsOnComponentAddrs {
			// NOTE: We're using the component _instance_ address parser
			// here, but we really want just components, so we'll need to
			// check afterwards to make sure we don't have an instance key.
			addr, diags := stackaddrs.ParseAbsComponentInstanceStr(rawAddr)
			if diags.HasErrors() {
				return fmt.Errorf("invalid component address syntax in %q", rawAddr)
			}
			if addr.Item.Key != addrs.NoKey {
				return fmt.Errorf("invalid component address syntax in %q: is actually a component instance address", rawAddr)
			}
			realAddr := stackaddrs.AbsComponent{
				Stack: addr.Stack,
				Item:  addr.Item.Component,
			}
			dependencies.Add(realAddr)
		}

		plannedAction, err := planproto.FromAction(msg.PlannedAction)
		if err != nil {
			return fmt.Errorf("decoding plan for %s: %w", addr, err)
		}

		inputVals := make(map[addrs.InputVariable]plans.DynamicValue)
		inputValMarks := make(map[addrs.InputVariable][]cty.PathValueMarks)
		for name, rawVal := range msg.PlannedInputValues {
			val := addrs.InputVariable{
				Name: name,
			}
			inputVals[val] = rawVal.Value.Msgpack
			inputValMarks[val] = make([]cty.PathValueMarks, len(rawVal.SensitivePaths))
			for _, path := range rawVal.SensitivePaths {
				path, err := planfile.PathFromProto(path)
				if err != nil {
					return fmt.Errorf("decoding sensitive path %q for %s: %w", val, addr, err)
				}
				inputValMarks[val] = append(inputValMarks[val], cty.PathValueMarks{
					Path:  path,
					Marks: cty.NewValueMarks(marks.Sensitive),
				})
			}
		}

		outputVals := make(map[addrs.OutputValue]cty.Value)
		for name, rawVal := range msg.PlannedOutputValues {
			v, err := tfstackdata1.DynamicValueFromTFStackData1(rawVal, cty.DynamicPseudoType)
			if err != nil {
				return fmt.Errorf("decoding output value %q for %s: %w", name, addr, err)
			}
			outputVals[addrs.OutputValue{Name: name}] = v
		}

		checkResults, err := planfile.CheckResultsFromPlanProto(msg.PlannedCheckResults)
		if err != nil {
			return fmt.Errorf("decoding check results: %w", err)
		}

		if !l.ret.Components.HasKey(addr) {
			l.ret.Components.Put(addr, &Component{
				PlannedAction:          plannedAction,
				PlanApplyable:          msg.PlanApplyable,
				PlanComplete:           msg.PlanComplete,
				Dependencies:           dependencies,
				Dependents:             collections.NewSet[stackaddrs.AbsComponent](),
				PlannedInputValues:     inputVals,
				PlannedInputValueMarks: inputValMarks,
				PlannedOutputValues:    outputVals,
				PlannedChecks:          checkResults,

				ResourceInstancePlanned:         addrs.MakeMap[addrs.AbsResourceInstanceObject, *plans.ResourceInstanceChangeSrc](),
				ResourceInstancePriorState:      addrs.MakeMap[addrs.AbsResourceInstanceObject, *states.ResourceInstanceObjectSrc](),
				ResourceInstanceProviderConfig:  addrs.MakeMap[addrs.AbsResourceInstanceObject, addrs.AbsProviderConfig](),
				DeferredResourceInstanceChanges: addrs.MakeMap[addrs.AbsResourceInstanceObject, *plans.DeferredResourceInstanceChangeSrc](),
			})
		}
		c := l.ret.Components.Get(addr)
		err = c.PlanTimestamp.UnmarshalText([]byte(msg.PlanTimestamp))
		if err != nil {
			return fmt.Errorf("invalid plan timestamp %q for %s", msg.PlanTimestamp, addr)
		}

	case *tfstackdata1.PlanResourceInstanceChangePlanned:
		c, fullAddr, providerConfigAddr, err := LoadComponentForResourceInstance(l.ret, msg)
		if err != nil {
			return err
		}
		c.ResourceInstanceProviderConfig.Put(fullAddr, providerConfigAddr)

		// Not all "planned changes" for resource instances are actually
		// changes in the plans.Change sense, confusingly: sometimes the
		// "change" we're recording is just to overwrite the state entry
		// with a refreshed copy, in which case riPlan is nil and
		// msg.PriorState is the main content of this change, handled below.
		if msg.Change != nil {
			riPlan, err := ValidateResourceInstanceChange(msg, fullAddr, providerConfigAddr)
			if err != nil {
				return err
			}
			c.ResourceInstancePlanned.Put(fullAddr, riPlan)
		}

		if msg.PriorState != nil {
			stateSrc, err := stackstate.DecodeProtoResourceInstanceObject(msg.PriorState)
			if err != nil {
				return fmt.Errorf("invalid prior state for %s: %w", fullAddr, err)
			}
			c.ResourceInstancePriorState.Put(fullAddr, stateSrc)
		} else {
			// We'll record an explicit nil just to affirm that there's
			// intentionally no prior state for this resource instance
			// object.
			c.ResourceInstancePriorState.Put(fullAddr, nil)
		}

	case *tfstackdata1.PlanDeferredResourceInstanceChange:
		if msg.Deferred == nil {
			return fmt.Errorf("missing deferred from PlanDeferredResourceInstanceChange")
		}

		c, fullAddr, providerConfigAddr, err := LoadComponentForPartialResourceInstance(l.ret, msg.Change)
		if err != nil {
			return err
		}

		riPlan, err := ValidatePartialResourceInstanceChange(msg.Change, fullAddr, providerConfigAddr)
		if err != nil {
			return err
		}

		// We'll just swallow the error here. A missing deferred reason
		// could be the only cause and we want to be forward and backward
		// compatible. This will just render as INVALID, which is fine.
		deferredReason, _ := planfile.DeferredReasonFromProto(msg.Deferred.Reason)

		c.DeferredResourceInstanceChanges.Put(fullAddr, &plans.DeferredResourceInstanceChangeSrc{
			ChangeSrc:      riPlan,
			DeferredReason: deferredReason,
		})

	default:
		// Should not get here, because a stack plan can only be loaded by
		// the same version of Terraform that created it, and the above
		// should cover everything this version of Terraform can possibly
		// emit during PlanStackChanges.
		return fmt.Errorf("unsupported raw message type %T", msg)
	}
	return nil
}

// Plan consumes the loaded plan, making the associated loader closed to
// further additions.
func (l *Loader) Plan() (*Plan, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// If we got through all of the messages without encountering at least
	// one *PlanHeader then we'll abort because we may have lost part of the
	// plan sequence somehow.
	if !l.foundHeader {
		return nil, fmt.Errorf("missing PlanHeader")
	}

	// Before we return we'll calculate the reverse dependency information
	// based on the forward dependency information we loaded earlier.
	for _, elem := range l.ret.Components.Elems() {
		dependentInstAddr := elem.K
		dependentAddr := stackaddrs.AbsComponent{
			Stack: dependentInstAddr.Stack,
			Item:  dependentInstAddr.Item.Component,
		}

		for _, dependencyAddr := range elem.V.Dependencies.Elems() {
			// FIXME: This is very inefficient because the current data structure doesn't
			// allow looking up all of the component instances that have a particular
			// component. This'll be okay as long as the number of components is
			// small, but we'll need to improve this if we ever want to support stacks
			// with a large number of components.
			for _, elem := range l.ret.Components.Elems() {
				maybeDependencyInstAddr := elem.K
				maybeDependencyAddr := stackaddrs.AbsComponent{
					Stack: maybeDependencyInstAddr.Stack,
					Item:  maybeDependencyInstAddr.Item.Component,
				}
				if dependencyAddr.UniqueKey() == maybeDependencyAddr.UniqueKey() {
					elem.V.Dependents.Add(dependentAddr)
				}
			}
		}
	}

	ret := l.ret
	l.ret = nil

	return ret, nil
}

func LoadFromProto(msgs []*anypb.Any) (*Plan, error) {
	loader := NewLoader()
	for i, rawMsg := range msgs {
		err := loader.AddRaw(rawMsg)
		if err != nil {
			return nil, fmt.Errorf("raw item %d: %w", i, err)
		}
	}
	return loader.Plan()
}

func ValidateResourceInstanceChange(change *tfstackdata1.PlanResourceInstanceChangePlanned, fullAddr addrs.AbsResourceInstanceObject, providerConfigAddr addrs.AbsProviderConfig) (*plans.ResourceInstanceChangeSrc, error) {
	riPlan, err := planfile.ResourceChangeFromProto(change.Change)
	if err != nil {
		return nil, fmt.Errorf("invalid resource instance change: %w", err)
	}
	// We currently have some redundant information in the nested
	// "change" object due to having reused some protobuf message
	// types from the traditional Terraform CLI planproto format.
	// We'll make sure the redundant information is consistent
	// here because otherwise they're likely to cause
	// difficult-to-debug problems downstream.
	if !riPlan.Addr.Equal(fullAddr.ResourceInstance) && riPlan.DeposedKey == fullAddr.DeposedKey {
		return nil, fmt.Errorf("planned change has inconsistent address to its containing object")
	}
	if !riPlan.ProviderAddr.Equal(providerConfigAddr) {
		return nil, fmt.Errorf("planned change has inconsistent provider configuration address to its containing object")
	}
	return riPlan, nil
}

func ValidatePartialResourceInstanceChange(change *tfstackdata1.PlanResourceInstanceChangePlanned, fullAddr addrs.AbsResourceInstanceObject, providerConfigAddr addrs.AbsProviderConfig) (*plans.ResourceInstanceChangeSrc, error) {
	riPlan, err := planfile.DeferredResourceChangeFromProto(change.Change)
	if err != nil {
		return nil, fmt.Errorf("invalid resource instance change: %w", err)
	}
	// We currently have some redundant information in the nested
	// "change" object due to having reused some protobuf message
	// types from the traditional Terraform CLI planproto format.
	// We'll make sure the redundant information is consistent
	// here because otherwise they're likely to cause
	// difficult-to-debug problems downstream.
	if !riPlan.Addr.Equal(fullAddr.ResourceInstance) && riPlan.DeposedKey == fullAddr.DeposedKey {
		return nil, fmt.Errorf("planned change has inconsistent address to its containing object")
	}
	if !riPlan.ProviderAddr.Equal(providerConfigAddr) {
		return nil, fmt.Errorf("planned change has inconsistent provider configuration address to its containing object")
	}
	return riPlan, nil
}

func LoadComponentForResourceInstance(plan *Plan, change *tfstackdata1.PlanResourceInstanceChangePlanned) (*Component, addrs.AbsResourceInstanceObject, addrs.AbsProviderConfig, error) {
	cAddr, diags := stackaddrs.ParseAbsComponentInstanceStr(change.ComponentInstanceAddr)
	if diags.HasErrors() {
		return nil, addrs.AbsResourceInstanceObject{}, addrs.AbsProviderConfig{}, fmt.Errorf("invalid component instance address syntax in %q", change.ComponentInstanceAddr)
	}

	providerConfigAddr, diags := addrs.ParseAbsProviderConfigStr(change.ProviderConfigAddr)
	if diags.HasErrors() {
		return nil, addrs.AbsResourceInstanceObject{}, addrs.AbsProviderConfig{}, fmt.Errorf("invalid provider configuration address syntax in %q", change.ProviderConfigAddr)
	}

	riAddr, diags := addrs.ParseAbsResourceInstanceStr(change.ResourceInstanceAddr)
	if diags.HasErrors() {
		return nil, addrs.AbsResourceInstanceObject{}, addrs.AbsProviderConfig{}, fmt.Errorf("invalid resource instance address syntax in %q", change.ResourceInstanceAddr)
	}

	var deposedKey addrs.DeposedKey
	if change.DeposedKey != "" {
		var err error
		deposedKey, err = addrs.ParseDeposedKey(change.DeposedKey)
		if err != nil {
			return nil, addrs.AbsResourceInstanceObject{}, addrs.AbsProviderConfig{}, fmt.Errorf("invalid deposed key syntax in %q", change.DeposedKey)
		}
	}
	fullAddr := addrs.AbsResourceInstanceObject{
		ResourceInstance: riAddr,
		DeposedKey:       deposedKey,
	}

	c, ok := plan.Components.GetOk(cAddr)
	if !ok {
		return nil, addrs.AbsResourceInstanceObject{}, addrs.AbsProviderConfig{}, fmt.Errorf("resource instance change for unannounced component instance %s", cAddr)
	}

	return c, fullAddr, providerConfigAddr, nil
}

func LoadComponentForPartialResourceInstance(plan *Plan, change *tfstackdata1.PlanResourceInstanceChangePlanned) (*Component, addrs.AbsResourceInstanceObject, addrs.AbsProviderConfig, error) {
	cAddr, diags := stackaddrs.ParsePartialComponentInstanceStr(change.ComponentInstanceAddr)
	if diags.HasErrors() {
		return nil, addrs.AbsResourceInstanceObject{}, addrs.AbsProviderConfig{}, fmt.Errorf("invalid component instance address syntax in %q", change.ComponentInstanceAddr)
	}

	providerConfigAddr, diags := addrs.ParseAbsProviderConfigStr(change.ProviderConfigAddr)
	if diags.HasErrors() {
		return nil, addrs.AbsResourceInstanceObject{}, addrs.AbsProviderConfig{}, fmt.Errorf("invalid provider configuration address syntax in %q", change.ProviderConfigAddr)
	}

	riAddr, diags := addrs.ParsePartialResourceInstanceStr(change.ResourceInstanceAddr)
	if diags.HasErrors() {
		return nil, addrs.AbsResourceInstanceObject{}, addrs.AbsProviderConfig{}, fmt.Errorf("invalid resource instance address syntax in %q", change.ResourceInstanceAddr)
	}

	var deposedKey addrs.DeposedKey
	if change.DeposedKey != "" {
		var err error
		deposedKey, err = addrs.ParseDeposedKey(change.DeposedKey)
		if err != nil {
			return nil, addrs.AbsResourceInstanceObject{}, addrs.AbsProviderConfig{}, fmt.Errorf("invalid deposed key syntax in %q", change.DeposedKey)
		}
	}
	fullAddr := addrs.AbsResourceInstanceObject{
		ResourceInstance: riAddr,
		DeposedKey:       deposedKey,
	}

	c, ok := plan.Components.GetOk(cAddr)
	if !ok {
		return nil, addrs.AbsResourceInstanceObject{}, addrs.AbsProviderConfig{}, fmt.Errorf("resource instance change for unannounced component instance %s", cAddr)
	}

	return c, fullAddr, providerConfigAddr, nil
}
