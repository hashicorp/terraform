package terraform

import (
	"fmt"
	"testing"

	backendInit "github.com/hashicorp/terraform/backend/init"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestState_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccState_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStateValue(
						"data.terraform_remote_state.foo", "foo", "bar"),
				),
			},
		},
	})
}

func TestState_backends(t *testing.T) {
	backendInit.Set("_ds_test", backendInit.Backend("local"))
	defer backendInit.Set("_ds_test", nil)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccState_backend,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStateValue(
						"data.terraform_remote_state.foo", "foo", "bar"),
				),
			},
		},
	})
}

func TestState_complexOutputs(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccState_complexOutputs,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStateValue("terraform_remote_state.foo", "backend", "local"),
					// This (adding the hash) should be reverted when merged into 0.12.
					// testAccCheckStateValue("terraform_remote_state.foo", "config.path", "./test-fixtures/complex_outputs.tfstate"),
					testAccCheckStateValue("terraform_remote_state.foo", "config.1952794129.path", "./test-fixtures/complex_outputs.tfstate"),
					testAccCheckStateValue("terraform_remote_state.foo", "computed_set.#", "2"),
					testAccCheckStateValue("terraform_remote_state.foo", `map.%`, "2"),
					testAccCheckStateValue("terraform_remote_state.foo", `map.key`, "test"),
				),
			},
		},
	})
}

// outputs should never have a null value, but don't crash if we ever encounter
// them.
func TestState_nullOutputs(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccState_nullOutputs,
			},
		},
	})
}

func TestEmptyState_defaults(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccEmptyState_defaults,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStateValue(
						"data.terraform_remote_state.foo", "foo", "bar"),
				),
			},
		},
	})
}

func TestState_defaults(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccEmptyState_defaults,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStateValue(
						"data.terraform_remote_state.foo", "foo", "bar"),
				),
			},
		},
	})
}

func TestState_version4(t *testing.T) {
	// This test is for our special support for reading state version 4 as
	// remote state even though this Terraform version doesn't support that
	// version for any other purpose.

	resource.UnitTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccState_version4,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStateValue("data.terraform_remote_state.foo", "from_resource", "7585419850792319264"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "string", "value"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "bool", "true"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "float", "1.2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "integer", "3"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "list.#", "2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "list.0", "value 1"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "list.1", "value 2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "set.#", "2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "set.0", "value 1"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "set.1", "value 2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "tuple.#", "2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "tuple.0", "value 1"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "tuple.1", "value 2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "mixed_tuple.#", "2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "mixed_tuple.0", "value 1"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "mixed_tuple.1.#", "1"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "mixed_tuple.1.0", "value 2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "map.%", "2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "map.a", "value 1"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "map.b", "value 2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "object.%", "2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "object.a", "value 1"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "object.b", "value 2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "mixed_object.%", "2"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "mixed_object.a", "value 1"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "mixed_object.b.#", "1"),
					testAccCheckStateValue("data.terraform_remote_state.foo", "mixed_object.b.0", "value 2"),
				),
			},
		},
	})
}

func testAccCheckStateValue(id, name, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("Not found: %s", id)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		v := rs.Primary.Attributes[name]
		if v != value {
			return fmt.Errorf(
				"Value for %s is %s, not %s", name, v, value)
		}

		return nil
	}
}

// make sure that the deprecated environment field isn't overridden by the
// default value for workspace.
func TestState_deprecatedEnvironment(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccState_deprecatedEnvironment,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStateValue(
						// if the workspace default value overrides the
						// environment, this will get the foo value from the
						// default state.
						"data.terraform_remote_state.foo", "foo", ""),
				),
			},
		},
	})
}

const testAccState_basic = `
data "terraform_remote_state" "foo" {
	backend = "local"

	config {
		path = "./test-fixtures/basic.tfstate"
	}
}`

const testAccState_backend = `
data "terraform_remote_state" "foo" {
	backend = "_ds_test"

	config {
		path = "./test-fixtures/basic.tfstate"
	}
}`

const testAccState_complexOutputs = `
resource "terraform_remote_state" "foo" {
	backend = "local"

	config {
		path = "./test-fixtures/complex_outputs.tfstate"
	}
}`

const testAccState_nullOutputs = `
resource "terraform_remote_state" "foo" {
	backend = "local"

	config {
		path = "./test-fixtures/null_outputs.tfstate"
	}
}`

const testAccEmptyState_defaults = `
data "terraform_remote_state" "foo" {
	backend = "local"

	config {
		path = "./test-fixtures/empty.tfstate"
	}

	defaults {
		foo = "bar"
	}
}`

const testAccState_defaults = `
data "terraform_remote_state" "foo" {
	backend = "local"

	config {
		path = "./test-fixtures/basic.tfstate"
	}

	defaults {
		foo = "not bar"
	}
}`

const testAccState_deprecatedEnvironment = `
data "terraform_remote_state" "foo" {
	backend = "local"
	environment = "deprecated"

	config {
		path = "./test-fixtures/basic.tfstate"
	}
}`

const testAccState_version4 = `
data "terraform_remote_state" "foo" {
	backend = "local"

	config {
		path = "./test-fixtures/version4.tfstate"
	}
}`
