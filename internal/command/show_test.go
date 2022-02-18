package command

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/version"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"
)

func TestShow_badArgs(t *testing.T) {
	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{
		"bad",
		"bad",
		"-no-color",
	}

	code := c.Run(args)
	output := done(t)

	if code != 1 {
		t.Fatalf("unexpected exit status %d; want 1\ngot: %s", code, output.Stdout())
	}
}

func TestShow_noArgsNoState(t *testing.T) {
	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	code := c.Run([]string{})
	output := done(t)

	if code != 0 {
		t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, output.Stderr())
	}

	got := output.Stdout()
	want := `No state.`
	if !strings.Contains(got, want) {
		t.Fatalf("unexpected output\ngot: %s\nwant: %s", got, want)
	}
}

func TestShow_noArgsWithState(t *testing.T) {
	// Get a temp cwd
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)
	// Create the default state
	testStateFileDefault(t, testState())

	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	code := c.Run([]string{})
	output := done(t)

	if code != 0 {
		t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, output.Stderr())
	}

	got := output.Stdout()
	want := `# test_instance.foo:`
	if !strings.Contains(got, want) {
		t.Fatalf("unexpected output\ngot: %s\nwant: %s", got, want)
	}
}

func TestShow_argsWithState(t *testing.T) {
	// Create the default state
	statePath := testStateFile(t, testState())
	stateDir := filepath.Dir(statePath)
	defer os.RemoveAll(stateDir)
	defer testChdir(t, stateDir)()

	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	path := filepath.Base(statePath)
	args := []string{
		path,
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)

	if code != 0 {
		t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, output.Stderr())
	}
}

// https://github.com/hashicorp/terraform/issues/21462
func TestShow_argsWithStateAliasedProvider(t *testing.T) {
	// Create the default state with aliased resource
	testState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				// The weird whitespace here is reflective of how this would
				// get written out in a real state file, due to the indentation
				// of all of the containing wrapping objects and arrays.
				AttrsJSON:    []byte("{\n            \"id\": \"bar\"\n          }"),
				Status:       states.ObjectReady,
				Dependencies: []addrs.ConfigResource{},
			},
			addrs.RootModuleInstance.ProviderConfigAliased(addrs.NewDefaultProvider("test"), "alias"),
		)
	})

	statePath := testStateFile(t, testState)
	stateDir := filepath.Dir(statePath)
	defer os.RemoveAll(stateDir)
	defer testChdir(t, stateDir)()

	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	path := filepath.Base(statePath)
	args := []string{
		path,
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)

	if code != 0 {
		t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, output.Stderr())
	}

	got := output.Stdout()
	want := `# missing schema for provider \"test.alias\"`
	if strings.Contains(got, want) {
		t.Fatalf("unexpected output\ngot: %s", got)
	}
}

func TestShow_argsPlanFileDoesNotExist(t *testing.T) {
	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{
		"doesNotExist.tfplan",
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)

	if code != 1 {
		t.Fatalf("unexpected exit status %d; want 1\ngot: %s", code, output.Stdout())
	}

	got := output.Stderr()
	want := `Plan read error: open doesNotExist.tfplan:`
	if !strings.Contains(got, want) {
		t.Errorf("unexpected output\ngot: %s\nwant:\n%s", got, want)
	}
}

func TestShow_argsStatefileDoesNotExist(t *testing.T) {
	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{
		"doesNotExist.tfstate",
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)

	if code != 1 {
		t.Fatalf("unexpected exit status %d; want 1\ngot: %s", code, output.Stdout())
	}

	got := output.Stderr()
	want := `State read error: Error loading statefile:`
	if !strings.Contains(got, want) {
		t.Errorf("unexpected output\ngot: %s\nwant:\n%s", got, want)
	}
}

func TestShow_json_argsPlanFileDoesNotExist(t *testing.T) {
	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{
		"-json",
		"doesNotExist.tfplan",
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)

	if code != 1 {
		t.Fatalf("unexpected exit status %d; want 1\ngot: %s", code, output.Stdout())
	}

	got := output.Stderr()
	want := `Plan read error: open doesNotExist.tfplan:`
	if !strings.Contains(got, want) {
		t.Errorf("unexpected output\ngot: %s\nwant:\n%s", got, want)
	}
}

