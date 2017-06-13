package command

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/plugin/discovery"
)

// mockGetProvider providers a GetProvider method for testing automatic
// provider downloads
type mockGetProvider struct {
	// A map of provider names to available versions.
	// The tests expect the versions to be in order from newest to oldest.
	Providers map[string][]string

	// optionally supply a path to a binary to source for the providers, or an
	// empty placeholder file will be created.
	Bins map[string]string
}

func (m mockGetProvider) FileName(provider, version string) string {
	return fmt.Sprintf("terraform-provider-%s_v%s_x4", provider, version)
}

// GetProvider will check the Providers map to see if it can find a suitable
// version, and put an empty file in the dst directory.
func (m mockGetProvider) GetProvider(dst, provider string, req discovery.Constraints, protoVersion uint) error {
	versions := m.Providers[provider]
	if len(versions) == 0 {
		return fmt.Errorf("provider %q not found", provider)
	}

	err := os.MkdirAll(dst, 0755)
	if err != nil {
		return fmt.Errorf("error creating plugins directory: %s", err)
	}

	for _, v := range versions {
		version, err := discovery.VersionStr(v).Parse()
		if err != nil {
			panic(err)
		}

		if req.Allows(version) {
			// provider filename
			name := m.FileName(provider, v)
			path := filepath.Join(dst, name)

			// create an empty file by default
			var source io.Reader = bytes.NewBuffer(nil)

			if bin := m.Bins[provider]; bin != "" {
				// copy the binary to the destination
				b, err := os.Open(bin)
				if err != nil {
					return err
				}
				defer b.Close()
				source = b
			}

			f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0755)
			if err != nil {
				return fmt.Errorf("error fetching provider: %s", err)
			}
			defer f.Close()

			if _, err := io.Copy(f, source); err != nil {
				return err
			}
			return nil
		}
	}

	return fmt.Errorf("no suitable version for provider %q found with constraints %s", provider, req)
}

func TestExecPlugins(t *testing.T) {
	td := tempDir(t)
	os.MkdirAll(td, 0755)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// make some invalid plugins
	if err := ioutil.WriteFile("terraform-provider-nonexecutable_v1.2.3", []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile("terraform-provider-notplugin_v1.2.3", []byte("#!/bin/bash\n"), 0755); err != nil {
		t.Fatal(err)
	}

	// create a valid plugin
	if err := createTestProviderBin("./terraform-provider-test_v2.3.4_x4"); err != nil {
		t.Fatal(err)
	}

	plugins := make(discovery.PluginMetaSet)
	nonExecMeta := discovery.PluginMeta{
		Name:    "notexecutable",
		Version: "1.2.3",
		Path:    "./terraform-provider-nonexecutable_v1.2.3",
	}
	notPluginMeta := discovery.PluginMeta{
		Name:    "notplugin",
		Version: "1.2.3",
		Path:    "./terraform-provider-notplugin_v1.2.3",
	}
	testPluginMeta := discovery.PluginMeta{
		Name:    "test",
		Version: "2.3.4",
		Path:    "./terraform-provider-test_v2.3.4_x4",
	}
	plugins.Add(nonExecMeta)
	plugins.Add(notPluginMeta)
	plugins.Add(testPluginMeta)

	filtered, err := execPlugins(plugins)
	if filtered.Count() != 1 {
		t.Fatalf("expected 1 plugins, got %d: %#v\n", filtered.Count(), filtered)
	}

	if err == nil {
		t.Fatal("expected errors")
	}

	if !filtered.Has(testPluginMeta) {
		t.Fatalf("the test plugin should be the only valid plugin. got: %#v", filtered)
	}
}

// compile a test provider
func createTestProviderBin(path string) error {
	args := []string{"build", "-o", path, "github.com/hashicorp/terraform/builtin/bins/provider-test"}
	cmd := exec.Command("go", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, output)
	}

	if len(output) > 0 {
		log.Printf("[INFO] provider compilation output: %s", string(output))
	}

	return nil
}
