// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"github.com/hashicorp/terraform/internal/providers"
)

// DeferredResourceInstanceChangeSrc tracks information about a resource that
// has been deferred for some reason.
type DeferredResourceInstanceChangeSrc struct {
	// DeferredReason is the reason why this resource instance was deferred.
	DeferredReason providers.DeferredReason

	// ChangeSrc contains any information we have about the deferred change.
	// This could be incomplete so must be parsed with care.
	ChangeSrc *ResourceInstanceChangeSrc
}

func (rcs *DeferredResourceInstanceChangeSrc) Decode(schema providers.Schema) (*DeferredResourceInstanceChange, error) {
	change, err := rcs.ChangeSrc.Decode(schema)
	if err != nil {
		return nil, err
	}

	return &DeferredResourceInstanceChange{
		DeferredReason: rcs.DeferredReason,
		Change:         change,
	}, nil
}

// DeferredResourceInstanceChange tracks information about a resource that
// has been deferred for some reason.
type DeferredResourceInstanceChange struct {
	// DeferredReason is the reason why this resource instance was deferred.
	DeferredReason providers.DeferredReason

	// Change contains any information we have about the deferred change. This
	// could be incomplete so must be parsed with care.
	Change *ResourceInstanceChange
}

func (rcs *DeferredResourceInstanceChange) Encode(schema providers.Schema) (*DeferredResourceInstanceChangeSrc, error) {
	change, err := rcs.Change.Encode(schema)
	if err != nil {
		return nil, err
	}

	return &DeferredResourceInstanceChangeSrc{
		DeferredReason: rcs.DeferredReason,
		ChangeSrc:      change,
	}, nil
}

// DeferredActionInvocation tracks information about an action invocation
// that has been deferred for some reason.
type DeferredActionInvocation struct {
	// DeferredReason is the reason why this action invocation was deferred.
	DeferredReason providers.DeferredReason

	// ActionInvocationInstance is the instance of the action invocation that was deferred.
	ActionInvocationInstance *ActionInvocationInstance
}

func (dai *DeferredActionInvocation) Encode(schema *providers.ActionSchema) (*DeferredActionInvocationSrc, error) {
	src, err := dai.ActionInvocationInstance.Encode(schema)
	if err != nil {
		return nil, err
	}

	return &DeferredActionInvocationSrc{
		DeferredReason:              dai.DeferredReason,
		ActionInvocationInstanceSrc: src,
	}, nil
}

// DeferredActionInvocationSrc tracks information about an action invocation
// that has been deferred for some reason.
type DeferredActionInvocationSrc struct {
	// DeferredReason is the reason why this action invocation was deferred.
	DeferredReason providers.DeferredReason

	// ActionInvocationInstanceSrc is the instance of the action invocation that was deferred.
	ActionInvocationInstanceSrc *ActionInvocationInstanceSrc
}

func (dais *DeferredActionInvocationSrc) Decode(schema *providers.ActionSchema) (*DeferredActionInvocation, error) {
	instance, err := dais.ActionInvocationInstanceSrc.Decode(schema)
	if err != nil {
		return nil, err
	}

	return &DeferredActionInvocation{
		DeferredReason:           dais.DeferredReason,
		ActionInvocationInstance: instance,
	}, nil
}

// DeferredPartialExpandedActionInvocation tracks information about an action
// invocation that has been deferred for some reason, where the underlying
// ActionInvocationInstance contains a partially expanded address (and
// LifecycleActionTrigger).
type DeferredPartialExpandedActionInvocation struct {
	// DeferredReason is the reason why this action invocation was deferred.
	DeferredReason providers.DeferredReason

	// ActionInvocationInstance is the (partially expanded) instance of the action
	// invocation that was deferred. Its Addr (and any embedded
	// LifecycleActionTrigger addresses) are partial.
	ActionInvocationInstance *PartialExpandedActionInvocationInstance
}

func (dai *DeferredPartialExpandedActionInvocation) Encode(schema *providers.ActionSchema) (*DeferredPartialExpandedActionInvocationSrc, error) {
	src, err := dai.ActionInvocationInstance.Encode(schema)
	if err != nil {
		return nil, err
	}

	return &DeferredPartialExpandedActionInvocationSrc{
		DeferredReason:              dai.DeferredReason,
		ActionInvocationInstanceSrc: src,
	}, nil
}

// DeferredPartialExpandedActionInvocationSrc is the serialized form of
// DeferredPartialExpandedActionInvocation.
type DeferredPartialExpandedActionInvocationSrc struct {
	// DeferredReason is the reason why this action invocation was deferred.
	DeferredReason providers.DeferredReason

	// ActionInvocationInstanceSrc is the (partially expanded) instance of the
	// action invocation that was deferred. Its Addr (and any embedded
	// LifecycleActionTrigger addresses) are partial.
	ActionInvocationInstanceSrc *PartialExpandedActionInvocationInstanceSrc
}

func (dais *DeferredPartialExpandedActionInvocationSrc) Decode(schema *providers.ActionSchema) (*DeferredPartialExpandedActionInvocation, error) {
	instance, err := dais.ActionInvocationInstanceSrc.Decode(schema)
	if err != nil {
		return nil, err
	}

	return &DeferredPartialExpandedActionInvocation{
		DeferredReason:           dais.DeferredReason,
		ActionInvocationInstance: instance,
	}, nil
}