func TestShow_json_argsStatefileDoesNotExist(t *testing.T) {
	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{
		"-json",
		"doesNotExist.tfstate",
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)

	if code != 1 {
		t.Fatalf("unexpected exit status %d; want 1\ngot: %s", code, output.Stdout())
	}

	got := output.Stderr()
	want := `State read error: Error loading statefile:`
	if !strings.Contains(got, want) {
		t.Errorf("unexpected output\ngot: %s\nwant:\n%s", got, want)
	}
}

func TestShow_planNoop(t *testing.T) {
	planPath := testPlanFileNoop(t)

	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{
		planPath,
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)

	if code != 0 {
		t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, output.Stderr())
	}

	got := output.Stdout()
	want := `No changes. Your infrastructure matches the configuration.`
	if !strings.Contains(got, want) {
		t.Errorf("unexpected output\ngot: %s\nwant:\n%s", got, want)
	}
}

func TestShow_planWithChanges(t *testing.T) {
	planPathWithChanges := showFixturePlanFile(t, plans.DeleteThenCreate)

	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(showFixtureProvider()),
			View:             view,
		},
	}

	args := []string{
		planPathWithChanges,
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)

	if code != 0 {
		t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, output.Stderr())
	}

	got := output.Stdout()
	want := `test_instance.foo must be replaced`
	if !strings.Contains(got, want) {
		t.Fatalf("unexpected output\ngot: %s\nwant: %s", got, want)
	}
}

func TestShow_planWithForceReplaceChange(t *testing.T) {
	// The main goal of this test is to see that the "replace by request"
	// resource instance action reason can round-trip through a plan file and
	// be reflected correctly in the "terraform show" output, the same way
	// as it would appear in "terraform plan" output.

	_, snap := testModuleWithSnapshot(t, "show")
	plannedVal := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.UnknownVal(cty.String),
		"ami": cty.StringVal("bar"),
	})
	priorValRaw, err := plans.NewDynamicValue(cty.NullVal(plannedVal.Type()), plannedVal.Type())
	if err != nil {
		t.Fatal(err)
	}
	plannedValRaw, err := plans.NewDynamicValue(plannedVal, plannedVal.Type())
	if err != nil {
		t.Fatal(err)
	}
	plan := testPlan(t)
	plan.Changes.SyncWrapper().AppendResourceInstanceChange(&plans.ResourceInstanceChangeSrc{
		Addr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: "foo",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
		ProviderAddr: addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
		ChangeSrc: plans.ChangeSrc{
			Action: plans.CreateThenDelete,
			Before: priorValRaw,
			After:  plannedValRaw,
		},
		ActionReason: plans.ResourceInstanceReplaceByRequest,
	})
	planFilePath := testPlanFile(
		t,
		snap,
		states.NewState(),
		plan,
	)

	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(showFixtureProvider()),
			View:             view,
		},
	}

	args := []string{
		planFilePath,
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)

	if code != 0 {
		t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, output.Stderr())
	}

	got := output.Stdout()
	want := `test_instance.foo will be replaced, as requested`
	if !strings.Contains(got, want) {
		t.Fatalf("unexpected output\ngot: %s\nwant: %s", got, want)
	}

	want = `Plan: 1 to add, 0 to change, 1 to destroy.`
	if !strings.Contains(got, want) {
		t.Fatalf("unexpected output\ngot: %s\nwant: %s", got, want)
	}

}

func TestShow_plan_json(t *testing.T) {
	planPath := showFixturePlanFile(t, plans.Create)

	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(showFixtureProvider()),
			View:             view,
		},
	}

	args := []string{
		"-json",
		planPath,
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)

	if code != 0 {
		t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, output.Stderr())
	}
}

func TestShow_state(t *testing.T) {
	originalState := testState()
	statePath := testStateFile(t, originalState)
	defer os.RemoveAll(filepath.Dir(statePath))

	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{
		statePath,
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)

	if code != 0 {
		t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, output.Stderr())
	}
}

