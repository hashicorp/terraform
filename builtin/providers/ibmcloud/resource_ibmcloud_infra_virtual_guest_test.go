package ibmcloud

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/services"
)

func TestAccIBMCloudInfraVirtualGuest_basic(t *testing.T) {
	var guest datatypes.Virtual_Guest

	hostname := acctest.RandString(16)
	domain := "terraform.ibm.com"
	networkSpeed1 := "10"
	networkSpeed2 := "100"
	cores1 := "1"
	cores2 := "2"
	memory1 := "1024"
	memory2 := "2048"
	tags1 := "collectd"
	tags2 := "mesos-master"
	userMetadata1 := "{\\\"value\\\":\\\"newvalue\\\"}"
	userMetadata1Unquoted, _ := strconv.Unquote(`"` + userMetadata1 + `"`)
	userMetadata2 := "updated"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIBMCloudInfraVirtualGuestDestroy,
		Steps: []resource.TestStep{
			{
				Config:  testAccIBMCloudInfraVirtualGuestConfigBasic(hostname, domain, networkSpeed1, cores1, memory1, userMetadata1, tags1),
				Destroy: false,
				Check: resource.ComposeTestCheckFunc(
					testAccIBMCloudInfraVirtualGuestExists("ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", &guest),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "hostname", hostname),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "domain", domain),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "datacenter", "wdc04"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "network_speed", networkSpeed1),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "hourly_billing", "true"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "private_network_only", "false"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "cores", cores1),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "memory", memory1),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "disks.0", "25"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "disks.1", "10"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "disks.2", "20"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "user_metadata", userMetadata1Unquoted),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "local_disk", "false"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "dedicated_acct_host_only", "true"),
					CheckStringSet(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1",
						"tags", []string{tags1},
					),
					resource.TestCheckResourceAttrSet(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "ipv6_enabled"),
					resource.TestCheckResourceAttrSet(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "ipv6_address"),
					resource.TestCheckResourceAttrSet(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "ipv6_address_id"),
					resource.TestCheckResourceAttrSet(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "public_ipv6_subnet"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "secondary_ip_count", "4"),
					resource.TestCheckResourceAttrSet(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "secondary_ip_addresses.3"),
				),
			},

			{
				Config:  testAccIBMCloudInfraVirtualGuestConfigBasic(hostname, domain, networkSpeed1, cores1, memory1, userMetadata2, tags2),
				Destroy: false,
				Check: resource.ComposeTestCheckFunc(
					testAccIBMCloudInfraVirtualGuestExists("ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", &guest),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "user_metadata", userMetadata2),
					CheckStringSet(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1",
						"tags", []string{tags2},
					),
				),
			},

			{
				Config: testAccIBMCloudInfraVirtualGuestConfigBasic(hostname, domain, networkSpeed2, cores2, memory2, userMetadata2, tags2),
				Check: resource.ComposeTestCheckFunc(
					testAccIBMCloudInfraVirtualGuestExists("ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", &guest),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "cores", cores2),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "memory", memory2),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-1", "network_speed", networkSpeed2),
				),
			},
		},
	})
}

func TestAccIBMCloudInfraVirtualGuest_BlockDeviceTemplateGroup(t *testing.T) {
	var guest datatypes.Virtual_Guest

	hostname := acctest.RandString(16)
	domain := "bdtg.terraform.ibm.com"

	imageID := os.Getenv("IBMCLOUD_VIRTUAL_GUEST_IMAGE_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIBMCloudInfraVirtualGuestDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIBMCloudInfraVirtualGuestConfigBlockDeviceTemplateGroup(hostname, domain, imageID),
				Check: resource.ComposeTestCheckFunc(
					// image_id value is hardcoded. If it's valid then virtual guest will be created well
					testAccIBMCloudInfraVirtualGuestExists("ibmcloud_infra_virtual_guest.terraform-acceptance-test-BDTGroup", &guest),
				),
			},
		},
	})
}

func TestAccIBMCloudInfraVirtualGuest_CustomImageMultipleDisks(t *testing.T) {
	var guest datatypes.Virtual_Guest
	hostname := acctest.RandString(16)
	domain := "mdisk.terraform.ibm.com"

	imageID := os.Getenv("IBMCLOUD_VIRTUAL_GUEST_IMAGE_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIBMCloudInfraVirtualGuestDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIBMCloudInfraVirtualGuestConfigCustomImageMultipleDisks(hostname, domain, imageID),
				Check: resource.ComposeTestCheckFunc(
					// image_id value is hardcoded. If it's valid then virtual guest will be created well
					testAccIBMCloudInfraVirtualGuestExists("ibmcloud_infra_virtual_guest.terraform-acceptance-test-disks", &guest),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-disks", "disks.0", "25"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-disks", "disks.1", "10"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-disks", "hostname", hostname),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_virtual_guest.terraform-acceptance-test-disks", "domain", domain),
				),
			},
		},
	})
}

