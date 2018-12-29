package swift

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

// verify that we are doing ACC tests or the Swift tests specifically
func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_SWIFT_TEST") == ""
	if skip {
		t.Log("swift backend tests require setting TF_ACC or TF_SWIFT_TEST")
		t.Skip()
	}
	t.Log("swift backend acceptance tests enabled")
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func testAccPreCheck(t *testing.T) {
	v := os.Getenv("OS_AUTH_URL")
	if v == "" {
		t.Fatal("OS_AUTH_URL must be set for acceptance tests")
	}
}

func TestBackendConfig(t *testing.T) {
	testACC(t)

	// Build config
	config := map[string]interface{}{
		"archive_container": "test-tfstate-archive",
		"container":         "test-tfstate",
	}

	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	if b.container != "test-tfstate" {
		t.Fatal("Incorrect path was provided.")
	}
	if b.archiveContainer != "test-tfstate-archive" {
		t.Fatal("Incorrect archivepath was provided.")
	}
}

func TestBackend(t *testing.T) {
	testACC(t)

	container := fmt.Sprintf("terraform-state-swift-test-%x", time.Now().Unix())

	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"container": container,
	}).(*Backend)

	defer deleteSwiftContainer(t, b.client, container)

	backend.TestBackendStates(t, b)
}

func TestBackendPath(t *testing.T) {
	testACC(t)

	path := fmt.Sprintf("terraform-state-swift-test-%x", time.Now().Unix())
	t.Logf("[DEBUG] Generating backend config")
	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"path": path,
	}).(*Backend)
	t.Logf("[DEBUG] Backend configured")

	defer deleteSwiftContainer(t, b.client, path)

	t.Logf("[DEBUG] Testing Backend")

	// Generate some state
	state1 := terraform.NewState()
	// state1Lineage := state1.Lineage
	t.Logf("state1 lineage = %s, serial = %d", state1.Lineage, state1.Serial)

	// RemoteClient to test with
	client := &RemoteClient{
		client:           b.client,
		archive:          b.archive,
		archiveContainer: b.archiveContainer,
		container:        b.container,
	}

	stateMgr := &remote.State{Client: client}
	stateMgr.WriteState(state1)
	if err := stateMgr.PersistState(); err != nil {
		t.Fatal(err)
	}

	if err := stateMgr.RefreshState(); err != nil {
		t.Fatal(err)
	}

	// Add some state
	state1.AddModuleState(&terraform.ModuleState{
		Path: []string{"root"},
		Outputs: map[string]*terraform.OutputState{
			"bar": &terraform.OutputState{
				Type:      "string",
				Sensitive: false,
				Value:     "baz",
			},
		},
	})
	stateMgr.WriteState(state1)
	if err := stateMgr.PersistState(); err != nil {
		t.Fatal(err)
	}

}

func TestBackendArchive(t *testing.T) {
	testACC(t)

	container := fmt.Sprintf("terraform-state-swift-test-%x", time.Now().Unix())
	archiveContainer := fmt.Sprintf("%s_archive", container)

	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"archive_container": archiveContainer,
		"container":         container,
	}).(*Backend)

	defer deleteSwiftContainer(t, b.client, container)
	defer deleteSwiftContainer(t, b.client, archiveContainer)

	// Generate some state
	state1 := terraform.NewState()
	// state1Lineage := state1.Lineage
	t.Logf("state1 lineage = %s, serial = %d", state1.Lineage, state1.Serial)

	// RemoteClient to test with
	client := &RemoteClient{
		client:           b.client,
		archive:          b.archive,
		archiveContainer: b.archiveContainer,
		container:        b.container,
	}

	stateMgr := &remote.State{Client: client}
	stateMgr.WriteState(state1)
	if err := stateMgr.PersistState(); err != nil {
		t.Fatal(err)
	}

	if err := stateMgr.RefreshState(); err != nil {
		t.Fatal(err)
	}

	// Add some state
	state1.AddModuleState(&terraform.ModuleState{
		Path: []string{"root"},
		Outputs: map[string]*terraform.OutputState{
			"bar": &terraform.OutputState{
				Type:      "string",
				Sensitive: false,
				Value:     "baz",
			},
		},
	})
	stateMgr.WriteState(state1)
	if err := stateMgr.PersistState(); err != nil {
		t.Fatal(err)
	}

	archiveObjects := getSwiftObjectNames(t, b.client, archiveContainer)
	t.Logf("archiveObjects len = %d. Contents = %+v", len(archiveObjects), archiveObjects)
	if len(archiveObjects) != 1 {
		t.Fatalf("Invalid number of archive objects. Expected 1, got %d", len(archiveObjects))
	}

	// Download archive state to validate
	archiveData := downloadSwiftObject(t, b.client, archiveContainer, archiveObjects[0])
	t.Logf("Archive data downloaded... Looks like: %+v", archiveData)
	archiveState, err := terraform.ReadState(archiveData)
	if err != nil {
		t.Fatalf("Error Reading State: %s", err)
	}

	t.Logf("Archive state lineage = %s, serial = %d, lineage match = %t", archiveState.Lineage, archiveState.Serial, stateMgr.State().SameLineage(archiveState))
	if !stateMgr.State().SameLineage(archiveState) {
		t.Fatal("Got a different lineage")
	}

}

// Helper function to download an object in a Swift container
func downloadSwiftObject(t *testing.T, osClient *gophercloud.ServiceClient, container, object string) (data io.Reader) {
	t.Logf("Attempting to download object %s from container %s", object, container)
	res := objects.Download(osClient, container, object, nil)
	if res.Err != nil {
		t.Fatalf("Error downloading object: %s", res.Err)
	}
	data = res.Body
	return
}

// Helper function to get a list of objects in a Swift container
func getSwiftObjectNames(t *testing.T, osClient *gophercloud.ServiceClient, container string) (objectNames []string) {
	_ = objects.List(osClient, container, nil).EachPage(func(page pagination.Page) (bool, error) {
		// Get a slice of object names
		names, err := objects.ExtractNames(page)
		if err != nil {
			t.Fatalf("Error extracting object names from page: %s", err)
		}
		for _, object := range names {
			objectNames = append(objectNames, object)
		}

		return true, nil
	})
	return
}

// Helper function to delete Swift container
func deleteSwiftContainer(t *testing.T, osClient *gophercloud.ServiceClient, container string) {
	warning := "WARNING: Failed to delete the test Swift container. It may have been left in your Openstack account and may incur storage charges. (error was %s)"

	// Remove any objects
	deleteSwiftObjects(t, osClient, container)

	// Delete the container
	deleteResult := containers.Delete(osClient, container)
	if deleteResult.Err != nil {
		if _, ok := deleteResult.Err.(gophercloud.ErrDefault404); !ok {
			t.Fatalf(warning, deleteResult.Err)
		}
	}
}

// Helper function to delete Swift objects within a container
func deleteSwiftObjects(t *testing.T, osClient *gophercloud.ServiceClient, container string) {
	// Get a slice of object names
	objectNames := getSwiftObjectNames(t, osClient, container)

	for _, object := range objectNames {
		result := objects.Delete(osClient, container, object, nil)
		if result.Err != nil {
			t.Fatalf("Error deleting object %s from container %s: %s", object, container, result.Err)
		}
	}

}
