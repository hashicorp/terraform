// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestGraph_planPhase_mermaid(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("graph"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	streams, closeStreams := terminal.StreamsForTesting(t)
	c := &GraphCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(applyFixtureProvider()),
			Ui:               ui,
			Streams:          streams,
		},
	}

	args := []string{"-type=plan", "-format=mermaid"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := closeStreams(t)
	if !strings.Contains(output.Stdout(), "flowchart") && !strings.Contains(output.Stdout(), "-->") {
		t.Fatalf("doesn't look like mermaid graph:\n%s\n\nstderr:\n%s", output.Stdout(), output.Stderr())
	}
}

// TestGraph_resourcesOnly_mermaid is a golden-fixture test for the default
// (resources-only) graph rendered as Mermaid flowchart syntax.  It uses the
// same "graph-interesting" fixture as TestGraph_resourcesOnly so the two
// tests document equivalent output for the dot and mermaid formats.
func TestGraph_resourcesOnly_mermaid(t *testing.T) {
	wd := tempWorkingDirFixture(t, "graph-interesting")
	t.Chdir(wd.RootModuleDir())

	loader, cleanupLoader := configload.NewLoaderForTests(t)
	t.Cleanup(cleanupLoader)
	err := os.MkdirAll(".terraform/modules", 0700)
	if err != nil {
		t.Fatal(err)
	}
	inst := initwd.NewModuleInstaller(".terraform/modules", loader, registry.NewClient(nil, nil), testModuleInstallerInitializer(loader))
	_, instDiags := inst.InstallModules(context.Background(), ".", "tests", true, false)
	if instDiags.HasErrors() {
		t.Fatal(instDiags.Err())
	}

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"foo": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"arg": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}

	ui := cli.NewMockUi()
	streams, closeStreams := terminal.StreamsForTesting(t)
	c := &GraphCommand{
		Meta: Meta{
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("foo"): providers.FactoryFixed(p),
				},
			},
			Ui:      ui,
			Streams: streams,
		},
	}

	args := []string{"-format=mermaid"}
	if code := c.Run(args); code != 0 {
		output := closeStreams(t)
		t.Fatalf("unexpected error: \n%s", output.Stderr())
	}

	output := closeStreams(t)
	gotGraph := strings.TrimSpace(output.Stdout())
	// Golden fixture: the resources-only Mermaid graph for the graph-interesting
	// configuration.  Node IDs are the full config addresses; labels are the
	// resource-local addresses (type.name).  module.child resources are wrapped
	// in a Mermaid subgraph block.
	wantGraph := strings.TrimSpace(`
flowchart LR
  foo.bar["foo.bar"]
  foo.baz["foo.baz"]
  foo.boop["foo.boop"]
  subgraph module.child
    module.child.foo.bleep["foo.bleep"]
  end
  foo.baz --> foo.bar
  foo.boop --> module.child.foo.bleep
  module.child.foo.bleep --> foo.bar
`)
	if diff := cmp.Diff(wantGraph, gotGraph); diff != "" {
		t.Fatalf("wrong mermaid graph output\n%s", diff)
	}
}
