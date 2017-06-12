package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"golang.org/x/net/context"
	"os"
	"testing"
)

func testBasicPreCheckSnapshot(t *testing.T) {

	testAccPreCheck(t)

	if v := os.Getenv("VSPHERE_VM_NAME"); v == "" {
		t.Fatal("env variable VSPHERE_VM_NAME must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_VM_FOLDER"); v == "" {
		t.Fatal("env variable VSPHERE_VM_FOLDER must be set for acceptance tests")
	}
}

func TestAccVmSnanpshot_Basic(t *testing.T) {
	snapshot_name := "SnapshotForTestingTerraform"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVmSnapshotDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckVmSnapshotConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVmSnapshotExists("vsphere_snapshot.Test_terraform_cases", snapshot_name),
					resource.TestCheckResourceAttr(
						"vsphere_snapshot.Test_terraform_cases", "snapshot_name", "SnapshotForTestingTerraform"),
				),
			},
		},
	})
}

func testAccCheckVmSnapshotDestroy(s *terraform.State) error {

	client := testAccProvider.Meta().(*govmomi.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vsphere_snapshot" {
			continue
		}
		dc, err := getDatacenter(client, "")
		if err != nil {
			return fmt.Errorf("error %s", err)
		}
		finder := find.NewFinder(client.Client, true)
		finder = finder.SetDatacenter(dc)
		vm, err := finder.VirtualMachine(context.TODO(), vmPath(rs.Primary.Attributes["folder"], rs.Primary.Attributes["vm_name"]))
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		snapshot, err := vm.FindSnapshot(context.TODO(), rs.Primary.Attributes["snapshot_name"])
		if err == nil {
			return fmt.Errorf("Vm Snapshot still exists: %v", snapshot)
		}
	}

	return nil
}

func testAccCheckVmSnapshotExists(n, snapshot_name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Vm Snapshot ID is set")
		}
		client := testAccProvider.Meta().(*govmomi.Client)

		dc, err := getDatacenter(client, "")
		if err != nil {
			return fmt.Errorf("error %s", err)
		}
		finder := find.NewFinder(client.Client, true)
		finder = finder.SetDatacenter(dc)
		vm, err := finder.VirtualMachine(context.TODO(), vmPath(os.Getenv("VSPHERE_VM_FOLDER"), os.Getenv("VSPHERE_VM_NAME")))
		if err != nil {
			return fmt.Errorf("error %s", err)
		}
		snapshot, err := vm.FindSnapshot(context.TODO(), snapshot_name)
		if err != nil {
			return fmt.Errorf("Error while getting the snapshot %v", snapshot)
		}

		return nil
	}
}

const testAccCheckVmSnapshotConfig_basic = `
resource "vsphere_snapshot" "Test_terraform_cases"{
 
	snapshot_name = "SnapshotForTestingTerraform"
	description = "This is snpashot created for testing and will be deleted."
	memory = "true"
	quiesce = "true"
	remove_children = "false"
	consolidate = "true"
}`
