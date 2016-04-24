package terraform

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestBasicGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(BasicGraphBuilder)
}

func TestBasicGraphBuilder(t *testing.T) {
	b := &BasicGraphBuilder{
		Steps: []GraphTransformer{
			&testBasicGraphBuilderTransform{1},
		},
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(g.Path, RootModulePath) {
		t.Fatalf("bad: %#v", g.Path)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testBasicGraphBuilderStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestBasicGraphBuilder_validate(t *testing.T) {
	b := &BasicGraphBuilder{
		Steps: []GraphTransformer{
			&testBasicGraphBuilderTransform{1},
			&testBasicGraphBuilderTransform{2},
		},
		Validate: true,
	}

	_, err := b.Build(RootModulePath)
	if err == nil {
		t.Fatal("should error")
	}
}

func TestBasicGraphBuilder_validateOff(t *testing.T) {
	b := &BasicGraphBuilder{
		Steps: []GraphTransformer{
			&testBasicGraphBuilderTransform{1},
			&testBasicGraphBuilderTransform{2},
		},
		Validate: false,
	}

	_, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
}

func TestBuiltinGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(BuiltinGraphBuilder)
}

// This test is not meant to test all the transforms but rather just
// to verify we get some basic sane graph out. Special tests to ensure
// specific ordering of steps should be added in other tests.
func TestBuiltinGraphBuilder(t *testing.T) {
	b := &BuiltinGraphBuilder{
		Root:     testModule(t, "graph-builder-basic"),
		Validate: true,
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testBuiltinGraphBuilderBasicStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestBuiltinGraphBuilder_Verbose(t *testing.T) {
	b := &BuiltinGraphBuilder{
		Root:     testModule(t, "graph-builder-basic"),
		Validate: true,
		Verbose:  true,
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testBuiltinGraphBuilderVerboseStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

// This tests that the CreateBeforeDestoryTransformer is not present when
// we perform a "terraform destroy" operation. We don't actually do anything
// else.
func TestBuiltinGraphBuilder_CreateBeforeDestroy_Destroy_Bypass(t *testing.T) {
	b := &BuiltinGraphBuilder{
		Root:     testModule(t, "graph-builder-basic"),
		Validate: true,
		Destroy:  true,
	}

	steps := b.Steps([]string{})

	actual := false
	expected := false
	for _, v := range steps {
		switch v.(type) {
		case *CreateBeforeDestroyTransformer:
			actual = true
		}
	}

	if actual != expected {
		t.Fatalf("bad: CreateBeforeDestroyTransformer still in root path")
	}
}

// This tests that the CreateBeforeDestoryTransformer *is* present
// during a non-destroy operation (ie: Destroy not set).
func TestBuiltinGraphBuilder_CreateBeforeDestroy_NonDestroy_Present(t *testing.T) {
	b := &BuiltinGraphBuilder{
		Root:     testModule(t, "graph-builder-basic"),
		Validate: true,
	}

	steps := b.Steps([]string{})

	actual := false
	expected := true
	for _, v := range steps {
		switch v.(type) {
		case *CreateBeforeDestroyTransformer:
			actual = true
		}
	}

	if actual != expected {
		t.Fatalf("bad: CreateBeforeDestroyTransformer not in root path")
	}
}

// This tests a cycle we got when a CBD resource depends on a non-CBD
// resource. This cycle shouldn't happen in the general case anymore.
func TestBuiltinGraphBuilder_cbdDepNonCbd(t *testing.T) {
	b := &BuiltinGraphBuilder{
		Root:     testModule(t, "graph-builder-cbd-non-cbd"),
		Validate: true,
	}

	_, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestBuiltinGraphBuilder_cbdDepNonCbd_errorsWhenVerbose(t *testing.T) {
	b := &BuiltinGraphBuilder{
		Root:     testModule(t, "graph-builder-cbd-non-cbd"),
		Validate: true,
		Verbose:  true,
	}

	_, err := b.Build(RootModulePath)
	if err == nil {
		t.Fatalf("expected err, got none")
	}
}

func TestBuiltinGraphBuilder_multiLevelModule(t *testing.T) {
	b := &BuiltinGraphBuilder{
		Root:     testModule(t, "graph-builder-multi-level-module"),
		Validate: true,
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testBuiltinGraphBuilderMultiLevelStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestBuiltinGraphBuilder_orphanDeps(t *testing.T) {
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.bar": &ResourceState{
						Type:         "aws_instance",
						Dependencies: []string{"aws_instance.foo"},
						Primary: &InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}

	b := &BuiltinGraphBuilder{
		Root:     testModule(t, "graph-builder-orphan-deps"),
		State:    state,
		Validate: true,
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testBuiltinGraphBuilderOrphanDepsStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

/*
TODO: This exposes a really bad bug we need to fix after we merge
the f-ast-branch. This bug still exists in master.

// This test tests that the graph builder properly expands modules.
func TestBuiltinGraphBuilder_modules(t *testing.T) {
	b := &BuiltinGraphBuilder{
		Root: testModule(t, "graph-builder-modules"),
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testBuiltinGraphBuilderModuleStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}
*/

type testBasicGraphBuilderTransform struct {
	V dag.Vertex
}

func (t *testBasicGraphBuilderTransform) Transform(g *Graph) error {
	g.Add(t.V)
	return nil
}

const testBasicGraphBuilderStr = `
1
`

const testBuiltinGraphBuilderBasicStr = `
aws_instance.db
  provider.aws
aws_instance.web
  aws_instance.db
provider.aws
provider.aws (close)
  aws_instance.web
`

const testBuiltinGraphBuilderVerboseStr = `
aws_instance.db
  aws_instance.db (destroy tainted)
  aws_instance.db (destroy)
aws_instance.db (destroy tainted)
  aws_instance.web (destroy tainted)
aws_instance.db (destroy)
  aws_instance.web (destroy)
aws_instance.web
  aws_instance.db
aws_instance.web (destroy tainted)
  provider.aws
aws_instance.web (destroy)
  provider.aws
provider.aws
provider.aws (close)
  aws_instance.web
`

const testBuiltinGraphBuilderMultiLevelStr = `
module.foo.module.bar.output.value
  module.foo.module.bar.var.bar
  module.foo.var.foo
module.foo.module.bar.plan-destroy
module.foo.module.bar.var.bar
  module.foo.var.foo
module.foo.plan-destroy
module.foo.var.foo
root
  module.foo.module.bar.output.value
  module.foo.module.bar.plan-destroy
  module.foo.module.bar.var.bar
  module.foo.plan-destroy
  module.foo.var.foo
`

const testBuiltinGraphBuilderOrphanDepsStr = `
aws_instance.bar (orphan)
  provider.aws
aws_instance.foo (orphan)
  aws_instance.bar (orphan)
provider.aws
provider.aws (close)
  aws_instance.foo (orphan)
`

/*
TODO: Commented out this const as it's likely this needs to
be updated when the TestBuiltinGraphBuilder_modules test is
enabled again.
const testBuiltinGraphBuilderModuleStr = `
aws_instance.web
  aws_instance.web (destroy)
aws_instance.web (destroy)
  aws_security_group.firewall
  module.consul (expanded)
  provider.aws
aws_security_group.firewall
  aws_security_group.firewall (destroy)
aws_security_group.firewall (destroy)
  provider.aws
module.consul (expanded)
  aws_security_group.firewall
  provider.aws
provider.aws
`
*/