func TestShow_json_output(t *testing.T) {
	fixtureDir := "testdata/show-json"
	testDirs, err := ioutil.ReadDir(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range testDirs {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			td := tempDir(t)
			inputDir := filepath.Join(fixtureDir, entry.Name())
			testCopyDir(t, inputDir, td)
			defer os.RemoveAll(td)
			defer testChdir(t, td)()

			expectError := strings.Contains(entry.Name(), "error")

			providerSource, close := newMockProviderSource(t, map[string][]string{
				"test":            {"1.2.3"},
				"hashicorp2/test": {"1.2.3"},
			})
			defer close()

			p := showFixtureProvider()

			// init
			ui := new(cli.MockUi)
			ic := &InitCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					Ui:               ui,
					ProviderSource:   providerSource,
				},
			}
			if code := ic.Run([]string{}); code != 0 {
				if expectError {
					// this should error, but not panic.
					return
				}
				t.Fatalf("init failed\n%s", ui.ErrorWriter)
			}

			// plan
			planView, planDone := testView(t)
			pc := &PlanCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					View:             planView,
					ProviderSource:   providerSource,
				},
			}

			args := []string{
				"-out=terraform.plan",
			}

			code := pc.Run(args)
			planOutput := planDone(t)

			if code != 0 {
				t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, planOutput.Stderr())
			}

			// show
			showView, showDone := testView(t)
			sc := &ShowCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					View:             showView,
					ProviderSource:   providerSource,
				},
			}

			args = []string{
				"-json",
				"terraform.plan",
			}
			defer os.Remove("terraform.plan")
			code = sc.Run(args)
			showOutput := showDone(t)

			if code != 0 {
				t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, showOutput.Stderr())
			}

			// compare view output to wanted output
			var got, want plan

			gotString := showOutput.Stdout()
			json.Unmarshal([]byte(gotString), &got)

			wantFile, err := os.Open("output.json")
			if err != nil {
				t.Fatalf("unexpected err: %s", err)
			}
			defer wantFile.Close()
			byteValue, err := ioutil.ReadAll(wantFile)
			if err != nil {
				t.Fatalf("unexpected err: %s", err)
			}
			json.Unmarshal([]byte(byteValue), &want)

			// Disregard format version to reduce needless test fixture churn
			want.FormatVersion = got.FormatVersion

			if !cmp.Equal(got, want) {
				t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, want))
			}
		})
	}
}

func TestShow_json_output_sensitive(t *testing.T) {
	td := tempDir(t)
	inputDir := "testdata/show-json-sensitive"
	testCopyDir(t, inputDir, td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	providerSource, close := newMockProviderSource(t, map[string][]string{"test": {"1.2.3"}})
	defer close()

	p := showFixtureSensitiveProvider()

	// init
	ui := new(cli.MockUi)
	ic := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			ProviderSource:   providerSource,
		},
	}
	if code := ic.Run([]string{}); code != 0 {
		t.Fatalf("init failed\n%s", ui.ErrorWriter)
	}

	// plan
	planView, planDone := testView(t)
	pc := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             planView,
			ProviderSource:   providerSource,
		},
	}

	args := []string{
		"-out=terraform.plan",
	}
	code := pc.Run(args)
	planOutput := planDone(t)

	if code != 0 {
		t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, planOutput.Stderr())
	}

	// show
	showView, showDone := testView(t)
	sc := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             showView,
			ProviderSource:   providerSource,
		},
	}

	args = []string{
		"-json",
		"terraform.plan",
	}
	defer os.Remove("terraform.plan")
	code = sc.Run(args)
	showOutput := showDone(t)

	if code != 0 {
		t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, showOutput.Stderr())
	}

	// compare ui output to wanted output
	var got, want plan

	gotString := showOutput.Stdout()
	json.Unmarshal([]byte(gotString), &got)

	wantFile, err := os.Open("output.json")
	if err != nil {
		t.Fatalf("unexpected err: %s", err)
	}
	defer wantFile.Close()
	byteValue, err := ioutil.ReadAll(wantFile)
	if err != nil {
		t.Fatalf("unexpected err: %s", err)
	}
	json.Unmarshal([]byte(byteValue), &want)

	// Disregard format version to reduce needless test fixture churn
	want.FormatVersion = got.FormatVersion

	if !cmp.Equal(got, want) {
		t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, want))
	}
}

