// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backendrun

import (
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Local implements additional behavior on a Backend that allows local
// operations in addition to remote operations.
//
// This enables more behaviors of Terraform that require more data such
// as `console`, `import`, `graph`. These require direct access to
// configurations, variables, and more. Not all backends may support this
// so we separate it out into its own optional interface.
type Local interface {
	// LocalRun uses information in the Operation to prepare a set of objects
	// needed to start running that operation.
	//
	// The operation doesn't need a Type set, but it needs various other
	// options set. This is a rather odd API that tries to treat all
	// operations as the same when they really aren't; see the local and remote
	// backend's implementations of this to understand what this actually
	// does, because this operation has no well-defined contract aside from
	// "whatever it already does".
	LocalRun(*Operation) (*LocalRun, statemgr.Full, tfdiags.Diagnostics)
}

// LocalRun represents the assortment of objects that we can collect or
// calculate from an Operation object, which we can then use for local
// operations.
//
// The operation methods on terraform.Context (Plan, Apply, Import, etc) each
// generate new artifacts which supersede parts of the LocalRun object that
// started the operation, so callers should be careful to use those subsequent
// artifacts instead of the fields of LocalRun where appropriate. The LocalRun
// data intentionally doesn't update as a result of calling methods on Context,
// in order to make data flow explicit.
//
// This type is a weird architectural wart resulting from the overly-general
// way our backend API models operations, whereby we behave as if all
// Terraform operations have the same inputs and outputs even though they
// are actually all rather different. The exact meaning of the fields in
// this type therefore vary depending on which OperationType was passed to
// Local.Context in order to create an object of this type.
type LocalRun struct {
	// Core is an already-initialized Terraform Core context, ready to be
	// used to run operations such as Plan and Apply.
	Core *terraform.Context

	// Config is the configuration we're working with, which typically comes
	// from either config files directly on local disk (when we're creating
	// a plan, or similar) or from a snapshot embedded in a plan file
	// (when we're applying a saved plan).
	Config *configs.Config

	// InputState is the state that should be used for whatever is the first
	// method call to a context created with CoreOpts. When creating a plan
	// this will be the previous run state, but when applying a saved plan
	// this will be the prior state recorded in that plan.
	InputState *states.State

	// PlanOpts are options to pass to a Plan or Plan-like operation.
	//
	// This is nil when we're applying a saved plan, because the plan itself
	// contains enough information about its options to apply it.
	PlanOpts *terraform.PlanOpts

	// Plan is a plan loaded from a saved plan file, if our operation is to
	// apply that saved plan.
	//
	// This is nil when we're not applying a saved plan.
	Plan *plans.Plan
}
