// Copyright (c) HashiCorp, Inc.
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
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestGraph_planPhase(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("graph"), td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	streams, closeStreams := terminal.StreamsForTesting(t)
	c := &GraphCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(applyFixtureProvider()),
			Ui:               ui,
			Streams:          streams,
		},
	}

	args := []string{"-type=plan"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := closeStreams(t)
	if !strings.Contains(output.Stdout(), `provider[\"registry.terraform.io/hashicorp/test\"]`) {
		t.Fatalf("doesn't look like digraph:\n%s\n\nstderr:\n%s", output.Stdout(), output.Stderr())
	}
}

func TestGraph_cyclic(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("graph-cyclic"), td)
	defer testChdir(t, td)()

	tests := []struct {
		name     string
		args     []string
		expected string

		// The cyclic errors do not maintain a consistent order, so we can't
		// predict the exact output. We'll just check that the error messages
		// are present for the things we know are cyclic.
		errors []string
	}{
		{
			name: "plan",
			args: []string{"-type=plan"},
			errors: []string{`Error: Cycle: test_instance.`,
				`Error: Cycle: local.`},
		},
		{
			name: "plan with -draw-cycles option",
			args: []string{"-draw-cycles", "-type=plan"},
			expected: `digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] provider[\"registry.terraform.io/hashicorp/test\"]" [label = "provider[\"registry.terraform.io/hashicorp/test\"]", shape = "diamond"]
		"[root] test_instance.bar (expand)" [label = "test_instance.bar", shape = "box"]
		"[root] test_instance.foo (expand)" [label = "test_instance.foo", shape = "box"]
		"[root] local.test1 (expand)" -> "[root] local.test2 (expand)"
		"[root] local.test2 (expand)" -> "[root] local.test1 (expand)"
		"[root] provider[\"registry.terraform.io/hashicorp/test\"] (close)" -> "[root] provider[\"registry.terraform.io/hashicorp/test\"]"
		"[root] provider[\"registry.terraform.io/hashicorp/test\"] (close)" -> "[root] test_instance.bar (expand)"
		"[root] provider[\"registry.terraform.io/hashicorp/test\"] (close)" -> "[root] test_instance.foo (expand)"
		"[root] root" -> "[root] provider[\"registry.terraform.io/hashicorp/test\"] (close)"
		"[root] test_instance.bar (expand)" -> "[root] provider[\"registry.terraform.io/hashicorp/test\"]"
		"[root] test_instance.bar (expand)" -> "[root] test_instance.foo (expand)" [color = "red", penwidth = "2.0"]
		"[root] test_instance.foo (expand)" -> "[root] provider[\"registry.terraform.io/hashicorp/test\"]"
		"[root] test_instance.foo (expand)" -> "[root] test_instance.bar (expand)" [color = "red", penwidth = "2.0"]
	}
}`,
		},
		{
			name: "apply",
			args: []string{"-type=apply"},
			// The cyclic errors do not maintain a consistent order, so we can't
			// predict the exact output. We'll just check that the error messages
			// are present for the things we know are cyclic.
			errors: []string{`Error: Cycle: test_instance.`,
				`Error: Cycle: local.`},
		},
		{
			name: "apply with -draw-cycles option",
			args: []string{"-draw-cycles", "-type=apply"},
			expected: `digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] provider[\"registry.terraform.io/hashicorp/test\"]" [label = "provider[\"registry.terraform.io/hashicorp/test\"]", shape = "diamond"]
		"[root] test_instance.bar (expand)" [label = "test_instance.bar", shape = "box"]
		"[root] test_instance.foo (expand)" [label = "test_instance.foo", shape = "box"]
		"[root] local.test1 (expand)" -> "[root] local.test2 (expand)"
		"[root] local.test2 (expand)" -> "[root] local.test1 (expand)"
		"[root] provider[\"registry.terraform.io/hashicorp/test\"] (close)" -> "[root] provider[\"registry.terraform.io/hashicorp/test\"]"
		"[root] provider[\"registry.terraform.io/hashicorp/test\"] (close)" -> "[root] test_instance.bar (expand)"
		"[root] provider[\"registry.terraform.io/hashicorp/test\"] (close)" -> "[root] test_instance.foo (expand)"
		"[root] root" -> "[root] provider[\"registry.terraform.io/hashicorp/test\"] (close)"
		"[root] test_instance.bar (expand)" -> "[root] provider[\"registry.terraform.io/hashicorp/test\"]"
		"[root] test_instance.bar (expand)" -> "[root] test_instance.foo (expand)" [color = "red", penwidth = "2.0"]
		"[root] test_instance.foo (expand)" -> "[root] provider[\"registry.terraform.io/hashicorp/test\"]"
		"[root] test_instance.foo (expand)" -> "[root] test_instance.bar (expand)" [color = "red", penwidth = "2.0"]
	}
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := new(cli.MockUi)
			streams, closeStreams := terminal.StreamsForTesting(t)
			c := &GraphCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(applyFixtureProvider()),
					Ui:               ui,
					Streams:          streams,
				},
			}

			code := c.Run(tt.args)
			// If we expect errors, make sure they are present
			if len(tt.errors) > 0 {
				if code == 0 {
					t.Fatalf("expected error, got none")
				}
				got := strings.TrimSpace(ui.ErrorWriter.String())
				for _, err := range tt.errors {
					if !strings.Contains(got, err) {
						t.Fatalf("expected error:\n%s\n\nactual error:\n%s", err, got)
					}
				}
				return
			}

			// If we don't expect errors, make sure the command ran successfully
			if code != 0 {
				t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
			}
			output := closeStreams(t)
			if strings.TrimSpace(output.Stdout()) != strings.TrimSpace(tt.expected) {
				t.Fatalf("expected dot graph to match:\n%s", cmp.Diff(output.Stdout(), tt.expected))
			}

		})
	}
}

func TestGraph_multipleArgs(t *testing.T) {
	ui := new(cli.MockUi)
	c := &GraphCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(applyFixtureProvider()),
			Ui:               ui,
		},
	}

	args := []string{
		"bad",
		"bad",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestGraph_noConfig(t *testing.T) {
	td := t.TempDir()
	os.MkdirAll(td, 0755)
	defer testChdir(t, td)()

	streams, closeStreams := terminal.StreamsForTesting(t)
	defer closeStreams(t)
	ui := cli.NewMockUi()
	c := &GraphCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(applyFixtureProvider()),
			Ui:               ui,
			Streams:          streams,
		},
	}

	// Running the graph command without a config should not panic,
	// but this may be an error at some point in the future.
	args := []string{"-type", "apply"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestGraph_resourcesOnly(t *testing.T) {
	wd := tempWorkingDirFixture(t, "graph-interesting")
	defer testChdir(t, wd.RootModuleDir())()

	// The graph-interesting fixture has a child module, so we'll need to
	// run the module installer just to get the working directory set up
	// properly, as if the user has run "terraform init". This is really
	// just building the working directory's index of module directories.
	loader, cleanupLoader := configload.NewLoaderForTests(t)
	t.Cleanup(cleanupLoader)
	err := os.MkdirAll(".terraform/modules", 0700)
	if err != nil {
		t.Fatal(err)
	}
	inst := initwd.NewModuleInstaller(".terraform/modules", loader, registry.NewClient(nil, nil))
	_, instDiags := inst.InstallModules(context.Background(), ".", "tests", true, false, initwd.ModuleInstallHooksImpl{})
	if instDiags.HasErrors() {
		t.Fatal(instDiags.Err())
	}

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"foo": {
				Block: &configschema.Block{
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

	// A "resources only" graph is the default behavior, with no extra arguments.
	args := []string{}
	if code := c.Run(args); code != 0 {
		output := closeStreams(t)
		t.Fatalf("unexpected error: \n%s", output.Stderr())
	}

	output := closeStreams(t)
	gotGraph := strings.TrimSpace(output.Stdout())
	wantGraph := strings.TrimSpace(`
digraph G {
  rankdir = "RL";
  node [shape = rect, fontname = "sans-serif"];
  "foo.bar" [label="foo.bar"];
  "foo.baz" [label="foo.baz"];
  "foo.boop" [label="foo.boop"];
  subgraph "cluster_module.child" {
    label = "module.child"
    fontname = "sans-serif"
    "module.child.foo.bleep" [label="foo.bleep"];
  }
  "foo.baz" -> "foo.bar";
  "foo.boop" -> "module.child.foo.bleep";
  "module.child.foo.bleep" -> "foo.bar";
}
`)
	if diff := cmp.Diff(wantGraph, gotGraph); diff != "" {
		t.Fatalf("wrong result\n%s", diff)
	}
}

func TestGraph_applyPhaseSavedPlan(t *testing.T) {
	testCwd(t)

	emptyObj, err := plans.NewDynamicValue(cty.EmptyObjectVal, cty.EmptyObject)
	if err != nil {
		t.Fatal(err)
	}

	nullEmptyObj, err := plans.NewDynamicValue(cty.NullVal((cty.EmptyObject)), cty.EmptyObject)
	if err != nil {
		t.Fatal(err)
	}

	plan := &plans.Plan{
		Changes: plans.NewChangesSrc(),
	}
	plan.Changes.Resources = append(plan.Changes.Resources, &plans.ResourceInstanceChangeSrc{
		Addr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: "bar",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
		ChangeSrc: plans.ChangeSrc{
			Action: plans.Delete,
			Before: emptyObj,
			After:  nullEmptyObj,
		},
		ProviderAddr: addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	})

	plan.Backend = plans.Backend{
		// Doesn't actually matter since we aren't going to activate the backend
		// for this command anyway, but we need something here for the plan
		// file writer to succeed.
		Type:   "placeholder",
		Config: emptyObj,
	}
	_, configSnap := testModuleWithSnapshot(t, "graph")

	planPath := testPlanFile(t, configSnap, states.NewState(), plan)

	streams, closeStreams := terminal.StreamsForTesting(t)
	ui := cli.NewMockUi()
	c := &GraphCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(applyFixtureProvider()),
			Ui:               ui,
			Streams:          streams,
		},
	}

	args := []string{
		"-plan", planPath,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := closeStreams(t)
	if !strings.Contains(output.Stdout(), `provider[\"registry.terraform.io/hashicorp/test\"]`) {
		t.Fatalf("doesn't look like digraph:\n%s\n\nstderr:\n%s", output.Stdout(), output.Stderr())
	}
}
