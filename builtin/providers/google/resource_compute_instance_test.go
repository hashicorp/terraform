package google

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeInstance_basic_deprecated_network(t *testing.T) {
	var instance compute.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic_deprecated_network,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceTag(&instance, "foo"),
					testAccCheckComputeInstanceMetadata(&instance, "foo", "bar"),
					testAccCheckComputeInstanceDisk(&instance, "terraform-test", true, true),
				),
			},
		},
	})
}

func TestAccComputeInstance_basic1(t *testing.T) {
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
					testAccCheckComputeInstanceMetadata(&instance, "baz", "qux"),
					testAccCheckComputeInstanceDisk(&instance, "terraform-test", true, true),
				),
			},
		},
	})
}

func TestAccComputeInstance_basic2(t *testing.T) {
	var instance compute.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceTag(&instance, "foo"),
					testAccCheckComputeInstanceMetadata(&instance, "foo", "bar"),
					testAccCheckComputeInstanceDisk(&instance, "terraform-test", true, true),
				),
			},
		},
	})
}

func TestAccComputeInstance_basic3(t *testing.T) {
	var instance compute.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic3,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceTag(&instance, "foo"),
					testAccCheckComputeInstanceMetadata(&instance, "foo", "bar"),
					testAccCheckComputeInstanceDisk(&instance, "terraform-test", true, true),
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
					testAccCheckComputeInstanceAccessConfigHasIP(&instance),
				),
			},
		},
	})
}

func TestAccComputeInstance_disks(t *testing.T) {
	var instance compute.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_disks,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceDisk(&instance, "terraform-test", true, true),
					testAccCheckComputeInstanceDisk(&instance, "terraform-test-disk", false, false),
				),
			},
		},
	})
}

func TestAccComputeInstance_local_ssd(t *testing.T) {
	var instance compute.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_local_ssd,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.local-ssd", &instance),
					testAccCheckComputeInstanceDisk(&instance, "terraform-test", true, true),
				),
			},
		},
	})
}

func TestAccComputeInstance_update_deprecated_network(t *testing.T) {
	var instance compute.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic_deprecated_network,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
				),
			},
			resource.TestStep{
				Config: testAccComputeInstance_update_deprecated_network,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceMetadata(
						&instance, "bar", "baz"),
					testAccCheckComputeInstanceTag(&instance, "baz"),
				),
			},
		},
	})
}

func TestAccComputeInstance_forceNewAndChangeMetadata(t *testing.T) {
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
				),
			},
			resource.TestStep{
				Config: testAccComputeInstance_forceNewAndChangeMetadata,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceMetadata(
						&instance, "qux", "true"),
				),
			},
		},
	})
}

func TestAccComputeInstance_update(t *testing.T) {
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
				),
			},
			resource.TestStep{
				Config: testAccComputeInstance_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceMetadata(
						&instance, "bar", "baz"),
					testAccCheckComputeInstanceTag(&instance, "baz"),
					testAccCheckComputeInstanceAccessConfig(&instance),
				),
			},
		},
	})
}

func TestAccComputeInstance_service_account(t *testing.T) {
	var instance compute.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_service_account,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceServiceAccount(&instance,
						"https://www.googleapis.com/auth/compute.readonly"),
					testAccCheckComputeInstanceServiceAccount(&instance,
						"https://www.googleapis.com/auth/devstorage.read_only"),
					testAccCheckComputeInstanceServiceAccount(&instance,
						"https://www.googleapis.com/auth/userinfo.email"),
				),
			},
		},
	})
}

func testAccCheckComputeInstanceDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_instance" {
			continue
		}

		_, err := config.clientCompute.Instances.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Instance still exists")
		}
	}

	return nil
}

func testAccCheckComputeInstanceExists(n string, instance *compute.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Instances.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
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

			if item.Value != nil && v == *item.Value {
				return nil
			}

			return fmt.Errorf("bad value for %s: %s", k, *item.Value)
		}

		return fmt.Errorf("metadata not found: %s", k)
	}
}

func testAccCheckComputeInstanceAccessConfig(instance *compute.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, i := range instance.NetworkInterfaces {
			if len(i.AccessConfigs) == 0 {
				return fmt.Errorf("no access_config")
			}
		}

		return nil
	}
}