// similar test as above, without the plan
func TestShow_json_output_state(t *testing.T) {
	fixtureDir := "testdata/show-json-state"
	testDirs, err := ioutil.ReadDir(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range testDirs {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			td := tempDir(t)
			inputDir := filepath.Join(fixtureDir, entry.Name())
			testCopyDir(t, inputDir, td)
			defer os.RemoveAll(td)
			defer testChdir(t, td)()

			providerSource, close := newMockProviderSource(t, map[string][]string{
				"test": {"1.2.3"},
			})
			defer close()

			p := showFixtureProvider()

			// init
			ui := new(cli.MockUi)
			ic := &InitCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					Ui:               ui,
					ProviderSource:   providerSource,
				},
			}
			if code := ic.Run([]string{}); code != 0 {
				t.Fatalf("init failed\n%s", ui.ErrorWriter)
			}

			// show
			showView, showDone := testView(t)
			sc := &ShowCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					View:             showView,
					ProviderSource:   providerSource,
				},
			}

			code := sc.Run([]string{"-json"})
			showOutput := showDone(t)

			if code != 0 {
				t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, showOutput.Stderr())
			}

			// compare ui output to wanted output
			type state struct {
				FormatVersion    string                 `json:"format_version,omitempty"`
				TerraformVersion string                 `json:"terraform_version"`
				Values           map[string]interface{} `json:"values,omitempty"`
				SensitiveValues  map[string]bool        `json:"sensitive_values,omitempty"`
			}
			var got, want state

			gotString := showOutput.Stdout()
			json.Unmarshal([]byte(gotString), &got)

			wantFile, err := os.Open("output.json")
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			defer wantFile.Close()
			byteValue, err := ioutil.ReadAll(wantFile)
			if err != nil {
				t.Fatalf("unexpected err: %s", err)
			}
			json.Unmarshal([]byte(byteValue), &want)

			if !cmp.Equal(got, want) {
				t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, want))
			}
		})
	}
}

func TestShow_planWithNonDefaultStateLineage(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	testCopyDir(t, testFixturePath("show"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// Write default state file with a testing lineage ("fake-for-testing")
	testStateFileDefault(t, testState())

	// Create a plan with a different lineage, which we should still be able
	// to show
	_, snap := testModuleWithSnapshot(t, "show")
	state := testState()
	plan := testPlan(t)
	stateMeta := statemgr.SnapshotMeta{
		Lineage:          "fake-for-plan",
		Serial:           1,
		TerraformVersion: version.SemVer,
	}
	planPath := testPlanFileMatchState(t, snap, state, plan, stateMeta)

	view, done := testView(t)
	c := &ShowCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			View:             view,
		},
	}

	args := []string{
		planPath,
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)

	if code != 0 {
		t.Fatalf("unexpected exit status %d; want 0\ngot: %s", code, output.Stderr())
	}

	got := output.Stdout()
	want := `No changes. Your infrastructure matches the configuration.`
	if !strings.Contains(got, want) {
		t.Fatalf("unexpected output\ngot: %s\nwant: %s", got, want)
	}
}

// showFixtureSchema returns a schema suitable for processing the configuration
// in testdata/show. This schema should be assigned to a mock provider
// named "test".
func showFixtureSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"region": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Optional: true, Computed: true},
						"ami": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
}

// showFixtureSensitiveSchema returns a schema suitable for processing the configuration
// in testdata/show. This schema should be assigned to a mock provider
// named "test". It includes a sensitive attribute.
func showFixtureSensitiveSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"region": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":       {Type: cty.String, Optional: true, Computed: true},
						"ami":      {Type: cty.String, Optional: true},
						"password": {Type: cty.String, Optional: true, Sensitive: true},
					},
				},
			},
		},
	}
}

// showFixtureProvider returns a mock provider that is configured for basic
// operation with the configuration in testdata/show. This mock has
// GetSchemaResponse, PlanResourceChangeFn, and ApplyResourceChangeFn populated,
// with the plan/apply steps just passing through the data determined by
// Terraform Core.
func showFixtureProvider() *terraform.MockProvider {
	p := testProvider()
	p.GetProviderSchemaResponse = showFixtureSchema()
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		idVal := req.PriorState.GetAttr("id")
		amiVal := req.PriorState.GetAttr("ami")
		if amiVal.RawEquals(cty.StringVal("refresh-me")) {
			amiVal = cty.StringVal("refreshed")
		}
		return providers.ReadResourceResponse{
			NewState: cty.ObjectVal(map[string]cty.Value{
				"id":  idVal,
				"ami": amiVal,
			}),
			Private: req.Private,
		}
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		idVal := req.ProposedNewState.GetAttr("id")
		amiVal := req.ProposedNewState.GetAttr("ami")
		if idVal.IsNull() {
			idVal = cty.UnknownVal(cty.String)
		}
		var reqRep []cty.Path
		if amiVal.RawEquals(cty.StringVal("force-replace")) {
			reqRep = append(reqRep, cty.GetAttrPath("ami"))
		}
		return providers.PlanResourceChangeResponse{
			PlannedState: cty.ObjectVal(map[string]cty.Value{
				"id":  idVal,
				"ami": amiVal,
			}),
			RequiresReplace: reqRep,
		}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		idVal := req.PlannedState.GetAttr("id")
		amiVal := req.PlannedState.GetAttr("ami")
		if !idVal.IsKnown() {
			idVal = cty.StringVal("placeholder")
		}
		return providers.ApplyResourceChangeResponse{
			NewState: cty.ObjectVal(map[string]cty.Value{
				"id":  idVal,
				"ami": amiVal,
			}),
		}
	}
	return p
}

