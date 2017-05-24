package google

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeInstance_basic_deprecated_network(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic_deprecated_network(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceTag(&instance, "foo"),
					testAccCheckComputeInstanceMetadata(&instance, "foo", "bar"),
					testAccCheckComputeInstanceDisk(&instance, instanceName, true, true),
				),
			},
		},
	})
}

func TestAccComputeInstance_basic1(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceTag(&instance, "foo"),
					testAccCheckComputeInstanceMetadata(&instance, "foo", "bar"),
					testAccCheckComputeInstanceMetadata(&instance, "baz", "qux"),
					testAccCheckComputeInstanceDisk(&instance, instanceName, true, true),
				),
			},
		},
	})
}

func TestAccComputeInstance_basic2(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceTag(&instance, "foo"),
					testAccCheckComputeInstanceMetadata(&instance, "foo", "bar"),
					testAccCheckComputeInstanceDisk(&instance, instanceName, true, true),
				),
			},
		},
	})
}

func TestAccComputeInstance_basic3(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic3(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceTag(&instance, "foo"),
					testAccCheckComputeInstanceMetadata(&instance, "foo", "bar"),
					testAccCheckComputeInstanceDisk(&instance, instanceName, true, true),
				),
			},
		},
	})
}

func TestAccComputeInstance_basic4(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic4(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceTag(&instance, "foo"),
					testAccCheckComputeInstanceMetadata(&instance, "foo", "bar"),
					testAccCheckComputeInstanceDisk(&instance, instanceName, true, true),
				),
			},
		},
	})
}

func TestAccComputeInstance_basic5(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic5(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceTag(&instance, "foo"),
					testAccCheckComputeInstanceMetadata(&instance, "foo", "bar"),
					testAccCheckComputeInstanceDisk(&instance, instanceName, true, true),
				),
			},
		},
	})
}

func TestAccComputeInstance_IP(t *testing.T) {
	var instance compute.Instance
	var ipName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_ip(ipName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceAccessConfigHasIP(&instance),
				),
			},
		},
	})
}

func TestAccComputeInstance_disksWithoutAutodelete(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))
	var diskName = fmt.Sprintf("instance-testd-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_disks(diskName, instanceName, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceDisk(&instance, instanceName, true, true),
					testAccCheckComputeInstanceDisk(&instance, diskName, false, false),
				),
			},
		},
	})
}

func TestAccComputeInstance_disksWithAutodelete(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))
	var diskName = fmt.Sprintf("instance-testd-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_disks(diskName, instanceName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceDisk(&instance, instanceName, true, true),
					testAccCheckComputeInstanceDisk(&instance, diskName, true, false),
				),
			},
		},
	})
}

func TestAccComputeInstance_diskEncryption(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))
	var diskName = fmt.Sprintf("instance-testd-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_disks_encryption(diskName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceDisk(&instance, instanceName, true, true),
					testAccCheckComputeInstanceDisk(&instance, diskName, true, false),
					testAccCheckComputeInstanceDiskEncryptionKey("google_compute_instance.foobar", &instance),
				),
			},
		},
	})
}

func TestAccComputeInstance_attachedDisk(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))
	var diskName = fmt.Sprintf("instance-testd-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_attachedDisk(diskName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceDisk(&instance, diskName, false, true),
				),
			},
		},
	})
}

func TestAccComputeInstance_noDisk(t *testing.T) {
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config:      testAccComputeInstance_noDisk(instanceName),
				ExpectError: regexp.MustCompile("At least one disk or attached_disk must be set"),
			},
		},
	})
}

func TestAccComputeInstance_local_ssd(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_local_ssd(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.local-ssd", &instance),
					testAccCheckComputeInstanceDisk(&instance, instanceName, true, true),
				),
			},
		},
	})
}

func TestAccComputeInstance_update_deprecated_network(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic_deprecated_network(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
				),
			},
			resource.TestStep{
				Config: testAccComputeInstance_update_deprecated_network(instanceName),
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
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
				),
			},
			resource.TestStep{
				Config: testAccComputeInstance_forceNewAndChangeMetadata(instanceName),
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
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
				),
			},
			resource.TestStep{
				Config: testAccComputeInstance_update(instanceName),
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
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_service_account(instanceName),
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

func TestAccComputeInstance_scheduling(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_scheduling(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
				),
			},
		},
	})
}

func TestAccComputeInstance_subnet_auto(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_subnet_auto(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceHasSubnet(&instance),
				),
			},
		},
	})
}

func TestAccComputeInstance_subnet_custom(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_subnet_custom(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceHasSubnet(&instance),
				),
			},
		},
	})
}

