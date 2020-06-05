package terraform

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
)

func TestRefreshGraphBuilder_configOrphans(t *testing.T) {

	m := testModule(t, "refresh-config-orphan")

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	deposedKey := states.DeposedKey("00000001")
	testSetResourceInstanceDeposed(root, "test_object.foo[0]", `{"id":"foo"}`, `provider["registry.terraform.io/hashicorp/test"]`, deposedKey)
	testSetResourceInstanceDeposed(root, "test_object.foo[1]", `{"id":"bar"}`, `provider["registry.terraform.io/hashicorp/test"]`, deposedKey)
	testSetResourceInstanceDeposed(root, "test_object.foo[2]", `{"id":"baz"}`, `provider["registry.terraform.io/hashicorp/test"]`, deposedKey)

	// NOTE: Real-world data resources don't get deposed
	testSetResourceInstanceDeposed(root, "data.test_object.foo[0]", `{"id":"foo"}`, `provider["registry.terraform.io/hashicorp/test"]`, deposedKey)
	testSetResourceInstanceDeposed(root, "data.test_object.foo[1]", `{"id":"bar"}`, `provider["registry.terraform.io/hashicorp/test"]`, deposedKey)
	testSetResourceInstanceDeposed(root, "data.test_object.foo[2]", `{"id":"baz"}`, `provider["registry.terraform.io/hashicorp/test"]`, deposedKey)

	b := &RefreshGraphBuilder{
		Config:     m,
		State:      state,
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
	}
	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("Error building graph: %s", err)
	}

	actual := strings.TrimSpace(g.StringWithNodeTypes())
	expected := strings.TrimSpace(`
data.test_object.foo[0] - *terraform.NodeRefreshableManagedResourceInstance
  provider["registry.terraform.io/hashicorp/test"] - *terraform.NodeApplyableProvider
data.test_object.foo[0] (deposed 00000001) - *terraform.NodePlanDeposedResourceInstanceObject
  provider["registry.terraform.io/hashicorp/test"] - *terraform.NodeApplyableProvider
data.test_object.foo[1] - *terraform.NodeRefreshableManagedResourceInstance
  provider["registry.terraform.io/hashicorp/test"] - *terraform.NodeApplyableProvider
data.test_object.foo[1] (deposed 00000001) - *terraform.NodePlanDeposedResourceInstanceObject
  provider["registry.terraform.io/hashicorp/test"] - *terraform.NodeApplyableProvider
data.test_object.foo[2] - *terraform.NodeRefreshableManagedResourceInstance
  provider["registry.terraform.io/hashicorp/test"] - *terraform.NodeApplyableProvider
data.test_object.foo[2] (deposed 00000001) - *terraform.NodePlanDeposedResourceInstanceObject
  provider["registry.terraform.io/hashicorp/test"] - *terraform.NodeApplyableProvider
provider["registry.terraform.io/hashicorp/test"] - *terraform.NodeApplyableProvider
provider["registry.terraform.io/hashicorp/test"] (close) - *terraform.graphNodeCloseProvider
  data.test_object.foo[0] - *terraform.NodeRefreshableManagedResourceInstance
  data.test_object.foo[0] (deposed 00000001) - *terraform.NodePlanDeposedResourceInstanceObject
  data.test_object.foo[1] - *terraform.NodeRefreshableManagedResourceInstance
  data.test_object.foo[1] (deposed 00000001) - *terraform.NodePlanDeposedResourceInstanceObject
  data.test_object.foo[2] - *terraform.NodeRefreshableManagedResourceInstance
  data.test_object.foo[2] (deposed 00000001) - *terraform.NodePlanDeposedResourceInstanceObject
  test_object.foo (expand) - *terraform.nodeExpandRefreshableManagedResource
  test_object.foo[0] (deposed 00000001) - *terraform.NodePlanDeposedResourceInstanceObject
  test_object.foo[1] (deposed 00000001) - *terraform.NodePlanDeposedResourceInstanceObject
  test_object.foo[2] (deposed 00000001) - *terraform.NodePlanDeposedResourceInstanceObject
root - *terraform.nodeCloseModule
  provider["registry.terraform.io/hashicorp/test"] (close) - *terraform.graphNodeCloseProvider
test_object.foo (expand) - *terraform.nodeExpandRefreshableManagedResource
  provider["registry.terraform.io/hashicorp/test"] - *terraform.NodeApplyableProvider
test_object.foo[0] (deposed 00000001) - *terraform.NodePlanDeposedResourceInstanceObject
  provider["registry.terraform.io/hashicorp/test"] - *terraform.NodeApplyableProvider
test_object.foo[1] (deposed 00000001) - *terraform.NodePlanDeposedResourceInstanceObject
  provider["registry.terraform.io/hashicorp/test"] - *terraform.NodeApplyableProvider
test_object.foo[2] (deposed 00000001) - *terraform.NodePlanDeposedResourceInstanceObject
  provider["registry.terraform.io/hashicorp/test"] - *terraform.NodeApplyableProvider
`)
	if expected != actual {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s\ndiff:\n%s", actual, expected, cmp.Diff(expected, actual))
	}
}
