package terraform

import (
	"fmt"
	"strings"
	"testing"
)

func TestContextImport_basic(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
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
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo[0]",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
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
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err == nil {
		t.Fatalf("err: %s", err)
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
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err == nil {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := "<no state>"
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_moduleProvider(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Module: m,
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
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
		Module: m,
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

	_, err := ctx.Import(&ImportOpts{
		Module: m,
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "module.child.aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
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
		Module: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: map[string]interface{}{
			"foo": "bar",
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
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
		Module: m,
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

	_, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err == nil {
		t.Fatal("should error")
	}
}

func TestContextImport_refresh(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
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
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err == nil {
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
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "module.foo.aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
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
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "module.a.module.b.aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
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
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "module.foo.aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
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
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "module.foo.aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
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
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
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
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportMultiSameStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_customProvider(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Module: m,
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

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr:     "aws_instance.foo",
				ID:       "bar",
				Provider: "aws.alias",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
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
  provider = aws
`

const testImportCountIndexStr = `
aws_instance.foo.0:
  ID = foo
  provider = aws
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
    provider = aws
`

const testImportModuleDepth2Str = `
<no state>
module.a.b:
  aws_instance.foo:
    ID = foo
    provider = aws
`

const testImportModuleDiffStr = `
module.bar:
  aws_instance.bar:
    ID = bar
module.foo:
  aws_instance.foo:
    ID = foo
    provider = aws
`

const testImportModuleExistingStr = `
module.foo:
  aws_instance.bar:
    ID = bar
  aws_instance.foo:
    ID = foo
    provider = aws
`

const testImportMultiStr = `
aws_instance.foo:
  ID = foo
  provider = aws
aws_instance_thing.foo:
  ID = bar
  provider = aws
`

const testImportMultiSameStr = `
aws_instance.foo:
  ID = foo
  provider = aws
aws_instance_thing.foo:
  ID = bar
  provider = aws
aws_instance_thing.foo-1:
  ID = qux
  provider = aws
`

const testImportRefreshStr = `
aws_instance.foo:
  ID = foo
  provider = aws
  foo = bar
`

const testImportCustomProviderStr = `
aws_instance.foo:
  ID = foo
  provider = aws.alias
`