func TestAccComputeInstance_subnet_xpn(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))
	var xpn_host = os.Getenv("GOOGLE_XPN_HOST_PROJECT")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_subnet_xpn(instanceName, xpn_host),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceHasSubnet(&instance),
				),
			},
		},
	})
}

func TestAccComputeInstance_address_auto(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_address_auto(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceHasAnyAddress(&instance),
				),
			},
		},
	})
}

func TestAccComputeInstance_address_custom(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))
	var address = "10.0.200.200"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_address_custom(instanceName, address),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceHasAddress(&instance, address),
				),
			},
		},
	})
}

func TestAccComputeInstance_private_image_family(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))
	var diskName = fmt.Sprintf("instance-testd-%s", acctest.RandString(10))
	var imageName = fmt.Sprintf("instance-testi-%s", acctest.RandString(10))
	var familyName = fmt.Sprintf("instance-testf-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_private_image_family(diskName, imageName, familyName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(
						"google_compute_instance.foobar", &instance),
				),
			},
		},
	})
}

func TestAccComputeInstance_invalid_disk(t *testing.T) {
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))
	var diskName = fmt.Sprintf("instance-testd-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config:      testAccComputeInstance_invalid_disk(diskName, instanceName),
				ExpectError: regexp.MustCompile("Error: cannot define both disk and type."),
			},
		},
	})
}

func TestAccComputeInstance_forceChangeMachineTypeManually(t *testing.T) {
	var instance compute.Instance
	var instanceName = fmt.Sprintf("instance-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists("google_compute_instance.foobar", &instance),
					testAccCheckComputeInstanceUpdateMachineType("google_compute_instance.foobar"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckComputeInstanceUpdateMachineType(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		op, err := config.clientCompute.Instances.Stop(config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return fmt.Errorf("Could not stop instance: %s", err)
		}
		err = computeOperationWaitZone(config, op, config.Project, rs.Primary.Attributes["zone"], "Waiting on stop")
		if err != nil {
			return fmt.Errorf("Could not stop instance: %s", err)
		}

		machineType := compute.InstancesSetMachineTypeRequest{
			MachineType: "zones/us-central1-a/machineTypes/f1-micro",
		}

		op, err = config.clientCompute.Instances.SetMachineType(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID, &machineType).Do()
		if err != nil {
			return fmt.Errorf("Could not change machine type: %s", err)
		}
		err = computeOperationWaitZone(config, op, config.Project, rs.Primary.Attributes["zone"], "Waiting machine type change")
		if err != nil {
			return fmt.Errorf("Could not change machine type: %s", err)
		}
		return nil
	}
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

func testAccCheckComputeInstanceDiskEncryptionKey(n string, instance *compute.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		for i, disk := range instance.Disks {
			attr := rs.Primary.Attributes[fmt.Sprintf("disk.%d.disk_encryption_key_sha256", i)]
			if disk.DiskEncryptionKey == nil && attr != "" {
				return fmt.Errorf("Disk %d has mismatched encryption key.\nTF State: %+v\nGCP State: <empty>", i, attr)
			}
			if disk.DiskEncryptionKey != nil && attr != disk.DiskEncryptionKey.Sha256 {
				return fmt.Errorf("Disk %d has mismatched encryption key.\nTF State: %+v\nGCP State: %+v",
					i, attr, disk.DiskEncryptionKey.Sha256)
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

func testAccCheckComputeInstanceHasSubnet(instance *compute.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, i := range instance.NetworkInterfaces {
			if i.Subnetwork == "" {
				return fmt.Errorf("no subnet")
			}
		}

		return nil
	}
}

func testAccCheckComputeInstanceHasAnyAddress(instance *compute.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, i := range instance.NetworkInterfaces {
			if i.NetworkIP == "" {
				return fmt.Errorf("no address")
			}
		}

		return nil
	}
}

func testAccCheckComputeInstanceHasAddress(instance *compute.Instance, address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, i := range instance.NetworkInterfaces {
			if i.NetworkIP != address {
				return fmt.Errorf("Wrong address found: expected %v, got %v", address, i.NetworkIP)
			}
		}

		return nil
	}
}

func testAccComputeInstance_basic_deprecated_network(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			image = "debian-8-jessie-v20160803"
		}

		network {
			source = "default"
		}

		metadata {
			foo = "bar"
		}
	}`, instance)
}

func testAccComputeInstance_update_deprecated_network(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"
		tags = ["baz"]

		disk {
			image = "debian-8-jessie-v20160803"
		}

		network {
			source = "default"
		}

		metadata {
			bar = "baz"
		}
	}`, instance)
}

