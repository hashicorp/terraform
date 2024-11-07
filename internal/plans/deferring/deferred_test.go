// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package deferring

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
)

func TestDeferred_externalDependency(t *testing.T) {
	deferred := NewDeferred(true)

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
	}, nil)
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

	dependencies := addrs.MakeMap[addrs.ConfigResource, []addrs.ConfigResource](
		addrs.MapElem[addrs.ConfigResource, []addrs.ConfigResource]{
			Key:   instCAddr.ConfigResource(),
			Value: []addrs.ConfigResource{instBAddr.ConfigResource(), instAAddr.ConfigResource()},
		})

	deferred := NewDeferred(true)

	// Before we report anything, all three addresses should indicate that
	// they don't need to have their actions deferred.
	t.Run("without any deferrals yet", func(t *testing.T) {
		for _, instAddr := range []addrs.AbsResourceInstance{instAAddr, instBAddr, instCAddr} {
			if deferred.ShouldDeferResourceInstanceChanges(instAddr, dependencies.Get(instAddr.ConfigResource())) {
				t.Errorf("%s reported as needing deferred; should not be, yet", instAddr)
			}
		}
	})

	// Instance A has its Create action deferred for some reason.
	deferred.ReportResourceInstanceDeferred(instAAddr, providers.DeferredReasonResourceConfigUnknown, &plans.ResourceInstanceChange{
		Addr: instAAddr,
		Change: plans.Change{
			Action: plans.Create,
			After:  cty.DynamicVal,
		},
	})

	t.Run("with one resource instance deferred", func(t *testing.T) {
		if !deferred.ShouldDeferResourceInstanceChanges(instCAddr, dependencies.Get(instCAddr.ConfigResource())) {
			t.Errorf("%s was not reported as needing deferred; should be deferred", instCAddr)
		}
		if deferred.ShouldDeferResourceInstanceChanges(instBAddr, dependencies.Get(instBAddr.ConfigResource())) {
			t.Errorf("%s reported as needing deferred; should not be", instCAddr)
		}
	})
}

func TestDeferred_absDataSourceInstanceDeferred(t *testing.T) {
	instAAddr := addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance.Child("foo", addrs.NoKey),
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.DataResourceMode,
				Type: "test",
				Name: "a",
			},
		},
	}
	instBAddr := addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance,
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.DataResourceMode,
				Type: "test",
				Name: "b",
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

	dependencies := addrs.MakeMap[addrs.ConfigResource, []addrs.ConfigResource](
		addrs.MapElem[addrs.ConfigResource, []addrs.ConfigResource]{
			Key:   instCAddr.ConfigResource(),
			Value: []addrs.ConfigResource{instBAddr.ConfigResource(), instAAddr.ConfigResource()},
		})
	deferred := NewDeferred(true)

	// Before we report anything, all three addresses should indicate that
	// they don't need to have their actions deferred.
	t.Run("without any deferrals yet", func(t *testing.T) {
		for _, instAddr := range []addrs.AbsResourceInstance{instAAddr, instBAddr, instCAddr} {
			if deferred.ShouldDeferResourceInstanceChanges(instAddr, dependencies.Get(instAddr.ConfigResource())) {
				t.Errorf("%s reported as needing deferred; should not be, yet", instAddr)
			}
		}
	})

	// Instance A has its Read action deferred for some reason.
	deferred.ReportDataSourceInstanceDeferred(instAAddr, providers.DeferredReasonProviderConfigUnknown, &plans.ResourceInstanceChange{
		Addr:        instAAddr,
		PrevRunAddr: instAAddr,
		Change: plans.Change{
			Action: plans.Read,
			After:  cty.DynamicVal,
		},
		ActionReason: plans.ResourceInstanceReadBecauseDependencyPending,
	})

	t.Run("with one resource instance deferred", func(t *testing.T) {
		if !deferred.ShouldDeferResourceInstanceChanges(instCAddr, dependencies.Get(instCAddr.ConfigResource())) {
			t.Errorf("%s was not reported as needing deferred; should be deferred", instCAddr)
		}
		if deferred.ShouldDeferResourceInstanceChanges(instBAddr, dependencies.Get(instBAddr.ConfigResource())) {
			t.Errorf("%s reported as needing deferred; should not be", instCAddr)
		}
	})
}

