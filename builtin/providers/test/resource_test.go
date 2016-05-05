package test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResource_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required = "yep"
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

// Targeted test in TestContext2Apply_ignoreChangesCreate
func TestResource_ignoreChangesRequired(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
       required = "yep"
       lifecycle {
               ignore_changes = ["required"]
       }
}
                               `),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func TestResource_ignoreChangesEmpty(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	optional_force_new = "one"
	lifecycle {
		ignore_changes = []
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	optional_force_new = "two"
	lifecycle {
		ignore_changes = []
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func TestResource_ignoreChangesForceNew(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	optional_force_new = "one"
	lifecycle {
		ignore_changes = ["optional_force_new"]
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	optional_force_new = "two"
	lifecycle {
		ignore_changes = ["optional_force_new"]
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

// Covers specific scenario in #6005, handled by normalizing boolean strings in
// helper/schema
func TestResource_ignoreChangesForceNewBoolean(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required           = "yep"
  optional_force_new = "one"
  optional_bool      = true
  lifecycle {
    ignore_changes = ["optional_force_new"]
  }
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required           = "yep"
  optional_force_new = "two"
  optional_bool      = true
  lifecycle {
    ignore_changes = ["optional_force_new"]
  }
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func TestResource_ignoreChangesMap(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	optional_computed_map {
		foo = "bar"
	}
	lifecycle {
		ignore_changes = ["optional_computed_map"]
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	optional_computed_map {
		foo = "bar"
		no  = "update"
	}
	lifecycle {
		ignore_changes = ["optional_computed_map"]
	}
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func testAccCheckResourceDestroy(s *terraform.State) error {
	return nil
}
