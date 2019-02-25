package swift

import (
	"fmt"
	"os"
	"testing"
	"time"

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

	be0 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"container": container,
	})).(*Backend)

	be1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"container": container,
	})).(*Backend)

	client := &RemoteClient{
		client:    be0.client,
		container: be0.container,
	}

	defer client.deleteContainer()

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
	backend.TestBackendStateForceUnlock(t, be0, be1)
}

func TestBackendArchive(t *testing.T) {
	testACC(t)

	container := fmt.Sprintf("terraform-state-swift-testarchive-%x", time.Now().Unix())
	archiveContainer := fmt.Sprintf("%s_archive", container)

	be0 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"archive_container": archiveContainer,
		"container":         container,
	})).(*Backend)

	be1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"archive_container": archiveContainer,
		"container":         container,
	})).(*Backend)

	defer func() {
		client := &RemoteClient{
			client:    be0.client,
			container: be0.container,
		}

		aclient := &RemoteClient{
			client:    be0.client,
			container: be0.archiveContainer,
		}

		defer client.deleteContainer()
		client.deleteContainer()
		aclient.deleteContainer()
	}()

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
	backend.TestBackendStateForceUnlock(t, be0, be1)
}
