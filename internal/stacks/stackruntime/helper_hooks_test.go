// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
)

type ExpectedHooks struct {
	ComponentExpanded                       []*hooks.ComponentInstances
	RemovedComponentExpanded                []*hooks.RemovedComponentInstances
	PendingComponentInstancePlan            collections.Set[stackaddrs.AbsComponentInstance]
	BeginComponentInstancePlan              collections.Set[stackaddrs.AbsComponentInstance]
	EndComponentInstancePlan                collections.Set[stackaddrs.AbsComponentInstance]
	ErrorComponentInstancePlan              collections.Set[stackaddrs.AbsComponentInstance]
	DeferComponentInstancePlan              collections.Set[stackaddrs.AbsComponentInstance]
	PendingComponentInstanceApply           collections.Set[stackaddrs.AbsComponentInstance]
	BeginComponentInstanceApply             collections.Set[stackaddrs.AbsComponentInstance]
	EndComponentInstanceApply               collections.Set[stackaddrs.AbsComponentInstance]
	ErrorComponentInstanceApply             collections.Set[stackaddrs.AbsComponentInstance]
	ReportResourceInstanceStatus            []*hooks.ResourceInstanceStatusHookData
	ReportResourceInstanceProvisionerStatus []*hooks.ResourceInstanceProvisionerHookData
	ReportResourceInstanceDrift             []*hooks.ResourceInstanceChange
	ReportResourceInstancePlanned           []*hooks.ResourceInstanceChange
	ReportResourceInstanceDeferred          []*hooks.DeferredResourceInstanceChange
	ReportComponentInstancePlanned          []*hooks.ComponentInstanceChange
	ReportComponentInstanceApplied          []*hooks.ComponentInstanceChange
}

