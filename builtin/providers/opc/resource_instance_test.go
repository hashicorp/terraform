package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCInstance_basic(t *testing.T) {
	resName := "opc_compute_instance.test"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceBasic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckInstanceExists,
					resource.TestCheckResourceAttr(resName, "name", fmt.Sprintf("acc-test-instance-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "label", "TestAccOPCInstance_basic"),
				),
			},
		},
	})
}

func TestAccOPCInstance_sharedNetworking(t *testing.T) {
	rInt := acctest.RandInt()
	resName := "opc_compute_instance.test"
	dataName := "data.opc_compute_network_interface.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceSharedNetworking(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckInstanceExists,
					resource.TestCheckResourceAttrSet(resName, "id"),
					resource.TestCheckResourceAttrSet(resName, "availability_domain"),
					resource.TestCheckResourceAttrSet(resName, "domain"),
					resource.TestCheckResourceAttrSet(resName, "hostname"),
					resource.TestCheckResourceAttrSet(resName, "ip_address"),
					resource.TestCheckResourceAttr(resName, "name", fmt.Sprintf("acc-test-instance-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "networking_info.#", "1"),
					// Default Placement Reqs
					resource.TestCheckResourceAttr(resName, "placement_requirements.#", "2"),
					resource.TestCheckResourceAttr(resName, "placement_requirements.0", "/system/compute/allow_instances"),
					resource.TestCheckResourceAttr(resName, "placement_requirements.1", "/system/compute/placement/default"),
					resource.TestCheckResourceAttr(resName, "platform", "linux"),
					resource.TestCheckResourceAttr(resName, "priority", "/oracle/public/default"),
					resource.TestCheckResourceAttr(resName, "reverse_dns", "true"),
					resource.TestCheckResourceAttr(resName, "state", "running"),
					resource.TestCheckResourceAttr(resName, "tags.#", "2"),
					resource.TestCheckResourceAttrSet(resName, "vcable"),
					resource.TestCheckResourceAttr(resName, "virtio", "false"),

					// Check Data Source to validate networking attributes
					resource.TestCheckResourceAttr(dataName, "shared_network", "true"),
					resource.TestCheckResourceAttr(dataName, "nat.#", "1"),
					resource.TestCheckResourceAttr(dataName, "sec_lists.#", "1"),
					resource.TestCheckResourceAttr(dataName, "name_servers.#", "0"),
					resource.TestCheckResourceAttr(dataName, "vnic_sets.#", "0"),
				),
			},
		},
	})
}

func TestAccOPCInstance_ipNetwork(t *testing.T) {
	rInt := acctest.RandInt()
	resName := "opc_compute_instance.test"
	dataName := "data.opc_compute_network_interface.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceIPNetworking(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckInstanceExists,
					resource.TestCheckResourceAttrSet(resName, "id"),
					resource.TestCheckResourceAttrSet(resName, "availability_domain"),
					resource.TestCheckResourceAttrSet(resName, "domain"),
					resource.TestCheckResourceAttrSet(resName, "ip_address"),
					resource.TestCheckResourceAttr(resName, "name", fmt.Sprintf("acc-test-instance-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "networking_info.#", "1"),
					// Default Placement Reqs
					resource.TestCheckResourceAttr(resName, "placement_requirements.#", "2"),
					resource.TestCheckResourceAttr(resName, "placement_requirements.0", "/system/compute/allow_instances"),
					resource.TestCheckResourceAttr(resName, "placement_requirements.1", "/system/compute/placement/default"),
					resource.TestCheckResourceAttr(resName, "platform", "linux"),
					resource.TestCheckResourceAttr(resName, "priority", "/oracle/public/default"),
					resource.TestCheckResourceAttr(resName, "reverse_dns", "true"),
					resource.TestCheckResourceAttr(resName, "state", "running"),
					resource.TestCheckResourceAttr(resName, "virtio", "false"),

					// Check Data Source to validate networking attributes
					resource.TestCheckResourceAttr(dataName, "ip_network", fmt.Sprintf("testing-ip-network-%d", rInt)),
					resource.TestCheckResourceAttr(dataName, "vnic", fmt.Sprintf("ip-network-test-%d", rInt)),
					resource.TestCheckResourceAttr(dataName, "shared_network", "false"),
				),
			},
		},
	})
}

