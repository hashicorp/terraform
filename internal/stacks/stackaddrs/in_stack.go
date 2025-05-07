// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StackItemConfig is a type set containing all of the address types that make
// sense to consider as belonging statically to a [Stack].
type StackItemConfig[T any] interface {
	inStackConfigSigil()
	String() string
	collections.UniqueKeyer[T]
}

// StackItemDynamic is a type set containing all of the address types that make
// sense to consider as belonging dynamically to a [StackInstance].
type StackItemDynamic[T any] interface {
	inStackInstanceSigil()
	String() string
	collections.UniqueKeyer[T]
}

// InStackConfig is the generic form of addresses representing configuration
// objects belonging to particular nodes in the static tree of stack
// configurations.
type InStackConfig[T StackItemConfig[T]] struct {
	Stack Stack
	Item  T
}

func Config[T StackItemConfig[T]](stackAddr Stack, relAddr T) InStackConfig[T] {
	return InStackConfig[T]{
		Stack: stackAddr,
		Item:  relAddr,
	}
}

func (ist InStackConfig[T]) String() string {
	if ist.Stack.IsRoot() {
		return ist.Item.String()
	}
	return ist.Stack.String() + "." + ist.Item.String()
}

func (ist InStackConfig[T]) UniqueKey() collections.UniqueKey[InStackConfig[T]] {
	return inStackConfigKey[T]{
		stackKey: ist.Stack.UniqueKey(),
		itemKey:  ist.Item.UniqueKey(),
	}
}

type inStackConfigKey[T StackItemConfig[T]] struct {
	stackKey collections.UniqueKey[Stack]
	itemKey  collections.UniqueKey[T]
}

// IsUniqueKey implements collections.UniqueKey.
func (inStackConfigKey[T]) IsUniqueKey(InStackConfig[T]) {}

// InStackInstance is the generic form of addresses representing dynamic
// instances of objects that exist within an instance of a stack.
type InStackInstance[T StackItemDynamic[T]] struct {
	Stack StackInstance
	Item  T
}

func Absolute[T StackItemDynamic[T]](stackAddr StackInstance, relAddr T) InStackInstance[T] {
	return InStackInstance[T]{
		Stack: stackAddr,
		Item:  relAddr,
	}
}

func (ist InStackInstance[T]) String() string {
	if ist.Stack.IsRoot() {
		return ist.Item.String()
	}
	return ist.Stack.String() + "." + ist.Item.String()
}

func (ist InStackInstance[T]) UniqueKey() collections.UniqueKey[InStackInstance[T]] {
	return inStackInstanceKey[T]{
		stackKey: ist.Stack.UniqueKey(),
		itemKey:  ist.Item.UniqueKey(),
	}
}

type inStackInstanceKey[T StackItemDynamic[T]] struct {
	stackKey collections.UniqueKey[StackInstance]
	itemKey  collections.UniqueKey[T]
}

// IsUniqueKey implements collections.UniqueKey.
func (inStackInstanceKey[T]) IsUniqueKey(InStackInstance[T]) {}

// ConfigForAbs returns the "in stack config" equivalent of the given
// "in stack instance" (absolute) address by just discarding any
// instance keys from the stack instance steps.
func ConfigForAbs[T interface {
	StackItemDynamic[T]
	StackItemConfig[T]
}](absAddr InStackInstance[T]) InStackConfig[T] {
	return Config(absAddr.Stack.ConfigAddr(), absAddr.Item)
}

// parseInStackInstancePrefix parses as many nested stack traversal steps
// as possible from the start of the given traversal, and then returns
// the resulting StackInstance address along with a relative traversal
// covering all of the remaining traversal steps, if any.
func parseInStackInstancePrefix(traversal hcl.Traversal) (StackInstance, hcl.Traversal, tfdiags.Diagnostics) {
	if len(traversal) == 0 {
		return RootStackInstance, nil, nil
	}

	const errSummary = "Invalid stack instance address"
	var diags tfdiags.Diagnostics
	var stackInst StackInstance
Steps:
	for len(traversal) > 0 {
		switch step := traversal[0].(type) {
		case hcl.TraverseRoot:
			if step.Name != "stack" {
				break Steps
			}
		case hcl.TraverseAttr:
			if step.Name != "stack" {
				break Steps
			}
		default:
			break Steps
		}

		// If we get here then we know that we're expecting a valid
		// stack instance step prefix, which always consists of the
		// literal step "stack" (which we found above) followed
		// by an embedded stack name. That might then be followed
		// by one optional index step for a multi-instance embedded stack.
		if len(traversal) < 2 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errSummary,
				Detail:   "The \"stack\" keyword must be followed by an attribute specifying the name of the embedded stack.",
				Subject:  traversal.SourceRange().Ptr(),
			})
			return nil, nil, diags
		}
		nameStep, ok := traversal[1].(hcl.TraverseAttr)
		if !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errSummary,
				Detail:   "The \"stack\" keyword must be followed by an attribute specifying the name of the embedded stack.",
				Subject:  traversal[1].SourceRange().Ptr(),
			})
			return nil, nil, diags
		}
		if !hclsyntax.ValidIdentifier(nameStep.Name) {
			// This check is redundant since the HCL parser should've caught
			// an invalid identifier while parsing this traversal, but this
			// is here for robustness in case we obtained this traversal
			// value in an unusual way.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errSummary,
				Detail:   "A stack name must be a valid identifier.",
				Subject:  nameStep.SourceRange().Ptr(),
			})
			return nil, nil, diags
		}
		addrStep := StackInstanceStep{
			Name: nameStep.Name,
			Key:  addrs.NoKey,
		}
		traversal = traversal[2:] // consume the first two steps that we already dealt with
		if len(traversal) > 0 {
			idxStep, ok := traversal[0].(hcl.TraverseIndex)
			if ok {
				var err error
				addrStep.Key, err = addrs.ParseInstanceKey(idxStep.Key)
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  errSummary,
						Detail:   fmt.Sprintf("Invalid instance key: %s.", err),
						Subject:  idxStep.SourceRange().Ptr(),
					})
					return nil, nil, diags
				}
				traversal = traversal[1:] // consume the step we just dealt with
			}
		}
		stackInst = append(stackInst, addrStep)
	}
	return stackInst, forceTraversalRelative(traversal), diags
}

// forceTraversalRelative takes any traversal and if it's absolute transforms
// it into a relative one by changing the first step from a TraverseRoot
// to an equivalent TraverseAttr.
func forceTraversalRelative(given hcl.Traversal) hcl.Traversal {
	if len(given) == 0 {
		return nil
	}
	firstStep, ok := given[0].(hcl.TraverseRoot)
	if !ok {
		return given
	}

	// If we get here then we have an absolute traversal. We shouldn't
	// mutate the backing array of the traversal because others might
	// still be using it, so we'll allocate a new traversal and copy
	// the steps into it.
	ret := make(hcl.Traversal, len(given))
	ret[0] = hcl.TraverseAttr{
		Name:     firstStep.Name,
		SrcRange: firstStep.SrcRange,
	}
	copy(ret[1:], given[1:])
	return ret
}
