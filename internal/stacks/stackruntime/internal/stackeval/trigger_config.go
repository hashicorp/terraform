// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"time"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ Validatable     = (*TriggerConfig)(nil)
	_ Plannable       = (*TriggerConfig)(nil)
	_ ExpressionScope = (*TriggerConfig)(nil)
)

type TriggerConfig struct {
	addr   stackaddrs.ConfigTrigger
	config *stackconfig.Trigger

	main *Main

	validate   promising.Once[tfdiags.Diagnostics]
	moduleTree promising.Once[withDiagnostics[*configs.Config]]
}

func newTriggerConfig(main *Main, addr stackaddrs.ConfigTrigger, config *stackconfig.Trigger) *TriggerConfig {
	return &TriggerConfig{
		addr:   addr,
		config: config,
		main:   main,
	}
}

func (c *TriggerConfig) Addr() stackaddrs.ConfigTrigger {
	return c.addr
}

func (c *TriggerConfig) Declaration(ctx context.Context) *stackconfig.Trigger {
	return c.config
}

func (c *TriggerConfig) DeclRange(_ context.Context) *hcl.Range {
	return c.config.DeclRange.ToHCL().Ptr()
}

func (c *TriggerConfig) StackConfig(ctx context.Context) *StackConfig {
	return c.main.mustStackConfig(ctx, c.addr.Stack)
}

func (c *TriggerConfig) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	// TODO: Validate check expressions against a fake context to check for errors
	return nil
}

// Validate implements Validatable.
func (c *TriggerConfig) Validate(ctx context.Context) tfdiags.Diagnostics {
	return c.checkValid(ctx, ValidatePhase)
}

// PlanChanges implements Plannable.
func (c *TriggerConfig) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, c.checkValid(ctx, PlanPhase)
}

func (c *TriggerConfig) tracingName() string {
	// TODO: Fix this
	// return c.Addr().String()
	return "trigger"
}

// reportNamedPromises implements namedPromiseReporter.
func (c *TriggerConfig) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	cb(c.validate.PromiseID(), c.Addr().String())
	cb(c.moduleTree.PromiseID(), c.Addr().String()+" modules")
}

// ResolveExpressionReference implements ExpressionScope.
func (c *TriggerConfig) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	repetition := instances.RepetitionData{}
	return c.StackConfig(ctx).resolveExpressionReference(ctx, ref, nil, repetition)
}

// ExternalFunctions implements ExpressionScope.
func (c *TriggerConfig) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return c.main.ProviderFunctions(ctx, c.StackConfig(ctx))
}

// PlanTimestamp implements ExpressionScope, providing the timestamp at which
// the current plan is being run.
func (c *TriggerConfig) PlanTimestamp() time.Time {
	return c.main.PlanTimestamp()
}
