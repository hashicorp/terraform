package test

import (
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceDiffSuppress_create(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_diff_suppress" "foo" {
	val_to_upper = "foo"
}
				`),
			},
		},
	})
}
func TestResourceDiffSuppress_update(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_diff_suppress" "foo" {
	val_to_upper = "foo"
}
				`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_diff_suppress" "foo" {
	val_to_upper = "bar"
	optional = "more"
}
				`),
			},
		},
	})
}

func TestResourceDiffSuppress_updateIgnoreChanges(t *testing.T) {
	// None of these steps should replace the instance
	id := ""
	checkFunc := func(s *terraform.State) error {
		root := s.ModuleByPath(addrs.RootModuleInstance)
		res := root.Resources["test_resource_diff_suppress.foo"]
		if id != "" && res.Primary.ID != id {
			return errors.New("expected no resource replacement")
		}
		id = res.Primary.ID
		return nil
	}

	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_diff_suppress" "foo" {
	val_to_upper = "foo"

	network    = "foo"
	subnetwork = "foo"

	node_pool {
	  name = "default-pool"
	}
	lifecycle {
		ignore_changes = ["node_pool"]
	}
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_diff_suppress" "foo" {
	val_to_upper = "foo"

	network    = "ignored"
	subnetwork = "ignored"

	node_pool {
		name = "default-pool"
	}
	lifecycle {
		ignore_changes = ["node_pool"]
	}
}
				`),
				Check: checkFunc,
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_diff_suppress" "foo" {
	val_to_upper = "foo"

	network    = "ignored"
	subnetwork = "ignored"

	node_pool {
		name = "ignored"
	}
	lifecycle {
		ignore_changes = ["node_pool"]
	}
}
			`),
				Check: checkFunc,
			},
		},
	})
}
