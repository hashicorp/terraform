package google

import (
	"fmt"
	"testing"

	"code.google.com/p/google-api-go-client/compute/v1"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeInstance_basic(t *testing.T) {
	var instance compute.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceTag(&instance, "foo"),
					testAccCheckComputeInstanceMetadata(&instance, "foo", "bar"),
				),
			},
		},
	})
}

func TestAccComputeInstance_IP(t *testing.T) {
	var instance compute.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_ip,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceNetwork(&instance),
				),
			},
		},
	})
}

func testAccCheckComputeInstanceDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.Resources {
		if rs.Type != "google_compute_instance" {
			continue
		}

		_, err := config.clientCompute.Instances.Get(
			config.Project, rs.Attributes["zone"], rs.ID).Do()
		if err == nil {
			return fmt.Errorf("Instance still exists")
		}
	}

	return nil
}

func testAccCheckComputeInstanceExists(n string, instance *compute.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Instances.Get(
			config.Project, rs.Attributes["zone"], rs.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.ID {
			return fmt.Errorf("Instance not found")
		}

		*instance = *found

		return nil
	}
}

func testAccCheckComputeInstanceMetadata(
	instance *compute.Instance,
	k string, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Metadata == nil {
			return fmt.Errorf("no metadata")
		}

		for _, item := range instance.Metadata.Items {
			if k != item.Key {
				continue
			}

			if v == item.Value {
				return nil
			}

			return fmt.Errorf("bad value for %s: %s", k, item.Value)
		}

		return fmt.Errorf("metadata not found: %s", k)
	}
}

func testAccCheckComputeInstanceNetwork(instance *compute.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, i := range instance.NetworkInterfaces {
			for _, c := range i.AccessConfigs {
				if c.NatIP == "" {
					return fmt.Errorf("no NAT IP")
				}
			}
		}

		return nil
	}
}

func testAccCheckComputeInstanceTag(instance *compute.Instance, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Tags == nil {
			return fmt.Errorf("no tags")
		}

		for _, k := range instance.Tags.Items {
			if k == n {
				return nil
			}
		}

		return fmt.Errorf("tag not found: %s", n)
	}
}

const testAccComputeInstance_basic = `
resource "google_compute_instance" "foobar" {
	name = "terraform-test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"
	tags = ["foo", "bar"]

	disk {
		image = "debian-7-wheezy-v20140814"
	}

	network {
		source = "default"
	}

	metadata {
		foo = "bar"
	}
}`

const testAccComputeInstance_ip = `
resource "google_compute_address" "foo" {
	name = "foo"
}

resource "google_compute_instance" "foobar" {
	name = "terraform-test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"
	tags = ["foo", "bar"]

	disk {
		image = "debian-7-wheezy-v20140814"
	}

	network {
		source = "default"
		address = "${google_compute_address.foo.address}"
	}

	metadata {
		foo = "bar"
	}
}`
