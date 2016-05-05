package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"golang.org/x/net/context"
)

// Basic top-level folder creation
func TestAccVSphereFolder_basic(t *testing.T) {
	var f folder
	datacenter := os.Getenv("VSPHERE_DATACENTER")
	testMethod := "basic"
	resourceName := "vsphere_folder." + testMethod
	path := "tf_test_basic"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereFolderDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereFolderConfig,
					testMethod,
					path,
					datacenter,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVSphereFolderExists(resourceName, &f),
					resource.TestCheckResourceAttr(
						resourceName, "path", path),
					resource.TestCheckResourceAttr(
						resourceName, "existing_path", ""),
				),
			},
		},
	})
}

func TestAccVSphereFolder_nested(t *testing.T) {

	var f folder
	datacenter := os.Getenv("VSPHERE_DATACENTER")
	testMethod := "nested"
	resourceName := "vsphere_folder." + testMethod
	path := "tf_test_nested/tf_test_folder"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereFolderDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereFolderConfig,
					testMethod,
					path,
					datacenter,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVSphereFolderExists(resourceName, &f),
					resource.TestCheckResourceAttr(
						resourceName, "path", path),
					resource.TestCheckResourceAttr(
						resourceName, "existing_path", ""),
				),
			},
		},
	})
}

func TestAccVSphereFolder_dontDeleteExisting(t *testing.T) {

	var f folder
	datacenter := os.Getenv("VSPHERE_DATACENTER")
	testMethod := "dontDeleteExisting"
	resourceName := "vsphere_folder." + testMethod
	existingPath := "tf_test_dontDeleteExisting/tf_existing"
	path := existingPath + "/tf_nested/tf_test"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			assertVSphereFolderExists(datacenter, existingPath),
			removeVSphereFolder(datacenter, existingPath, ""),
		),
		Steps: []resource.TestStep{
			resource.TestStep{
				PreConfig: func() {
					createVSphereFolder(datacenter, existingPath)
				},
				Config: fmt.Sprintf(
					testAccCheckVSphereFolderConfig,
					testMethod,
					path,
					datacenter,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVSphereFolderExistingPathExists(resourceName, &f),
					resource.TestCheckResourceAttr(
						resourceName, "path", path),
					resource.TestCheckResourceAttr(
						resourceName, "existing_path", existingPath),
				),
			},
		},
	})
}

func testAccCheckVSphereFolderDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*govmomi.Client)
	finder := find.NewFinder(client.Client, true)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vsphere_folder" {
			continue
		}

		dc, err := finder.Datacenter(context.TODO(), rs.Primary.Attributes["datacenter"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		dcFolders, err := dc.Folders(context.TODO())
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		f, err := object.NewSearchIndex(client.Client).FindChild(context.TODO(), dcFolders.VmFolder, rs.Primary.Attributes["path"])
		if f != nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckVSphereFolderExists(n string, f *folder) resource.TestCheckFunc {
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

		dcFolders, err := dc.Folders(context.TODO())
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		_, err = object.NewSearchIndex(client.Client).FindChild(context.TODO(), dcFolders.VmFolder, rs.Primary.Attributes["path"])

		*f = folder{
			path: rs.Primary.Attributes["path"],
		}

		return nil
	}
}

func testAccCheckVSphereFolderExistingPathExists(n string, f *folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource %s not found in %#v", n, s.RootModule().Resources)
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

		dcFolders, err := dc.Folders(context.TODO())
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		_, err = object.NewSearchIndex(client.Client).FindChild(context.TODO(), dcFolders.VmFolder, rs.Primary.Attributes["existing_path"])

		*f = folder{
			path: rs.Primary.Attributes["path"],
		}

		return nil
	}
}

func assertVSphereFolderExists(datacenter string, folder_name string) resource.TestCheckFunc {

	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*govmomi.Client)
		folder, err := object.NewSearchIndex(client.Client).FindByInventoryPath(
			context.TODO(), fmt.Sprintf("%v/vm/%v", datacenter, folder_name))
		if err != nil {
			return fmt.Errorf("Error: %s", err)
		} else if folder == nil {
			return fmt.Errorf("Folder %s does not exist!", folder_name)
		}

		return nil
	}
}

func createVSphereFolder(datacenter string, folder_name string) error {

	client := testAccProvider.Meta().(*govmomi.Client)

	f := folder{path: folder_name, datacenter: datacenter}

	folder, err := object.NewSearchIndex(client.Client).FindByInventoryPath(
		context.TODO(), fmt.Sprintf("%v/vm/%v", datacenter, folder_name))
	if err != nil {
		return fmt.Errorf("error %s", err)
	}

	if folder == nil {
		createFolder(client, &f)
	} else {
		return fmt.Errorf("Folder %s already exists", folder_name)
	}

	return nil
}

func removeVSphereFolder(datacenter string, folder_name string, existing_path string) resource.TestCheckFunc {

	f := folder{path: folder_name, datacenter: datacenter, existingPath: existing_path}

	return func(s *terraform.State) error {

		client := testAccProvider.Meta().(*govmomi.Client)
		// finder := find.NewFinder(client.Client, true)

		folder, _ := object.NewSearchIndex(client.Client).FindByInventoryPath(
			context.TODO(), fmt.Sprintf("%v/vm/%v", datacenter, folder_name))
		if folder != nil {
			deleteFolder(client, &f)
		}

		return nil
	}
}

const testAccCheckVSphereFolderConfig = `
resource "vsphere_folder" "%s" {
	path = "%s"
	datacenter = "%s"
}
`