func TestAccIBMCloudInfraVirtualGuest_PostInstallScriptUri(t *testing.T) {
	var guest datatypes.Virtual_Guest

	hostname := acctest.RandString(16)
	domain := "pis.terraform.ibm.com"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIBMCloudInfraVirtualGuestDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIBMCloudInfraVirtualGuestConfigPostInstallScriptURI(hostname, domain),
				Check: resource.ComposeTestCheckFunc(
					// image_id value is hardcoded. If it's valid then virtual guest will be created well
					testAccIBMCloudInfraVirtualGuestExists("ibmcloud_infra_virtual_guest.terraform-acceptance-test-pISU", &guest),
				),
			},
		},
	})
}

func testAccIBMCloudInfraVirtualGuestDestroy(s *terraform.State) error {
	service := services.GetVirtualGuestService(testAccProvider.Meta().(ClientSession).SoftLayerSession())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ibmcloud_infra_virtual_guest" {
			continue
		}

		guestID, _ := strconv.Atoi(rs.Primary.ID)

		// Try to find the guest
		_, err := service.Id(guestID).GetObject()

		// Wait

		if err != nil && !strings.Contains(err.Error(), "404") {
			return fmt.Errorf(
				"Error waiting for virtual guest (%s) to be destroyed: %s",
				rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccIBMCloudInfraVirtualGuestExists(n string, guest *datatypes.Virtual_Guest) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No virtual guest ID is set")
		}

		id, err := strconv.Atoi(rs.Primary.ID)

		if err != nil {
			return err
		}

		service := services.GetVirtualGuestService(testAccProvider.Meta().(ClientSession).SoftLayerSession())
		retrieveVirtGuest, err := service.Id(id).GetObject()

		if err != nil {
			return err
		}

		fmt.Printf("The ID is %d\n", id)

		if *retrieveVirtGuest.Id != id {
			return errors.New("Virtual guest not found")
		}

		*guest = retrieveVirtGuest

		return nil
	}
}

func CheckStringSet(n string, name string, set []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		values := []string{}
		setLengthKey := fmt.Sprintf("%s.#", name)
		prefix := fmt.Sprintf("%s.", name)
		for k, v := range rs.Primary.Attributes {
			if k != setLengthKey && strings.HasPrefix(k, prefix) {
				values = append(values, v)
			}
		}

		if len(values) == 0 {
			return fmt.Errorf("Could not find %s.%s", n, name)
		}

		for _, s := range set {
			found := false
			for _, v := range values {
				if s == v {
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("%s was not found in the set %s", s, name)
			}
		}

		return nil
	}
}

func testAccIBMCloudInfraVirtualGuestConfigBasic(hostname, domain, networkSpeed, cores, memory, userMetadata, tags string) string {
	return fmt.Sprintf(`
resource "ibmcloud_infra_virtual_guest" "terraform-acceptance-test-1" {
    hostname = "%s"
    domain = "%s"
    os_reference_code = "DEBIAN_7_64"
    datacenter = "wdc04"
    network_speed = %s
    hourly_billing = true
    private_network_only = false
    cores = %s
    memory = %s
    disks = [25, 10, 20]
    user_metadata = "%s"
    tags = ["%s"]
    dedicated_acct_host_only = true
    local_disk = false
    ipv6_enabled = true
    secondary_ip_count = 4
}`, hostname, domain, networkSpeed, cores, memory, userMetadata, tags)
}

func testAccIBMCloudInfraVirtualGuestConfigPostInstallScriptURI(hostname, domain string) string {
	return fmt.Sprintf(`
resource "ibmcloud_infra_virtual_guest" "terraform-acceptance-test-pISU" {
    hostname = "%s"
    domain = "%s"
    os_reference_code = "DEBIAN_7_64"
    datacenter = "wdc04"
    network_speed = 10
    hourly_billing = true
	private_network_only = false
    cores = 1
    memory = 1024
    disks = [25, 10, 20]
    user_metadata = "{\"value\":\"newvalue\"}"
    dedicated_acct_host_only = true
    local_disk = false
    post_install_script_uri = "https://www.google.com"
}`, hostname, domain)
}

func testAccIBMCloudInfraVirtualGuestConfigBlockDeviceTemplateGroup(hostname, domain, imageID string) string {
	return fmt.Sprintf(`
resource "ibmcloud_infra_virtual_guest" "terraform-acceptance-test-BDTGroup" {
    hostname = "%s"
    domain = "%s"
    datacenter = "wdc04"
    network_speed = 10
    hourly_billing = false
    cores = 1
    memory = 1024
    local_disk = false
    image_id = %s
}`, hostname, domain, imageID)
}

func testAccIBMCloudInfraVirtualGuestConfigCustomImageMultipleDisks(hostname, domain, imageID string) string {
	return fmt.Sprintf(`
resource "ibmcloud_infra_virtual_guest" "terraform-acceptance-test-disks" {
    hostname = "%s"
    domain = "%s"
    datacenter = "wdc04"
    network_speed = 10
    hourly_billing = false
    cores = 1
    memory = 1024
    local_disk = false
    image_id = %s
    disks = [25, 10]
}`, hostname, domain, imageID)

}
