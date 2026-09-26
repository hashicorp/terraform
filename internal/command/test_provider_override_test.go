// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"path"
	"strings"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	testing_command "github.com/hashicorp/terraform/internal/command/testing"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestCollectTestProviderVersionOverrides(t *testing.T) {
	aws := addrs.NewDefaultProvider("aws")
	random := addrs.NewDefaultProvider("random")

	config := &configs.Config{
		Module: &configs.Module{
			Tests: map[string]*configs.TestFile{
				"tests/latest.tftest.hcl": {
					RequiredProviders: &configs.RequiredProviders{
						RequiredProviders: map[string]*configs.RequiredProvider{
							"aws": {
								Name:        "aws",
								Type:        aws,
								Requirement: mustVersionConstraint(t, ">= 5.0.0"),
							},
						},
					},
				},
				"tests/pinned.tftest.hcl": {
					RequiredProviders: &configs.RequiredProviders{
						RequiredProviders: map[string]*configs.RequiredProvider{
							"random": {
								Name:        "random",
								Type:        random,
								Requirement: mustVersionConstraint(t, "3.6.0"),
							},
						},
					},
				},
			},
		},
	}

	t.Run("all files", func(t *testing.T) {
		got := collectTestProviderVersionOverrides(config, nil, nil)
		if len(got) != 2 {
			t.Fatalf("expected 2 overrides, got %d", len(got))
		}
	})

	t.Run("filter uses only matching files", func(t *testing.T) {
		got := collectTestProviderVersionOverrides(config, nil, []string{"tests/latest.tftest.hcl"})
		if len(got) != 1 {
			t.Fatalf("expected 1 override, got %d", len(got))
		}
		if got[0].Addr != aws {
			t.Fatalf("expected aws override, got %s", got[0].Addr)
		}
	})

	t.Run("cli flag replaces file override", func(t *testing.T) {
		cli := []arguments.ProviderVersionOverride{
			{
				Raw:        "hashicorp/aws=5.70.0",
				Addr:       aws,
				Version:    "5.70.0",
				Constraint: providerreqs.MustParseVersionConstraints("5.70.0"),
			},
		}
		got := collectTestProviderVersionOverrides(config, cli, []string{"tests/latest.tftest.hcl"})
		if len(got) != 1 {
			t.Fatalf("expected 1 override, got %d", len(got))
		}
		if !strings.Contains(providerreqs.VersionConstraintsString(got[0].Constraint), "5.70.0") {
			t.Fatalf("expected CLI constraint to win, got %s", providerreqs.VersionConstraintsString(got[0].Constraint))
		}
		if !strings.Contains(got[0].Origin, "-override-provider=") {
			t.Fatalf("expected origin to mention CLI flag, got %s", got[0].Origin)
		}
	})
}

func TestTest_OverrideProviderFlag(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "simple_pass")), td)
	t.Chdir(td)

	providerSource := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := testUiWrapped(t)
	store := &testing_command.ResourceStore{
		Data: make(map[string]cty.Value),
	}

	meta := Meta{
		testingOverrides: &testingOverrides{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): func() (providers.Interface, error) {
					return testing_command.NewProvider(store).Provider, nil
				},
			},
		},
		Ui:             ui,
		View:           view,
		Streams:        streams,
		ProviderSource: providerSource,
	}

	if code := (&InitCommand{Meta: meta}).Run(nil); code != 0 {
		t.Fatalf("init failed: %s", done(t).All())
	}
	done(t)

	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	code := (&TestCommand{Meta: meta}).Run([]string{
		"-no-color",
		"-override-provider=hashicorp/test=1.0.0",
	})
	output := done(t)
	if code != 0 {
		t.Fatalf("expected success, got %d\n%s", code, output.All())
	}
	if !strings.Contains(output.Stdout(), "1 passed") {
		t.Fatalf("expected passing test output, got:\n%s", output.Stdout())
	}
}

func TestTest_OverrideProviderFlag_Invalid(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "simple_pass")), td)
	t.Chdir(td)

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	c := &TestCommand{
		Meta: Meta{
			View:    view,
			Streams: streams,
		},
	}

	code := c.Run([]string{"-no-color", "-override-provider=hashicorp/aws"})
	output := done(t)
	if code == 0 {
		t.Fatalf("expected failure, got success\n%s", output.All())
	}
	if !strings.Contains(output.All(), "Invalid -override-provider value") {
		t.Fatalf("expected invalid override diagnostic, got:\n%s", output.All())
	}
}

func mustVersionConstraint(t *testing.T, raw string) configs.VersionConstraint {
	t.Helper()
	constraint, err := version.NewConstraint(raw)
	if err != nil {
		t.Fatal(err)
	}
	return configs.VersionConstraint{Required: constraint}
}
