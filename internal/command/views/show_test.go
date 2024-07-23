// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/cloud/cloudplan"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/terminal"

	"github.com/zclconf/go-cty/cty"
)

func TestShowHuman(t *testing.T) {
	redactedPath := "./testdata/plans/redacted-plan.json"
	redactedPlanJson, err := os.ReadFile(redactedPath)
	if err != nil {
		t.Fatalf("couldn't read json plan test data at %s for showing a cloud plan. Did the file get moved?", redactedPath)
	}
	testCases := map[string]struct {
		plan       *plans.Plan
		jsonPlan   *cloudplan.RemotePlanJSON
		stateFile  *statefile.File
		schemas    *terraform.Schemas
		wantExact  bool
		wantString string
	}{
		"plan file": {
			testPlan(t),
			nil,
			nil,
			testSchemas(),
			false,
			"# test_resource.foo will be created",
		},
		"cloud plan file": {
			nil,
			&cloudplan.RemotePlanJSON{
				JSONBytes: redactedPlanJson,
				Redacted:  true,
				Mode:      plans.NormalMode,
				Qualities: []plans.Quality{},
				RunHeader: "[reset][yellow]To view this run in a browser, visit:\nhttps://app.terraform.io/app/example_org/example_workspace/runs/run-run-bugsBUGSbugsBUGS[reset]",
				RunFooter: "[reset][green]Run status: planned and saved (confirmable)[reset]\n[green]Workspace is unlocked[reset]",
			},
			nil,
			nil,
			false,
			"# null_resource.foo will be created",
		},
		"statefile": {
			nil,
			nil,
			&statefile.File{
				Serial:  0,
				Lineage: "fake-for-testing",
				State:   testState(),
			},
			testSchemas(),
			false,
			"# test_resource.foo:",
		},
		"empty statefile": {
			nil,
			nil,
			&statefile.File{
				Serial:  0,
				Lineage: "fake-for-testing",
				State:   states.NewState(),
			},
			testSchemas(),
			true,
			"The state file is empty. No resources are represented.\n",
		},
		"nothing": {
			nil,
			nil,
			nil,
			nil,
			true,
			"No state.\n",
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewShow(arguments.ViewHuman, view)

			code := v.Display(nil, testCase.plan, testCase.jsonPlan, testCase.stateFile, testCase.schemas)
			if code != 0 {
				t.Errorf("expected 0 return code, got %d", code)
			}

			output := done(t)
			got := output.Stdout()
			want := testCase.wantString
			if (testCase.wantExact && got != want) || (!testCase.wantExact && !strings.Contains(got, want)) {
				t.Fatalf("unexpected output\ngot: %s\nwant: %s", got, want)
			}
		})
	}
}

func TestShowJSON(t *testing.T) {
	unredactedPath := "../testdata/show-json/basic-create/output.json"
	unredactedPlanJson, err := os.ReadFile(unredactedPath)
	if err != nil {
		t.Fatalf("couldn't read json plan test data at %s for showing a cloud plan. Did the file get moved?", unredactedPath)
	}
	testCases := map[string]struct {
		plan      *plans.Plan
		jsonPlan  *cloudplan.RemotePlanJSON
		stateFile *statefile.File
	}{
		"plan file": {
			testPlan(t),
			nil,
			nil,
		},
		"cloud plan file": {
			nil,
			&cloudplan.RemotePlanJSON{
				JSONBytes: unredactedPlanJson,
				Redacted:  false,
				Mode:      plans.NormalMode,
				Qualities: []plans.Quality{},
				RunHeader: "[reset][yellow]To view this run in a browser, visit:\nhttps://app.terraform.io/app/example_org/example_workspace/runs/run-run-bugsBUGSbugsBUGS[reset]",
				RunFooter: "[reset][green]Run status: planned and saved (confirmable)[reset]\n[green]Workspace is unlocked[reset]",
			},
			nil,
		},
		"statefile": {
			nil,
			nil,
			&statefile.File{
				Serial:  0,
				Lineage: "fake-for-testing",
				State:   testState(),
			},
		},
		"empty statefile": {
			nil,
			nil,
			&statefile.File{
				Serial:  0,
				Lineage: "fake-for-testing",
				State:   states.NewState(),
			},
		},
		"nothing": {
			nil,
			nil,
			nil,
		},
	}

	config, _, configCleanup := initwd.MustLoadConfigForTests(t, "./testdata/show", "tests")
	defer configCleanup()

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewShow(arguments.ViewJSON, view)

			schemas := &terraform.Schemas{
				Providers: map[addrs.Provider]providers.ProviderSchema{
					addrs.NewDefaultProvider("test"): {
						ResourceTypes: map[string]providers.Schema{
							"test_resource": {
								Block: &configschema.Block{
									Attributes: map[string]*configschema.Attribute{
										"id":  {Type: cty.String, Optional: true, Computed: true},
										"foo": {Type: cty.String, Optional: true},
									},
								},
							},
						},
					},
				},
			}

			code := v.Display(config, testCase.plan, testCase.jsonPlan, testCase.stateFile, schemas)

			if code != 0 {
				t.Errorf("expected 0 return code, got %d", code)
			}

			// Make sure the result looks like JSON; we comprehensively test
			// the structure of this output in the command package tests.
			var result map[string]interface{}
			got := done(t).All()
			t.Logf("output: %s", got)
			if err := json.Unmarshal([]byte(got), &result); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// testState returns a test State structure.
func testState() *states.State {
	return states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_resource",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		// DeepCopy is used here to ensure our synthetic state matches exactly
		// with a state that will have been copied during the command
		// operation, and all fields have been copied correctly.
	}).DeepCopy()
}
