// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
)

func TestTransformForTest(t *testing.T) {

	str := func(providers map[string]string) string {
		var buffer bytes.Buffer
		for key, config := range providers {
			buffer.WriteString(fmt.Sprintf("%s: %s\n", key, config))
		}
		return buffer.String()
	}

	convertToProviders := func(t *testing.T, contents map[string]string) map[string]*configs.Provider {
		t.Helper()

		providers := make(map[string]*configs.Provider)
		for key, content := range contents {
			parser := hclparse.NewParser()
			file, diags := parser.ParseHCL([]byte(content), fmt.Sprintf("%s.hcl", key))
			if diags.HasErrors() {
				t.Fatal(diags.Error())
			}

			provider := &configs.Provider{
				Config: file.Body,
			}

			parts := strings.Split(key, ".")
			provider.Name = parts[0]
			if len(parts) > 1 {
				provider.Alias = parts[1]
			}

			providers[key] = provider
		}
		return providers
	}

	tcs := map[string]struct {
		configProviders   map[string]string
		fileProviders     map[string]string
		runProviders      []configs.PassedProviderConfig
		expectedProviders map[string]string
		expectedErrors    []string
	}{
		"empty": {
			configProviders:   make(map[string]string),
			expectedProviders: make(map[string]string),
		},
		"only providers in config": {
			configProviders: map[string]string{
				"foo": "source = \"config\"",
				"bar": "source = \"config\"",
			},
			expectedProviders: map[string]string{
				"foo": "source = \"config\"",
				"bar": "source = \"config\"",
			},
		},
		"only providers in test file": {
			configProviders: make(map[string]string),
			fileProviders: map[string]string{
				"foo": "source = \"testfile\"",
				"bar": "source = \"testfile\"",
			},
			expectedProviders: map[string]string{
				"foo": "source = \"testfile\"",
				"bar": "source = \"testfile\"",
			},
		},
		"only providers in run block": {
			configProviders: make(map[string]string),
			runProviders: []configs.PassedProviderConfig{
				{
					InChild: &configs.ProviderConfigRef{
						Name: "foo",
					},
					InParent: &configs.ProviderConfigRef{
						Name: "bar",
					},
				},
			},
			expectedProviders: make(map[string]string),
			expectedErrors: []string{
				":0,0-0: Missing provider definition for bar; This provider block references a provider definition that does not exist.",
			},
		},
		"subset of providers in test file": {
			configProviders: make(map[string]string),
			fileProviders: map[string]string{
				"bar": "source = \"testfile\"",
			},
			runProviders: []configs.PassedProviderConfig{
				{
					InChild: &configs.ProviderConfigRef{
						Name: "foo",
					},
					InParent: &configs.ProviderConfigRef{
						Name: "bar",
					},
				},
			},
			expectedProviders: map[string]string{
				"foo": "source = \"testfile\"",
			},
		},
		"overrides providers in config": {
			configProviders: map[string]string{
				"foo": "source = \"config\"",
				"bar": "source = \"config\"",
			},
			fileProviders: map[string]string{
				"bar": "source = \"testfile\"",
			},
			expectedProviders: map[string]string{
				"foo": "source = \"config\"",
				"bar": "source = \"testfile\"",
			},
		},
		"overrides subset of providers in config": {
			configProviders: map[string]string{
				"foo": "source = \"config\"",
				"bar": "source = \"config\"",
			},
			fileProviders: map[string]string{
				"foo": "source = \"testfile\"",
				"bar": "source = \"testfile\"",
			},
			runProviders: []configs.PassedProviderConfig{
				{
					InChild: &configs.ProviderConfigRef{
						Name: "bar",
					},
					InParent: &configs.ProviderConfigRef{
						Name: "bar",
					},
				},
			},
			expectedProviders: map[string]string{
				"foo": "source = \"config\"",
				"bar": "source = \"testfile\"",
			},
		},
		"handles aliases": {
			configProviders: map[string]string{
				"foo.primary":   "source = \"config\"",
				"foo.secondary": "source = \"config\"",
			},
			fileProviders: map[string]string{
				"foo": "source = \"testfile\"",
			},
			runProviders: []configs.PassedProviderConfig{
				{
					InChild: &configs.ProviderConfigRef{
						Name: "foo.secondary",
					},
					InParent: &configs.ProviderConfigRef{
						Name: "foo",
					},
				},
			},
			expectedProviders: map[string]string{
				"foo.primary":   "source = \"config\"",
				"foo.secondary": "source = \"testfile\"",
			},
		},
		"ignores unexpected providers in test file": {
			configProviders: make(map[string]string),
			fileProviders: map[string]string{
				"foo": "source = \"testfile\"",
				"bar": "source = \"testfile\"",
			},
			expectedProviders: map[string]string{
				"foo": "source = \"testfile\"",
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			config := &configs.Config{
				Module: &configs.Module{
					ProviderConfigs: convertToProviders(t, tc.configProviders),
				},
			}

			file := &moduletest.File{
				Config: &configs.TestFile{
					Providers: convertToProviders(t, tc.fileProviders),
				},
			}

			run := &moduletest.Run{
				Config: &configs.TestRun{
					Providers: tc.runProviders,
				},
				ModuleConfig: config,
			}

			availableProviders := make(map[string]bool, len(tc.expectedProviders))
			for provider := range tc.expectedProviders {
				availableProviders[provider] = true
			}

			ctx := NewEvalContext(&EvalContextOpts{
				CancelCtx: context.Background(),
				StopCtx:   context.Background(),
			})
			ctx.configProviders = map[string]map[string]bool{
				run.GetModuleConfigID(): availableProviders,
			}

			diags := TransformConfigForRun(ctx, run, file)

			var actualErrs []string
			for _, err := range diags.Errs() {
				actualErrs = append(actualErrs, err.Error())
			}
			if diff := cmp.Diff(actualErrs, tc.expectedErrors, cmpopts.IgnoreUnexported()); len(diff) > 0 {
				t.Errorf("unmatched errors\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", strings.Join(tc.expectedErrors, "\n"), strings.Join(actualErrs, "\n"), diff)
			}

			converted := make(map[string]string)
			for key, provider := range config.Module.ProviderConfigs {
				content, err := provider.Config.Content(&hcl.BodySchema{
					Attributes: []hcl.AttributeSchema{
						{Name: "source", Required: true},
					},
				})
				if err != nil {
					t.Fatal(err)
				}

				source, diags := content.Attributes["source"].Expr.Value(nil)
				if diags.HasErrors() {
					t.Fatal(diags.Error())
				}
				converted[key] = fmt.Sprintf("source = %q", source.AsString())
			}

			if diff := cmp.Diff(tc.expectedProviders, converted); len(diff) > 0 {
				t.Errorf("%s\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", "after transform mismatch", str(tc.expectedProviders), str(converted), diff)
			}
		})
	}
}
