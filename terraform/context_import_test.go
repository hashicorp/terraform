package terraform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/zclconf/go-cty/cty"
)

func TestContextImport_basic(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_countIndex(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.IntKey(0),
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportCountIndexStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_collision(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),

		State: &State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: []string{"root"},
					Resources: map[string]*ResourceState{
						"aws_instance.foo": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "bar",
							},
						},
					},
				},
			},
		},
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want an error indicating that the resource already exists in state")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportCollisionStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_missingType(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID: "foo",
		},
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if !diags.HasErrors() {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := "<no state>"
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_moduleProvider(t *testing.T) {
	p := mockProviderWithResourceTypeSchema("aws_instance", &configschema.Block{})
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	configured := false
	p.ConfigureFn = func(c *ResourceConfig) error {
		configured = true

		if v, ok := c.Get("foo"); !ok || v.(string) != "bar" {
			return fmt.Errorf("bad")
		}

		return nil
	}

	state, diags := ctx.Import(&ImportOpts{
		Config: m,
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !configured {
		t.Fatal("didn't configure provider")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

// Importing into a module requires a provider config in that module.
func TestContextImport_providerModule(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider-module")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	configured := false
	p.ConfigureFn = func(c *ResourceConfig) error {
		configured = true

		if v, ok := c.Get("foo"); !ok || v.(string) != "bar" {
			return fmt.Errorf("bad")
		}

		return nil
	}

	_, diags := ctx.Import(&ImportOpts{
		Config: m,
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.Child("child", addrs.NoKey).ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !configured {
		t.Fatal("didn't configure provider")
	}
}

// Test that import will interpolate provider configuration and use
// that configuration for import.
func TestContextImport_providerVarConfig(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider-vars")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("foo"),
				SourceType: ValueFromCaller,
			},
		},
	})

	configured := false
	p.ConfigureFn = func(c *ResourceConfig) error {
		configured = true

		if v, ok := c.Get("foo"); !ok || v.(string) != "bar" {
			return fmt.Errorf("bad value: %#v", v)
		}

		return nil
	}

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !configured {
		t.Fatal("didn't configure provider")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

// Test that provider configs can't reference resources.
func TestContextImport_providerNonVarConfig(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider-non-vars")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	_, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if !diags.HasErrors() {
		t.Fatal("should error")
	}
}

func TestContextImport_refresh(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	p.RefreshFn = func(info *InstanceInfo, s *InstanceState) (*InstanceState, error) {
		return &InstanceState{
			ID:         "foo",
			Attributes: map[string]string{"foo": "bar"},
		}, nil
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportRefreshStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_refreshNil(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	p.RefreshFn = func(info *InstanceInfo, s *InstanceState) (*InstanceState, error) {
		return nil, nil
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if !diags.HasErrors() {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := "<no state>"
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_module(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.Child("foo", addrs.NoKey).ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportModuleStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_moduleDepth2(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.Child("a", addrs.NoKey).Child("b", addrs.NoKey).ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportModuleDepth2Str)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_moduleDiff(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),

		State: &State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: []string{"root", "bar"},
					Resources: map[string]*ResourceState{
						"aws_instance.bar": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "bar",
							},
						},
					},
				},
			},
		},
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.Child("foo", addrs.NoKey).ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportModuleDiffStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_moduleExisting(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),

		State: &State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: []string{"root", "foo"},
					Resources: map[string]*ResourceState{
						"aws_instance.bar": &ResourceState{
							Type: "aws_instance",
							Primary: &InstanceState{
								ID: "bar",
							},
						},
					},
				},
			},
		},
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.Child("foo", addrs.NoKey).ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportModuleExistingStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_multiState(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
		&InstanceState{
			ID:        "bar",
			Ephemeral: EphemeralState{Type: "aws_instance_thing"},
		},
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportMultiStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_multiStateSame(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
		&InstanceState{
			ID:        "bar",
			Ephemeral: EphemeralState{Type: "aws_instance_thing"},
		},
		&InstanceState{
			ID:        "qux",
			Ephemeral: EphemeralState{Type: "aws_instance_thing"},
		},
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault("aws"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportMultiSameStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

// import missing a provider alias should fail
func TestContextImport_customProviderMissing(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	_, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigAliased("aws", "alias"),
			},
		},
	})
	if !diags.HasErrors() {
		t.Fatal("expected error")
	}
}

func TestContextImport_customProvider(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider-alias")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	state, diags := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				ID:           "bar",
				ProviderAddr: addrs.RootModuleInstance.ProviderConfigAliased("aws", "alias"),
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportCustomProviderStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

const testImportStr = `
aws_instance.foo:
  ID = foo
  provider = provider.aws
`

const testImportCountIndexStr = `
aws_instance.foo.0:
  ID = foo
  provider = provider.aws
`

const testImportCollisionStr = `
aws_instance.foo:
  ID = bar
`

const testImportModuleStr = `
<no state>
module.foo:
  aws_instance.foo:
    ID = foo
    provider = provider.aws
`

const testImportModuleDepth2Str = `
<no state>
module.a.b:
  aws_instance.foo:
    ID = foo
    provider = provider.aws
`

const testImportModuleDiffStr = `
module.bar:
  aws_instance.bar:
    ID = bar
module.foo:
  aws_instance.foo:
    ID = foo
    provider = provider.aws
`

const testImportModuleExistingStr = `
module.foo:
  aws_instance.bar:
    ID = bar
  aws_instance.foo:
    ID = foo
    provider = provider.aws
`

const testImportMultiStr = `
aws_instance.foo:
  ID = foo
  provider = provider.aws
aws_instance_thing.foo:
  ID = bar
  provider = provider.aws
`

const testImportMultiSameStr = `
aws_instance.foo:
  ID = foo
  provider = provider.aws
aws_instance_thing.foo:
  ID = bar
  provider = provider.aws
aws_instance_thing.foo-1:
  ID = qux
  provider = provider.aws
`

const testImportRefreshStr = `
aws_instance.foo:
  ID = foo
  provider = provider.aws
  foo = bar
`

const testImportCustomProviderStr = `
aws_instance.foo:
  ID = foo
  provider = provider.aws.alias
`
