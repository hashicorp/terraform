package earlyconfig

import (
	"log"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestConfigProviderRequirements(t *testing.T) {
	cfg := testConfig(t, "testdata/provider-reqs")

	impliedProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost,
		"hashicorp", "implied",
	)
	nullProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost,
		"hashicorp", "null",
	)
	randomProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost,
		"hashicorp", "random",
	)
	tlsProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost,
		"hashicorp", "tls",
	)
	happycloudProvider := addrs.NewProvider(
		svchost.Hostname("tf.example.com"),
		"awesomecorp", "happycloud",
	)

	got, diags := cfg.ProviderRequirements()
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %s", diags.Err().Error())
	}
	want := getproviders.Requirements{
		// the nullProvider constraints from the two modules are merged
		nullProvider:       getproviders.MustParseVersionConstraints("~> 2.0.0, 2.0.1"),
		randomProvider:     getproviders.MustParseVersionConstraints("~> 1.2.0"),
		tlsProvider:        getproviders.MustParseVersionConstraints("~> 3.0"),
		impliedProvider:    nil,
		happycloudProvider: nil,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func testConfig(t *testing.T, baseDir string) *Config {
	rootMod, diags := LoadModule(baseDir)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %s", diags.Err().Error())
	}

	cfg, diags := BuildConfig(rootMod, ModuleWalkerFunc(testModuleWalkerFunc))
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %s", diags.Err().Error())
	}

	return cfg
}

// testModuleWalkerFunc is a simple implementation of ModuleWalkerFunc that
// only understands how to resolve relative filesystem paths, using source
// location information from the call.
func testModuleWalkerFunc(req *ModuleRequest) (*tfconfig.Module, *version.Version, tfdiags.Diagnostics) {
	callFilename := req.CallPos.Filename
	sourcePath := req.SourceAddr.String()
	finalPath := filepath.Join(filepath.Dir(callFilename), sourcePath)
	log.Printf("[TRACE] %s in %s -> %s", sourcePath, callFilename, finalPath)

	newMod, diags := LoadModule(finalPath)
	return newMod, version.Must(version.NewVersion("0.0.0")), diags
}
