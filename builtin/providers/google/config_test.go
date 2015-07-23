package google

import (
	"io/ioutil"
	"testing"
)

const testFakeAccountFilePath = "./test-fixtures/fake_account.json"

func TestConfigLoadAndValidate_accountFile(t *testing.T) {
	config := Config{
		AccountFile: testFakeAccountFilePath,
		Project:     "my-gce-project",
		Region:      "us-central1",
	}

	err := config.loadAndValidate()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestConfigLoadAndValidate_accountFileContents(t *testing.T) {
	contents, err := ioutil.ReadFile(testFakeAccountFilePath)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	config := Config{
		AccountFileContents: string(contents),
		Project:             "my-gce-project",
		Region:              "us-central1",
	}

	err = config.loadAndValidate()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestConfigLoadAndValidate_none(t *testing.T) {
	config := Config{
		Project: "my-gce-project",
		Region:  "us-central1",
	}

	err := config.loadAndValidate()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestConfigLoadAndValidate_both(t *testing.T) {
	config := Config{
		AccountFile:         testFakeAccountFilePath,
		AccountFileContents: "{}",
		Project:             "my-gce-project",
		Region:              "us-central1",
	}

	if config.loadAndValidate() == nil {
		t.Fatalf("expected error, but got nil")
	}
}
