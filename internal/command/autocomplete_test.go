// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"reflect"
	"testing"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/posener/complete"
)

func TestMetaCompletePredictWorkspaceName(t *testing.T) {

	t.Run("test autocompletion using the local backend", func(t *testing.T) {
		// Create a temporary working directory that is empty
		td := t.TempDir()
		t.Chdir(td)

		ui := new(cli.MockUi)
		meta := &Meta{Ui: ui}

		predictor := meta.completePredictWorkspaceName()

		got := predictor.Predict(complete.Args{
			Last: "",
		})
		want := []string{"default"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})

	t.Run("test autocompletion using a state store", func(t *testing.T) {
		// Create a temporary working directory with state_store config
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-unchanged"), td)
		t.Chdir(td)

		// Set up pluggable state store provider mock
		mockProvider := mockPluggableStateStorageProvider(t)
		// Mock the existence of workspaces
		mockProvider.MockStates = map[string]interface{}{
			"default": true,
			"foobar":  true,
		}
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.0.0"},
		})
		defer close()

		ui := new(cli.MockUi)
		view, _ := testView(t)
		wd := workdir.NewDir(".")
		wd.OverrideOriginalWorkingDir(td)
		meta := Meta{
			WorkingDir:                wd, // Use the test's temp dir
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}

		predictor := meta.completePredictWorkspaceName()

		got := predictor.Predict(complete.Args{
			Last: "",
		})
		want := []string{"default", "foobar"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})

	t.Run("test autocompletion using a state store containing no workspaces", func(t *testing.T) {
		// Create a temporary working directory with state_store config
		td := t.TempDir()
		testCopyDir(t, testFixturePath("state-store-unchanged"), td)
		t.Chdir(td)

		// Set up pluggable state store provider mock
		mockProvider := mockPluggableStateStorageProvider(t)
		// No workspaces exist in the mock
		mockProvider.MockStates = map[string]interface{}{}
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.0.0"},
		})
		defer close()

		ui := new(cli.MockUi)
		view, _ := testView(t)
		wd := workdir.NewDir(".")
		wd.OverrideOriginalWorkingDir(td)
		meta := Meta{
			WorkingDir:                wd, // Use the test's temp dir
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddress: providers.FactoryFixed(mockProvider),
				},
			},
			ProviderSource: providerSource,
		}

		predictor := meta.completePredictWorkspaceName()

		got := predictor.Predict(complete.Args{
			Last: "",
		})
		if got != nil {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, nil)
		}
	})
}
