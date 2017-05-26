package command

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	atlas "github.com/hashicorp/atlas-go/v1"
	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestPush_good(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create remote state file, this should be pulled
	conf, srv := testRemoteState(t, testState(), 200)
	defer srv.Close()

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	s.Remote = conf
	testStateFileRemote(t, s)

	// Path where the archive will be "uploaded" to
	archivePath := testTempFile(t)
	defer os.Remove(archivePath)

	client := &mockPushClient{File: archivePath}
	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},

		client: client,
	}

	args := []string{
		"-vcs=false",
		testFixturePath("push"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := testArchiveStr(t, archivePath)
	expected := []string{
		".terraform/",
		".terraform/terraform.tfstate",
		"main.tf",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	variables := make(map[string]interface{})
	if !reflect.DeepEqual(client.UpsertOptions.Variables, variables) {
		t.Fatalf("bad: %#v", client.UpsertOptions)
	}

	if client.UpsertOptions.Name != "foo" {
		t.Fatalf("bad: %#v", client.UpsertOptions)
	}
}

func TestPush_goodBackendInit(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("push-backend-new"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// init backend
	ui := new(cli.MockUi)
	ci := &InitCommand{
		Meta: Meta{
			Ui: ui,
		},
	}
	if code := ci.Run(nil); code != 0 {
		t.Fatalf("bad: %d\n%s", code, ui.ErrorWriter)
	}

	// Path where the archive will be "uploaded" to
	archivePath := testTempFile(t)
	defer os.Remove(archivePath)

	client := &mockPushClient{File: archivePath}
	ui = new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},

		client: client,
	}

	args := []string{
		"-vcs=false",
		td,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := testArchiveStr(t, archivePath)
	expected := []string{
		// Expected weird behavior, doesn't affect unpackaging
		".terraform/",
		".terraform/",
		".terraform/terraform.tfstate",
		".terraform/terraform.tfstate",
		"main.tf",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	variables := make(map[string]interface{})
	if !reflect.DeepEqual(client.UpsertOptions.Variables, variables) {
		t.Fatalf("bad: %#v", client.UpsertOptions)
	}

	if client.UpsertOptions.Name != "hello" {
		t.Fatalf("bad: %#v", client.UpsertOptions)
	}
}

func TestPush_noUploadModules(t *testing.T) {
	// Path where the archive will be "uploaded" to
	archivePath := testTempFile(t)
	defer os.Remove(archivePath)

	client := &mockPushClient{File: archivePath}
	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},

		client: client,
	}

	// Path of the test. We have to do some renaming to avoid our own
	// VCS getting in the way.
	path := testFixturePath("push-no-upload")
	defer os.RemoveAll(filepath.Join(path, ".terraform"))

	// Move into that directory
	defer testChdir(t, path)()

	// Do a "terraform get"
	{
		ui := new(cli.MockUi)
		c := &GetCommand{
			Meta: Meta{
				ContextOpts: testCtxConfig(testProvider()),
				Ui:          ui,
			},
		}

		if code := c.Run([]string{}); code != 0 {
			t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
		}
	}

	// Create remote state file, this should be pulled
	conf, srv := testRemoteState(t, testState(), 200)
	defer srv.Close()

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	s.Remote = conf
	defer os.Remove(testStateFileRemote(t, s))

	args := []string{
		"-vcs=false",
		"-name=mitchellh/tf-test",
		"-upload-modules=false",
		path,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// NOTE: The duplicates below are not ideal but are how things work
	// currently due to how we manually add the files to the archive. This
	// is definitely a "bug" we can fix in the future.
	actual := testArchiveStr(t, archivePath)
	expected := []string{
		".terraform/",
		".terraform/",
		".terraform/terraform.tfstate",
		".terraform/terraform.tfstate",
		"child/",
		"child/main.tf",
		"main.tf",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestPush_input(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create remote state file, this should be pulled
	conf, srv := testRemoteState(t, testState(), 200)
	defer srv.Close()

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	s.Remote = conf
	testStateFileRemote(t, s)

	// Path where the archive will be "uploaded" to
	archivePath := testTempFile(t)
	defer os.Remove(archivePath)

	client := &mockPushClient{File: archivePath}
	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},

		client: client,
	}

	// Disable test mode so input would be asked and setup the
	// input reader/writers.
	test = false
	defer func() { test = true }()
	defaultInputReader = bytes.NewBufferString("foo\n")
	defaultInputWriter = new(bytes.Buffer)

	args := []string{
		"-vcs=false",
		testFixturePath("push-input"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	variables := map[string]interface{}{
		"foo": "foo",
	}

	if !reflect.DeepEqual(client.UpsertOptions.Variables, variables) {
		t.Fatalf("bad: %#v", client.UpsertOptions.Variables)
	}
}

// We want a variable from atlas to fill a missing variable locally
func TestPush_inputPartial(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create remote state file, this should be pulled
	conf, srv := testRemoteState(t, testState(), 200)
	defer srv.Close()

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	s.Remote = conf
	testStateFileRemote(t, s)

	// Path where the archive will be "uploaded" to
	archivePath := testTempFile(t)
	defer os.Remove(archivePath)

	client := &mockPushClient{
		File: archivePath,
		GetResult: map[string]atlas.TFVar{
			"foo": atlas.TFVar{Key: "foo", Value: "bar"},
		},
	}
	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},

		client: client,
	}

	// Disable test mode so input would be asked and setup the
	// input reader/writers.
	test = false
	defer func() { test = true }()
	defaultInputReader = bytes.NewBufferString("foo\n")
	defaultInputWriter = new(bytes.Buffer)

	args := []string{
		"-vcs=false",
		testFixturePath("push-input-partial"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	expectedTFVars := []atlas.TFVar{
		{Key: "bar", Value: "foo"},
		{Key: "foo", Value: "bar"},
	}
	if !reflect.DeepEqual(client.UpsertOptions.TFVars, expectedTFVars) {
		t.Logf("expected: %#v", expectedTFVars)
		t.Fatalf("got:      %#v", client.UpsertOptions.TFVars)
	}
}

// This tests that the push command will override Atlas variables
// if requested.
func TestPush_localOverride(t *testing.T) {
	// Disable test mode so input would be asked and setup the
	// input reader/writers.
	test = false
	defer func() { test = true }()
	defaultInputReader = bytes.NewBufferString("nope\n")
	defaultInputWriter = new(bytes.Buffer)

	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create remote state file, this should be pulled
	conf, srv := testRemoteState(t, testState(), 200)
	defer srv.Close()

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	s.Remote = conf
	testStateFileRemote(t, s)

	// Path where the archive will be "uploaded" to
	archivePath := testTempFile(t)
	defer os.Remove(archivePath)

	client := &mockPushClient{File: archivePath}
	// Provided vars should override existing ones
	client.GetResult = map[string]atlas.TFVar{
		"foo": atlas.TFVar{
			Key:   "foo",
			Value: "old",
		},
	}
	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},

		client: client,
	}

	path := testFixturePath("push-tfvars")
	args := []string{
		"-var-file", path + "/terraform.tfvars",
		"-vcs=false",
		"-overwrite=foo",
		path,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := testArchiveStr(t, archivePath)
	expected := []string{
		".terraform/",
		".terraform/terraform.tfstate",
		"main.tf",
		"terraform.tfvars",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	if client.UpsertOptions.Name != "foo" {
		t.Fatalf("bad: %#v", client.UpsertOptions)
	}

	expectedTFVars := pushTFVars()

	if !reflect.DeepEqual(client.UpsertOptions.TFVars, expectedTFVars) {
		t.Logf("expected: %#v", expectedTFVars)
		t.Fatalf("got:    %#v", client.UpsertOptions.TFVars)
	}
}

// This tests that the push command will override Atlas variables
// even if we don't have it defined locally
func TestPush_remoteOverride(t *testing.T) {
	// Disable test mode so input would be asked and setup the
	// input reader/writers.
	test = false
	defer func() { test = true }()
	defaultInputReader = bytes.NewBufferString("nope\n")
	defaultInputWriter = new(bytes.Buffer)

	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create remote state file, this should be pulled
	conf, srv := testRemoteState(t, testState(), 200)
	defer srv.Close()

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	s.Remote = conf
	testStateFileRemote(t, s)

	// Path where the archive will be "uploaded" to
	archivePath := testTempFile(t)
	defer os.Remove(archivePath)

	client := &mockPushClient{File: archivePath}
	// Provided vars should override existing ones
	client.GetResult = map[string]atlas.TFVar{
		"remote": atlas.TFVar{
			Key:   "remote",
			Value: "old",
		},
	}
	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},

		client: client,
	}

	path := testFixturePath("push-tfvars")
	args := []string{
		"-var-file", path + "/terraform.tfvars",
		"-vcs=false",
		"-overwrite=remote",
		"-var",
		"remote=new",
		path,
	}

	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := testArchiveStr(t, archivePath)
	expected := []string{
		".terraform/",
		".terraform/terraform.tfstate",
		"main.tf",
		"terraform.tfvars",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	if client.UpsertOptions.Name != "foo" {
		t.Fatalf("bad: %#v", client.UpsertOptions)
	}

	found := false
	// find the "remote" var and make sure we're going to set it
	for _, tfVar := range client.UpsertOptions.TFVars {
		if tfVar.Key == "remote" {
			found = true
			if tfVar.Value != "new" {
				t.Log("'remote' variable should be set to 'new'")
				t.Fatalf("sending instead: %#v", tfVar)
			}
		}
	}

	if !found {
		t.Fatal("'remote' variable not being sent to atlas")
	}
}

// This tests that the push command prefers Atlas variables over
// local ones.
func TestPush_preferAtlas(t *testing.T) {
	// Disable test mode so input would be asked and setup the
	// input reader/writers.
	test = false
	defer func() { test = true }()
	defaultInputReader = bytes.NewBufferString("nope\n")
	defaultInputWriter = new(bytes.Buffer)

	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create remote state file, this should be pulled
	conf, srv := testRemoteState(t, testState(), 200)
	defer srv.Close()

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	s.Remote = conf
	testStateFileRemote(t, s)

	// Path where the archive will be "uploaded" to
	archivePath := testTempFile(t)
	defer os.Remove(archivePath)

	client := &mockPushClient{File: archivePath}
	// Provided vars should override existing ones
	client.GetResult = map[string]atlas.TFVar{
		"foo": atlas.TFVar{
			Key:   "foo",
			Value: "old",
		},
	}
	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},

		client: client,
	}

	path := testFixturePath("push-tfvars")
	args := []string{
		"-var-file", path + "/terraform.tfvars",
		"-vcs=false",
		path,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := testArchiveStr(t, archivePath)
	expected := []string{
		".terraform/",
		".terraform/terraform.tfstate",
		"main.tf",
		"terraform.tfvars",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	if client.UpsertOptions.Name != "foo" {
		t.Fatalf("bad: %#v", client.UpsertOptions)
	}

	// change the expected response to match our change
	expectedTFVars := pushTFVars()
	for i, v := range expectedTFVars {
		if v.Key == "foo" {
			expectedTFVars[i] = atlas.TFVar{Key: "foo", Value: "old"}
		}
	}

	if !reflect.DeepEqual(expectedTFVars, client.UpsertOptions.TFVars) {
		t.Logf("expected: %#v", expectedTFVars)
		t.Fatalf("got:      %#v", client.UpsertOptions.TFVars)
	}
}

// This tests that the push command will send the variables in tfvars
func TestPush_tfvars(t *testing.T) {
	// Disable test mode so input would be asked and setup the
	// input reader/writers.
	test = false
	defer func() { test = true }()
	defaultInputReader = bytes.NewBufferString("nope\n")
	defaultInputWriter = new(bytes.Buffer)

	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create remote state file, this should be pulled
	conf, srv := testRemoteState(t, testState(), 200)
	defer srv.Close()

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	s.Remote = conf
	testStateFileRemote(t, s)

	// Path where the archive will be "uploaded" to
	archivePath := testTempFile(t)
	defer os.Remove(archivePath)

	client := &mockPushClient{File: archivePath}
	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},

		client: client,
	}

	path := testFixturePath("push-tfvars")
	args := []string{
		"-var-file", path + "/terraform.tfvars",
		"-vcs=false",
		"-var",
		"bar=[1,2]",
		path,
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	actual := testArchiveStr(t, archivePath)
	expected := []string{
		".terraform/",
		".terraform/terraform.tfstate",
		"main.tf",
		"terraform.tfvars",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}

	if client.UpsertOptions.Name != "foo" {
		t.Fatalf("bad: %#v", client.UpsertOptions)
	}

	//now check TFVars
	tfvars := pushTFVars()
	// update bar to match cli value
	for i, v := range tfvars {
		if v.Key == "bar" {
			tfvars[i].Value = "[1, 2]"
			tfvars[i].IsHCL = true
		}
	}

	for i, expected := range tfvars {
		got := client.UpsertOptions.TFVars[i]
		if got != expected {
			t.Logf("%2d expected: %#v", i, expected)
			t.Fatalf("        got: %#v", got)
		}
	}
}

func TestPush_name(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create remote state file, this should be pulled
	conf, srv := testRemoteState(t, testState(), 200)
	defer srv.Close()

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	s.Remote = conf
	testStateFileRemote(t, s)

	// Path where the archive will be "uploaded" to
	archivePath := testTempFile(t)
	defer os.Remove(archivePath)

	client := &mockPushClient{File: archivePath}
	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},

		client: client,
	}

	args := []string{
		"-name", "bar",
		"-vcs=false",
		testFixturePath("push"),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if client.UpsertOptions.Name != "bar" {
		t.Fatalf("bad: %#v", client.UpsertOptions)
	}
}

func TestPush_noState(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}
}

