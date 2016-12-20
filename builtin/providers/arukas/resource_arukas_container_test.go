package arukas

import (
	"fmt"
	API "github.com/arukasio/cli"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestAccArukasContainer_Basic(t *testing.T) {
	var container API.Container
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckArukasContainerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckArukasContainerConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArukasContainerExists("arukas_container.foobar", &container),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "name", "terraform_for_arukas_test_foobar"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "image", "nginx:latest"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "instances", "1"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "memory", "256"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "endpoint", "terraform-for-arukas-test-endpoint"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.#", "1"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.0.number", "80"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "environments.#", "1"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "environments.0.key", "key"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "environments.0.value", "value"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "port_mappings.#", "1"),
				),
			},
		},
	})
}

func TestAccArukasContainer_Update(t *testing.T) {
	var container API.Container
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckArukasContainerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckArukasContainerConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArukasContainerExists("arukas_container.foobar", &container),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "name", "terraform_for_arukas_test_foobar"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "image", "nginx:latest"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "instances", "1"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "memory", "256"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "endpoint", "terraform-for-arukas-test-endpoint"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.#", "1"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.0.number", "80"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "environments.#", "1"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "environments.0.key", "key"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "environments.0.value", "value"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "port_mappings.#", "1"),
				),
			},
			resource.TestStep{
				Config: testAccCheckArukasContainerConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArukasContainerExists("arukas_container.foobar", &container),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "name", "terraform_for_arukas_test_foobar_upd"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "image", "nginx:latest"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "instances", "2"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "memory", "512"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "endpoint", "terraform-for-arukas-test-endpoint-upd"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.#", "2"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.0.number", "80"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.1.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.1.number", "443"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "environments.#", "2"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "environments.0.key", "key"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "environments.0.value", "value"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "environments.1.key", "key_upd"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "environments.1.value", "value_upd"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "port_mappings.#", "4"),
				),
			},
		},
	})
}

func TestAccArukasContainer_Minimum(t *testing.T) {
	var container API.Container
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckArukasContainerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckArukasContainerConfig_minimum,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArukasContainerExists("arukas_container.foobar", &container),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "name", "terraform_for_arukas_test_foobar"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "image", "nginx:latest"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "instances", "1"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "memory", "256"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.#", "1"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "ports.0.number", "80"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "port_mappings.#", "1"),
				),
			},
		},
	})
}

func TestAccArukasContainer_Import(t *testing.T) {
	resourceName := "arukas_container.foobar"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckArukasContainerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckArukasContainerConfig_basic,
			},
			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckArukasContainerExists(n string, container *API.Container) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Container ID is set")
		}
		client := testAccProvider.Meta().(*ArukasClient)
		var foundContainer API.Container
		err := client.Get(&foundContainer, fmt.Sprintf("/containers/%s", rs.Primary.ID))

		if err != nil {
			return err
		}

		if foundContainer.ID != rs.Primary.ID {
			return fmt.Errorf("Container not found")
		}

		*container = foundContainer

		return nil
	}
}

func testAccCheckArukasContainerDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ArukasClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "arukas_container" {
			continue
		}

		err := client.Get(nil, fmt.Sprintf("/containers/%s", rs.Primary.ID))

		if err == nil {
			return fmt.Errorf("Note still exists")
		}
	}

	return nil
}

const testAccCheckArukasContainerConfig_basic = `
resource "arukas_container" "foobar" {
    name = "terraform_for_arukas_test_foobar"
    image = "nginx:latest"
    instances = 1
    memory = 256
    endpoint = "terraform-for-arukas-test-endpoint"
    ports = {
        protocol = "tcp"
        number = "80"
    }
    environments {
        key = "key"
        value = "value"
    }
}`

const testAccCheckArukasContainerConfig_update = `
resource "arukas_container" "foobar" {
    name = "terraform_for_arukas_test_foobar_upd"
    image = "nginx:latest"
    instances = 2
    memory = 512
    endpoint = "terraform-for-arukas-test-endpoint-upd"
    ports = {
        protocol = "tcp"
        number = "80"
    }
    ports = {
        protocol = "tcp"
        number = "443"
    }
    environments {
        key = "key"
        value = "value"
    }
    environments {
        key = "key_upd"
        value = "value_upd"
    }
}`

const testAccCheckArukasContainerConfig_minimum = `
resource "arukas_container" "foobar" {
    name = "terraform_for_arukas_test_foobar"
    image = "nginx:latest"
    ports = {
        number = "80"
    }
}`
