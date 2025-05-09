// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackplan

import (
	"iter"
	"time"

	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

// Plan is the main type in this package, representing an entire stack plan,
// or at least the subset of the information that Terraform needs to reliably
// apply the plan and detect any inconsistencies during the apply process.
//
// However, the process of _creating_ a plan doesn't actually produce a single
// object of this type, and instead produces fragments of it gradually as the
// planning process proceeds. The caller of the stack runtime must retain
// all of the raw parts in the order they were emitted and provide them back
// during the apply phase, and then we will finally construct a single instance
// of Plan covering the entire set of changes before we begin applying it.
type Plan struct {
	// Applyable is true for a plan that was successfully created in full and
	// is sufficient to be applied, or false if the plan is incomplete for
	// some reason, such as if an error occurred during planning and so
	// the planning process did not entirely run.
	Applyable bool

	// Complete is true for a plan that shouldn't need any follow-up plans to
	// converge.
	Complete bool

	// Mode is the original mode of the plan.
	Mode plans.Mode

	// Root is the root StackInstance for the configuration being planned.
	// The StackInstance object wraps the specific components for each stack
	// instance.
	Root *StackInstance

	// The raw representation of the raw state that was provided in the request
	// to create the plan. We use this primarily to perform mundane state
	// data structure maintenence operations, such as discarding keys that
	// are no longer needed or replacing data in old formats with the
	// equivalent new representations.
	PrevRunStateRaw map[string]*anypb.Any

	// RootInputValues are the input variable values provided to calculate
	// the plan. We must use the same values during the apply step to
	// sure that the actions taken can be consistent with what was planned.
	RootInputValues map[stackaddrs.InputVariable]cty.Value

	// ApplyTimeInputVariables are the names of the root input variable
	// values whose values must be re-supplied during the apply phase,
	// instead of being persisted in [Plan.RootInputValues].
	ApplyTimeInputVariables collections.Set[stackaddrs.InputVariable]

	// DeletedInputVariables tracks the set of input variables that are being
	// deleted by this plan. The apply operation will miss any values
	// that are not defined in the configuration, but should still emit
	// deletion events to remove them from the state.
	DeletedInputVariables collections.Set[stackaddrs.InputVariable]

	// DeletedOutputValues tracks the set of output values that are being
	// deleted by this plan. The apply operation will miss any output values
	// that are not defined in the configuration, but should still emit
	// deletion events to remove them from the state. Output values not being
	// deleted will be recomputed during the apply so are not needed.
	DeletedOutputValues collections.Set[stackaddrs.OutputValue]

	// DeletedComponents are a set of components that are in the state that
	// should just be removed without any apply operation. This is typically
	// because they are not referenced in the configuration and have no
	// associated resources.
	DeletedComponents collections.Set[stackaddrs.AbsComponentInstance]

	// FunctionResults is a shared table of results from calling
	// provider functions. This is stored and loaded from during the planning
	// stage to use during apply operations.
	FunctionResults []lang.FunctionResultHash

	// PlanTimestamp is the time at which the plan was created.
	PlanTimestamp time.Time
}

func (p *Plan) AllComponents() iter.Seq2[stackaddrs.AbsComponentInstance, *Component] {
	return func(yield func(stackaddrs.AbsComponentInstance, *Component) bool) {
		p.Root.iterate(yield)
	}
}

func (p *Plan) ComponentInstanceAddresses(addr stackaddrs.AbsComponent) iter.Seq[stackaddrs.ComponentInstance] {
	return func(yield func(stackaddrs.ComponentInstance) bool) {
		stack := p.Root.GetDescendentStack(addr.Stack)
		if stack != nil {
			components := stack.Components[addr.Item]
			for key := range components {
				proceed := yield(stackaddrs.ComponentInstance{
					Component: addr.Item,
					Key:       key,
				})
				if !proceed {
					return
				}
			}
		}
	}
}

// ComponentInstances returns a set of the component instances that belong to
// the given component.
func (p *Plan) ComponentInstances(addr stackaddrs.AbsComponent) iter.Seq2[stackaddrs.ComponentInstance, *Component] {
	return func(yield func(stackaddrs.ComponentInstance, *Component) bool) {
		stack := p.Root.GetDescendentStack(addr.Stack)
		if stack != nil {
			components := stack.Components[addr.Item]
			for key, component := range components {
				proceed := yield(stackaddrs.ComponentInstance{
					Component: addr.Item,
					Key:       key,
				}, component)
				if !proceed {
					return
				}
			}
		}
	}
}

func (p *Plan) StackInstances(addr stackaddrs.AbsStackCall) iter.Seq[stackaddrs.StackInstance] {
	return func(yield func(stackaddrs.StackInstance) bool) {
		stack := p.Root.GetDescendentStack(addr.Stack)
		if stack != nil {
			stacks := stack.Children[addr.Item.Name]
			for key := range stacks {
				proceed := yield(append(addr.Stack, stackaddrs.StackInstanceStep{
					Name: addr.Item.Name,
					Key:  key,
				}))
				if !proceed {
					return
				}
			}
		}
	}
}

func (p *Plan) GetOrCreate(addr stackaddrs.AbsComponentInstance, component *Component) *Component {
	targetStackInstance := p.Root.GetOrCreateDescendentStack(addr.Stack)
	return targetStackInstance.GetOrCreateComponent(addr.Item, component)
}

func (p *Plan) GetComponent(addr stackaddrs.AbsComponentInstance) *Component {
	targetStackInstance := p.Root.GetDescendentStack(addr.Stack)
	return targetStackInstance.GetComponent(addr.Item)
}

func (p *Plan) GetStack(addr stackaddrs.StackInstance) *StackInstance {
	return p.Root.GetDescendentStack(addr)
}

// RequiredProviderInstances returns a description of all of the provider
// instance slots that are required to satisfy the resource instances
// belonging to the given component instance.
//
// See also stackeval.ComponentConfig.RequiredProviderInstances for a similar
// function that operates on the configuration of a component instance rather
// than the plan of one.
func (p *Plan) RequiredProviderInstances(addr stackaddrs.AbsComponentInstance) addrs.Set[addrs.RootProviderConfig] {
	stack := p.Root.GetDescendentStack(addr.Stack)
	if stack == nil {
		return addrs.MakeSet[addrs.RootProviderConfig]()
	}

	components, ok := stack.Components[addr.Item.Component]
	if !ok {
		return addrs.MakeSet[addrs.RootProviderConfig]()
	}

	component, ok := components[addr.Item.Key]
	if !ok {
		return addrs.MakeSet[addrs.RootProviderConfig]()
	}
	return component.RequiredProviderInstances()
}

// StackInstance stores the components and embedded stacks for a single stack
// instance.
type StackInstance struct {
	Address    stackaddrs.StackInstance
	Children   map[string]map[addrs.InstanceKey]*StackInstance
	Components map[stackaddrs.Component]map[addrs.InstanceKey]*Component
}

func newStackInstance(address stackaddrs.StackInstance) *StackInstance {
	return &StackInstance{
		Address:    address,
		Components: make(map[stackaddrs.Component]map[addrs.InstanceKey]*Component),
		Children:   make(map[string]map[addrs.InstanceKey]*StackInstance),
	}
}

func (stack *StackInstance) GetComponent(addr stackaddrs.ComponentInstance) *Component {
	components, ok := stack.Components[addr.Component]
	if !ok {
		return nil
	}
	return components[addr.Key]
}

func (stack *StackInstance) GetOrCreateComponent(addr stackaddrs.ComponentInstance, component *Component) *Component {
	components, ok := stack.Components[addr.Component]
	if !ok {
		components = make(map[addrs.InstanceKey]*Component)
	}
	existing, ok := components[addr.Key]
	if ok {
		return existing
	}
	components[addr.Key] = component
	stack.Components[addr.Component] = components
	return component
}

func (stack *StackInstance) GetOrCreateDescendentStack(addr stackaddrs.StackInstance) *StackInstance {
	if len(addr) == 0 {
		return stack
	}
	next := stack.GetOrCreateChildStack(addr[0])
	return next.GetOrCreateDescendentStack(addr[1:])
}

func (stack *StackInstance) GetOrCreateChildStack(step stackaddrs.StackInstanceStep) *StackInstance {
	child := stack.GetChildStack(step)
	if child == nil {
		child = stack.CreateChildStack(step)
	}
	return child
}

func (stack *StackInstance) GetDescendentStack(addr stackaddrs.StackInstance) *StackInstance {
	if len(addr) == 0 {
		return stack
	}

	next := stack.GetChildStack(addr[0])
	if next == nil {
		return nil
	}
	return next.GetDescendentStack(addr[1:])
}

func (stack *StackInstance) GetChildStack(step stackaddrs.StackInstanceStep) *StackInstance {
	insts, ok := stack.Children[step.Name]
	if !ok {
		return nil
	}
	return insts[step.Key]
}

func (stack *StackInstance) CreateChildStack(step stackaddrs.StackInstanceStep) *StackInstance {
	stacks, ok := stack.Children[step.Name]
	if !ok {
		stacks = make(map[addrs.InstanceKey]*StackInstance)
	}
	stacks[step.Key] = newStackInstance(append(stack.Address, step))
	stack.Children[step.Name] = stacks
	return stacks[step.Key]
}

func (stack *StackInstance) GetOk(addr stackaddrs.AbsComponentInstance) (*Component, bool) {
	if len(addr.Stack) == 0 {
		component, ok := stack.Components[addr.Item.Component]
		if !ok {
			return nil, false
		}

		instance, ok := component[addr.Item.Key]
		return instance, ok
	}

	stacks, ok := stack.Children[addr.Stack[0].Name]
	if !ok {
		return nil, false
	}
	next, ok := stacks[addr.Stack[0].Key]
	if !ok {
		return nil, false
	}
	return next.GetOk(stackaddrs.AbsComponentInstance{
		Stack: addr.Stack[1:],
		Item:  addr.Item,
	})
}

func (stack *StackInstance) iterate(yield func(stackaddrs.AbsComponentInstance, *Component) bool) bool {
	for name, components := range stack.Components {
		for key, component := range components {
			proceed := yield(stackaddrs.AbsComponentInstance{
				Stack: stack.Address,
				Item: stackaddrs.ComponentInstance{
					Component: name,
					Key:       key,
				},
			}, component)
			if !proceed {
				return false
			}
		}
	}

	for _, stacks := range stack.Children {
		for _, inst := range stacks {
			proceed := inst.iterate(yield)
			if !proceed {
				return false
			}
		}
	}

	return true
}