// showFixtureSensitiveProvider returns a mock provider that is configured for basic
// operation with the configuration in testdata/show. This mock has
// GetSchemaResponse, PlanResourceChangeFn, and ApplyResourceChangeFn populated,
// with the plan/apply steps just passing through the data determined by
// Terraform Core. It also has a sensitive attribute in the provider schema.
func showFixtureSensitiveProvider() *terraform.MockProvider {
	p := testProvider()
	p.GetProviderSchemaResponse = showFixtureSensitiveSchema()
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		idVal := req.ProposedNewState.GetAttr("id")
		if idVal.IsNull() {
			idVal = cty.UnknownVal(cty.String)
		}
		return providers.PlanResourceChangeResponse{
			PlannedState: cty.ObjectVal(map[string]cty.Value{
				"id":       idVal,
				"ami":      req.ProposedNewState.GetAttr("ami"),
				"password": req.ProposedNewState.GetAttr("password"),
			}),
		}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		idVal := req.PlannedState.GetAttr("id")
		if !idVal.IsKnown() {
			idVal = cty.StringVal("placeholder")
		}
		return providers.ApplyResourceChangeResponse{
			NewState: cty.ObjectVal(map[string]cty.Value{
				"id":       idVal,
				"ami":      req.PlannedState.GetAttr("ami"),
				"password": req.PlannedState.GetAttr("password"),
			}),
		}
	}
	return p
}

// showFixturePlanFile creates a plan file at a temporary location containing a
// single change to create or update the test_instance.foo that is included in the "show"
// test fixture, returning the location of that plan file.
// `action` is the planned change you would like to elicit
func showFixturePlanFile(t *testing.T, action plans.Action) string {
	_, snap := testModuleWithSnapshot(t, "show")
	plannedVal := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.UnknownVal(cty.String),
		"ami": cty.StringVal("bar"),
	})
	priorValRaw, err := plans.NewDynamicValue(cty.NullVal(plannedVal.Type()), plannedVal.Type())
	if err != nil {
		t.Fatal(err)
	}
	plannedValRaw, err := plans.NewDynamicValue(plannedVal, plannedVal.Type())
	if err != nil {
		t.Fatal(err)
	}
	plan := testPlan(t)
	plan.Changes.SyncWrapper().AppendResourceInstanceChange(&plans.ResourceInstanceChangeSrc{
		Addr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: "foo",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
		ProviderAddr: addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
		ChangeSrc: plans.ChangeSrc{
			Action: action,
			Before: priorValRaw,
			After:  plannedValRaw,
		},
	})
	return testPlanFile(
		t,
		snap,
		states.NewState(),
		plan,
	)
}

// this simplified plan struct allows us to preserve field order when marshaling
// the command output. NOTE: we are leaving "terraform_version" out of this test
// to avoid needing to constantly update the expected output; as a potential
// TODO we could write a jsonplan compare function.
type plan struct {
	FormatVersion   string                 `json:"format_version,omitempty"`
	Variables       map[string]interface{} `json:"variables,omitempty"`
	PlannedValues   map[string]interface{} `json:"planned_values,omitempty"`
	ResourceDrift   []interface{}          `json:"resource_drift,omitempty"`
	ResourceChanges []interface{}          `json:"resource_changes,omitempty"`
	OutputChanges   map[string]interface{} `json:"output_changes,omitempty"`
	PriorState      priorState             `json:"prior_state,omitempty"`
	Config          map[string]interface{} `json:"configuration,omitempty"`
}

type priorState struct {
	FormatVersion   string                 `json:"format_version,omitempty"`
	Values          map[string]interface{} `json:"values,omitempty"`
	SensitiveValues map[string]bool        `json:"sensitive_values,omitempty"`
}
