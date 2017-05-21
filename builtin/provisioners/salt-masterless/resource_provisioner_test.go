package saltmasterless

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	return terraform.NewResourceConfig(r)
}

func TestResourceProvisioner_impl(t *testing.T) {
	var _ terraform.ResourceProvisioner = Provisioner()
}

func TestProvisioner(t *testing.T) {
	if err := Provisioner().(*schema.Provisioner).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestResourceProvisioner_Validate_good(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	defer os.RemoveAll(dir) // clean up

	c := testConfig(t, map[string]interface{}{
		"local_state_tree": dir,
	})
	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestResourceProvider_Validate_missing_required(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"remote_state_tree": "_default",
	})
	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

func TestResourceProvider_Validate_LocalStateTree_doesnt_exist(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"local_state_tree": "/i/dont/exist",
	})
	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

func TestResourceProvisioner_Validate_invalid(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	defer os.RemoveAll(dir) // clean up

	c := testConfig(t, map[string]interface{}{
		"local_state_tree": dir,
		"i_am_not_valid":   "_invalid",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

func TestProvisionerPrepare_CustomState(t *testing.T) {
	c := map[string]interface{}{
		"local_state_tree": "/tmp/local_state_tree",
	}

	p, err := decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if !strings.Contains(p.CmdArgs, "state.highstate") {
		t.Fatal("CmdArgs should contain state.highstate")
	}

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	c = map[string]interface{}{
		"local_state_tree": "/tmp/local_state_tree",
		"custom_state":     "custom",
	}

	p, err = decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if !strings.Contains(p.CmdArgs, "state.sls custom") {
		t.Fatal("CmdArgs should contain state.sls custom")
	}

	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvisionerPrepare_MinionConfig(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	defer os.RemoveAll(dir) // clean up

	c := testConfig(t, map[string]interface{}{
		"local_state_tree": dir,
		"minion_config":    "i/dont/exist",
	})

	warns, errs := Provisioner().Validate(c)

	if len(warns) > 0 {
		t.Fatalf("Warnings: %v", warns)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have error")
	}

	tf, err := ioutil.TempFile("", "minion")
	if err != nil {
		t.Fatalf("error tempfile: %s", err)
	}

	defer os.Remove(tf.Name())

	c = testConfig(t, map[string]interface{}{
		"local_state_tree": dir,
		"minion_config":    tf.Name(),
	})

	warns, errs = Provisioner().Validate(c)

	if len(warns) > 0 {
		t.Fatalf("Warnings: %v", warns)
	}
	if len(errs) > 0 {
		t.Fatalf("errs: %s", errs)
	}
}

func TestProvisionerPrepare_MinionConfig_RemoteStateTree(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	c := testConfig(t, map[string]interface{}{
		"local_state_tree":  dir,
		"minion_config":     "i/dont/exist",
		"remote_state_tree": "i/dont/exist/remote_state_tree",
	})

	warns, errs := Provisioner().Validate(c)
	if len(warns) > 0 {
		t.Fatalf("Warnings: %v", warns)
	}
	if len(errs) == 0 {
		t.Fatalf("Should be error")
	}
}

func TestProvisionerPrepare_MinionConfig_RemotePillarRoots(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	c := testConfig(t, map[string]interface{}{
		"local_state_tree":    dir,
		"minion_config":       "i/dont/exist",
		"remote_pillar_roots": "i/dont/exist/remote_pillar_roots",
	})

	warns, errs := Provisioner().Validate(c)
	if len(warns) > 0 {
		t.Fatalf("Warnings: %v", warns)
	}
	if len(errs) == 0 {
		t.Fatalf("Should be error")
	}
}

func TestProvisionerPrepare_LocalPillarRoots(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	c := testConfig(t, map[string]interface{}{
		"local_state_tree":   dir,
		"minion_config":      "i/dont/exist",
		"local_pillar_roots": "i/dont/exist/local_pillar_roots",
	})

	warns, errs := Provisioner().Validate(c)
	if len(warns) > 0 {
		t.Fatalf("Warnings: %v", warns)
	}
	if len(errs) == 0 {
		t.Fatalf("Should be error")
	}
}

func TestProvisionerSudo(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	c := map[string]interface{}{
		"local_state_tree": dir,
	}

	p, err := decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	withSudo := p.sudo("echo hello")
	if withSudo != "sudo echo hello" {
		t.Fatalf("sudo command not generated correctly")
	}

	c = map[string]interface{}{
		"local_state_tree": dir,
		"disable_sudo":     "true",
	}

	p, err = decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	if err != nil {
		t.Fatalf("err: %s", err)
	}
	withoutSudo := p.sudo("echo hello")
	if withoutSudo != "echo hello" {
		t.Fatalf("sudo-less command not generated correctly")
	}
}

func TestProvisionerPrepare_RemoteStateTree(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	c := map[string]interface{}{
		"local_state_tree":  dir,
		"remote_state_tree": "/remote_state_tree",
	}

	p, err := decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !strings.Contains(p.CmdArgs, "--file-root=/remote_state_tree") {
		t.Fatal("--file-root should be set in CmdArgs")
	}
}

func TestProvisionerPrepare_RemotePillarRoots(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	c := map[string]interface{}{
		"local_state_tree":    dir,
		"remote_pillar_roots": "/remote_pillar_roots",
	}

	p, err := decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !strings.Contains(p.CmdArgs, "--pillar-root=/remote_pillar_roots") {
		t.Fatal("--pillar-root should be set in CmdArgs")
	}
}

func TestProvisionerPrepare_RemoteStateTree_Default(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	c := map[string]interface{}{
		"local_state_tree": dir,
	}

	p, err := decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !strings.Contains(p.CmdArgs, "--file-root=/srv/salt") {
		t.Fatal("--file-root should be set in CmdArgs")
	}
}

func TestProvisionerPrepare_RemotePillarRoots_Default(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	c := map[string]interface{}{
		"local_state_tree": dir,
	}

	p, err := decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !strings.Contains(p.CmdArgs, "--pillar-root=/srv/pillar") {
		t.Fatal("--pillar-root should be set in CmdArgs")
	}
}

func TestProvisionerPrepare_NoExitOnFailure(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	c := map[string]interface{}{
		"local_state_tree": dir,
	}

	p, err := decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !strings.Contains(p.CmdArgs, "--retcode-passthrough") {
		t.Fatal("--retcode-passthrough should be set in CmdArgs")
	}

	c = map[string]interface{}{
		"no_exit_on_failure": true,
	}

	p, err = decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if strings.Contains(p.CmdArgs, "--retcode-passthrough") {
		t.Fatal("--retcode-passthrough should not be set in CmdArgs")
	}
}

func TestProvisionerPrepare_LogLevel(t *testing.T) {
	dir, err := ioutil.TempDir("", "_terraform_saltmasterless_test")
	if err != nil {
		t.Fatalf("Error when creating temp dir: %v", err)
	}

	c := map[string]interface{}{
		"local_state_tree": dir,
	}

	p, err := decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !strings.Contains(p.CmdArgs, "-l info") {
		t.Fatal("-l info should be set in CmdArgs")
	}

	c = map[string]interface{}{
		"log_level": "debug",
	}

	p, err = decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !strings.Contains(p.CmdArgs, "-l debug") {
		t.Fatal("-l debug should be set in CmdArgs")
	}
}
