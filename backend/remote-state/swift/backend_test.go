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
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
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
	container := fmt.Sprintf("terraform-state-swift-testconfig-%x", time.Now().Unix())
	archiveContainer := fmt.Sprintf("%s_archive", container)
	// Build config
	config := map[string]interface{}{
		"archive_container": archiveContainer,
		"container":         container,
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)

	if b.container != container {
		t.Fatal("Incorrect container was provided.")
	}
	if b.archiveContainer != archiveContainer {
		t.Fatal("Incorrect archive_container was provided.")
	}
}

func TestBackend(t *testing.T) {
	testACC(t)

	container := fmt.Sprintf("terraform-state-swift-testbackend-%x", time.Now().Unix())

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"container": container,
	})).(*Backend)

	defer deleteSwiftContainer(t, b.client, container)

	backend.TestBackendStates(t, b)
}

func TestBackendPath(t *testing.T) {
	testACC(t)

	path := fmt.Sprintf("terraform-state-swift-test-%x", time.Now().Unix())
	t.Logf("[DEBUG] Generating backend config")
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"path": path,
	})).(*Backend)
	t.Logf("[DEBUG] Backend configured")

	defer deleteSwiftContainer(t, b.client, path)

	t.Logf("[DEBUG] Testing Backend")

	// Generate some state
	state1 := states.NewState()

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
	mod := state1.EnsureModule(addrs.RootModuleInstance)
	mod.SetOutputValue("bar", cty.StringVal("baz"), false)
	stateMgr.WriteState(state1)
	if err := stateMgr.PersistState(); err != nil {
		t.Fatal(err)
	}

}

func TestBackendArchive(t *testing.T) {
	testACC(t)

	container := fmt.Sprintf("terraform-state-swift-testarchive-%x", time.Now().Unix())
	archiveContainer := fmt.Sprintf("%s_archive", container)

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"archive_container": archiveContainer,
		"container":         container,
	})).(*Backend)

	defer func() {
		deleteSwiftContainer(t, b.client, container)
		deleteSwiftContainer(t, b.client, archiveContainer)
	}()

	// RemoteClient to test with
	client := &RemoteClient{
		client:           b.client,
		archive:          b.archive,
		archiveContainer: b.archiveContainer,
		container:        b.container,
	}

	stateMgr := &remote.State{Client: client}

	workspaces, err := b.Workspaces()
	if err != nil {
		t.Fatalf("Error Reading States: %s", err)
	}

	// Generate some state
	state1 := states.NewState()

	// there should always be at least one default state
	s2Mgr, err := b.StateMgr(workspaces[0])
	if err != nil {
		t.Fatal(err)
	}

	s2Mgr.WriteState(state1)
	if err := s2Mgr.PersistState(); err != nil {
		t.Fatal(err)
	}

	if err := s2Mgr.RefreshState(); err != nil {
		t.Fatal(err)
	}

	// Add some state
	mod := state1.EnsureModule(addrs.RootModuleInstance)
	mod.SetOutputValue("bar", cty.StringVal("baz"), false)
	s2Mgr.WriteState(state1)
	if err := s2Mgr.PersistState(); err != nil {
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
	archiveStateFile, err := statefile.Read(archiveData)
	if err != nil {
		t.Fatalf("Error Reading State: %s", err)
	}

	t.Logf("Archive state lineage = %s, serial = %d", archiveStateFile.Lineage, archiveStateFile.Serial)
	if stateMgr.StateSnapshotMeta().Lineage != archiveStateFile.Lineage {
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

var (
	// The amount of time we will retry to delete a container waiting for the objects
	// to be deleted.
	deleteContainerRetryTimeout = 60 * time.Second

	// delay when polling the objects
	deleteContainerRetryPollInterval = 5 * time.Second
)

// Helper function to delete Swift container
func deleteSwiftContainer(t *testing.T, osClient *gophercloud.ServiceClient, container string) {
	warning := "WARNING: Failed to delete the test Swift container. It may have been left in your Openstack account and may incur storage charges. (error was %s)"
	deadline := time.Now().Add(deleteContainerRetryTimeout)

	// Swift is eventually consistent; we have to retry until
	// all objects are effectively deleted to delete the container
	// If we still have objects in the container, or raise
	// an error if deadline is reached
	for {
		if time.Now().Before(deadline) {
			// Remove any objects
			deleteSwiftObjects(t, osClient, container)

			// Delete the container
			t.Logf("Deleting container %s", container)
			deleteResult := containers.Delete(osClient, container)
			if deleteResult.Err != nil {
				// container is not found, thus has been deleted
				if _, ok := deleteResult.Err.(gophercloud.ErrDefault404); ok {
					return
				}

				// 409 http error is raised when deleting a container with
				// remaining objects
				if respErr, ok := deleteResult.Err.(gophercloud.ErrUnexpectedResponseCode); ok && respErr.Actual == 409 {
					time.Sleep(deleteContainerRetryPollInterval)
					t.Logf("Remaining objects, failed to delete container, retrying...")
					continue
				}

				t.Fatalf(warning, deleteResult.Err)
			}
			return
		}

		t.Fatalf(warning, "timeout reached")
	}

}

// Helper function to delete Swift objects within a container
func deleteSwiftObjects(t *testing.T, osClient *gophercloud.ServiceClient, container string) {
	// Get a slice of object names
	objectNames := getSwiftObjectNames(t, osClient, container)

	for _, object := range objectNames {
		t.Logf("Deleting object %s from container %s", object, container)
		result := objects.Delete(osClient, container, object, nil)
		if result.Err == nil {
			continue
		}

		// if object is not found, it has already been deleted
		if _, ok := result.Err.(gophercloud.ErrDefault404); !ok {
			t.Fatalf("Error deleting object %s from container %s: %v", object, container, result.Err)
		}
	}

}
