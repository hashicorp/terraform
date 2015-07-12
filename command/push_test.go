package command

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"reflect"
	"sort"
	"testing"

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

	variables := make(map[string]string)
	if !reflect.DeepEqual(client.UpsertOptions.Variables, variables) {
		t.Fatalf("bad: %#v", client.UpsertOptions)
	}

	if client.UpsertOptions.Name != "foo" {
		t.Fatalf("bad: %#v", client.UpsertOptions)
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

	variables := map[string]string{
		"foo": "foo",
	}
	if !reflect.DeepEqual(client.UpsertOptions.Variables, variables) {
		t.Fatalf("bad: %#v", client.UpsertOptions.Variables)
	}
}

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
		File:      archivePath,
		GetResult: map[string]string{"foo": "bar"},
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

	variables := map[string]string{
		"foo": "bar",
		"bar": "foo",
	}
	if !reflect.DeepEqual(client.UpsertOptions.Variables, variables) {
		t.Fatalf("bad: %#v", client.UpsertOptions)
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
	client.GetResult = map[string]string{
		"foo": "old",
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

	variables := map[string]string{
		"foo": "bar",
		"bar": "foo",
	}
	if !reflect.DeepEqual(client.UpsertOptions.Variables, variables) {
		t.Fatalf("bad: %#v", client.UpsertOptions)
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
	client.GetResult = map[string]string{
		"foo": "old",
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

	variables := map[string]string{
		"foo": "old",
		"bar": "foo",
	}
	if !reflect.DeepEqual(client.UpsertOptions.Variables, variables) {
		t.Fatalf("bad: %#v", client.UpsertOptions)
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

	variables := map[string]string{
		"foo": "bar",
		"bar": "foo",
	}
	if !reflect.DeepEqual(client.UpsertOptions.Variables, variables) {
		t.Fatalf("bad: %#v", client.UpsertOptions)
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

	ui := new(cli.MockUi)
	c := &PushCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		"-state", statePath,
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
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
