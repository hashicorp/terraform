package terraform

import (
	"strings"
	"testing"
)

func TestContextImport_basic(t *testing.T) {
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
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
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_missingType(t *testing.T) {
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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
	expected := "<nil>"
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_refresh(t *testing.T) {
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
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

func TestContextImport_module(t *testing.T) {
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
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
	expected := strings.TrimSpace(testImportModuleStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_moduleDepth2(t *testing.T) {
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
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

func TestContextImport_moduleExisting(t *testing.T) {
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},

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

const testImportStr = `
aws_instance.foo:
  ID = foo
  provider = aws
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

const testImportModuleExistingStr = `
module.foo:
  aws_instance.bar:
    ID = bar
  aws_instance.foo:
    ID = foo
    provider = aws
`

const testImportRefreshStr = `
aws_instance.foo:
  ID = foo
  provider = aws
  foo = bar
`
