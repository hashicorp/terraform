package remote

import (
	"os"
	"testing"
)

func TestMantaClient_impl(t *testing.T) {
	var _ Client = new(MantaClient)
}

func TestMantaClient(t *testing.T) {
	// This test creates an object in Manta in the root directory of
	// the current MANTA_USER.
	//
	// It may incur costs, so it will only run if Manta credential environment
	// variables are present.

	mantaUser := os.Getenv("MANTA_USER")
	if mantaUser == "" {
		t.Skipf("skipping, MANTA_USER and friends must be set")
	}

	testPath := "terraform-remote-state-test"

	client, err := mantaFactory(map[string]string{
		"path":       testPath,
		"objectName": "terraform-test-state.tf",
	})

	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	mantaClient := client.(*MantaClient)

	err = mantaClient.Client.PutDirectory(mantaClient.Path)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	defer func() {
		err = mantaClient.Client.DeleteDirectory(mantaClient.Path)
		if err != nil {
			t.Fatalf("bad: %s", err)
		}
	}()

	testClient(t, client)
}
