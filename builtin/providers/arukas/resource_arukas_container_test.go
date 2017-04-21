package arukas

import (
	"fmt"
	API "github.com/arukasio/cli"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestAccArukasContainer_Basic(t *testing.T) {
	var container API.Container
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("terraform_acc_test_%s", randString)
	endpoint := fmt.Sprintf("terraform-acc-test-endpoint-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckArukasContainerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckArukasContainerConfig_basic(randString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArukasContainerExists("arukas_container.foobar", &container),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "name", name),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "image", "nginx:latest"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "instances", "1"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "memory", "256"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "endpoint", endpoint),
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
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("terraform_acc_test_%s", randString)
	updatedName := fmt.Sprintf("terraform_acc_test_update_%s", randString)
	endpoint := fmt.Sprintf("terraform-acc-test-endpoint-%s", randString)
	updatedEndpoint := fmt.Sprintf("terraform-acc-test-endpoint-update-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckArukasContainerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckArukasContainerConfig_basic(randString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArukasContainerExists("arukas_container.foobar", &container),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "name", name),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "image", "nginx:latest"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "instances", "1"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "memory", "256"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "endpoint", endpoint),
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
				Config: testAccCheckArukasContainerConfig_update(randString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArukasContainerExists("arukas_container.foobar", &container),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "name", updatedName),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "image", "nginx:latest"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "instances", "2"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "memory", "512"),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "endpoint", updatedEndpoint),
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
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("terraform_acc_test_minimum_%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckArukasContainerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckArukasContainerConfig_minimum(randString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArukasContainerExists("arukas_container.foobar", &container),
					resource.TestCheckResourceAttr(
						"arukas_container.foobar", "name", name),
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
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckArukasContainerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckArukasContainerConfig_basic(randString),
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

func testAccCheckArukasContainerConfig_basic(randString string) string {
	return fmt.Sprintf(`
resource "arukas_container" "foobar" {
    name = "terraform_acc_test_%s"
    image = "nginx:latest"
    instances = 1
    memory = 256
    endpoint = "terraform-acc-test-endpoint-%s"
    ports = {
        protocol = "tcp"
        number = "80"
    }
    environments {
        key = "key"
        value = "value"
    }
}`, randString, randString)
}

func testAccCheckArukasContainerConfig_update(randString string) string {
	return fmt.Sprintf(`
resource "arukas_container" "foobar" {
    name = "terraform_acc_test_update_%s"
    image = "nginx:latest"
    instances = 2
    memory = 512
    endpoint = "terraform-acc-test-endpoint-update-%s"
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
}`, randString, randString)
}

func testAccCheckArukasContainerConfig_minimum(randString string) string {
	return fmt.Sprintf(`
resource "arukas_container" "foobar" {
    name = "terraform_acc_test_minimum_%s"
    image = "nginx:latest"
    ports = {
        number = "80"
    }
}`, randString)
}
