package test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
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
	required = "yep"
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
	required = "yep"
	required_map = {
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
	required_map = {
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
	optional_computed_map = {
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
	optional_computed_map = {
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
	required_map = {
		key = "value"
	}

	optional_force_new = "one"
	lifecycle {
		ignore_changes = ["optional_force_new"]
	}
}
resource "test_resource" "bar" {
	count = 2
	required = "yep"
	required_map = {
		key = "value"
	}
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
	required_map = {
		key = "value"
	}

	optional_force_new = "two"
	lifecycle {
		ignore_changes = ["optional_force_new"]
	}
}
resource "test_resource" "bar" {
	count = 2
	required = "yep"
	required_map = {
		key = "value"
	}
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

// Reproduces plan-time panic when the wrong type is interpolated in a list of
// maps.
// TODO: this should return a type error, rather than silently setting an empty
//       list
func TestResource_dataSourceListMapPanic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required = "val"
  required_map = {x = "y"}
  list_of_map = "${var.maplist}"
}

variable "maplist" {
  type = "list"

  default = [
    {a = "b"}
  ]
}
				`),
				ExpectError: nil,
				Check: func(s *terraform.State) error {
					return nil
				},
			},
		},
	})
}

func TestResource_dataSourceIndexMapList(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
  required = "val"

  required_map = {
    x = "y"
  }

  list_of_map = [
    {
      a = "1"
      b = "2"
    },
    {
      c = "3"
      d = "4"
    },
  ]
}

output "map_from_list" {
  value = "${test_resource.foo.list_of_map[0]}"
}

output "value_from_map_from_list" {
  value = "${lookup(test_resource.foo.list_of_map[1], "d")}"
}
				`),
				ExpectError: nil,
				Check: func(s *terraform.State) error {
					root := s.ModuleByPath(addrs.RootModuleInstance)
					mapOut := root.Outputs["map_from_list"].Value
					expectedMapOut := map[string]interface{}{
						"a": "1",
						"b": "2",
					}

					valueOut := root.Outputs["value_from_map_from_list"].Value
					expectedValueOut := "4"

					if !reflect.DeepEqual(mapOut, expectedMapOut) {
						t.Fatalf("Expected: %#v\nGot: %#v", expectedMapOut, mapOut)
					}
					if !reflect.DeepEqual(valueOut, expectedValueOut) {
						t.Fatalf("Expected: %#v\nGot: %#v", valueOut, expectedValueOut)
					}
					return nil
				},
			},
		},
	})
}

func testAccCheckResourceDestroy(s *terraform.State) error {
	return nil
}

func TestResource_removeForceNew(t *testing.T) {
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
	optional_force_new = "here"
}
				`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "yep"
	required_map = {
	  key = "value"
	}
}
				`),
			},
		},
	})
}

func TestResource_unknownFuncInMap(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource" "foo" {
	required           = "ok"
	required_map = {
	  key = "${uuid()}"
	}
}
				`),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// Verify that we can destroy when a managed resource references something with
// a count of 1.
func TestResource_countRefDestroyError(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: strings.TrimSpace(`
resource "test_resource" "one" {
	count = 1
	required     = "ok"
	required_map = {
	  key = "val"
	}
}

resource "test_resource" "two" {
	required     = test_resource.one[0].id
	required_map = {
	  key = "val"
	}
}
				`),
			},
		},
	})
}
