// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package deferring

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/zclconf/go-cty/cty"
)

func TestDeferred_externalDependency(t *testing.T) {
	// The resource graph is irrelevant for this case, because we're going to
	// defer any resource instance changes regardless. Therefore an empty
	// graph is just fine.
	resourceGraph := addrs.NewDirectedGraph[addrs.ConfigResource]()
	deferred := NewDeferred(resourceGraph)

	// This reports that something outside of the modules runtime knows that
	// everything in this configuration depends on some elsewhere-action
	// that has been deferred, and so the modules runtime must respect that
	// even though it doesn't know the details of why it is so.
	deferred.SetExternalDependencyDeferred()

	// With the above flag set, now ShouldDeferResourceInstanceChanges should
	// return true regardless of any other information.
	got := deferred.ShouldDeferResourceInstanceChanges(addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance,
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "anything",
				Name: "really-anything",
			},
		},
	})
	if !got {
		t.Errorf("did not report that the instance should have its changes deferred; should have")
	}
}

func TestDeferred_absResourceInstanceDeferred(t *testing.T) {
	instAAddr := addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance.Child("foo", addrs.NoKey),
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test",
				Name: "a",
			},
		},
	}
	instBAddr := addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance,
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test",
				Name: "a",
			},
		},
	}
	instCAddr := addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance,
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test",
				Name: "c",
			},
		},
	}

	resourceGraph := addrs.NewDirectedGraph[addrs.ConfigResource]()
	resourceGraph.AddDependency(instCAddr.ConfigResource(), instBAddr.ConfigResource())
	resourceGraph.AddDependency(instCAddr.ConfigResource(), instAAddr.ConfigResource())
	deferred := NewDeferred(resourceGraph)

	// Before we report anything, all three addresses should indicate that
	// they don't need to have their actions deferred.
	t.Run("without any deferrals yet", func(t *testing.T) {
		for _, instAddr := range []addrs.AbsResourceInstance{instAAddr, instBAddr, instCAddr} {
			if deferred.ShouldDeferResourceInstanceChanges(instAddr) {
				t.Errorf("%s reported as needing deferred; should not be, yet", instAddr)
			}
		}
	})

	// Instance A has its Create action deferred for some reason.
	deferred.ReportResourceInstanceDeferred(instAAddr, plans.Create, cty.DynamicVal)

	t.Run("with one resource instance deferred", func(t *testing.T) {
		if !deferred.ShouldDeferResourceInstanceChanges(instCAddr) {
			t.Errorf("%s was not reported as needing deferred; should be deferred", instCAddr)
		}
		if deferred.ShouldDeferResourceInstanceChanges(instBAddr) {
			t.Errorf("%s reported as needing deferred; should not be", instCAddr)
		}
	})
}

func TestDeferred_partialExpandedResource(t *testing.T) {
	instAAddr := addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance.Child("foo", addrs.NoKey),
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test",
				Name: "a",
			},
		},
	}
	instBAddr := addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance,
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test",
				Name: "a",
			},
		},
	}
	instCAddr := addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance,
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test",
				Name: "c",
			},
		},
	}
	instAPartial := addrs.RootModuleInstance.
		UnexpandedChild(addrs.ModuleCall{Name: "foo"}).
		Resource(instAAddr.Resource.Resource)

	resourceGraph := addrs.NewDirectedGraph[addrs.ConfigResource]()
	resourceGraph.AddDependency(instCAddr.ConfigResource(), instBAddr.ConfigResource())
	resourceGraph.AddDependency(instCAddr.ConfigResource(), instAAddr.ConfigResource())
	deferred := NewDeferred(resourceGraph)

	// Before we report anything, all three addresses should indicate that
	// they don't need to have their actions deferred.
	t.Run("without any deferrals yet", func(t *testing.T) {
		for _, instAddr := range []addrs.AbsResourceInstance{instAAddr, instBAddr, instCAddr} {
			if deferred.ShouldDeferResourceInstanceChanges(instAddr) {
				t.Errorf("%s reported as needing deferred; should not be, yet", instAddr)
			}
		}
	})

	// Resource A hasn't been expanded fully, so is deferred.
	deferred.ReportResourceExpansionDeferred(instAPartial, cty.DynamicVal)

	t.Run("with one resource instance deferred", func(t *testing.T) {
		if !deferred.ShouldDeferResourceInstanceChanges(instCAddr) {
			t.Errorf("%s was not reported as needing deferred; should be deferred", instCAddr)
		}
		if deferred.ShouldDeferResourceInstanceChanges(instBAddr) {
			t.Errorf("%s reported as needing deferred; should not be", instCAddr)
		}
	})
}
