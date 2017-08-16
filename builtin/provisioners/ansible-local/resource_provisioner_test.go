package ansible_local

import (
	"fmt"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestResourceProvisioner_impl(t *testing.T) {
	var _ terraform.ResourceProvisioner = Provisioner()
}

func TestProvisioner(t *testing.T) {
	if err := Provisioner().(*schema.Provisioner).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestResourceProvider_Validate_good(t *testing.T) {

	validDir, err := ioutil.TempDir("", "ansible-local")
	if err != nil {
		t.Fatalf("Unable to create temporary directory: %s", err)
	}
	defer os.RemoveAll(validDir)

	validFile, err := ioutil.TempFile("", "ansible-local")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %s", err)
	}
	defer os.RemoveAll(validFile.Name())

	c := testConfig(t, map[string]interface{}{
		"group_vars":         validDir,
		"host_vars":          validDir,
		"playbook_directory": validDir,
		"playbook_file":      validFile.Name(),
		"playbook_paths":     []interface{}{validDir},
		"role_paths":         []interface{}{validDir},
		"inventory_file":     validFile.Name(),
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestResourceProvider_Validate_bad(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"group_vars":         "/this/path/should/not/exist",
		"host_vars":          "/this/path/should/not/exist",
		"playbook_directory": "/this/path/should/not/exist",
		"playbook_file":      "/this/path/should/not/exist",
		"playbook_paths":     []interface{}{"/this/path/should/not/exist"},
		"role_paths":         []interface{}{"/this/path/should/not/exist"},
		"inventory_file":     "/this/path/should/not/exist",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

func TestResourceProvider_Provision(t *testing.T) {
	playbookFile, err := ioutil.TempFile("", "ansible-local")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %s", err)
	}
	defer os.RemoveAll(playbookFile.Name())

	inventoryFile, err := ioutil.TempFile("", "ansible-local")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %s", err)
	}
	defer os.RemoveAll(inventoryFile.Name())

	stagingDir, err := ioutil.TempDir("", "ansible-local")
	if err != nil {
		t.Fatalf("Unable to create temporary directory: %s", err)
	}
	defer os.RemoveAll(stagingDir)

	roleDir, err := ioutil.TempDir("", "ansible-local")
	if err != nil {
		t.Fatalf("Unable to create temporary directory: %s", err)
	}
	defer os.RemoveAll(roleDir)

	conf := map[string]interface{}{
		"playbook_file":     playbookFile.Name(),
		"role_paths":        []interface{}{roleDir},
		"staging_directory": stagingDir,
		"inventory_groups":  []interface{}{"main"},
		"extra_arguments":   []interface{}{"--verbose"},
	}

	uploadedPlaybookFile := filepath.ToSlash(filepath.Join(stagingDir, filepath.Base(playbookFile.Name())))
	uploadedInventoryFile := filepath.ToSlash(filepath.Join(stagingDir, filepath.Base(inventoryFile.Name())))
	uploadedRolePath := filepath.ToSlash(filepath.Join(stagingDir, "roles", filepath.Base(roleDir)))

	mockComm := new(communicator.MockCommunicator)
	mockComm.Commands = map[string]bool{
		fmt.Sprintf("mkdir -p '%s'", stagingDir):       true,
		fmt.Sprintf("mkdir -p '%s'", uploadedRolePath): true,
		fmt.Sprintf("cd %s && ANSIBLE_FORCE_COLOR=1 PYTHONUNBUFFERED=1 ansible-playbook %s --verbose -c local -i %s",
			stagingDir, uploadedPlaybookFile, uploadedInventoryFile): true,
	}

	mockComm.Uploads = map[string]string{
		uploadedPlaybookFile:  "expected content",
		uploadedInventoryFile: "[main]\n127.0.0.1\n",
	}

	mockComm.UploadDirs = map[string]string{
		roleDir + string(os.PathSeparator): uploadedRolePath,
	}

	if err := ioutil.WriteFile(playbookFile.Name(), []byte("expected content"), 0644); err != nil {
		t.Fatalf("Failed to write to temporary file %s: %s", playbookFile.Name(), err)
	}

	if err := ioutil.WriteFile(inventoryFile.Name(), []byte("expected content"), 0644); err != nil {
		t.Fatalf("Failed to write to temporary file %s: %s", playbookFile.Name(), err)
	}

	if err := provisionWithAnsible(mockComm, schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, conf),
		new(terraform.MockUIOutput), func() (*os.File, error) {
			return inventoryFile, nil
		}); err != nil {
		t.Fatalf("provisioning with Ansible failed: %s", err)
	}
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	return terraform.NewResourceConfig(r)
}
