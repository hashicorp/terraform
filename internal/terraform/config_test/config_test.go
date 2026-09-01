package config_test

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/terraform"
)

func TestModuleOverrideResourceFQNs(t *testing.T) {
	mod, diags := testModuleFromDirWithInitGraph("testdata/valid-modules/override-resource-provider")
	assertNoDiagnostics(t, diags)

	got := mod.ManagedResources["test_instance.explicit"]
	wantProvider := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "bar", "test")
	wantProviderCfg := &configs.ProviderConfigRef{
		Name: "bar-test",
		NameRange: hcl.Range{
			Filename: "testdata/valid-modules/override-resource-provider/a_override.tf",
			Start:    hcl.Pos{Line: 2, Column: 14, Byte: 51},
			End:      hcl.Pos{Line: 2, Column: 22, Byte: 59},
		},
	}

	if !got.Provider.Equals(wantProvider) {
		t.Fatalf("wrong provider %s, want %s", got.Provider, wantProvider)
	}
	assertResultDeepEqual(t, got.ProviderConfigRef, wantProviderCfg)

	// now verify that a resource with no provider config falls back to default
	got = mod.ManagedResources["test_instance.default"]
	wantProvider = addrs.NewDefaultProvider("test")
	if !got.Provider.Equals(wantProvider) {
		t.Fatalf("wrong provider %s, want %s", got.Provider, wantProvider)
	}
	if got.ProviderConfigRef != nil {
		t.Fatalf("wrong result: found provider config ref %s, expected nil", got.ProviderConfigRef)
	}
}

// This tests the override behavior of action blocks and action_triggers inside resources.
func TestModuleOverride_action_and_trigger(t *testing.T) {
	mod, diags := testModuleFromDirWithExperiments("testdata/valid-modules/override-action-and-trigger")
	assertNoDiagnostics(t, diags)

	if len(mod.Actions) != 2 {
		t.Fatalf("wrong number of actions: %d\n", len(mod.Actions))
	}

	// verify that the action has attr foo = baz (override)
	got := mod.Actions["action.test_action.test"]
	want := &configs.Action{
		Name:              "test",
		Type:              "test_action",
		Config:            nil,
		Count:             nil,
		ForEach:           nil,
		ProviderConfigRef: nil,
		Provider:          addrs.NewProvider(addrs.DefaultProviderRegistryHost, "hashicorp", "test"),
		DeclRange: hcl.Range{
			Filename: "testdata/valid-modules/override-action-and-trigger/main.tf",
			Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
			End:      hcl.Pos{Line: 1, Column: 28, Byte: 27},
		},
		TypeRange: hcl.Range{
			Filename: "testdata/valid-modules/override-action-and-trigger/main.tf",
			Start:    hcl.Pos{Line: 1, Column: 8, Byte: 7},
			End:      hcl.Pos{Line: 1, Column: 21, Byte: 20},
		},
		Body: mod.Actions["action.test_action.test"].Body,
	}

	// We're going to extract and nil out our hcl.Body here because DeepEqual
	// is not a useful way to assert on that.
	gotConfig := got.Config
	got.Config = nil

	assertResultDeepEqual(t, got, want)

	// now to check that config
	type content struct {
		Foo *string `hcl:"foo"`
	}
	var gotArgs content
	diags = gohcl.DecodeBody(gotConfig, nil, &gotArgs)
	assertNoDiagnostics(t, diags)

	wantArgs := content{
		Foo: stringPtr("baz"),
	}
	assertResultDeepEqual(t, gotArgs, wantArgs)

	if _, exists := mod.ManagedResources["test_instance.test"]; !exists {
		t.Fatalf("no resource 'test_instance.test'")
	}
	if len(mod.ManagedResources) != 1 {
		t.Fatalf("wrong number of managed resources in result %d; want 1", len(mod.ManagedResources))
	}

	r := mod.ManagedResources["test_instance.test"].Managed
	assertResultDeepEqual(t, len(r.ActionTriggers), 1)

	// verify the resource action trigger event changed
	at := mod.ManagedResources["test_instance.test"].Managed.ActionTriggers[0]
	assertResultDeepEqual(t, at.Events, []configs.ActionTriggerEvent{configs.BeforeCreate})
}

// testModuleFromDirWithExperiments reads configuration from the given directory
// path as a module and returns it. The parser is configured to allow language
// experiments. This is a helper for use in unit tests.
func testModuleFromDirWithExperiments(path string) (*configs.Module, hcl.Diagnostics) {
	parser := configs.NewParser(nil)
	parser.AllowLanguageExperiments(true)
	mod, diags := parser.LoadConfigDir(path)
	if diags != nil {
		return nil, diags
	}
	mod, diagnostics := terraform.BuildModuleWithGraph(mod, nil)
	if diagnostics != nil {
		return nil, diagnostics.ToHCL()
	}
	return mod, nil
}

func testModuleFromDirWithInitGraph(path string) (*configs.Module, hcl.Diagnostics) {
	parser := configs.NewParser(nil)
	mod, diags := parser.LoadConfigDir(path)
	if diags != nil {
		return nil, diags
	}
	mod, diagnostics := terraform.BuildModuleWithGraph(mod, nil)
	if diagnostics != nil {
		return nil, diagnostics.ToHCL()
	}
	return mod, nil
}

func assertResultDeepEqual(t *testing.T, got, want interface{}) bool {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(want))
		return true
	}
	return false
}
func assertNoDiagnostics(t *testing.T, diags hcl.Diagnostics) bool {
	t.Helper()
	return assertDiagnosticCount(t, diags, 0)
}

func assertDiagnosticCount(t *testing.T, diags hcl.Diagnostics, want int) bool {
	t.Helper()
	if len(diags) != want {
		t.Errorf("wrong number of diagnostics %d; want %d", len(diags), want)
		for _, diag := range diags {
			t.Logf("- %s", diag)
		}
		return true
	}
	return false
}

func stringPtr(s string) *string {
	return &s
}
