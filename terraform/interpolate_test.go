package terraform

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/lang/ast"
)

func TestInterpolater_countIndex(t *testing.T) {
	i := &Interpolater{}

	scope := &InterpolationScope{
		Path:     rootModulePath,
		Resource: &Resource{CountIndex: 42},
	}

	testInterpolate(t, i, scope, "count.index", ast.Variable{
		Value: 42,
		Type:  ast.TypeInt,
	})
}

func TestInterpolater_countIndexInWrongContext(t *testing.T) {
	i := &Interpolater{}

	scope := &InterpolationScope{
		Path: rootModulePath,
	}

	n := "count.index"

	v, err := config.NewInterpolatedVariable(n)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expectedErr := fmt.Errorf("foo: count.index is only valid within resources")

	_, err = i.Values(scope, map[string]config.InterpolatedVariable{
		"foo": v,
	})

	if !reflect.DeepEqual(expectedErr, err) {
		t.Fatalf("expected: %#v, got %#v", expectedErr, err)
	}
}

func TestInterpolater_moduleVariable(t *testing.T) {
	lock := new(sync.RWMutex)
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
					},
				},
			},
			&ModuleState{
				Path: []string{RootModuleName, "child"},
				Outputs: map[string]string{
					"foo": "bar",
				},
			},
		},
	}

	i := &Interpolater{
		State:     state,
		StateLock: lock,
	}

	scope := &InterpolationScope{
		Path: rootModulePath,
	}

	testInterpolate(t, i, scope, "module.child.foo", ast.Variable{
		Value: "bar",
		Type:  ast.TypeString,
	})
}

func TestInterpolater_pathCwd(t *testing.T) {
	i := &Interpolater{}
	scope := &InterpolationScope{}

	expected, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testInterpolate(t, i, scope, "path.cwd", ast.Variable{
		Value: expected,
		Type:  ast.TypeString,
	})
}

func TestInterpolater_pathModule(t *testing.T) {
	mod := testModule(t, "interpolate-path-module")
	i := &Interpolater{
		Module: mod,
	}
	scope := &InterpolationScope{
		Path: []string{RootModuleName, "child"},
	}

	path := mod.Child([]string{"child"}).Config().Dir
	testInterpolate(t, i, scope, "path.module", ast.Variable{
		Value: path,
		Type:  ast.TypeString,
	})
}

func TestInterpolater_pathRoot(t *testing.T) {
	mod := testModule(t, "interpolate-path-module")
	i := &Interpolater{
		Module: mod,
	}
	scope := &InterpolationScope{
		Path: []string{RootModuleName, "child"},
	}

	path := mod.Config().Dir
	testInterpolate(t, i, scope, "path.root", ast.Variable{
		Value: path,
		Type:  ast.TypeString,
	})
}

func TestInterpolater_computedSet(t *testing.T) {
	lock := new(sync.RWMutex)
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"resource.name": &ResourceState{
						Type: "resource",
						Primary: &InstanceState{
							ID: "qux",
							Attributes: map[string]string{
								"foo.#": "4",
								"foo.298374.bar1": "baz1",
								"foo.298374.bar2": "baz12",
								"foo.233489.bar1": "baz2",
								"foo.233489.bar2": "baz22",
								"foo.872348.bar1": "baz3",
								"foo.872348.bar2": "baz32",
								"foo.348573.bar1": "baz4",
								"foo.348573.bar2": "baz42",
							},
						},
					},
				},
			},
		},
	}

	mod := testModule(t, "resource-computed-set")
	i := &Interpolater{
		Module:    mod,
		State:     state,
		StateLock: lock,
	}

	scope := &InterpolationScope{
		Path: rootModulePath,
	}

	testInterpolate(t, i, scope, "resource.name.foo.#", ast.Variable{
		Value: "4",
		Type:  ast.TypeString,
	})

	expectedValues := []string{"baz1", "baz2", "baz3", "baz4"}
	testInterpolate(t, i, scope, "resource.name.foo.*.bar1", ast.Variable{
		Value: strings.Join(expectedValues, config.InterpSplitDelim),
		Type:  ast.TypeString,
	})

	expectedValues = []string{"baz12", "baz22", "baz32", "baz42"}
	testInterpolate(t, i, scope, "resource.name.foo.*.bar2", ast.Variable{
		Value: strings.Join(expectedValues, config.InterpSplitDelim),
		Type:  ast.TypeString,
	})

	testInterpolate(t, i, scope, "resource.name.foo.0.bar1", ast.Variable{
		Value: "baz1",
		Type:  ast.TypeString,
	})

	testInterpolate(t, i, scope, "resource.name.foo.4.bar1", ast.Variable{
		Value: "",
		Type:  ast.TypeString,
	})
}

func TestInterpolater_computedList(t *testing.T) {
	lock := new(sync.RWMutex)
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"resource.name": &ResourceState{
						Type: "resource",
						Primary: &InstanceState{
							ID: "qux",
							Attributes: map[string]string{
								"foo.#": "4",
								"foo.298374": "bar1",
								"foo.233489": "bar2",
								"foo.872348": "bar3",
								"foo.348573": "bar4",
							},
						},
					},
				},
			},
		},
	}

	mod := testModule(t, "resource-computed-list")
	i := &Interpolater{
		Module:    mod,
		State:     state,
		StateLock: lock,
	}

	scope := &InterpolationScope{
		Path: rootModulePath,
	}

	testInterpolate(t, i, scope, "resource.name.foo.#", ast.Variable{
		Value: "4",
		Type:  ast.TypeString,
	})

	expectedValues := []string{"bar1", "bar2", "bar3", "bar4"}
	testInterpolate(t, i, scope, "resource.name.foo.*", ast.Variable{
		Value: strings.Join(expectedValues, config.InterpSplitDelim),
		Type:  ast.TypeString,
	})
}

func testInterpolate(
	t *testing.T, i *Interpolater,
	scope *InterpolationScope,
	n string, expectedVar ast.Variable) {
	v, err := config.NewInterpolatedVariable(n)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual, err := i.Values(scope, map[string]config.InterpolatedVariable{
		"foo": v,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := map[string]ast.Variable{
		"foo": expectedVar,
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}
