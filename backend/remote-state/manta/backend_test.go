package manta

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/joyent/triton-go/storage"
)

func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_MANTA_TEST") == ""
	if skip {
		t.Log("Manta backend tests require setting TF_ACC or TF_MANTA_TEST")
		t.Skip()
	}
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackend(t *testing.T) {
	testACC(t)

	directory := fmt.Sprintf("terraform-remote-manta-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"path":        directory,
		"object_name": keyName,
	})).(*Backend)

	createMantaFolder(t, b.storageClient, directory)
	defer deleteMantaFolder(t, b.storageClient, directory)

	backend.TestBackendStates(t, b)
}

func TestBackendLocked(t *testing.T) {
	testACC(t)

	directory := fmt.Sprintf("terraform-remote-manta-test-%x", time.Now().Unix())
	keyName := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"path":        directory,
		"object_name": keyName,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"path":        directory,
		"object_name": keyName,
	})).(*Backend)

	createMantaFolder(t, b1.storageClient, directory)
	defer deleteMantaFolder(t, b1.storageClient, directory)

	backend.TestBackendStateLocks(t, b1, b2)
	backend.TestBackendStateForceUnlock(t, b1, b2)
}

func createMantaFolder(t *testing.T, mantaClient *storage.StorageClient, directoryName string) {
	// Be clear about what we're doing in case the user needs to clean
	// this up later.
	//t.Logf("creating Manta directory %s", directoryName)
	err := mantaClient.Dir().Put(context.Background(), &storage.PutDirectoryInput{
		DirectoryName: path.Join(mantaDefaultRootStore, directoryName),
	})
	if err != nil {
		t.Fatal("failed to create test Manta directory:", err)
	}
}

func deleteMantaFolder(t *testing.T, mantaClient *storage.StorageClient, directoryName string) {
	//warning := "WARNING: Failed to delete the test Manta directory. It may have been left in your Manta account and may incur storage charges. (error was %s)"

	// first we have to get rid of the env objects, or we can't delete the directory
	objs, err := mantaClient.Dir().List(context.Background(), &storage.ListDirectoryInput{
		DirectoryName: path.Join(mantaDefaultRootStore, directoryName),
	})
	if err != nil {
		t.Fatal("failed to retrieve directory listing")
	}

	for _, obj := range objs.Entries {
		if obj.Type == "directory" {
			ojs, err := mantaClient.Dir().List(context.Background(), &storage.ListDirectoryInput{
				DirectoryName: path.Join(mantaDefaultRootStore, directoryName, obj.Name),
			})
			if err != nil {
				t.Fatal("failed to retrieve directory listing")
			}
			for _, oj := range ojs.Entries {
				err := mantaClient.Objects().Delete(context.Background(), &storage.DeleteObjectInput{
					ObjectPath: path.Join(mantaDefaultRootStore, directoryName, obj.Name, oj.Name),
				})
				if err != nil {
					t.Fatal(err)
				}
			}
		}

		err := mantaClient.Objects().Delete(context.Background(), &storage.DeleteObjectInput{
			ObjectPath: path.Join(mantaDefaultRootStore, directoryName, obj.Name),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	err = mantaClient.Dir().Delete(context.Background(), &storage.DeleteDirectoryInput{
		DirectoryName: path.Join(mantaDefaultRootStore, directoryName),
	})
	if err != nil {
		t.Fatal("failed to delete manta directory")
	}
}
