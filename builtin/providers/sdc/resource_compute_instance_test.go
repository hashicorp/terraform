package sdc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/joyent/gosdc/cloudapi"
)

func TestAccInstance_basic(t *testing.T) {
	var instance cloudapi.Machine

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstance_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"sdc_instance.foobar", &instance),
					testAccCheckInstanceTag(&instance, "foo"),
				),
			},
		},
	})
}

func TestAccInstance_update(t *testing.T) {
	var instance cloudapi.Machine

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstance_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"sdc_instance.foobar", &instance),
					testAccCheckInstancePackage(&instance, "g3-standard-1.75-smartos"),
				),
			},
			resource.TestStep{
				Config: testAccInstance_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"sdc_instance.foobar", &instance),
					testAccCheckInstancePackage(&instance, "g3-highcpu-1.75-smartos"),
					testAccCheckInstanceTag(&instance, "bar"),
					testAccCheckInstanceName(&instance, "foobar"),
				),
			},
		},
	})
}

func testAccCheckInstanceExists(n string, instance *cloudapi.Machine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.sdc_client.GetMachine(rs.Primary.ID)
		if err != nil {
			return err
		}

		if found.Id != rs.Primary.ID {
			return fmt.Errorf("Instance not found")
		}

		*instance = *found

		return nil
	}
}

func testAccCheckInstanceDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "sdc_instance" {
			continue
		}

		_, err := config.sdc_client.GetMachine(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Instance still exists")
		}
	}

	return nil
}

func testAccCheckInstanceTag(instance *cloudapi.Machine, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Tags == nil {
			return fmt.Errorf("no tags")
		}

		for k, _ := range instance.Tags {
			if k == n {
				return nil
			}
		}

		return fmt.Errorf("tag not found: %s", n)
	}
}

func testAccCheckInstanceName(instance *cloudapi.Machine, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Name == "" {
			return fmt.Errorf("no name")
		}

		if instance.Name == n {
			return nil
		}

		return fmt.Errorf("instance has wrong name: %s", instance.Name)
	}
}

func testAccCheckInstancePackage(instance *cloudapi.Machine, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Package == "" {
			return fmt.Errorf("no package")
		}

		if instance.Package == n {
			return nil
		}

		return fmt.Errorf("instance has wrong package: %s", instance.Package)
	}
}

const testAccInstance_basic = `
resource "sdc_instance" "foobar" {
  image = "d34c301e-10c3-11e4-9b79-5f67ca448df0"
  package = "g3-standard-1.75-smartos"

  network {
    source = "1e7bb0e1-25a9-43b6-bb19-f79ae9540b39"
    name = "SDC Public"
  }

  tags {
    foo = "bar"
  }
}`

const testAccInstance_update = `
resource "sdc_instance" "foobar" {
	name = "foobar"

  image = "d34c301e-10c3-11e4-9b79-5f67ca448df0"
  package = "g3-highcpu-1.75-smartos"

  network {
    source = "1e7bb0e1-25a9-43b6-bb19-f79ae9540b39"
    name = "SDC Public"
  }

  tags {
    foo = "bar"
    bar = "baz"
  }
}`
