package vsphere

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"golang.org/x/net/context"
)

// Basic file creation
func TestAccVSphereFile_basic(t *testing.T) {
	testVmdkFileData := []byte("# Disk DescriptorFile\n")
	testVmdkFile := "/tmp/tf_test.vmdk"
	err := ioutil.WriteFile(testVmdkFile, testVmdkFileData, 0644)
	if err != nil {
		t.Errorf("error %s", err)
		return
	}

	datacenter := os.Getenv("VSPHERE_DATACENTER")
	datastore := os.Getenv("VSPHERE_DATASTORE")
	testMethod := "basic"
	resourceName := "vsphere_file." + testMethod
	destinationFile := "tf_file_test.vmdk"
	sourceFile := testVmdkFile

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereFileDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccCheckVSphereFileConfig,
					testMethod,
					datacenter,
					datastore,
					sourceFile,
					destinationFile,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVSphereFileExists(resourceName, destinationFile, true),
					resource.TestCheckResourceAttr(resourceName, "destination_file", destinationFile),
				),
			},
		},
	})
	os.Remove(testVmdkFile)
}

// file creation followed by a rename of file (update)
func TestAccVSphereFile_renamePostCreation(t *testing.T) {
	testVmdkFileData := []byte("# Disk DescriptorFile\n")
	testVmdkFile := "/tmp/tf_test.vmdk"
	err := ioutil.WriteFile(testVmdkFile, testVmdkFileData, 0644)
	if err != nil {
		t.Errorf("error %s", err)
		return
	}

	datacenter := os.Getenv("VSPHERE_DATACENTER")
	datastore := os.Getenv("VSPHERE_DATASTORE")
	testMethod := "basic"
	resourceName := "vsphere_file." + testMethod
	destinationFile := "tf_test_file.vmdk"
	destinationFileMoved := "tf_test_file_moved.vmdk"
	sourceFile := testVmdkFile

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereFolderDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccCheckVSphereFileConfig,
					testMethod,
					datacenter,
					datastore,
					sourceFile,
					destinationFile,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVSphereFileExists(resourceName, destinationFile, true),
					testAccCheckVSphereFileExists(resourceName, destinationFileMoved, false),
					resource.TestCheckResourceAttr(resourceName, "destination_file", destinationFile),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccCheckVSphereFileConfig,
					testMethod,
					datacenter,
					datastore,
					sourceFile,
					destinationFileMoved,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVSphereFileExists(resourceName, destinationFile, false),
					testAccCheckVSphereFileExists(resourceName, destinationFileMoved, true),
					resource.TestCheckResourceAttr(resourceName, "destination_file", destinationFileMoved),
				),
			},
		},
	})
	os.Remove(testVmdkFile)
}

func testAccCheckVSphereFileDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*govmomi.Client)
	finder := find.NewFinder(client.Client, true)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vsphere_file" {
			continue
		}

		dc, err := finder.Datacenter(context.TODO(), rs.Primary.Attributes["datacenter"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		finder = finder.SetDatacenter(dc)

		ds, err := getDatastore(finder, rs.Primary.Attributes["datastore"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		_, err = ds.Stat(context.TODO(), rs.Primary.Attributes["destination_file"])
		if err != nil {
			switch e := err.(type) {
			case object.DatastoreNoSuchFileError:
				fmt.Printf("Expected error received: %s\n", e.Error())
				return nil
			default:
				return err
			}
		} else {
			return fmt.Errorf("File %s still exists", rs.Primary.Attributes["destination_file"])
		}
	}

	return nil
}

func testAccCheckVSphereFileExists(n string, df string, exists bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
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

		ds, err := getDatastore(finder, rs.Primary.Attributes["datastore"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		_, err = ds.Stat(context.TODO(), df)
		if err != nil {
			switch e := err.(type) {
			case object.DatastoreNoSuchFileError:
				if exists {
					return fmt.Errorf("File does not exist: %s", e.Error())
				}
				fmt.Printf("Expected error received: %s\n", e.Error())
				return nil
			default:
				return err
			}
		}
		return nil
	}
}

const testAccCheckVSphereFileConfig = `
resource "vsphere_file" "%s" {
	datacenter = "%s"
	datastore = "%s"
	source_file = "%s"
	destination_file = "%s"
}
`
