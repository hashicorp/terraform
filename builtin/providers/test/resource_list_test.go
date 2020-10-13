package test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

// an empty config should be ok, because no deprecated/removed fields are set.
func TestResourceList_changed(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
	list_block {
		string = "a"
		int = 1
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.#", "1",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.0.string", "a",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.0.int", "1",
					),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
	list_block {
		string = "a"
		int = 1
	}

	list_block {
		string = "b"
		int = 2
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.#", "2",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.0.string", "a",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.0.int", "1",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.1.string", "b",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.1.int", "2",
					),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
	list_block {
		string = "a"
		int = 1
	}

	list_block {
		string = "c"
		int = 2
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.#", "2",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.0.string", "a",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.0.int", "1",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.1.string", "c",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.1.int", "2",
					),
				),
			},
		},
	})
}

func TestResourceList_mapList(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
variable "map" {
  type = map(string)
  default = {}
}

resource "test_resource_list" "foo" {
	map_list = [
	  {
	    a = "1"
	  },
	  var.map
	]
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "map_list.1", "",
					),
				),
			},
		},
	})
}

func TestResourceList_sublist(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
	list_block {
		sublist_block {
			string = "a"
			int = 1
		}
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.0.sublist_block.#", "1",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.0.sublist_block.0.string", "a",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.0.sublist_block.0.int", "1",
					),
				),
			},
		},
	})
}

func TestResourceList_interpolationChanges(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
	list_block {
		string = "x"
	}
}
resource "test_resource_list" "bar" {
	list_block {
		string = test_resource_list.foo.id
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.0.string", "x",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.bar", "list_block.0.string", "testId",
					),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "baz" {
	list_block {
		string = "x"
		int = 1
	}
}
resource "test_resource_list" "bar" {
	list_block {
		string = test_resource_list.baz.id
		int = 3
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.baz", "list_block.0.string", "x",
					),
					resource.TestCheckResourceAttr(
						"test_resource_list.bar", "list_block.0.string", "testId",
					),
				),
			},
		},
	})
}

func TestResourceList_removedForcesNew(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
	list_block {
		force_new = "ok"
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.foo", "list_block.0.force_new", "ok",
					),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
}
				`),
				Check: resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestResourceList_emptyStrings(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
  list_block {
    sublist = ["a", ""]
  }

  list_block {
    sublist = [""]
  }

  list_block {
    sublist = ["", "c", ""]
  }
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_list.foo", "list_block.0.sublist.0", "a"),
					resource.TestCheckResourceAttr("test_resource_list.foo", "list_block.0.sublist.1", ""),
					resource.TestCheckResourceAttr("test_resource_list.foo", "list_block.1.sublist.0", ""),
					resource.TestCheckResourceAttr("test_resource_list.foo", "list_block.2.sublist.0", ""),
					resource.TestCheckResourceAttr("test_resource_list.foo", "list_block.2.sublist.1", "c"),
					resource.TestCheckResourceAttr("test_resource_list.foo", "list_block.2.sublist.2", ""),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
  list_block {
    sublist = [""]
  }

  list_block {
    sublist = []
  }

  list_block {
    sublist = ["", "c"]
  }
}
			`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_list.foo", "list_block.0.sublist.#", "1"),
					resource.TestCheckResourceAttr("test_resource_list.foo", "list_block.0.sublist.0", ""),
					resource.TestCheckResourceAttr("test_resource_list.foo", "list_block.1.sublist.#", "0"),
					resource.TestCheckResourceAttr("test_resource_list.foo", "list_block.2.sublist.1", "c"),
					resource.TestCheckResourceAttr("test_resource_list.foo", "list_block.2.sublist.#", "2"),
				),
			},
		},
	})
}

func TestResourceList_addRemove(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_list.foo", "computed_list.#", "0"),
					resource.TestCheckResourceAttr("test_resource_list.foo", "dependent_list.#", "0"),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
	dependent_list {
		val = "a"
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_list.foo", "computed_list.#", "1"),
					resource.TestCheckResourceAttr("test_resource_list.foo", "dependent_list.#", "1"),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test_resource_list.foo", "computed_list.#", "0"),
					resource.TestCheckResourceAttr("test_resource_list.foo", "dependent_list.#", "0"),
				),
			},
		},
	})
}

func TestResourceList_planUnknownInterpolation(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
	list_block {
		string = "x"
	}
}
resource "test_resource_list" "bar" {
	list_block {
		sublist = [
			test_resource_list.foo.list_block[0].string,
		]
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.bar", "list_block.0.sublist.0", "x",
					),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
	list_block {
		string = "x"
	}
	dependent_list {
		val = "y"
	}
}
resource "test_resource_list" "bar" {
	list_block {
		sublist = [
			test_resource_list.foo.computed_list[0],
		]
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.bar", "list_block.0.sublist.0", "y",
					),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
	list_block {
		string = "x"
	}
	dependent_list {
		val = "z"
	}
}
resource "test_resource_list" "bar" {
	list_block {
		sublist = [
			test_resource_list.foo.computed_list[0],
		]
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.bar", "list_block.0.sublist.0", "z",
					),
				),
			},
		},
	})
}

func TestResourceList_planUnknownInterpolationList(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
	dependent_list {
		val = "y"
	}
}
resource "test_resource_list" "bar" {
	list_block {
		sublist_block_optional {
			list = test_resource_list.foo.computed_list
		}
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.bar", "list_block.0.sublist_block_optional.0.list.0", "y",
					),
				),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "foo" {
	dependent_list {
		val = "z"
	}
}
resource "test_resource_list" "bar" {
	list_block {
		sublist_block_optional {
			list = test_resource_list.foo.computed_list
		}
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"test_resource_list.bar", "list_block.0.sublist_block_optional.0.list.0", "z",
					),
				),
			},
		},
	})
}

func TestResourceList_dynamicList(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "a" {
	dependent_list {
		val = "a"
	}

	dependent_list {
		val = "b"
	}
}
resource "test_resource_list" "b" {
	list_block {
		string = "constant"
	}
	dynamic "list_block" {
		for_each = test_resource_list.a.computed_list
		content {
		  string = list_block.value
		}
	}
}
				`),
				Check: resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestResourceList_dynamicMinItems(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: strings.TrimSpace(`
variable "a" {
  type = list(number)
  default = [1]
}

resource "test_resource_list" "b" {
	dynamic "min_items" {
		for_each = var.a
		content {
		  val = "foo"
		}
	}
}
				`),
				ExpectError: regexp.MustCompile(`attribute supports 2`),
			},
			resource.TestStep{
				Config: strings.TrimSpace(`
resource "test_resource_list" "a" {
	dependent_list {
		val = "a"
	}

	dependent_list {
		val = "b"
	}
}
resource "test_resource_list" "b" {
	list_block {
		string = "constant"
	}
	dynamic "min_items" {
		for_each = test_resource_list.a.computed_list
		content {
		  val = min_items.value
		}
	}
}
				`),
			},
		},
	})
}
