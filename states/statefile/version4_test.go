package statefile

import (
	"sort"
	"testing"
)

// This test verifies that modules are sorted before resources:
// https://github.com/hashicorp/terraform/issues/21552
func TestVersion4_sort(t *testing.T) {
	resources := sortResourcesV4{
		{
			Module: "module.child",
			Type:   "test_instance",
			Name:   "foo",
		},
		{
			Type: "test_instance",
			Name: "foo",
		},
		{
			Module: "module.kinder",
			Type:   "test_instance",
			Name:   "foo",
		},
		{
			Module: "module.child.grandchild",
			Type:   "test_instance",
			Name:   "foo",
		},
	}
	sort.Stable(resources)

	moduleOrder := []string{"", "module.child", "module.child.grandchild", "module.kinder"}

	for i, resource := range resources {
		if resource.Module != moduleOrder[i] {
			t.Errorf("wrong sort order: expected %q, got %q\n", moduleOrder[i], resource.Module)
		}
	}
}