func TestAccOPCInstance_storage(t *testing.T) {
	resName := "opc_compute_instance.test"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceStorage(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckInstanceExists,
					resource.TestCheckResourceAttr(resName, "storage.#", "2"),
				),
			},
		},
	})
}

func TestAccOPCInstance_emptyLabel(t *testing.T) {
	resName := "opc_compute_instance.test"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceEmptyLabel(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckInstanceExists,
					resource.TestCheckResourceAttr(resName, "name", fmt.Sprintf("acc-test-instance-%d", rInt)),
					resource.TestCheckResourceAttrSet(resName, "label"),
				),
			},
		},
	})
}

func testAccOPCCheckInstanceExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).Instances()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_instance" {
			continue
		}

		input := &compute.GetInstanceInput{
			ID:   rs.Primary.ID,
			Name: rs.Primary.Attributes["name"],
		}
		_, err := client.GetInstance(input)
		if err != nil {
			return fmt.Errorf("Error retrieving state of Instance %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccOPCCheckInstanceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).Instances()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_instance" {
			continue
		}

		input := &compute.GetInstanceInput{
			ID:   rs.Primary.ID,
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetInstance(input); err == nil {
			return fmt.Errorf("Instance %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

func testAccInstanceBasic(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_instance" "test" {
	name = "acc-test-instance-%d"
	label = "TestAccOPCInstance_basic"
	shape = "oc3"
	image_list = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
	instance_attributes = <<JSON
{
  "foo": "bar"
}
JSON
}`, rInt)
}

func testAccInstanceSharedNetworking(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_instance" "test" {
  name = "acc-test-instance-%d"
  label = "TestAccOPCInstance_sharedNetworking"
  shape = "oc3"
  image_list = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
  tags = ["tag1", "tag2"]
  networking_info {
    index = 0
    nat = ["ippool:/oracle/public/ippool"]
    shared_network = true
  }
}

data "opc_compute_network_interface" "test" {
  instance_name = "${opc_compute_instance.test.name}"
  instance_id = "${opc_compute_instance.test.id}"
  interface = "eth0"
}
`, rInt)
}

func testAccInstanceIPNetworking(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_network" "foo" {
  name = "testing-ip-network-%d"
  description = "testing-ip-network-instance"
  ip_address_prefix = "10.1.12.0/24"
}

resource "opc_compute_instance" "test" {
  name = "acc-test-instance-%d"
  label = "TestAccOPCInstance_ipNetwork"
  shape = "oc3"
  image_list = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
  networking_info {
    index = 0
    ip_network = "${opc_compute_ip_network.foo.id}"
    vnic = "ip-network-test-%d"
    shared_network = false
  }
}

data "opc_compute_network_interface" "test" {
  instance_id = "${opc_compute_instance.test.id}"
  instance_name = "${opc_compute_instance.test.name}"
  interface = "eth0"
}
`, rInt, rInt, rInt)
}

func testAccInstanceStorage(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_storage_volume" "foo" {
  name = "acc-test-instance-%d"
  size = 1
}

resource "opc_compute_storage_volume" "bar" {
  name = "acc-test-instance-2-%d"
  size = 1
}

resource "opc_compute_instance" "test" {
	name = "acc-test-instance-%d"
	label = "TestAccOPCInstance_basic"
	shape = "oc3"
	image_list = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
	storage {
		volume = "${opc_compute_storage_volume.foo.name}"
		index = 1
	}
	storage {
	  volume = "${opc_compute_storage_volume.bar.name}"
	  index = 2
	}
}`, rInt, rInt, rInt)
}

func testAccInstanceEmptyLabel(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_instance" "test" {
	name = "acc-test-instance-%d"
	shape = "oc3"
	image_list = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
	instance_attributes = <<JSON
{
  "foo": "bar"
}
JSON
}`, rInt)
}