func TestDeferred_absEphemeralResourceInstanceDeferred(t *testing.T) {
	instAAddr := addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance.Child("foo", addrs.NoKey),
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.EphemeralResourceMode,
				Type: "test",
				Name: "a",
			},
		},
	}
	instBAddr := addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance,
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.EphemeralResourceMode,
				Type: "test",
				Name: "b",
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

	dependencies := addrs.MakeMap[addrs.ConfigResource, []addrs.ConfigResource](
		addrs.MapElem[addrs.ConfigResource, []addrs.ConfigResource]{
			Key:   instCAddr.ConfigResource(),
			Value: []addrs.ConfigResource{instBAddr.ConfigResource(), instAAddr.ConfigResource()},
		})
	deferred := NewDeferred(true)

	// Before we report anything, all three addresses should indicate that
	// they don't need to have their actions deferred.
	t.Run("without any deferrals yet", func(t *testing.T) {
		for _, instAddr := range []addrs.AbsResourceInstance{instAAddr, instBAddr, instCAddr} {
			if deferred.ShouldDeferResourceInstanceChanges(instAddr, dependencies.Get(instAddr.ConfigResource())) {
				t.Errorf("%s reported as needing deferred; should not be, yet", instAddr)
			}
		}
	})

	// Instance A has e.g. the open action deferred
	deferred.ReportEphemeralResourceInstanceDeferred(instAAddr, providers.DeferredReasonProviderConfigUnknown)

	t.Run("with one resource instance deferred", func(t *testing.T) {
		if !deferred.ShouldDeferResourceInstanceChanges(instCAddr, dependencies.Get(instCAddr.ConfigResource())) {
			t.Errorf("%s was not reported as needing deferred; should be deferred", instCAddr)
		}
		if deferred.ShouldDeferResourceInstanceChanges(instBAddr, dependencies.Get(instBAddr.ConfigResource())) {
			t.Errorf("%s reported as needing deferred; should not be", instCAddr)
		}
	})
}

func TestDeferred_partialExpandedDatasource(t *testing.T) {
	instAAddr := addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance.Child("foo", addrs.NoKey),
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.DataResourceMode,
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
				Mode: addrs.DataResourceMode,
				Type: "test",
				Name: "c",
			},
		},
	}
	instAPartial := addrs.RootModuleInstance.
		UnexpandedChild(addrs.ModuleCall{Name: "foo"}).
		Resource(instAAddr.Resource.Resource)

	dependencies := addrs.MakeMap[addrs.ConfigResource, []addrs.ConfigResource](
		addrs.MapElem[addrs.ConfigResource, []addrs.ConfigResource]{
			Key:   instCAddr.ConfigResource(),
			Value: []addrs.ConfigResource{instBAddr.ConfigResource(), instAAddr.ConfigResource()},
		})
	deferred := NewDeferred(true)

	// Before we report anything, all three addresses should indicate that
	// they don't need to have their actions deferred.
	t.Run("without any deferrals yet", func(t *testing.T) {
		for _, instAddr := range []addrs.AbsResourceInstance{instAAddr, instBAddr, instCAddr} {
			if deferred.ShouldDeferResourceInstanceChanges(instAddr, dependencies.Get(instAddr.ConfigResource())) {
				t.Errorf("%s reported as needing deferred; should not be, yet", instAddr)
			}
		}
	})

	// Resource A hasn't been expanded fully, so is deferred.
	deferred.ReportDataSourceExpansionDeferred(instAPartial, &plans.ResourceInstanceChange{
		Addr: instAAddr,
		Change: plans.Change{
			Action: plans.Read,
			After:  cty.DynamicVal,
		},
	})

	t.Run("with one resource instance deferred", func(t *testing.T) {
		if !deferred.ShouldDeferResourceInstanceChanges(instCAddr, dependencies.Get(instCAddr.ConfigResource())) {
			t.Errorf("%s was not reported as needing deferred; should be deferred", instCAddr)
		}
		if deferred.ShouldDeferResourceInstanceChanges(instBAddr, dependencies.Get(instBAddr.ConfigResource())) {
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

	dependencies := addrs.MakeMap[addrs.ConfigResource, []addrs.ConfigResource](
		addrs.MapElem[addrs.ConfigResource, []addrs.ConfigResource]{
			Key:   instCAddr.ConfigResource(),
			Value: []addrs.ConfigResource{instBAddr.ConfigResource(), instAAddr.ConfigResource()},
		})
	deferred := NewDeferred(true)

	// Before we report anything, all three addresses should indicate that
	// they don't need to have their actions deferred.
	t.Run("without any deferrals yet", func(t *testing.T) {
		for _, instAddr := range []addrs.AbsResourceInstance{instAAddr, instBAddr, instCAddr} {
			if deferred.ShouldDeferResourceInstanceChanges(instAddr, dependencies.Get(instAddr.ConfigResource())) {
				t.Errorf("%s reported as needing deferred; should not be, yet", instAddr)
			}
		}
	})

	// Resource A hasn't been expanded fully, so is deferred.
	deferred.ReportResourceExpansionDeferred(instAPartial, &plans.ResourceInstanceChange{
		Addr: instAAddr,
		Change: plans.Change{
			Action: plans.Create,
			After:  cty.DynamicVal,
		},
	})

	t.Run("with one resource instance deferred", func(t *testing.T) {
		if !deferred.ShouldDeferResourceInstanceChanges(instCAddr, dependencies.Get(instCAddr.ConfigResource())) {
			t.Errorf("%s was not reported as needing deferred; should be deferred", instCAddr)
		}
		if deferred.ShouldDeferResourceInstanceChanges(instBAddr, dependencies.Get(instBAddr.ConfigResource())) {
			t.Errorf("%s reported as needing deferred; should not be", instCAddr)
		}
	})
}
