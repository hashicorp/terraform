package terraform

import (
	"os"
	"reflect"
	"sync"
	"testing"

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
