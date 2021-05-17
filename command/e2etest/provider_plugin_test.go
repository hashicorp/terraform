package e2etest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// TestProviderProtocols verifies that Terraform can execute provider plugins
// with both supported protocol versions.
func TestProviderProtocols(t *testing.T) {
	t.Parallel()

	tf := e2e.NewBinary(terraformBin, "testdata/provider-plugin")
	defer tf.Close()

	// In order to do a decent end-to-end test for this case we will need a real
	// enough provider plugin to try to run and make sure we are able to
	// actually run it. Here will build the simple and simple6 (built with
	// protocol v6) providers.
	simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

	simpleProvider := filepath.Join(tf.WorkDir(), "terraform-provider-simple")
	simpleProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple/main", simpleProvider)

	// Move the provider binaries into a directory that we will point terraform
	// to using the -plugin-dir cli flag.
	platform := getproviders.CurrentPlatform.String()
	hashiDir := "cache/registry.terraform.io/hashicorp/"
	if err := os.MkdirAll(tf.Path(hashiDir, "simple6/0.0.1/", platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simple6ProviderExe, tf.Path(hashiDir, "simple6/0.0.1/", platform, "terraform-provider-simple6")); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(tf.Path(hashiDir, "simple/0.0.1/", platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simpleProviderExe, tf.Path(hashiDir, "simple/0.0.1/", platform, "terraform-provider-simple")); err != nil {
		t.Fatal(err)
	}

	//// INIT
	_, stderr, err := tf.Run("init", "-plugin-dir=cache")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	//// PLAN
	_, stderr, err = tf.Run("plan", "-out=tfplan")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
	}

	//// APPLY
	stdout, stderr, err := tf.Run("apply", "tfplan")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Apply complete! Resources: 2 added, 0 changed, 0 destroyed.") {
		t.Fatalf("wrong output:\n%s", stdout)
	}
}