func (eh *ExpectedHooks) Validate(t *testing.T, expectedHooks *ExpectedHooks) {
	sort.SliceStable(expectedHooks.ComponentExpanded, func(i, j int) bool {
		return expectedHooks.ComponentExpanded[i].ComponentAddr.String() < expectedHooks.ComponentExpanded[j].ComponentAddr.String()
	})
	sort.SliceStable(expectedHooks.RemovedComponentExpanded, func(i, j int) bool {
		return expectedHooks.RemovedComponentExpanded[i].Source.String() < expectedHooks.RemovedComponentExpanded[j].Source.String()
	})
	sort.SliceStable(expectedHooks.ReportResourceInstanceStatus, func(i, j int) bool {
		return expectedHooks.ReportResourceInstanceStatus[i].Addr.String() < expectedHooks.ReportResourceInstanceStatus[j].Addr.String()
	})
	sort.SliceStable(expectedHooks.ReportResourceInstanceProvisionerStatus, func(i, j int) bool {
		return expectedHooks.ReportResourceInstanceProvisionerStatus[i].Addr.String() < expectedHooks.ReportResourceInstanceProvisionerStatus[j].Addr.String()
	})
	sort.SliceStable(expectedHooks.ReportResourceInstanceDrift, func(i, j int) bool {
		return expectedHooks.ReportResourceInstanceDrift[i].Addr.String() < expectedHooks.ReportResourceInstanceDrift[j].Addr.String()
	})
	sort.SliceStable(expectedHooks.ReportResourceInstancePlanned, func(i, j int) bool {
		return expectedHooks.ReportResourceInstancePlanned[i].Addr.String() < expectedHooks.ReportResourceInstancePlanned[j].Addr.String()
	})
	sort.SliceStable(expectedHooks.ReportResourceInstanceDeferred, func(i, j int) bool {
		return expectedHooks.ReportResourceInstanceDeferred[i].Change.Addr.String() < expectedHooks.ReportResourceInstanceDeferred[j].Change.Addr.String()
	})
	sort.SliceStable(expectedHooks.ReportComponentInstancePlanned, func(i, j int) bool {
		return expectedHooks.ReportComponentInstancePlanned[i].Addr.String() < expectedHooks.ReportComponentInstancePlanned[j].Addr.String()
	})
	sort.SliceStable(expectedHooks.ReportComponentInstanceApplied, func(i, j int) bool {
		return expectedHooks.ReportComponentInstanceApplied[i].Addr.String() < expectedHooks.ReportComponentInstanceApplied[j].Addr.String()
	})

	if diff := cmp.Diff(expectedHooks.ComponentExpanded, eh.ComponentExpanded); len(diff) > 0 {
		t.Errorf("wrong ComponentExpanded hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.RemovedComponentExpanded, eh.RemovedComponentExpanded); len(diff) > 0 {
		t.Errorf("wrong RemovedComponentExpanded hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.PendingComponentInstancePlan, eh.PendingComponentInstancePlan, collections.CmpOptions); len(diff) > 0 {
		t.Errorf("wrong PendingComponentInstancePlan hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.BeginComponentInstancePlan, eh.BeginComponentInstancePlan, collections.CmpOptions); len(diff) > 0 {
		t.Errorf("wrong BeginComponentInstancePlan hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.EndComponentInstancePlan, eh.EndComponentInstancePlan, collections.CmpOptions); len(diff) > 0 {
		t.Errorf("wrong EndComponentInstancePlan hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.ErrorComponentInstancePlan, eh.ErrorComponentInstancePlan, collections.CmpOptions); len(diff) > 0 {
		t.Errorf("wrong ErrorComponentInstancePlan hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.DeferComponentInstancePlan, eh.DeferComponentInstancePlan, collections.CmpOptions); len(diff) > 0 {
		t.Errorf("wrong DeferComponentInstancePlan hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.PendingComponentInstanceApply, eh.PendingComponentInstanceApply, collections.CmpOptions); len(diff) > 0 {
		t.Errorf("wrong PendingComponentInstanceApply hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.BeginComponentInstanceApply, eh.BeginComponentInstanceApply, collections.CmpOptions); len(diff) > 0 {
		t.Errorf("wrong BeginComponentInstanceApply hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.EndComponentInstanceApply, eh.EndComponentInstanceApply, collections.CmpOptions); len(diff) > 0 {
		t.Errorf("wrong EndComponentInstanceApply hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.ErrorComponentInstanceApply, eh.ErrorComponentInstanceApply, collections.CmpOptions); len(diff) > 0 {
		t.Errorf("wrong ErrorComponentInstanceApply hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.ReportResourceInstanceStatus, eh.ReportResourceInstanceStatus); len(diff) > 0 {
		t.Errorf("wrong ReportResourceInstanceStatus hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.ReportResourceInstanceProvisionerStatus, eh.ReportResourceInstanceProvisionerStatus); len(diff) > 0 {
		t.Errorf("wrong ReportResourceInstanceProvisionerStatus hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.ReportResourceInstanceDrift, eh.ReportResourceInstanceDrift); len(diff) > 0 {
		t.Errorf("wrong ReportResourceInstanceDrift hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.ReportResourceInstancePlanned, eh.ReportResourceInstancePlanned); len(diff) > 0 {
		t.Errorf("wrong ReportResourceInstancePlanned hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.ReportResourceInstanceDeferred, eh.ReportResourceInstanceDeferred); len(diff) > 0 {
		t.Errorf("wrong ReportResourceInstanceDeferred hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.ReportComponentInstancePlanned, eh.ReportComponentInstancePlanned); len(diff) > 0 {
		t.Errorf("wrong ReportComponentInstancePlanned hooks: %s", diff)
	}
	if diff := cmp.Diff(expectedHooks.ReportComponentInstanceApplied, eh.ReportComponentInstanceApplied); len(diff) > 0 {
		t.Errorf("wrong ReportComponentInstanceApplied hooks: %s", diff)
	}
}

type CapturedHooks struct {
	ExpectedHooks

	sync.Mutex
	Planning bool
}

func NewCapturedHooks(planning bool) *CapturedHooks {
	return &CapturedHooks{
		Planning: planning,
		ExpectedHooks: ExpectedHooks{
			PendingComponentInstancePlan:  collections.NewSet[stackaddrs.AbsComponentInstance](),
			BeginComponentInstancePlan:    collections.NewSet[stackaddrs.AbsComponentInstance](),
			EndComponentInstancePlan:      collections.NewSet[stackaddrs.AbsComponentInstance](),
			ErrorComponentInstancePlan:    collections.NewSet[stackaddrs.AbsComponentInstance](),
			DeferComponentInstancePlan:    collections.NewSet[stackaddrs.AbsComponentInstance](),
			PendingComponentInstanceApply: collections.NewSet[stackaddrs.AbsComponentInstance](),
			BeginComponentInstanceApply:   collections.NewSet[stackaddrs.AbsComponentInstance](),
			EndComponentInstanceApply:     collections.NewSet[stackaddrs.AbsComponentInstance](),
			ErrorComponentInstanceApply:   collections.NewSet[stackaddrs.AbsComponentInstance](),
		},
	}
}

func (ch *CapturedHooks) ComponentInstancePending(addr stackaddrs.AbsComponentInstance) bool {
	if ch.Planning {
		return ch.PendingComponentInstancePlan.Has(addr)
	}
	return ch.PendingComponentInstanceApply.Has(addr)
}

func (ch *CapturedHooks) ComponentInstanceBegun(addr stackaddrs.AbsComponentInstance) bool {
	if ch.Planning {
		return ch.BeginComponentInstancePlan.Has(addr)
	}
	return ch.BeginComponentInstanceApply.Has(addr)
}

func (ch *CapturedHooks) ComponentInstanceFinished(addr stackaddrs.AbsComponentInstance) bool {
	if ch.Planning {
		return ch.EndComponentInstancePlan.Has(addr) || ch.ErrorComponentInstancePlan.Has(addr) || ch.DeferComponentInstancePlan.Has(addr)
	}
	return ch.EndComponentInstanceApply.Has(addr) || ch.ErrorComponentInstanceApply.Has(addr)
}

func (ch *CapturedHooks) captureHooks() *Hooks {
	return &Hooks{
		ComponentExpanded: func(ctx context.Context, instances *hooks.ComponentInstances) {
			ch.Lock()
			defer ch.Unlock()
			ch.ComponentExpanded = append(ch.ComponentExpanded, instances)
		},
		RemovedComponentExpanded: func(ctx context.Context, instances *hooks.RemovedComponentInstances) {
			ch.Lock()
			defer ch.Unlock()
			ch.RemovedComponentExpanded = append(ch.RemovedComponentExpanded, instances)
		},
		PendingComponentInstancePlan: func(ctx context.Context, instance stackaddrs.AbsComponentInstance) {
			ch.Lock()
			defer ch.Unlock()
			if ch.ComponentInstancePending(instance) {
				panic("tried to add pending component instance plan twice")
			}
			ch.PendingComponentInstancePlan.Add(instance)
		},
		BeginComponentInstancePlan: func(ctx context.Context, instance stackaddrs.AbsComponentInstance) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstancePending(instance) {
				panic("tried to begin component instance plan before ending")
			}

			if ch.ComponentInstanceBegun(instance) {
				panic("tried to add begin component instance plan twice")
			}
			ch.BeginComponentInstancePlan.Add(instance)
			return nil
		},
		EndComponentInstancePlan: func(ctx context.Context, a any, instance stackaddrs.AbsComponentInstance) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.BeginComponentInstancePlan.Has(instance) {
				panic("tried to end component instance plan before beginning")
			}

			if ch.EndComponentInstancePlan.Has(instance) || ch.ErrorComponentInstancePlan.Has(instance) || ch.DeferComponentInstancePlan.Has(instance) {
				panic("tried to add end component instance plan twice")
			}
			ch.EndComponentInstancePlan.Add(instance)
			return a
		},
		ErrorComponentInstancePlan: func(ctx context.Context, a any, instance stackaddrs.AbsComponentInstance) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstanceBegun(instance) {
				panic("tried to end component instance plan before beginning")
			}

			if ch.ComponentInstanceFinished(instance) {
				panic("tried to add end component instance plan twice")
			}
			ch.ErrorComponentInstancePlan.Add(instance)
			return a
		},
		DeferComponentInstancePlan: func(ctx context.Context, a any, instance stackaddrs.AbsComponentInstance) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstanceBegun(instance) {
				panic("tried to end component instance plan before beginning")
			}

			if ch.ComponentInstanceFinished(instance) {
				panic("tried to add end component instance plan twice")
			}
			ch.DeferComponentInstancePlan.Add(instance)
			return a
		},
		PendingComponentInstanceApply: func(ctx context.Context, instance stackaddrs.AbsComponentInstance) {
			ch.Lock()
			defer ch.Unlock()

			if ch.ComponentInstancePending(instance) {
				panic("tried to add pending component instance apply twice")
			}
			ch.PendingComponentInstanceApply.Add(instance)
		},
		BeginComponentInstanceApply: func(ctx context.Context, instance stackaddrs.AbsComponentInstance) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstancePending(instance) {
				panic("tried to begin component before pending")
			}

			if ch.ComponentInstanceBegun(instance) {
				panic("tried to add begin component instance apply twice")
			}
			ch.BeginComponentInstanceApply.Add(instance)
			return nil
		},
		EndComponentInstanceApply: func(ctx context.Context, a any, instance stackaddrs.AbsComponentInstance) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstanceBegun(instance) {
				panic("tried to end component before beginning")
			}

			if ch.ComponentInstanceFinished(instance) {
				panic("tried to add end component instance apply twice")
			}
			ch.EndComponentInstanceApply.Add(instance)
			return a
		},
		ErrorComponentInstanceApply: func(ctx context.Context, a any, instance stackaddrs.AbsComponentInstance) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstanceBegun(instance) {
				panic("tried to end component before beginning")
			}

			if ch.ComponentInstanceFinished(instance) {
				panic("tried to add error component instance apply twice")
			}
			ch.ErrorComponentInstanceApply.Add(instance)
			return a
		},
		ReportResourceInstanceStatus: func(ctx context.Context, a any, data *hooks.ResourceInstanceStatusHookData) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstanceBegun(data.Addr.Component) {
				panic("tried to report resource instance status before component")
			}

			if ch.ComponentInstanceFinished(data.Addr.Component) {
				panic("tried to report resource instance status after component")
			}

			ch.ReportResourceInstanceStatus = append(ch.ReportResourceInstanceStatus, data)
			return a
		},
		ReportResourceInstanceProvisionerStatus: func(ctx context.Context, a any, data *hooks.ResourceInstanceProvisionerHookData) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstanceBegun(data.Addr.Component) {
				panic("tried to report resource instance provisioner status before component")
			}

			if ch.ComponentInstanceFinished(data.Addr.Component) {
				panic("tried to report resource instance provisioner status after component")
			}

			ch.ReportResourceInstanceProvisionerStatus = append(ch.ReportResourceInstanceProvisionerStatus, data)
			return a
		},
		ReportResourceInstanceDrift: func(ctx context.Context, a any, change *hooks.ResourceInstanceChange) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstanceBegun(change.Addr.Component) {
				panic("tried to report resource instance drift before component")
			}

			if ch.ComponentInstanceFinished(change.Addr.Component) {
				panic("tried to report resource instance drift after component")
			}

			ch.ReportResourceInstanceDrift = append(ch.ReportResourceInstanceDrift, change)
			return a
		},
		ReportResourceInstancePlanned: func(ctx context.Context, a any, change *hooks.ResourceInstanceChange) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstanceBegun(change.Addr.Component) {
				panic("tried to report resource instance planned before component")
			}

			if ch.ComponentInstanceFinished(change.Addr.Component) {
				panic("tried to report resource instance planned after component")
			}

			ch.ReportResourceInstancePlanned = append(ch.ReportResourceInstancePlanned, change)
			return a
		},
		ReportResourceInstanceDeferred: func(ctx context.Context, a any, change *hooks.DeferredResourceInstanceChange) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstanceBegun(change.Change.Addr.Component) {
				panic("tried to report resource instance deferred before component")
			}

			if ch.ComponentInstanceFinished(change.Change.Addr.Component) {
				panic("tried to report resource instance deferred after component")
			}

			ch.ReportResourceInstanceDeferred = append(ch.ReportResourceInstanceDeferred, change)
			return a
		},
		ReportComponentInstancePlanned: func(ctx context.Context, a any, change *hooks.ComponentInstanceChange) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstanceBegun(change.Addr) {
				panic("tried to report component instance planned before component")
			}

			if ch.ComponentInstanceFinished(change.Addr) {
				panic("tried to report component instance planned after component")
			}

			ch.ReportComponentInstancePlanned = append(ch.ReportComponentInstancePlanned, change)
			return a
		},
		ReportComponentInstanceApplied: func(ctx context.Context, a any, change *hooks.ComponentInstanceChange) any {
			ch.Lock()
			defer ch.Unlock()

			if !ch.ComponentInstanceBegun(change.Addr) {
				panic("tried to report component instance planned before component")
			}

			if ch.ComponentInstanceFinished(change.Addr) {
				panic("tried to report component instance planned after component")
			}

			ch.ReportComponentInstanceApplied = append(ch.ReportComponentInstanceApplied, change)
			return a
		},
	}
}