func TestPush_noRemoteState(t *testing.T) {
	// Create a temporary working directory that is empty
	td := tempDir(t)
	copy.CopyDir(testFixturePath("push-no-remote"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	state := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}
	statePath := testStateFile(t, state)

	// Path where the archive will be "uploaded" to
	archivePath := testTempFile(t)
	defer os.Remove(archivePath)

	client := &mockPushClient{File: archivePath}
	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			Ui: ui,
		},
		client: client,
	}

	args := []string{
		"-vcs=false",
		"-state", statePath,
		td,
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	errStr := ui.ErrorWriter.String()
	if !strings.Contains(errStr, "remote backend") {
		t.Fatalf("bad: %s", errStr)
	}
}

func TestPush_plan(t *testing.T) {
	tmp, cwd := testCwd(t)
	defer testFixCwd(t, tmp, cwd)

	// Create remote state file, this should be pulled
	conf, srv := testRemoteState(t, testState(), 200)
	defer srv.Close()

	// Persist local remote state
	s := terraform.NewState()
	s.Serial = 5
	s.Remote = conf
	testStateFileRemote(t, s)

	// Create a plan
	planPath := testPlanFile(t, &terraform.Plan{
		Module: testModule(t, "apply"),
	})

	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{planPath}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func testArchiveStr(t *testing.T, path string) []string {
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	// Ungzip
	gzipR, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Accumulator
	result := make([]string, 0, 10)

	// Untar
	tarR := tar.NewReader(gzipR)
	for {
		header, err := tarR.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		result = append(result, header.Name)
	}

	sort.Strings(result)
	return result
}

// we always quote map keys to be safe
func pushTFVars() []atlas.TFVar {
	return []atlas.TFVar{
		{Key: "bar", Value: "foo", IsHCL: false},
		{Key: "baz", Value: `{
  "A" = "a"
}`, IsHCL: true},
		{Key: "fob", Value: `["a", "quotes \"in\" quotes"]`, IsHCL: true},
		{Key: "foo", Value: "bar", IsHCL: false},
	}
}

// the structure returned from the push-tfvars test fixture
func pushTFVarsMap() map[string]atlas.TFVar {
	vars := make(map[string]atlas.TFVar)
	for _, v := range pushTFVars() {
		vars[v.Key] = v
	}
	return vars
}
