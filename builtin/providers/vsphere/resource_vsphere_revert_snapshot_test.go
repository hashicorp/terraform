package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/mo"
	"golang.org/x/net/context"
	"os"
	"testing"
)

func testBasicPreCheckSnapshotRevert(t *testing.T) {

	testAccPreCheck(t)

	if v := os.Getenv("VSPHERE_VM_NAME"); v == "" {
		t.Fatal("env variable VSPHERE_VM_NAME must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_VM_FOLDER"); v == "" {
		t.Fatal("env variable VSPHERE_VM_FOLDER must be set for acceptance tests")
	}
}

func TestAccVmSnanpshotRevert_Basic(t *testing.T) {
	snapshot_name := "SnapshotForTestingTerraform"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVmSnapshotRevertDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckVmSnapshotRevertConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVmCurrentSnapshot("vsphere_snapshot_revert.Test_terraform_cases", snapshot_name),
				),
			},
		},
	})
}

func testAccCheckVmSnapshotRevertDestroy(s *terraform.State) error {

	return nil
}

func testAccCheckVmCurrentSnapshot(n, snapshot_name string) resource.TestCheckFunc {
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

		var vm_object mo.VirtualMachine

		err = vm.Properties(context.TODO(), vm.Reference(), []string{"snapshot"}, &vm_object)

		if err != nil {
			return nil
		}
		current_snap := vm_object.Snapshot.CurrentSnapshot
		snapshot, err := vm.FindSnapshot(context.TODO(), snapshot_name)

		if err != nil {
			return fmt.Errorf("Error while getting the snapshot %v", snapshot)
		}
		if fmt.Sprintf("<%s>", snapshot) == fmt.Sprintf("<%s>", current_snap) {
			return nil
		}

		return fmt.Errorf("Test Case failed for revert snapshot. Current snapshot does not match to reverted snapshot")
	}
}

const testAccCheckVmSnapshotRevertConfig_basic = `
resource "vsphere_snapshot_revert" "Test_terraform_cases"{
 
	snapshot_name = "SnapshotForTestingTerraform"
	suppress_power_on = "true"
}`
