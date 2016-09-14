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
	mantaKeyId := os.Getenv("MANTA_KEY_ID")
	mantaUrl := os.Getenv("MANTA_URL")

	if mantaUser == "" || mantaKeyId == "" || mantaUrl == "" {
		t.Skipf("skipping; MANTA_USER, MANTA_KEY_ID and MANTA_URL must all be set")
	}

	testPath := "terraform-remote-state-test"

	client, err := mantaFactory(map[string]string{
		"path":       testPath,
		"objectName": "terraform-test-state.tfstate",
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
