package google

import (
	"io/ioutil"
	"testing"
)

const testFakeCredentialsPath = "./test-fixtures/fake_account.json"

func TestConfigLoadAndValidate_accountFilePath(t *testing.T) {
	config := Config{
		Credentials: testFakeCredentialsPath,
		Project:     "my-gce-project",
		Region:      "us-central1",
	}

	err := config.loadAndValidate()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestConfigLoadAndValidate_accountFileJSON(t *testing.T) {
	contents, err := ioutil.ReadFile(testFakeCredentialsPath)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	config := Config{
		Credentials: string(contents),
		Project:     "my-gce-project",
		Region:      "us-central1",
	}

	err = config.loadAndValidate()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestConfigLoadAndValidate_accountFileJSONInvalid(t *testing.T) {
	config := Config{
		Credentials: "{this is not json}",
		Project:     "my-gce-project",
		Region:      "us-central1",
	}

	if config.loadAndValidate() == nil {
		t.Fatalf("expected error, but got nil")
	}
}
