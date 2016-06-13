package vsphere

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"golang.org/x/net/context"
)

func testVirtualDiskPreCheck(t *testing.T) {
	if v := os.Getenv("VSPHERE_INIT_TYPE"); v == "" {
		t.Fatal("VSPHERE_INIT_TYPE must be set to test")
	}
	if v := os.Getenv("VSPHERE_ADAPTER_TYPE"); v == "" {
		t.Fatal("VSPHERE_ADAPTER_TYPE must be set to test")
	}
	testAccPreCheck(t)
}

func TestAccVSphereVirtualDisk_basic(t *testing.T) {
	var datacenterOpt string
	var datastoreOpt string
	var initTypeOpt string
	var adapterTypeOpt string

	if v := os.Getenv("VSPHERE_DATACENTER"); v != "" {
		datacenterOpt = v
	}
	if v := os.Getenv("VSPHERE_DATASTORE"); v != "" {
		datastoreOpt = v
	}
	if v := os.Getenv("VSPHERE_INIT_TYPE"); v != "" {
		initTypeOpt += fmt.Sprintf("    type = \"%s\"\n", v)
	}
	if v := os.Getenv("VSPHERE_ADAPTER_TYPE"); v != "" {
		adapterTypeOpt += fmt.Sprintf("    adapter_type = \"%s\"\n", v)
	}

	config := fmt.Sprintf(
		testAccCheckVSphereVirtuaDiskConfig_basic,
		initTypeOpt,
		adapterTypeOpt,
		datacenterOpt,
		datastoreOpt,
	)

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtuaDiskConfig_basic)
	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testVirtualDiskPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereVirtualDiskExists("vsphere_virtual_disk.foo"),
				),
			},
		},
	})
}

func testAccVSphereVirtualDiskExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*govmomi.Client)
		finder := find.NewFinder(client.Client, true)

		dc, err := finder.Datacenter(context.TODO(), rs.Primary.Attributes["datacenter"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}
		finder = finder.SetDatacenter(dc)

		ds, err := finder.Datastore(context.TODO(), rs.Primary.Attributes["datastore"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		_, err = ds.Stat(context.TODO(), rs.Primary.Attributes["vmdk_path"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		return nil
	}
}

func testAccCheckVSphereVirtualDiskDestroy(s *terraform.State) error {
	log.Printf("[FINDME] test Destroy")
	client := testAccProvider.Meta().(*govmomi.Client)
	finder := find.NewFinder(client.Client, true)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vsphere_virtual_disk" {
			continue
		}

		dc, err := finder.Datacenter(context.TODO(), rs.Primary.Attributes["datacenter"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		finder = finder.SetDatacenter(dc)

		ds, err := finder.Datastore(context.TODO(), rs.Primary.Attributes["datastore"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		_, err = ds.Stat(context.TODO(), rs.Primary.Attributes["vmdk_path"])
		if err == nil {
			return fmt.Errorf("error %s", err)
		}
	}

	return nil
}

const testAccCheckVSphereVirtuaDiskConfig_basic = `
resource "vsphere_virtual_disk" "foo" {
    size = 1
    vmdk_path = "tfTestDisk.vmdk"
%s
%s
    datacenter = "%s"
    datastore = "%s"
}
`
