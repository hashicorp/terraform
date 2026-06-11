// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
)

// smokeOutput is a lenient view of the providers-schema JSON document used by
// the command-level wiring/smoke tests. The exhaustive pruning matrix lives in
// the pure filter-function tests; these only verify end-to-end wiring.
type smokeOutput struct {
	FormatVersion string `json:"format_version"`
	Filters       *struct {
		Provider string `json:"provider"`
		Kind     string `json:"kind"`
		Type     string `json:"type"`
	} `json:"filters"`
	Schemas map[string]map[string]json.RawMessage `json:"provider_schemas"`
}

// runProvidersSchemaFixture inits the given providers-schema fixture and runs
// `terraform providers schema` with the provided args, returning the exit code
// and the command's stdout/stderr.
func runProvidersSchemaFixture(t *testing.T, fixture string, args []string) (int, string, string) {
	t.Helper()

	td := t.TempDir()
	inputDir := filepath.Join("testdata/providers-schema", fixture)
	testCopyDir(t, inputDir, td)
	t.Chdir(td)

	providerSource := newMockProviderSource(t, map[string][]string{
		"test": {"1.2.3"},
	})

	p := providersSchemaFixtureProvider()
	ui := new(cli.MockUi)
	view, done := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(p),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	ic := &InitCommand{Meta: m}
	if code := ic.Run([]string{}); code != 0 {
		t.Fatalf("init failed\n%s", done(t).Stderr())
	}

	pc := &ProvidersSchemaCommand{Meta: m}
	code := pc.Run(args)
	return code, ui.OutputWriter.String(), ui.ErrorWriter.String()
}

// parseSmokeOutput parses command stdout into a smokeOutput.
func parseSmokeOutput(t *testing.T, out string) smokeOutput {
	t.Helper()
	var got smokeOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("failed to parse command output as JSON: %s\noutput: %s", err, out)
	}
	return got
}

const testProviderFQN = "registry.terraform.io/hashicorp/test"

func TestProvidersSchemaCommand_provider(t *testing.T) {
	t.Run("shorthand source normalizes to the FQN", func(t *testing.T) {
		code, out, stderr := runProvidersSchemaFixture(t, "basic", []string{"-json", "-provider=hashicorp/test"})
		if code != 0 {
			t.Fatalf("wrong exit %d; stderr: %s", code, stderr)
		}
		got := parseSmokeOutput(t, out)
		if got.Filters == nil || got.Filters.Provider != testProviderFQN {
			t.Fatalf("expected filters.provider=%q, got %+v", testProviderFQN, got.Filters)
		}
		if _, ok := got.Schemas[testProviderFQN]; !ok {
			t.Fatalf("expected the test provider in output, got %v", got.Schemas)
		}
		// A provider-only selection keeps all categories the provider exposes.
		if _, ok := got.Schemas[testProviderFQN]["resource_schemas"]; !ok {
			t.Errorf("expected resource_schemas to be retained")
		}
		if _, ok := got.Schemas[testProviderFQN]["provider"]; !ok {
			t.Errorf("expected provider config to be retained for a provider-only selection")
		}
	})

	t.Run("full source form", func(t *testing.T) {
		code, out, stderr := runProvidersSchemaFixture(t, "basic", []string{"-json", "-provider=" + testProviderFQN})
		if code != 0 {
			t.Fatalf("wrong exit %d; stderr: %s", code, stderr)
		}
		got := parseSmokeOutput(t, out)
		if got.Filters == nil || got.Filters.Provider != testProviderFQN {
			t.Fatalf("expected filters.provider=%q, got %+v", testProviderFQN, got.Filters)
		}
	})

	t.Run("no-match exits 1 and lists the loaded providers", func(t *testing.T) {
		code, _, stderr := runProvidersSchemaFixture(t, "basic", []string{"-json", "-provider=hashicorp/google"})
		if code != 1 {
			t.Fatalf("expected exit 1, got %d", code)
		}
		if !strings.Contains(stderr, "was not found in the loaded provider schemas") {
			t.Errorf("expected a no-match diagnostic, got: %s", stderr)
		}
		if !strings.Contains(stderr, testProviderFQN) {
			t.Errorf("expected the loaded providers to be listed, got: %s", stderr)
		}
	})
}

