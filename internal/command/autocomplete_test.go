// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	pTesting "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/posener/complete"
)

func TestMetaCompletePredictWorkspaceName(t *testing.T) {
	t.Run("test autocompletion using the local backend", func(t *testing.T) {
		// Create a temporary working directory that is empty
		td := tempWorkingDir(t)
		t.Chdir(td.RootModuleDir())

		ui := testUiWrapped(t)
		meta := &Meta{
			Ui:         ui,
			WorkingDir: td,
		}

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
		td := tempWorkingDirFixture(t, "state-store-unchanged/provider-managed-by-terraform")
		t.Chdir(td.RootModuleDir())

		// Set up pluggable state store provider mock
		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		// Mock the existence of workspaces
		mockProvider.MockStates = pTesting.NewMockStateBytesWithStateIds("test_store", []string{
			"default",
			"foobar",
		})
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.0.0"},
		})

		ui := testUiWrapped(t)
		view, _ := testView(t)
		meta := Meta{
			WorkingDir:                td,
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
		td := tempWorkingDirFixture(t, "state-store-unchanged/provider-managed-by-terraform")
		t.Chdir(td.RootModuleDir())

		// Set up pluggable state store provider mock
		mockProvider := mockPluggableStateStorageProvider(mockSingleStateStoreSchema("test_store"))
		mockProviderAddress := addrs.NewDefaultProvider("test")
		providerSource := newMockProviderSource(t, map[string][]string{
			"hashicorp/test": {"1.0.0"},
		})

		ui := testUiWrapped(t)
		view, _ := testView(t)
		meta := Meta{
			WorkingDir:                td,
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