func testAccCheckComputeInstanceAccessConfigHasIP(instance *compute.Instance) resource.TestCheckFunc {
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

func testAccCheckComputeInstanceDisk(instance *compute.Instance, source string, delete bool, boot bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Disks == nil {
			return fmt.Errorf("no disks")
		}

		for _, disk := range instance.Disks {
			if strings.LastIndex(disk.Source, "/"+source) == len(disk.Source)-len(source)-1 && disk.AutoDelete == delete && disk.Boot == boot {
				return nil
			}
		}

		return fmt.Errorf("Disk not found: %s", source)
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

func testAccCheckComputeInstanceServiceAccount(instance *compute.Instance, scope string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if count := len(instance.ServiceAccounts); count != 1 {
			return fmt.Errorf("Wrong number of ServiceAccounts: expected 1, got %d", count)
		}

		for _, val := range instance.ServiceAccounts[0].Scopes {
			if val == scope {
				return nil
			}
		}

		return fmt.Errorf("Scope not found: %s", scope)
	}
}

const testAccComputeInstance_basic_deprecated_network = `
resource "google_compute_instance" "foobar" {
	name = "terraform-test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"
	can_ip_forward = false
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

const testAccComputeInstance_update_deprecated_network = `
resource "google_compute_instance" "foobar" {
	name = "terraform-test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"
	tags = ["baz"]

	disk {
		image = "debian-7-wheezy-v20140814"
	}

	network {
		source = "default"
	}

	metadata {
		bar = "baz"
	}
}`

const testAccComputeInstance_basic = `
resource "google_compute_instance" "foobar" {
	name = "terraform-test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"
	can_ip_forward = false
	tags = ["foo", "bar"]

	disk {
		image = "debian-7-wheezy-v20140814"
	}

	network_interface {
		network = "default"
	}

	metadata {
		foo = "bar"
		baz = "qux"
	}

	metadata_startup_script = "echo Hello"
}`

const testAccComputeInstance_basic2 = `
resource "google_compute_instance" "foobar" {
	name = "terraform-test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"
	can_ip_forward = false
	tags = ["foo", "bar"]

	disk {
		image = "debian-cloud/debian-7-wheezy-v20140814"
	}

	network_interface {
		network = "default"
	}


	metadata {
		foo = "bar"
	}
}`

const testAccComputeInstance_basic3 = `
resource "google_compute_instance" "foobar" {
	name = "terraform-test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"
	can_ip_forward = false
	tags = ["foo", "bar"]

	disk {
		image = "https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/debian-7-wheezy-v20140814"
	}

	network_interface {
		network = "default"
	}

	metadata {
		foo = "bar"
	}
}`

// Update zone to ForceNew, and change metadata k/v entirely
// Generates diff mismatch
const testAccComputeInstance_forceNewAndChangeMetadata = `
resource "google_compute_instance" "foobar" {
	name = "terraform-test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"
	zone = "us-central1-b"
	tags = ["baz"]

	disk {
		image = "debian-7-wheezy-v20140814"
	}

	network_interface {
		network = "default"
		access_config { }
	}

	metadata {
		qux = "true"
	}
}`

// Update metadata, tags, and network_interface
const testAccComputeInstance_update = `
resource "google_compute_instance" "foobar" {
	name = "terraform-test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"
	tags = ["baz"]

	disk {
		image = "debian-7-wheezy-v20140814"
	}

	network_interface {
		network = "default"
		access_config { }
	}

	metadata {
		bar = "baz"
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

	network_interface {
		network = "default"
		access_config {
			nat_ip = "${google_compute_address.foo.address}"
		}
	}

	metadata {
		foo = "bar"
	}
}`

const testAccComputeInstance_disks = `
resource "google_compute_disk" "foobar" {
	name = "terraform-test-disk"
	size = 10
	type = "pd-ssd"
	zone = "us-central1-a"
}

resource "google_compute_instance" "foobar" {
	name = "terraform-test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"

	disk {
		image = "debian-7-wheezy-v20140814"
	}

	disk {
		disk = "${google_compute_disk.foobar.name}"
		auto_delete = false
	}

	network_interface {
		network = "default"
	}

	metadata {
		foo = "bar"
	}
}`

const testAccComputeInstance_local_ssd = `
resource "google_compute_instance" "local-ssd" {
	name = "terraform-test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"

	disk {
		image = "debian-7-wheezy-v20140814"
	}

	disk {
        type = "local-ssd"
        scratch = true
	}

	network_interface {
		network = "default"
	}

}`

const testAccComputeInstance_service_account = `
resource "google_compute_instance" "foobar" {
	name = "terraform-test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"

	disk {
		image = "debian-7-wheezy-v20140814"
	}

	network_interface {
		network = "default"
	}

	service_account {
		scopes = [
			"userinfo-email",
			"compute-ro",
			"storage-ro",
		]
	}
}`
