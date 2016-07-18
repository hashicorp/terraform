package test

import (
	"regexp"
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
	required_map = {
	    key = "value"
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
	required_map = {
	    key = "value"
	}
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
	required_map = {
	    key = "value"
	}
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
	required_map {
	    key = "value"
	}
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
	required_map {
	    key = "value"
	}
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
	required_map = {
	    key = "value"
	}
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
  required_map = {
    key = "value"
  }
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
  required_map = {
    key = "value"
  }
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

// Reproduces plan-time panic described in GH-7170
func TestResource_dataSourceListPlanPanic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
data "test_data_source" "foo" {}
resource "test_resource" "foo" {
  required = "${data.test_data_source.foo.list}"
  required_map = {
    key = "value"
  }
}
				`),
				ExpectError: regexp.MustCompile(`must be a single value, not a list`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

// Reproduces apply-time panic described in GH-7170
func TestResource_dataSourceListApplyPanic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required = "ok"
  required_map = {
    key = "value"
  }
}
resource "test_resource" "bar" {
  required = "${test_resource.foo.computed_list}"
  required_map = {
    key = "value"
  }
}
				`),
				ExpectError: regexp.MustCompile(`must be a single value, not a list`),
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
	required_map = {
	  key = "value"
	}
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
	required_map = {
	  key = "value"
	}
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

func TestResource_ignoreChangesDependent(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	count = 2
	required = "yep"
	required_map { key = "value" }

	optional_force_new = "one"
	lifecycle {
		ignore_changes = ["optional_force_new"]
	}
}
resource "test_resource" "bar" {
	count = 2
	required = "yep"
	required_map { key = "value" }
	optional = "${element(test_resource.foo.*.id, count.index)}"
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	count = 2
	required = "yep"
	required_map { key = "value" }

	optional_force_new = "two"
	lifecycle {
		ignore_changes = ["optional_force_new"]
	}
}
resource "test_resource" "bar" {
	count = 2
	required = "yep"
	required_map { key = "value" }
	optional = "${element(test_resource.foo.*.id, count.index)}"
}
				`),
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func TestResource_ignoreChangesStillReplaced(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required     = "yep"
  required_map = {
    key = "value"
  }
  optional_force_new = "one"
  optional_bool      = true
  lifecycle {
    ignore_changes = ["optional_bool"]
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
  required     = "yep"
  required_map = {
    key = "value"
  }
  optional_force_new = "two"
  optional_bool      = false
  lifecycle {
    ignore_changes = ["optional_bool"]
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