func TestProvidersSchemaCommand_kind(t *testing.T) {
	t.Run("kind=resource keeps only resource_schemas and drops the provider config", func(t *testing.T) {
		code, out, stderr := runProvidersSchemaFixture(t, "basic", []string{"-json", "-kind=resource"})
		if code != 0 {
			t.Fatalf("wrong exit %d; stderr: %s", code, stderr)
		}
		got := parseSmokeOutput(t, out)
		if got.Filters == nil || got.Filters.Kind != "resource" {
			t.Fatalf("expected filters.kind=resource, got %+v", got.Filters)
		}
		cats := got.Schemas[testProviderFQN]
		if _, ok := cats["resource_schemas"]; !ok {
			t.Errorf("expected resource_schemas, got %v", cats)
		}
		if _, ok := cats["provider"]; ok {
			t.Errorf("expected the provider config to be dropped for -kind=resource, got %v", cats)
		}
		if _, ok := cats["resource_identity_schemas"]; ok {
			t.Errorf("expected resource_identity_schemas to be absent for -kind=resource, got %v", cats)
		}
	})

	t.Run("kind=data-source selects nothing here and is an empty success", func(t *testing.T) {
		code, out, stderr := runProvidersSchemaFixture(t, "basic", []string{"-json", "-kind=data-source"})
		if code != 0 {
			t.Fatalf("wrong exit %d; stderr: %s", code, stderr)
		}
		got := parseSmokeOutput(t, out)
		if got.Filters == nil || got.Filters.Kind != "data-source" {
			t.Fatalf("expected filters.kind=data-source, got %+v", got.Filters)
		}
		if len(got.Schemas) != 0 {
			t.Errorf("expected no provider schemas, got %v", got.Schemas)
		}
	})

	t.Run("provider and kind compose", func(t *testing.T) {
		code, out, stderr := runProvidersSchemaFixture(t, "basic", []string{"-json", "-provider=hashicorp/test", "-kind=resource"})
		if code != 0 {
			t.Fatalf("wrong exit %d; stderr: %s", code, stderr)
		}
		got := parseSmokeOutput(t, out)
		if got.Filters == nil || got.Filters.Provider != testProviderFQN || got.Filters.Kind != "resource" {
			t.Fatalf("expected provider+kind filters echo, got %+v", got.Filters)
		}
		cats := got.Schemas[testProviderFQN]
		if _, ok := cats["resource_schemas"]; !ok {
			t.Errorf("expected resource_schemas, got %v", cats)
		}
		if _, ok := cats["provider"]; ok {
			t.Errorf("expected the provider config to be dropped, got %v", cats)
		}
	})
}

func TestProvidersSchemaCommand_fullComposition(t *testing.T) {
	// The basic fixture provider exposes a single resource, test_instance.
	code, out, stderr := runProvidersSchemaFixture(t, "basic", []string{
		"-json",
		"-provider=hashicorp/test",
		"-kind=resource",
		"-type=test_instance",
	})
	if code != 0 {
		t.Fatalf("wrong exit %d; stderr: %s", code, stderr)
	}
	got := parseSmokeOutput(t, out)
	if got.Filters == nil ||
		got.Filters.Provider != testProviderFQN ||
		got.Filters.Kind != "resource" ||
		got.Filters.Type != "test_instance" {
		t.Fatalf("expected all three filters echoed, got %+v", got.Filters)
	}
	cats := got.Schemas[testProviderFQN]
	if _, ok := cats["resource_schemas"]; !ok {
		t.Fatalf("expected resource_schemas, got %v", cats)
	}
	if _, ok := cats["provider"]; ok {
		t.Errorf("expected the provider config to be dropped, got %v", cats)
	}

	// Confirm the single retained resource is the requested type.
	var resources map[string]json.RawMessage
	if err := json.Unmarshal(cats["resource_schemas"], &resources); err != nil {
		t.Fatalf("failed to parse resource_schemas: %s", err)
	}
	if _, ok := resources["test_instance"]; !ok || len(resources) != 1 {
		t.Errorf("expected only test_instance, got %v", resources)
	}
}

func TestProvidersSchemaCommand_typeNoMatch(t *testing.T) {
	// A valid type that selects nothing is an empty success (exit 0).
	code, out, stderr := runProvidersSchemaFixture(t, "basic", []string{"-json", "-type=nonexistent_thing"})
	if code != 0 {
		t.Fatalf("expected exit 0 for a type no-match, got %d; stderr: %s", code, stderr)
	}
	got := parseSmokeOutput(t, out)
	if got.Filters == nil || got.Filters.Type != "nonexistent_thing" {
		t.Fatalf("expected filters.type echo, got %+v", got.Filters)
	}
	if len(got.Schemas) != 0 {
		t.Errorf("expected no provider schemas, got %v", got.Schemas)
	}
}

func TestProvidersSchemaCommand_help(t *testing.T) {
	c := &ProvidersSchemaCommand{}
	help := c.Help()
	for _, want := range []string{"-provider", "-kind", "-type"} {
		if !strings.Contains(help, want) {
			t.Errorf("help should mention %q:\n%s", want, help)
		}
	}
	// The -kind help lists the supported canonical labels.
	for _, label := range []string{"resource", "data-source", "resource-identity", "state-store"} {
		if !strings.Contains(help, label) {
			t.Errorf("help should list the %q kind label:\n%s", label, help)
		}
	}
	// The -type help notes the exact, case-sensitive contract.
	if !strings.Contains(help, "case-sensitive") {
		t.Errorf("help should note that -type is case-sensitive:\n%s", help)
	}
}