func testAccComputeInstance_basic(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			image = "debian-8-jessie-v20160803"
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
			baz = "qux"
		}

		create_timeout = 5

		metadata_startup_script = "echo Hello"
	}`, instance)
}

func testAccComputeInstance_basic2(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			image = "debian-8"
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}
	}`, instance)
}

func testAccComputeInstance_basic3(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			image = "debian-cloud/debian-8-jessie-v20160803"
		}

		network_interface {
			network = "default"
		}


		metadata {
			foo = "bar"
		}
	}`, instance)
}

func testAccComputeInstance_basic4(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			image = "debian-cloud/debian-8"
		}

		network_interface {
			network = "default"
		}


		metadata {
			foo = "bar"
		}
	}`, instance)
}

func testAccComputeInstance_basic5(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			image = "https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/debian-8-jessie-v20160803"
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}
	}`, instance)
}

// Update zone to ForceNew, and change metadata k/v entirely
// Generates diff mismatch
func testAccComputeInstance_forceNewAndChangeMetadata(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"
		zone = "us-central1-b"
		tags = ["baz"]

		disk {
			image = "debian-8-jessie-v20160803"
		}

		network_interface {
			network = "default"
			access_config { }
		}

		metadata {
			qux = "true"
		}
	}`, instance)
}

// Update metadata, tags, and network_interface
func testAccComputeInstance_update(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"
		tags = ["baz"]

		disk {
			image = "debian-8-jessie-v20160803"
		}

		network_interface {
			network = "default"
			access_config { }
		}

		metadata {
			bar = "baz"
		}
	}`, instance)
}

func testAccComputeInstance_ip(ip, instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_address" "foo" {
		name = "%s"
	}

	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"
		tags = ["foo", "bar"]

		disk {
			image = "debian-8-jessie-v20160803"
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
	}`, ip, instance)
}

func testAccComputeInstance_disks(disk, instance string, autodelete bool) string {
	return fmt.Sprintf(`
	resource "google_compute_disk" "foobar" {
		name = "%s"
		size = 10
		type = "pd-ssd"
		zone = "us-central1-a"
	}

	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"

		disk {
			image = "debian-8-jessie-v20160803"
		}

		disk {
			disk = "${google_compute_disk.foobar.name}"
			auto_delete = %v
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}
	}`, disk, instance, autodelete)
}

func testAccComputeInstance_disks_encryption(disk, instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_disk" "foobar" {
		name = "%s"
		size = 10
		type = "pd-ssd"
		zone = "us-central1-a"
	}

	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"

		disk {
			image = "debian-8-jessie-v20160803"
			disk_encryption_key_raw = "SGVsbG8gZnJvbSBHb29nbGUgQ2xvdWQgUGxhdGZvcm0="
		}

		disk {
			disk = "${google_compute_disk.foobar.name}"
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}
	}`, disk, instance)
}

func testAccComputeInstance_attachedDisk(disk, instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_disk" "foobar" {
		name = "%s"
		size = 10
		type = "pd-ssd"
		zone = "us-central1-a"
	}

	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"

		attached_disk {
			source = "${google_compute_disk.foobar.self_link}"
		}

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}
	}`, disk, instance)
}

func testAccComputeInstance_noDisk(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"

		network_interface {
			network = "default"
		}

		metadata {
			foo = "bar"
		}
	}`, instance)
}

func testAccComputeInstance_local_ssd(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "local-ssd" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"

		disk {
			image = "debian-8-jessie-v20160803"
		}

		disk {
			type = "local-ssd"
			scratch = true
		}

		network_interface {
			network = "default"
		}

	}`, instance)
}

func testAccComputeInstance_service_account(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"

		disk {
			image = "debian-8-jessie-v20160803"
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
	}`, instance)
}

func testAccComputeInstance_scheduling(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"

		disk {
			image = "debian-8-jessie-v20160803"
		}

		network_interface {
			network = "default"
		}

		scheduling {
		}
	}`, instance)
}

func testAccComputeInstance_subnet_auto(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_network" "inst-test-network" {
		name = "inst-test-network-%s"
		auto_create_subnetworks = true
	}

	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"

		disk {
			image = "debian-8-jessie-v20160803"
		}

		network_interface {
			network = "${google_compute_network.inst-test-network.name}"
			access_config {	}
		}

	}`, acctest.RandString(10), instance)
}

func testAccComputeInstance_subnet_custom(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_network" "inst-test-network" {
		name = "inst-test-network-%s"
		auto_create_subnetworks = false
	}

	resource "google_compute_subnetwork" "inst-test-subnetwork" {
		name = "inst-test-subnetwork-%s"
		ip_cidr_range = "10.0.0.0/16"
		region = "us-central1"
		network = "${google_compute_network.inst-test-network.self_link}"
	}

	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"

		disk {
			image = "debian-8-jessie-v20160803"
		}

		network_interface {
			subnetwork = "${google_compute_subnetwork.inst-test-subnetwork.name}"
			access_config {	}
		}

	}`, acctest.RandString(10), acctest.RandString(10), instance)
}

