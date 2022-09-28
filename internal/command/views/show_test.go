package views

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"

	"github.com/zclconf/go-cty/cty"
)

func TestShowHuman(t *testing.T) {
	testCases := map[string]struct {
		plan       *plans.Plan
		stateFile  *statefile.File
		schemas    *terraform.Schemas
		wantExact  bool
		wantString string
	}{
		"plan file": {
			testPlan(t),
			nil,
			testSchemas(),
			false,
			"# test_resource.foo will be created",
		},
		"statefile": {
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
			&statefile.File{
				Serial:  0,
				Lineage: "fake-for-testing",
				State:   states.NewState(),
			},
			testSchemas(),
			true,
			"\n",
		},
		"nothing": {
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

			code := v.Display(nil, testCase.plan, testCase.stateFile, testCase.schemas)
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
	testCases := map[string]struct {
		plan      *plans.Plan
		stateFile *statefile.File
	}{
		"plan file": {
			testPlan(t),
			nil,
		},
		"statefile": {
			nil,
			&statefile.File{
				Serial:  0,
				Lineage: "fake-for-testing",
				State:   testState(),
			},
		},
		"empty statefile": {
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
		},
	}

	config, _, configCleanup := initwd.MustLoadConfigForTests(t, "./testdata/show")
	defer configCleanup()

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewShow(arguments.ViewJSON, view)

			schemas := &terraform.Schemas{
				Providers: map[addrs.Provider]*terraform.ProviderSchema{
					addrs.NewDefaultProvider("test"): {
						ResourceTypes: map[string]*configschema.Block{
							"test_resource": {
								Attributes: map[string]*configschema.Attribute{
									"id":  {Type: cty.String, Optional: true, Computed: true},
									"foo": {Type: cty.String, Optional: true},
								},
							},
						},
					},
				},
			}

			code := v.Display(config, testCase.plan, testCase.stateFile, schemas)

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
