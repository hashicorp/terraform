package swift

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/hashicorp/terraform/backend"
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

	backend.TestBackendStates(t, b)

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