func testAccComputeInstance_subnet_xpn(instance, xpn_host string) string {
	return fmt.Sprintf(`
	resource "google_compute_network" "inst-test-network" {
		name = "inst-test-network-%s"
		auto_create_subnetworks = false
		project = "%s"
	}

	resource "google_compute_subnetwork" "inst-test-subnetwork" {
		name = "inst-test-subnetwork-%s"
		ip_cidr_range = "10.0.0.0/16"
		region = "us-central1"
		network = "${google_compute_network.inst-test-network.self_link}"
		project = "%s"
	}

	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"

		disk {
			image = "debian-8-jessie-v20160803"
		}

		network_interface {
			subnetwork = "${google_compute_subnetwork.inst-test-subnetwork.name}"
			subnetwork_project = "${google_compute_subnetwork.inst-test-subnetwork.project}"
			access_config {	}
		}

	}`, acctest.RandString(10), xpn_host, acctest.RandString(10), xpn_host, instance)
}

func testAccComputeInstance_address_auto(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_network" "inst-test-network" {
		name = "inst-test-network-%s"
	}
	resource "google_compute_subnetwork" "inst-test-subnetwork" {
		name = "inst-test-subnetwork-%s"
		ip_cidr_range = "10.0.0.0/16"
		region = "us-central1"
		network = "${google_compute_network.inst-test-network.self_link}"
	}
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"

		disk {
			image = "debian-8-jessie-v20160803"
		}

		network_interface {
			subnetwork = "${google_compute_subnetwork.inst-test-subnetwork.name}"
			access_config {	}
		}

	}`, acctest.RandString(10), acctest.RandString(10), instance)
}

func testAccComputeInstance_address_custom(instance, address string) string {
	return fmt.Sprintf(`
	resource "google_compute_network" "inst-test-network" {
		name = "inst-test-network-%s"
	}
	resource "google_compute_subnetwork" "inst-test-subnetwork" {
		name = "inst-test-subnetwork-%s"
		ip_cidr_range = "10.0.0.0/16"
		region = "us-central1"
		network = "${google_compute_network.inst-test-network.self_link}"
	}
	resource "google_compute_instance" "foobar" {
		name = "%s"
		machine_type = "n1-standard-1"
		zone = "us-central1-a"

		disk {
			image = "debian-8-jessie-v20160803"
		}

		network_interface {
			subnetwork = "${google_compute_subnetwork.inst-test-subnetwork.name}"
		    address = "%s"
			access_config {	}
		}

	}`, acctest.RandString(10), acctest.RandString(10), instance, address)
}

func testAccComputeInstance_private_image_family(disk, image, family, instance string) string {
	return fmt.Sprintf(`
		resource "google_compute_disk" "foobar" {
			name = "%s"
			zone = "us-central1-a"
			image = "debian-8-jessie-v20160803"
		}

		resource "google_compute_image" "foobar" {
			name = "%s"
			source_disk = "${google_compute_disk.foobar.self_link}"
			family = "%s"
		}

		resource "google_compute_instance" "foobar" {
			name = "%s"
			machine_type = "n1-standard-1"
			zone = "us-central1-a"

			disk {
				image = "${google_compute_image.foobar.family}"
			}

			network_interface {
				network = "default"
			}

			metadata {
				foo = "bar"
			}
		}`, disk, image, family, instance)
}

func testAccComputeInstance_invalid_disk(disk, instance string) string {
	return fmt.Sprintf(`
		resource "google_compute_instance" "foobar" {
		  name         = "%s"
		  machine_type = "f1-micro"
		  zone         = "us-central1-a"

		  disk {
		    image = "ubuntu-os-cloud/ubuntu-1604-lts"
		    type  = "pd-standard"
		  }

		  disk {
		    disk        = "${google_compute_disk.foobar.name}"
		    type        = "pd-standard"
		    device_name = "xvdb"
		  }

		  network_interface {
		    network = "default"
		  }
		}

		resource "google_compute_disk" "foobar" {
		  name = "%s"
		  zone = "us-central1-a"
		  type = "pd-standard"
		  size = "1"
		}`, instance, disk)
}
