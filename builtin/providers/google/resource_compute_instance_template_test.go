package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeInstanceTemplate_basic(t *testing.T) {
	var instanceTemplate compute.InstanceTemplate

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceTemplateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstanceTemplate_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceTemplateExists(
						"google_compute_instance_template.foobar", &instanceTemplate),
					testAccCheckComputeInstanceTemplateTag(&instanceTemplate, "foo"),
					testAccCheckComputeInstanceTemplateMetadata(&instanceTemplate, "foo", "bar"),
					testAccCheckComputeInstanceTemplateDisk(&instanceTemplate, "https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/debian-7-wheezy-v20140814", true, true),
				),
			},
		},
	})
}

func TestAccComputeInstanceTemplate_IP(t *testing.T) {
	var instanceTemplate compute.InstanceTemplate

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceTemplateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstanceTemplate_ip,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceTemplateExists(
						"google_compute_instance_template.foobar", &instanceTemplate),
					testAccCheckComputeInstanceTemplateNetwork(&instanceTemplate),
				),
			},
		},
	})
}

func TestAccComputeInstanceTemplate_disks(t *testing.T) {
	var instanceTemplate compute.InstanceTemplate

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceTemplateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstanceTemplate_disks,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceTemplateExists(
						"google_compute_instance_template.foobar", &instanceTemplate),
					testAccCheckComputeInstanceTemplateDisk(&instanceTemplate, "https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/debian-7-wheezy-v20140814", true, true),
					testAccCheckComputeInstanceTemplateDisk(&instanceTemplate, "terraform-test-foobar", false, false),
				),
			},
		},
	})
}

func testAccCheckComputeInstanceTemplateDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_instance_template" {
			continue
		}

		_, err := config.clientCompute.InstanceTemplates.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Instance template still exists")
		}
	}

	return nil
}

func testAccCheckComputeInstanceTemplateExists(n string, instanceTemplate *compute.InstanceTemplate) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.InstanceTemplates.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Instance template not found")
		}

		*instanceTemplate = *found

		return nil
	}
}

func testAccCheckComputeInstanceTemplateMetadata(
	instanceTemplate *compute.InstanceTemplate,
	k string, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instanceTemplate.Properties.Metadata == nil {
			return fmt.Errorf("no metadata")
		}

		for _, item := range instanceTemplate.Properties.Metadata.Items {
			if k != item.Key {
				continue
			}

			if item.Value != nil && v == *item.Value {
				return nil
			}

			return fmt.Errorf("bad value for %s: %s", k, *item.Value)
		}

		return fmt.Errorf("metadata not found: %s", k)
	}
}

func testAccCheckComputeInstanceTemplateNetwork(instanceTemplate *compute.InstanceTemplate) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, i := range instanceTemplate.Properties.NetworkInterfaces {
			for _, c := range i.AccessConfigs {
				if c.NatIP == "" {
					return fmt.Errorf("no NAT IP")
				}
			}
		}

		return nil
	}
}

func testAccCheckComputeInstanceTemplateDisk(instanceTemplate *compute.InstanceTemplate, source string, delete bool, boot bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instanceTemplate.Properties.Disks == nil {
			return fmt.Errorf("no disks")
		}

		for _, disk := range instanceTemplate.Properties.Disks {
			if disk.InitializeParams == nil {
				// Check disk source
				if disk.Source == source {
					if disk.AutoDelete == delete && disk.Boot == boot {
						return nil
					}
				}
			} else {
				// Check source image
				if disk.InitializeParams.SourceImage == source {
					if disk.AutoDelete == delete && disk.Boot == boot {
						return nil
					}
				}
			}
		}

		return fmt.Errorf("Disk not found: %s", source)
	}
}

func testAccCheckComputeInstanceTemplateTag(instanceTemplate *compute.InstanceTemplate, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instanceTemplate.Properties.Tags == nil {
			return fmt.Errorf("no tags")
		}

		for _, k := range instanceTemplate.Properties.Tags.Items {
			if k == n {
				return nil
			}
		}

		return fmt.Errorf("tag not found: %s", n)
	}
}

var testAccComputeInstanceTemplate_basic = fmt.Sprintf(`
resource "google_compute_instance_template" "foobar" {
	name = "instancet-test-%s"
	machine_type = "n1-standard-1"
	can_ip_forward = false
	tags = ["foo", "bar"]

	disk {
		source_image = "debian-7-wheezy-v20140814"
		auto_delete = true
		boot = true
	}

	network_interface {
		network = "default"
	}

	scheduling {
		preemptible = false
		automatic_restart = true
	}

	metadata {
		foo = "bar"
	}

	service_account {
		scopes = ["userinfo-email", "compute-ro", "storage-ro"]
	}
}`, acctest.RandString(10))

var testAccComputeInstanceTemplate_ip = fmt.Sprintf(`
resource "google_compute_address" "foo" {
	name = "instancet-test-%s"
}

resource "google_compute_instance_template" "foobar" {
	name = "instancet-test-%s"
	machine_type = "n1-standard-1"
	tags = ["foo", "bar"]

	disk {
		source_image = "debian-7-wheezy-v20140814"
	}

	network_interface {
		network = "default"
		access_config {
			nat_ip = "${google_compute_address.foo.address}"
		}
	}

	metadata {
		foo = "bar"
	}
}`, acctest.RandString(10), acctest.RandString(10))

var testAccComputeInstanceTemplate_disks = fmt.Sprintf(`
resource "google_compute_disk" "foobar" {
	name = "instancet-test-%s"
	image = "debian-7-wheezy-v20140814"
	size = 10
	type = "pd-ssd"
	zone = "us-central1-a"
}

resource "google_compute_instance_template" "foobar" {
	name = "instancet-test-%s"
	machine_type = "n1-standard-1"

	disk {
		source_image = "debian-7-wheezy-v20140814"
		auto_delete = true
		disk_size_gb = 100
		boot = true
	}

	disk {
		source = "terraform-test-foobar"
		auto_delete = false
		boot = false
	}

	network_interface {
		network = "default"
	}

	metadata {
		foo = "bar"
	}
}`, acctest.RandString(10), acctest.RandString(10))
